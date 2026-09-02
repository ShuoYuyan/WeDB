// Package sql implements a small SQL-92 subset parser tailored to the
// WeDB ODBC driver. The goal is to translate the limited surface that
// ODBC clients emit (SELECT/INSERT/UPDATE/DELETE/CREATE TABLE/DROP
// TABLE) into calls on the WeDB storage API. It is *not* a general
// SQL engine: aggregates, JOINs, subqueries, UNION, and most DDL are
// not supported and produce explicit errors.
package sql

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// Statement is the parsed representation of a single SQL statement.
type Statement struct {
	Kind StmtKind

	// SELECT
	Columns   []string        // empty means "*"
	AllCols   bool
	From      string
	Where     string
	OrderBy   []OrderItem
	Limit     int
	Offset    int
	Distinct  bool
	Aggregates []AggExpr // count/min/max/sum/avg

	// INSERT
	InsertCols []string
	InsertVals [][]interface{}

	// UPDATE
	UpdateTable string
	UpdateSet   map[string]interface{}
	UpdateWhere string

	// DELETE
	DeleteFrom  string
	DeleteWhere string

	// CREATE TABLE
	CreateTable string
	CreateCols  []ColumnDef
	IfNotExists bool

	// DROP TABLE / DROP INDEX
	DropTarget string
	IfExists   bool
	IsIndex    bool

	// CREATE INDEX
	CreateIndex      string
	IndexTable       string
	IndexCols        []string
	IndexUnique      bool

	// PRAGMA / SET
	IsPragma bool
	Pragma   string

	// Misc
	Raw     string
}

// StmtKind is the top-level statement classifier.
type StmtKind int

const (
	StmtUnknown StmtKind = iota
	StmtSelect
	StmtInsert
	StmtUpdate
	StmtDelete
	StmtCreateTable
	StmtDropTable
	StmtCreateIndex
	StmtDropIndex
	StmtBegin
	StmtCommit
	StmtRollback
	StmtPragma
)

// OrderItem is one ORDER BY column with direction.
type OrderItem struct {
	Column string
	Desc   bool
}

// AggExpr is one aggregate call, e.g. COUNT(*) or SUM(amount).
type AggExpr struct {
	Func string // COUNT, SUM, AVG, MIN, MAX
	Arg  string // column or "*"
}

// ColumnDef is one column in CREATE TABLE.
type ColumnDef struct {
	Name       string
	Type       string
	NotNull    bool
	PrimaryKey bool
	AutoInc    bool
	Unique     bool
}

// Parse parses a SQL string and returns a Statement.
func Parse(input string) (*Statement, error) {
	p := &parser{src: input, pos: 0}
	p.skipWS()
	if p.eof() {
		return nil, fmt.Errorf("empty statement")
	}
	stmt := &Statement{Raw: input}
	if err := p.parseStmt(stmt); err != nil {
		return nil, err
	}
	return stmt, nil
}

// MustParse is for tests.
func MustParse(s string) *Statement {
	st, err := Parse(s)
	if err != nil {
		panic(err)
	}
	return st
}

type parser struct {
	src string
	pos int
}

func (p *parser) eof() bool { return p.pos >= len(p.src) }

func (p *parser) peek() byte {
	if p.eof() {
		return 0
	}
	return p.src[p.pos]
}

func (p *parser) peekAt(off int) byte {
	if p.pos+off >= len(p.src) {
		return 0
	}
	return p.src[p.pos+off]
}

func (p *parser) next() byte {
	b := p.peek()
	if !p.eof() {
		p.pos++
	}
	return b
}

func (p *parser) skipWS() {
	for !p.eof() {
		c := p.peek()
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			p.pos++
		} else {
			break
		}
	}
}

func (p *parser) matchWord(w string) bool {
	p.skipWS()
	if p.pos+len(w) > len(p.src) {
		return false
	}
	if !strings.EqualFold(p.src[p.pos:p.pos+len(w)], w) {
		return false
	}
	// word boundary
	if p.pos+len(w) < len(p.src) {
		next := p.src[p.pos+len(w)]
		if isIdentChar(rune(next)) {
			return false
		}
	}
	p.pos += len(w)
	return true
}

func (p *parser) expectWord(w string) error {
	if !p.matchWord(w) {
		return fmt.Errorf("expected %q at position %d", w, p.pos)
	}
	return nil
}

func (p *parser) expect(ch byte) error {
	p.skipWS()
	if p.peek() != ch {
		return fmt.Errorf("expected %q at position %d, got %q", ch, p.pos, p.peek())
	}
	p.pos++
	return nil
}

func (p *parser) parseIdent() (string, error) {
	p.skipWS()
	if p.eof() {
		return "", fmt.Errorf("expected identifier at position %d", p.pos)
	}
	start := p.pos
	// optional quote
	if p.peek() == '"' || p.peek() == '`' {
		q := p.next()
		var b strings.Builder
		for !p.eof() {
			c := p.next()
			if c == q {
				return b.String(), nil
			}
			b.WriteByte(c)
		}
		return "", fmt.Errorf("unterminated quoted identifier")
	}
	if !isIdentStart(rune(p.peek())) {
		return "", fmt.Errorf("invalid identifier at position %d", p.pos)
	}
	for !p.eof() && isIdentChar(rune(p.peek())) {
		p.pos++
	}
	return p.src[start:p.pos], nil
}

func isIdentStart(r rune) bool {
	return unicode.IsLetter(r) || r == '_'
}
func isIdentChar(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

func (p *parser) parseStmt(stmt *Statement) error {
	// top-level dispatcher
	p.skipWS()
	if p.eof() {
		return fmt.Errorf("empty statement")
	}
	upper := strings.ToUpper(p.src[p.pos:])
	switch {
	case strings.HasPrefix(upper, "SELECT"):
		stmt.Kind = StmtSelect
		return p.parseSelect(stmt)
	case strings.HasPrefix(upper, "INSERT"):
		stmt.Kind = StmtInsert
		return p.parseInsert(stmt)
	case strings.HasPrefix(upper, "UPDATE"):
		stmt.Kind = StmtUpdate
		return p.parseUpdate(stmt)
	case strings.HasPrefix(upper, "DELETE"):
		stmt.Kind = StmtDelete
		return p.parseDelete(stmt)
	case strings.HasPrefix(upper, "CREATE TABLE"), strings.HasPrefix(upper, "CREATE INDEX"), strings.HasPrefix(upper, "CREATE UNIQUE"):
		return p.parseCreate(stmt)
	case strings.HasPrefix(upper, "DROP"):
		return p.parseDrop(stmt)
	case strings.HasPrefix(upper, "BEGIN"), strings.HasPrefix(upper, "START TRANSACTION"):
		stmt.Kind = StmtBegin
		p.pos += len("BEGIN")
		if strings.HasPrefix(upper, "START TRANSACTION") {
			p.pos += len("START TRANSACTION")
		}
		return nil
	case strings.HasPrefix(upper, "COMMIT"), strings.HasPrefix(upper, "END"):
		stmt.Kind = StmtCommit
		p.pos += len("COMMIT")
		if strings.HasPrefix(upper, "END") {
			p.pos += len("END")
		}
		return nil
	case strings.HasPrefix(upper, "ROLLBACK"):
		stmt.Kind = StmtRollback
		p.pos += len("ROLLBACK")
		return nil
	case strings.HasPrefix(upper, "PRAGMA") || strings.HasPrefix(upper, "SET"):
		stmt.Kind = StmtPragma
		stmt.IsPragma = true
		stmt.Pragma = p.src[p.pos:]
		p.pos = len(p.src)
		return nil
	default:
		return fmt.Errorf("unsupported SQL statement at position %d", p.pos)
	}
}

func (p *parser) parseSelect(stmt *Statement) error {
	if err := p.expectWord("SELECT"); err != nil {
		return err
	}
	p.skipWS()
	if p.matchWord("DISTINCT") {
		stmt.Distinct = true
	}
	p.skipWS()
	// column list
	if p.peek() == '*' {
		p.next()
		stmt.AllCols = true
		stmt.Columns = nil
	} else {
		for {
			col, agg, err := p.parseSelectCol()
			if err != nil {
				return err
			}
			if agg != nil {
				stmt.Aggregates = append(stmt.Aggregates, *agg)
				if col != "" && col != "*" {
					stmt.Columns = append(stmt.Columns, col)
				}
			} else if col == "*" {
				stmt.AllCols = true
			} else if col != "" {
				stmt.Columns = append(stmt.Columns, col)
			}
			p.skipWS()
			if p.peek() == ',' {
				p.next()
				continue
			}
			break
		}
	}
	// FROM
	if err := p.expectWord("FROM"); err != nil {
		return fmt.Errorf("SELECT requires FROM: %w", err)
	}
	t, err := p.parseIdent()
	if err != nil {
		return err
	}
	stmt.From = t
	// optional WHERE / ORDER BY / LIMIT
	if err := p.parseTailClauses(stmt); err != nil {
		return err
	}
	return nil
}

func (p *parser) parseSelectCol() (string, *AggExpr, error) {
	p.skipWS()
	// aggregate?
	upper := strings.ToUpper(p.src[p.pos:])
	for _, fn := range []string{"COUNT", "SUM", "AVG", "MIN", "MAX"} {
		if strings.HasPrefix(upper, fn+"(") {
			p.pos += len(fn)
			if err := p.expect('('); err != nil {
				return "", nil, err
			}
			p.skipWS()
			arg := "*"
			if p.peek() != '*' {
				a, err := p.parseIdent()
				if err != nil {
					return "", nil, err
				}
				arg = a
			} else {
				p.next()
			}
			if err := p.expect(')'); err != nil {
				return "", nil, err
			}
			return "", &AggExpr{Func: fn, Arg: arg}, nil
		}
	}
	// plain ident
	if p.peek() == '*' {
		p.next()
		return "*", nil, nil
	}
	ident, err := p.parseIdent()
	if err != nil {
		return "", nil, err
	}
	// optional alias
	p.skipWS()
	if p.matchWord("AS") {
		p.skipWS()
		alias, err := p.parseIdent()
		if err != nil {
			return "", nil, err
		}
		return alias, nil, nil
	}
	return ident, nil, nil
}

func (p *parser) parseTailClauses(stmt *Statement) error {
	for {
		p.skipWS()
		upper := strings.ToUpper(p.src[p.pos:])
		switch {
		case strings.HasPrefix(upper, "WHERE "):
			p.pos += len("WHERE")
			p.skipWS()
			stmt.Where = strings.TrimSpace(p.remainingUntilKeyword("ORDER", "LIMIT", "OFFSET", ";"))
		case strings.HasPrefix(upper, "ORDER BY "):
			p.pos += len("ORDER BY")
			p.skipWS()
			for {
				col, err := p.parseIdent()
				if err != nil {
					return err
				}
				oi := OrderItem{Column: col}
				p.skipWS()
				if p.matchWord("ASC") {
				} else if p.matchWord("DESC") {
					oi.Desc = true
				}
				stmt.OrderBy = append(stmt.OrderBy, oi)
				p.skipWS()
				if p.peek() == ',' {
					p.next()
					continue
				}
				break
			}
		case strings.HasPrefix(upper, "LIMIT "):
			p.pos += len("LIMIT")
			p.skipWS()
			n, err := p.parseInteger()
			if err != nil {
				return err
			}
			stmt.Limit = n
			p.skipWS()
			if p.matchWord("OFFSET") {
				p.skipWS()
				off, err := p.parseInteger()
				if err != nil {
					return err
				}
				stmt.Offset = off
			}
		case strings.HasPrefix(upper, "OFFSET "):
			p.pos += len("OFFSET")
			p.skipWS()
			off, err := p.parseInteger()
			if err != nil {
				return err
			}
			stmt.Offset = off
		case p.eof(), strings.HasPrefix(upper, ";"):
			return nil
		default:
			// unknown tail: skip to terminator
			for !p.eof() && p.peek() != ';' {
				p.pos++
			}
			if !p.eof() {
				p.pos++
			}
			return nil
		}
	}
}

func (p *parser) remainingUntilKeyword(words ...string) string {
	p.skipWS()
	start := p.pos
	upper := strings.ToUpper(p.src)
	for !p.eof() {
		if p.peek() == ';' {
			s := strings.TrimSpace(p.src[start:p.pos])
			p.pos++
			return s
		}
		// check if any keyword starts at current position
		match := false
		for _, w := range words {
			if p.pos+len(w) <= len(upper) && upper[p.pos:p.pos+len(w)] == w {
				// word boundary
				if p.pos+len(w) == len(upper) || !isIdentChar(rune(p.src[p.pos+len(w)])) {
					match = true
					break
				}
			}
		}
		if match {
			return strings.TrimSpace(p.src[start:p.pos])
		}
		p.pos++
	}
	return strings.TrimSpace(p.src[start:p.pos])
}

func (p *parser) parseInteger() (int, error) {
	p.skipWS()
	start := p.pos
	if p.eof() || !unicode.IsDigit(rune(p.peek())) {
		return 0, fmt.Errorf("expected integer at %d", p.pos)
	}
	for !p.eof() && unicode.IsDigit(rune(p.peek())) {
		p.pos++
	}
	n, err := strconv.Atoi(p.src[start:p.pos])
	if err != nil {
		return 0, err
	}
	return n, nil
}

func (p *parser) parseInsert(stmt *Statement) error {
	if err := p.expectWord("INSERT"); err != nil {
		return err
	}
	if err := p.expectWord("INTO"); err != nil {
		return err
	}
	t, err := p.parseIdent()
	if err != nil {
		return err
	}
	stmt.From = t
	stmt.InsertCols = nil
	stmt.InsertVals = nil
	p.skipWS()
	if p.peek() == '(' {
		p.next()
		for {
			c, err := p.parseIdent()
			if err != nil {
				return err
			}
			stmt.InsertCols = append(stmt.InsertCols, c)
			p.skipWS()
			if p.peek() == ',' {
				p.next()
				continue
			}
			break
		}
		if err := p.expect(')'); err != nil {
			return err
		}
	}
	if err := p.expectWord("VALUES"); err != nil {
		return err
	}
	for {
		if err := p.expect('('); err != nil {
			return err
		}
		var row []interface{}
		for {
			v, err := p.parseValue()
			if err != nil {
				return err
			}
			row = append(row, v)
			p.skipWS()
			if p.peek() == ',' {
				p.next()
				continue
			}
			break
		}
		if err := p.expect(')'); err != nil {
			return err
		}
		stmt.InsertVals = append(stmt.InsertVals, row)
		p.skipWS()
		if p.matchWord(",") {
			p.skipWS()
			if p.peek() == '(' {
				continue
			}
		}
		break
	}
	return nil
}

func (p *parser) parseValue() (interface{}, error) {
	p.skipWS()
	if p.eof() {
		return nil, fmt.Errorf("expected value at %d", p.pos)
	}
	c := p.peek()
	if c == '\'' || c == '"' {
		p.next()
		var b strings.Builder
		for !p.eof() {
			ch := p.next()
			if ch == c {
				// doubled escape: ''
				if p.peek() == c {
					b.WriteByte(c)
					p.next()
					continue
				}
				return b.String(), nil
			}
			b.WriteByte(ch)
		}
		return nil, fmt.Errorf("unterminated string literal")
	}
	if c == '-' || unicode.IsDigit(rune(c)) {
		start := p.pos
		if c == '-' {
			p.pos++
		}
		for !p.eof() && (unicode.IsDigit(rune(p.peek())) || p.peek() == '.') {
			p.pos++
		}
		tok := p.src[start:p.pos]
		if strings.Contains(tok, ".") {
			return strconv.ParseFloat(tok, 64)
		}
		return strconv.ParseInt(tok, 10, 64)
	}
	// NULL / TRUE / FALSE / ident-as-default
	rest := strings.ToUpper(p.src[p.pos:])
	switch {
	case strings.HasPrefix(rest, "NULL"):
		p.pos += 4
		return nil, nil
	case strings.HasPrefix(rest, "TRUE"):
		p.pos += 4
		return true, nil
	case strings.HasPrefix(rest, "FALSE"):
		p.pos += 5
		return false, nil
	}
	// bare identifier: treat as default value
	ident, err := p.parseIdent()
	if err != nil {
		return nil, err
	}
	return ident, nil
}

func (p *parser) parseUpdate(stmt *Statement) error {
	if err := p.expectWord("UPDATE"); err != nil {
		return err
	}
	t, err := p.parseIdent()
	if err != nil {
		return err
	}
	stmt.UpdateTable = t
	if err := p.expectWord("SET"); err != nil {
		return err
	}
	stmt.UpdateSet = map[string]interface{}{}
	for {
		p.skipWS()
		col, err := p.parseIdent()
		if err != nil {
			return err
		}
		if err := p.expect('='); err != nil {
			return err
		}
		v, err := p.parseValue()
		if err != nil {
			return err
		}
		stmt.UpdateSet[col] = v
		p.skipWS()
		if p.peek() == ',' {
			p.next()
			continue
		}
		break
	}
	p.skipWS()
	if p.matchWord("WHERE") {
		p.skipWS()
		stmt.UpdateWhere = strings.TrimSpace(p.remainingUntilKeyword(";"))
	}
	return nil
}

func (p *parser) parseDelete(stmt *Statement) error {
	if err := p.expectWord("DELETE"); err != nil {
		return err
	}
	if err := p.expectWord("FROM"); err != nil {
		return err
	}
	t, err := p.parseIdent()
	if err != nil {
		return err
	}
	stmt.DeleteFrom = t
	p.skipWS()
	if p.matchWord("WHERE") {
		p.skipWS()
		stmt.DeleteWhere = strings.TrimSpace(p.remainingUntilKeyword(";"))
	}
	return nil
}

func (p *parser) parseCreate(stmt *Statement) error {
	if err := p.expectWord("CREATE"); err != nil {
		return err
	}
	p.skipWS()
	upper := strings.ToUpper(p.src[p.pos:])
	if strings.HasPrefix(upper, "TABLE") {
		stmt.Kind = StmtCreateTable
		return p.parseCreateTable(stmt)
	}
	if strings.HasPrefix(upper, "UNIQUE INDEX") || strings.HasPrefix(upper, "INDEX") {
		stmt.Kind = StmtCreateIndex
		return p.parseCreateIndex(stmt)
	}
	return fmt.Errorf("CREATE: only TABLE/INDEX supported")
}

func (p *parser) parseCreateTable(stmt *Statement) error {
	if err := p.expectWord("TABLE"); err != nil {
		return err
	}
	if p.matchWord("IF") {
		if err := p.expectWord("NOT"); err != nil {
			return err
		}
		if err := p.expectWord("EXISTS"); err != nil {
			return err
		}
		stmt.IfNotExists = true
	}
	t, err := p.parseIdent()
	if err != nil {
		return err
	}
	stmt.CreateTable = t
	if err := p.expect('('); err != nil {
		return err
	}
	for {
		col, err := p.parseColumnDef()
		if err != nil {
			return err
		}
		stmt.CreateCols = append(stmt.CreateCols, col)
		p.skipWS()
		if p.peek() == ',' {
			p.next()
			continue
		}
		break
	}
	if err := p.expect(')'); err != nil {
		return err
	}
	return nil
}

func (p *parser) parseColumnDef() (ColumnDef, error) {
	p.skipWS()
	name, err := p.parseIdent()
	if err != nil {
		return ColumnDef{}, err
	}
	col := ColumnDef{Name: name}
	typ, err := p.parseTypeName()
	if err != nil {
		return col, err
	}
	col.Type = typ
	p.skipWS()
	// column constraints
	for {
		upper := strings.ToUpper(p.src[p.pos:])
		switch {
		case strings.HasPrefix(upper, "PRIMARY KEY"):
			p.pos += len("PRIMARY KEY")
			col.PrimaryKey = true
		case strings.HasPrefix(upper, "NOT NULL"):
			p.pos += len("NOT NULL")
			col.NotNull = true
		case strings.HasPrefix(upper, "UNIQUE"):
			p.pos += len("UNIQUE")
			col.Unique = true
		case strings.HasPrefix(upper, "AUTOINCREMENT") || strings.HasPrefix(upper, "AUTO_INCREMENT"):
			// consume longest match
			if strings.HasPrefix(upper, "AUTOINCREMENT") {
				p.pos += len("AUTOINCREMENT")
			} else {
				p.pos += len("AUTO_INCREMENT")
			}
			col.AutoInc = true
		default:
			return col, nil
		}
		p.skipWS()
	}
}

func (p *parser) parseTypeName() (string, error) {
	p.skipWS()
	start := p.pos
	if p.eof() {
		return "", fmt.Errorf("expected type")
	}
	if !isIdentStart(rune(p.peek())) {
		return "", fmt.Errorf("expected type at %d", p.pos)
	}
	for !p.eof() && isIdentChar(rune(p.peek())) {
		p.pos++
	}
	if !p.eof() && p.peek() == '(' {
		depth := 1
		p.pos++
		for !p.eof() && depth > 0 {
			if p.peek() == '(' {
				depth++
			} else if p.peek() == ')' {
				depth--
			}
			p.pos++
		}
	}
	raw := p.src[start:p.pos]
	up := strings.ToUpper(raw)
	switch {
	case strings.HasPrefix(up, "INT"):
		return "INTEGER", nil
	case strings.HasPrefix(up, "CHAR") || strings.HasPrefix(up, "VARCHAR") || strings.HasPrefix(up, "TEXT") || strings.HasPrefix(up, "CLOB"):
		return "TEXT", nil
	case strings.HasPrefix(up, "DOUBLE") || strings.HasPrefix(up, "REAL") || strings.HasPrefix(up, "FLOAT"):
		return "REAL", nil
	case strings.HasPrefix(up, "BLOB"):
		return "BLOB", nil
	case strings.HasPrefix(up, "BOOL"):
		return "INTEGER", nil
	case strings.HasPrefix(up, "DATETIME") || strings.HasPrefix(up, "TIMESTAMP"):
		return "TEXT", nil
	}
	if raw == "" {
		return "", fmt.Errorf("expected type at %d", start)
	}
	return up, nil
}

func (p *parser) parseCreateIndex(stmt *Statement) error {
	stmt.IsIndex = true
	if p.matchWord("UNIQUE") {
		stmt.IndexUnique = true
	}
	if err := p.expectWord("INDEX"); err != nil {
		return err
	}
	if p.matchWord("IF") {
		// we don't fully support IF NOT EXISTS for index
		for !p.eof() && p.peek() != ' ' {
			p.pos++
		}
	}
	name, err := p.parseIdent()
	if err != nil {
		return err
	}
	stmt.CreateIndex = name
	if err := p.expectWord("ON"); err != nil {
		return err
	}
	t, err := p.parseIdent()
	if err != nil {
		return err
	}
	stmt.IndexTable = t
	if err := p.expect('('); err != nil {
		return err
	}
	for {
		c, err := p.parseIdent()
		if err != nil {
			return err
		}
		stmt.IndexCols = append(stmt.IndexCols, c)
		p.skipWS()
		if p.peek() == ',' {
			p.next()
			continue
		}
		break
	}
	if err := p.expect(')'); err != nil {
		return err
	}
	return nil
}

func (p *parser) parseDrop(stmt *Statement) error {
	if err := p.expectWord("DROP"); err != nil {
		return err
	}
	p.skipWS()
	upper := strings.ToUpper(p.src[p.pos:])
	if strings.HasPrefix(upper, "TABLE") {
		if err := p.expectWord("TABLE"); err != nil {
			return err
		}
		stmt.Kind = StmtDropTable
	} else if strings.HasPrefix(upper, "INDEX") {
		if err := p.expectWord("INDEX"); err != nil {
			return err
		}
		stmt.Kind = StmtDropIndex
		stmt.IsIndex = true
	} else {
		return fmt.Errorf("DROP: only TABLE/INDEX supported")
	}
	if p.matchWord("IF") {
		if err := p.expectWord("EXISTS"); err != nil {
			return err
		}
		stmt.IfExists = true
	}
	t, err := p.parseIdent()
	if err != nil {
		return err
	}
	stmt.DropTarget = t
	return nil
}
