package sql

import (
	"fmt"
	"sort"
	"strings"
)

// ResultSet is the in-memory result of executing a SELECT.
type ResultSet struct {
	Columns []ColMeta
	Rows    [][]interface{}
	// Cursor position used by SQLFetch. -1 means before first row.
	Cursor int
	RowsAffected int64
}

// ColMeta describes one output column.
type ColMeta struct {
	Name     string
	DataType int16 // SQL type code (we map from WeDB type)
	Nullable bool
	Size     int64
}

// Executor runs a Statement against a Database (handle.Database) and
// returns either a *ResultSet (for SELECT) or rows-affected count.
type Executor struct {
	DB Database
}

// NewExecutor returns an Executor bound to db.
func NewExecutor(db Database) *Executor { return &Executor{DB: db} }

// Database is the minimal interface an Executor needs. Mirrors
// handle.Database; duplicated here to avoid the import cycle.
type Database interface {
	ListTables() []string
	TableExists(name string) bool
	GetTableSchema(name string) (interface{}, error)
	CreateTable(schema interface{}) error
	DropTable(name string) error
	ScanTable(name string) ([]map[string]interface{}, error)
	ScanTableWithColumns(name string, cols []string) ([]map[string]interface{}, error)
	InsertRow(name string, row map[string]interface{}) error
	UpdateRow(name string, row map[string]interface{}, where string) error
	DeleteRow(name string, where string) error
	Count(name, where string) (int64, error)
	Min(name, col, where string) (interface{}, error)
	Max(name, col, where string) (interface{}, error)
	Sum(name, col, where string) (float64, error)
	Avg(name, col, where string) (float64, error)
}

// Execute runs a parsed statement. Returns (result, rowsAffected, err).
// result is nil for non-SELECT statements.
func (e *Executor) Execute(s *Statement) (*ResultSet, int64, error) {
	switch s.Kind {
	case StmtSelect:
		rs, err := e.runSelect(s)
		if rs == nil {
			// runSelect returned a nil result with an error (e.g.
			// unbound parameter placeholder). Return an empty result
			// so callers can rely on a non-nil ResultSet.
			rs = &ResultSet{Cursor: -1}
		}
		return rs, int64(len(rs.Rows)), err
	case StmtInsert:
		n, err := e.runInsert(s)
		return nil, n, err
	case StmtUpdate:
		n, err := e.runUpdate(s)
		return nil, n, err
	case StmtDelete:
		n, err := e.runDelete(s)
		return nil, n, err
	case StmtCreateTable:
		err := e.runCreateTable(s)
		return nil, 0, err
	case StmtDropTable:
		err := e.runDropTable(s)
		return nil, 0, err
	case StmtCreateIndex:
		err := e.runCreateIndex(s)
		return nil, 0, err
	case StmtDropIndex:
		err := e.runDropIndex(s)
		return nil, 0, err
	case StmtBegin, StmtCommit, StmtRollback, StmtPragma:
		return nil, 0, fmt.Errorf("transaction/pragma not yet wired through the SQL executor")
	default:
		return nil, 0, fmt.Errorf("unsupported statement kind %v", s.Kind)
	}
}

func (e *Executor) runSelect(s *Statement) (*ResultSet, error) {
	if !e.DB.TableExists(s.From) {
		return nil, fmt.Errorf("table not found: %s", s.From)
	}
	// 1) collect source rows
	var rows []map[string]interface{}
	var err error
	if len(s.Columns) > 0 && !s.AllCols {
		rows, err = e.DB.ScanTableWithColumns(s.From, s.Columns)
	} else {
		rows, err = e.DB.ScanTable(s.From)
	}
	if err != nil {
		return nil, err
	}
	// 2) WHERE
	if s.Where != "" {
		filtered := rows[:0]
		for _, r := range rows {
			match, err := evalWhere(s.Where, r)
			if err != nil {
				return nil, err
			}
			if match {
				filtered = append(filtered, r)
			}
		}
		rows = filtered
	}
	// 3) aggregates
	if len(s.Aggregates) > 0 {
		return e.runAggregates(s, rows)
	}
	// 4) ORDER BY
	if len(s.OrderBy) > 0 {
		sortRows(rows, s.OrderBy)
	}
	// 5) LIMIT / OFFSET
	if s.Offset > 0 && s.Offset < len(rows) {
		rows = rows[s.Offset:]
	} else if s.Offset >= len(rows) {
		rows = nil
	}
	if s.Limit > 0 && len(rows) > s.Limit {
		rows = rows[:s.Limit]
	}
	// 6) build result
	rs := &ResultSet{Cursor: -1}
	colNames := unionColumns(s.Columns, rows)
	rs.Columns = make([]ColMeta, 0, len(colNames))
	for _, n := range colNames {
		rs.Columns = append(rs.Columns, ColMeta{
			Name:     n,
			DataType: inferColumnType(rows, n),
			Nullable: true,
		})
	}
	rs.Rows = make([][]interface{}, 0, len(rows))
	for _, r := range rows {
		out := make([]interface{}, len(colNames))
		for i, n := range colNames {
			out[i] = r[n]
		}
		rs.Rows = append(rs.Rows, out)
	}
	return rs, nil
}

// inferColumnType picks an SQL type code based on the actual values
// observed in a result set column. We scan the first non-nil value
// to decide between INTEGER, REAL, and VARCHAR.
func inferColumnType(rows []map[string]interface{}, colName string) int16 {
	for _, r := range rows {
		v, ok := r[colName]
		if !ok || v == nil {
			continue
		}
		switch v.(type) {
		case int, int32, int64, uint, uint32, uint64, bool:
			return 4 // SQL_INTEGER
		case float32, float64:
			return 8 // SQL_DOUBLE
		case []byte:
			return -4 // SQL_LONGVARBINARY
		}
		return 12 // SQL_VARCHAR
	}
	return 12
}

func unionColumns(cols []string, rows []map[string]interface{}) []string {
	if len(cols) > 0 {
		return cols
	}
	seen := map[string]struct{}{}
	out := make([]string, 0)
	for _, r := range rows {
		for k := range r {
			if _, ok := seen[k]; ok {
				continue
			}
			seen[k] = struct{}{}
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

func (e *Executor) runAggregates(s *Statement, rows []map[string]interface{}) (*ResultSet, error) {
	rs := &ResultSet{Cursor: -1}
	for _, a := range s.Aggregates {
		col := ColMeta{Name: a.Func + "(" + a.Arg + ")", DataType: 0, Nullable: true}
		rs.Columns = append(rs.Columns, col)
	}
	row := make([]interface{}, 0, len(s.Aggregates))
	for _, a := range s.Aggregates {
		v, err := aggregate(e.DB, s.From, a, s.Where)
		if err != nil {
			return nil, err
		}
		row = append(row, v)
	}
	rs.Rows = [][]interface{}{row}
	return rs, nil
}

func aggregate(db Database, table string, a AggExpr, where string) (interface{}, error) {
	switch a.Func {
	case "COUNT":
		if a.Arg == "*" {
			return db.Count(table, where)
		}
		return db.Count(table, where) // COUNT(col) approximated as COUNT(*)
	case "SUM":
		return db.Sum(table, a.Arg, where)
	case "AVG":
		return db.Avg(table, a.Arg, where)
	case "MIN":
		return db.Min(table, a.Arg, where)
	case "MAX":
		return db.Max(table, a.Arg, where)
	}
	return nil, fmt.Errorf("unsupported aggregate: %s", a.Func)
}

func (e *Executor) runInsert(s *Statement) (int64, error) {
	if !e.DB.TableExists(s.From) && s.Kind != StmtInsert {
		// insert: From is empty; we read CreateTable instead
	}
	// We use UpdateTable=="" for inserts, table is the FROM table? No: for
	// INSERT INTO t ... we stored the table into InsertCols; but we need
	// a field. Use s.From? We didn't fill it. Set it in parseInsert.
	if s.From == "" {
		return 0, fmt.Errorf("internal: insert target missing")
	}
	cols := s.InsertCols
	var n int64
	for _, vals := range s.InsertVals {
		row := map[string]interface{}{}
		if len(cols) == 0 {
			// positional values: schema lookup happens inside InsertRow;
			// build a row with ordinal keys "0","1",...
			for i, v := range vals {
				row[fmt.Sprintf("%d", i)] = v
			}
		} else {
			if len(vals) != len(cols) {
				return n, fmt.Errorf("column/value count mismatch")
			}
			for i, c := range cols {
				row[c] = vals[i]
			}
		}
		if err := e.DB.InsertRow(s.From, row); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}

func (e *Executor) runUpdate(s *Statement) (int64, error) {
	if err := e.DB.UpdateRow(s.UpdateTable, s.UpdateSet, s.UpdateWhere); err != nil {
		return 0, err
	}
	// The engine's UpdateRow doesn't return a count; assume one row
	// affected on success. Callers needing an exact count should issue
	// SELECT COUNT(*) first.
	return 1, nil
}

func (e *Executor) runDelete(s *Statement) (int64, error) {
	if err := e.DB.DeleteRow(s.DeleteFrom, s.DeleteWhere); err != nil {
		return 0, err
	}
	return 1, nil
}

func (e *Executor) runCreateTable(s *Statement) error {
	if s.IfNotExists && e.DB.TableExists(s.CreateTable) {
		return nil
	}
	schema := buildTableSchema(s)
	return e.DB.CreateTable(schema)
}

func (e *Executor) runDropTable(s *Statement) error {
	if s.IfExists && !e.DB.TableExists(s.DropTarget) {
		return nil
	}
	return e.DB.DropTable(s.DropTarget)
}

func (e *Executor) runCreateIndex(s *Statement) error {
	// CREATE INDEX is implemented via api.IndexInfo; we don't have a
	// direct method on the Database interface for non-row callers, so
	// route through CreateIndex. This is a no-op for the minimal driver
	// surface; full wiring lives in pkg/adapter.
	return fmt.Errorf("CREATE INDEX not exposed in this driver build")
}

func (e *Executor) runDropIndex(s *Statement) error {
	return fmt.Errorf("DROP INDEX not exposed in this driver build")
}

func sortRows(rows []map[string]interface{}, order []OrderItem) {
	// We don't know column types at compile time; sort by string
	// representation with numeric awareness via fmt.
	sort.SliceStable(rows, func(i, j int) bool {
		for _, o := range order {
			a, b := rows[i][o.Column], rows[j][o.Column]
			c := compareValues(a, b)
			if c == 0 {
				continue
			}
			if o.Desc {
				return c > 0
			}
			return c < 0
		}
		return false
	})
}

func compareValues(a, b interface{}) int {
	if a == nil && b == nil {
		return 0
	}
	if a == nil {
		return -1
	}
	if b == nil {
		return 1
	}
	switch va := a.(type) {
	case int:
		if vb, ok := b.(int); ok {
			switch {
			case va < vb:
				return -1
			case va > vb:
				return 1
			}
			return 0
		}
	case int64:
		if vb, ok := b.(int64); ok {
			switch {
			case va < vb:
				return -1
			case va > vb:
				return 1
			}
			return 0
		}
	case float64:
		if vb, ok := b.(float64); ok {
			switch {
			case va < vb:
				return -1
			case va > vb:
				return 1
			}
			return 0
		}
	}
	as, bs := fmt.Sprintf("%v", a), fmt.Sprintf("%v", b)
	return strings.Compare(as, bs)
}

// buildTableSchema converts a CREATE TABLE into a map the storage
// adapter can interpret. We avoid importing internal/api here to keep
// the driver module light; the storage layer accepts a flexible shape.
func buildTableSchema(s *Statement) map[string]interface{} {
	cols := make([]map[string]interface{}, 0, len(s.CreateCols))
	pk := ""
	for _, c := range s.CreateCols {
		entry := map[string]interface{}{
			"name":          c.Name,
			"type":          normalizeType(c.Type),
			"nullable":      !c.NotNull && !c.PrimaryKey,
			"primary_key":   c.PrimaryKey,
			"auto_increment": c.AutoInc,
			"unique":        c.Unique,
		}
		cols = append(cols, entry)
		if c.PrimaryKey {
			pk = c.Name
		}
	}
	return map[string]interface{}{
		"table_name":    s.CreateTable,
		"columns":       cols,
		"primary_key":   pk,
		"auto_increment": false,
	}
}

func normalizeType(t string) string {
	switch strings.ToUpper(t) {
	case "INTEGER", "INT", "BIGINT", "SMALLINT", "TINYINT":
		return "INTEGER"
	case "REAL", "FLOAT", "DOUBLE":
		return "REAL"
	case "TEXT", "VARCHAR", "CHAR", "CLOB":
		return "TEXT"
	case "BLOB":
		return "BLOB"
	}
	return "TEXT"
}

// evalWhere is a thin adapter to the engine's WHERE parser.
func evalWhere(where string, row map[string]interface{}) (bool, error) {
	// Reuse internal/storage parsing by routing through a compile-time
	// import would create a cycle, so we re-implement the tiny subset
	// we need: column OP literal [AND/OR ...].
	// The full WHERE evaluator lives in internal/storage/where.go.
	// For the ODBC driver we accept a simple comparator: parse the
	// first comparison and short-circuit.
	expr, err := parseWhereExpr(where)
	if err != nil {
		return false, err
	}
	return expr.Eval(row), nil
}
