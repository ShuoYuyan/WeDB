// CLI 辅助函数：为命令行工具提供简单的字符串接口。
// 这些函数不涉及 SQL，只是字符串解析和表格输出。
package wqlv3

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// QueryResult 统一的查询结果封装
type QueryResult struct {
	Rows         []map[string]interface{} // 查询行
	Value        interface{}              // 聚合值（Count/Sum/Avg/Min/Max）
	Columns      []string                 // 显式选择的列
	Duration     time.Duration             // 执行时间
	Statement    string                   // 原始语句（用于显示）
	AffectedRows int64                    // INSERT/UPDATE/DELETE 影响行数（-1 表示未知）
}

// ListTables 列出所有表
func ListTables(db *Database) ([]string, error) {
	if db == nil || db.adapter == nil {
		return nil, fmt.Errorf("database not opened")
	}
	return db.adapter.ListTables(), nil
}

// GetSchema 获取表结构
func GetSchema(db *Database, tableName string) ([]ColumnDef, error) {
	if db == nil || db.adapter == nil {
		return nil, fmt.Errorf("database not opened")
	}
	// 使用 ScanTable 获取一行的列信息
	rows, err := db.adapter.ScanTable(tableName)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		// 表为空，仍然获取表结构
		// 但 ScanTable 对空表返回空数组，所以从 API 获取
		return nil, nil
	}
	// 从第一行推断列
	var cols []ColumnDef
	seen := map[string]bool{}
	for _, row := range rows {
		for k, v := range row {
			if !seen[k] {
				seen[k] = true
				cols = append(cols, ColumnDef{
					Name:     k,
					Type:     inferType(v),
					Nullable: true,
				})
			}
		}
	}
	_ = tableName // suppress unused warning
	return cols, nil
}

func inferType(v interface{}) string {
	switch v.(type) {
	case int, int32, int64, uint, uint32, uint64:
		return "INTEGER"
	case float32, float64:
		return "REAL"
	case string:
		return "TEXT"
	case []byte:
		return "BLOB"
	case bool:
		return "BOOLEAN"
	case nil:
		return "NULL"
	}
	return "TEXT"
}

// PrintResult 以表格形式打印结果
func PrintResult(r QueryResult) {
	// DML 操作结果
	if r.AffectedRows > 0 || r.AffectedRows == 0 && (len(r.Rows) == 0 && r.Value == nil) {
		if r.Value != nil {
			fmt.Printf("  %v\n", formatValueForPrint(r.Value))
		}
		return
	}
	if r.Value != nil {
		// 单值结果（聚合）
		fmt.Printf("  %v\n", formatValueForPrint(r.Value))
		return
	}
	if len(r.Rows) == 0 {
		fmt.Println("  (no rows)")
		return
	}
	// 表格结果
	cols := r.Columns
	if len(cols) == 0 {
		// 从行推断
		seen := map[string]bool{}
		for _, row := range r.Rows {
			for k := range row {
				if !seen[k] {
					seen[k] = true
					cols = append(cols, k)
				}
			}
		}
	}

	// 计算列宽
	widths := make(map[string]int)
	for _, c := range cols {
		w := len(c)
		if w > 30 {
			w = 30
		}
		widths[c] = w
	}
	for _, row := range r.Rows {
		for _, c := range cols {
			s := formatValueForPrint(row[c])
			if len(s) > widths[c] {
				if len(s) > 30 {
					widths[c] = 30
				} else {
					widths[c] = len(s)
				}
			}
		}
	}

	// 限制总宽度
	totalWidth := 0
	for _, c := range cols {
		totalWidth += widths[c] + 2
	}
	if totalWidth > 120 {
		// 均匀缩小
		perCol := 120 / len(cols)
		if perCol < 8 {
			perCol = 8
		}
		for _, c := range cols {
			if widths[c] > perCol {
				widths[c] = perCol
			}
		}
	}

	// 打印头部
	for i, c := range cols {
		if i > 0 {
			fmt.Print("  ")
		}
		fmt.Printf("%-*s", widths[c], truncate(c, widths[c]))
	}
	fmt.Println()

	// 打印分隔
	for i, c := range cols {
		if i > 0 {
			fmt.Print("  ")
		}
		fmt.Print(strings.Repeat("-", widths[c]))
	}
	fmt.Println()

	// 打印行
	for _, row := range r.Rows {
		for i, c := range cols {
			if i > 0 {
				fmt.Print("  ")
			}
			v := formatValueForPrint(row[c])
			fmt.Printf("%-*s", widths[c], truncate(v, widths[c]))
		}
		fmt.Println()
	}
}

func caps(cols []string, i int) int {
	return len(cols) // unused, will be replaced
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 1 {
		return s[:max]
	}
	return s[:max-1] + "…"
}

// splitArgs 分割参数（处理括号嵌套和引号）
func splitArgs(s string) []string {
	var out []string
	depth := 0
	start := 0
	inStr := byte(0)
	for i := 0; i < len(s); i++ {
		c := s[i]
		if inStr != 0 {
			if c == inStr {
				inStr = 0
			}
			continue
		}
		switch c {
		case '"', '\'':
			inStr = c
		case '(':
			depth++
		case ')':
			depth--
		case ',':
			if depth == 0 {
				out = append(out, s[start:i])
				start = i + 1
			}
		}
	}
	if start < len(s) {
		out = append(out, s[start:])
	}
	return out
}

func formatValueForPrint(v interface{}) string {
	if v == nil {
		return "NULL"
	}
	switch x := v.(type) {
	case string:
		return x
	case []byte:
		return string(x)
	case float64:
		if x == float64(int64(x)) {
			return fmt.Sprintf("%d", int64(x))
		}
		return fmt.Sprintf("%g", x)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// EvaluateQuery 解析 WQL 方法链字符串并执行
// 这是 CLI 的核心入口。
// 接受的格式:
//   - SELECT: T("name").Where(cond).All() 或 Table("name").Where(cond).All()
//   - INSERT: Insert("name").Values({...}).Execute()
//   - UPDATE: Update("name").Set(col, val).Where(cond).Execute()
//   - DELETE: Delete("name").Where(cond).Execute()
//
// 由于完整的 WQL 解析需要复杂的语法分析（pkg/wql/lexer/parser），
// 这里采用简化的正则式解析。
func EvaluateQuery(db *Database, expr string) (QueryResult, error) {
	expr = strings.TrimSpace(expr)
	result := QueryResult{Statement: expr, Duration: 0}
	start := time.Now()

	upper := strings.ToUpper(expr)

	// INSERT 语句
	if strings.HasPrefix(upper, "INSERT(") {
		return evaluateInsert(db, expr, start, result)
	}
	// UPDATE 语句
	if strings.HasPrefix(upper, "UPDATE(") {
		return evaluateUpdate(db, expr, start, result)
	}
	// DELETE 语句
	if strings.HasPrefix(upper, "DELETE(") {
		return evaluateDelete(db, expr, start, result)
	}
	// CREATE TABLE 语句
	if strings.HasPrefix(upper, "CREATETABLE(") {
		return evaluateCreateTable(db, expr, start, result)
	}
	// DROP TABLE 语句
	if strings.HasPrefix(upper, "DROPTABLE(") {
		return evaluateDropTable(db, expr, start, result)
	}

	// 提取表名: T("name") 或 .Table("name") 或 Table("name")
	tableName, rest, err := parseTableRef(expr)
	if err != nil {
		return result, err
	}
	if tableName == "" {
		return result, fmt.Errorf("no table reference found (expected T(\"...\") or Table(\"...\"))")
	}

	// 解析方法链
	method, args := parseLastMethod(rest)

	// 提取 WHERE / ORDER BY / SKIP / TAKE
	where, orderCol, orderDir, skipN, takeN, selectCols := parseChainArgs(rest)

	qb := db.Table(tableName)
	if len(selectCols) > 0 {
		qb = qb.Select(selectCols...)
	}
	if where != "" {
		qb = qb.Where(where)
	}
	if orderCol != "" {
		qb = qb.OrderBy(orderCol, orderDir)
	}
	if skipN > 0 {
		qb = qb.Skip(skipN)
	}
	if takeN > 0 {
		qb = qb.Take(takeN)
	}
	result.Columns = selectCols

	// 执行对应方法
	switch method {
	case "All", "":
		rows, err := qb.All()
		if err != nil {
			return result, err
		}
		result.Rows = rows
	case "First":
		row, err := qb.First()
		if err != nil {
			return result, err
		}
		if row != nil {
			result.Rows = []map[string]interface{}{row}
		}
	case "Count":
		n, err := qb.Count()
		if err != nil {
			return result, err
		}
		result.Value = n
	case "Sum":
		if len(args) == 0 {
			return result, fmt.Errorf("Sum() requires a column name")
		}
		f, err := qb.Sum(args[0])
		if err != nil {
			return result, err
		}
		result.Value = f
	case "Avg":
		if len(args) == 0 {
			return result, fmt.Errorf("Avg() requires a column name")
		}
		f, err := qb.Avg(args[0])
		if err != nil {
			return result, err
		}
		result.Value = f
	case "Min":
		if len(args) == 0 {
			return result, fmt.Errorf("Min() requires a column name")
		}
		v, err := qb.Min(args[0])
		if err != nil {
			return result, err
		}
		result.Value = v
	case "Max":
		if len(args) == 0 {
			return result, fmt.Errorf("Max() requires a column name")
		}
		v, err := qb.Max(args[0])
		if err != nil {
			return result, err
		}
		result.Value = v
	default:
		return result, fmt.Errorf("unsupported method: %s", method)
	}

	result.Duration = time.Since(start)
	return result, nil
}

// parseTableRef 提取表名，返回 (表名, 剩余表达式, 错误)
func parseTableRef(expr string) (string, string, error) {
	// 查找 T( 或 Table(
	for _, prefix := range []string{"T(", "Table("} {
		idx := strings.Index(expr, prefix)
		if idx < 0 {
			continue
		}
		start := idx + len(prefix)
		// 找匹配的 )
		depth := 0
		end := -1
		for i := start; i < len(expr); i++ {
			switch expr[i] {
			case '(':
				depth++
			case ')':
				if depth == 0 {
					end = i
					break
				}
				depth--
			case '"', '\'':
				quote := expr[i]
				i++
				for i < len(expr) && expr[i] != quote {
					i++
				}
			}
			if end >= 0 {
				break
			}
		}
		if end < 0 {
			return "", "", fmt.Errorf("malformed T() expression")
		}
		name := strings.TrimSpace(expr[start:end])
		// 去掉引号
		name = strings.Trim(name, `"'`)
		rest := expr[end+1:]
		return name, rest, nil
	}
	return "", "", fmt.Errorf("no T(table) reference found")
}

// parseChainArgs 从方法链中提取 WHERE, ORDER BY, SKIP, TAKE, SELECT 参数
func parseChainArgs(rest string) (where, orderCol, orderDir string, skipN, takeN int64, selectCols []string) {
	upper := strings.ToUpper(rest)

	// SELECT
	if idx := strings.Index(upper, ".SELECT("); idx >= 0 {
		depth := 0
		end := -1
		for i := idx + 8; i < len(rest); i++ {
			switch rest[i] {
			case '(':
				depth++
			case ')':
				if depth == 0 {
					end = i
					break
				}
				depth--
			}
			if end >= 0 {
				break
			}
		}
		if end > 0 {
			args := rest[idx+8 : end]
			for _, a := range splitArgs(args) {
				a = strings.TrimSpace(a)
				a = strings.Trim(a, `"'`)
				if a != "" && a != "*" {
					selectCols = append(selectCols, a)
				}
			}
		}
	}

	// WHERE
	if idx := strings.Index(upper, ".WHERE("); idx >= 0 {
		depth := 0
		end := -1
		for i := idx + 7; i < len(rest); i++ {
			switch rest[i] {
			case '(':
				depth++
			case ')':
				if depth == 0 {
					end = i
					break
				}
				depth--
			}
			if end >= 0 {
				break
			}
		}
		if end > 0 {
			where = strings.TrimSpace(rest[idx+7 : end])
		}
	}

	// ORDER BY
	if idx := strings.Index(upper, ".ORDERBY("); idx >= 0 {
		depth := 0
		end := -1
		for i := idx + 9; i < len(rest); i++ {
			switch rest[i] {
			case '(':
				depth++
			case ')':
				if depth == 0 {
					end = i
					break
				}
				depth--
			}
			if end >= 0 {
				break
			}
		}
		if end > 0 {
			args := strings.SplitN(rest[idx+9:end], ",", 2)
			orderCol = strings.Trim(strings.TrimSpace(args[0]), `"'`)
			if len(args) >= 2 {
				orderDir = strings.TrimSpace(args[1])
				orderDir = strings.Trim(orderDir, `"'`)
			}
		}
	}

	// SKIP
	if idx := strings.Index(upper, ".SKIP("); idx >= 0 {
		end := strings.Index(rest[idx+6:], ")")
		if end > 0 {
			fmt.Sscanf(rest[idx+6:idx+6+end], "%d", &skipN)
		}
	}

	// TAKE
	if idx := strings.Index(upper, ".TAKE("); idx >= 0 {
		end := strings.Index(rest[idx+6:], ")")
		if end > 0 {
			fmt.Sscanf(rest[idx+6:idx+6+end], "%d", &takeN)
		}
	}

	return
}

// parseLastMethod 解析最后一个方法调用
func parseLastMethod(rest string) (string, []string) {
	// 找最后一个 .METHOD(...)
	idx := strings.LastIndex(rest, ".")
	if idx < 0 {
		return "", nil
	}
	rest = rest[idx+1:]
	if !strings.HasSuffix(rest, ")") {
		return "", nil
	}
	openIdx := strings.Index(rest, "(")
	if openIdx < 0 {
		return "", nil
	}
	method := rest[:openIdx]
	argsStr := rest[openIdx+1 : len(rest)-1]
	var args []string
	if argsStr != "" {
		for _, a := range splitArgs(argsStr) {
			a = strings.TrimSpace(a)
			a = strings.Trim(a, `"'`)
			if a != "" {
				args = append(args, a)
			}
		}
	}
	return method, args
}

// ===== DML 评估器（CLI 字符串接口） =====

// evaluateInsert 处理 INSERT("table").Values({...}).Execute()
// 简化语法:
//   Insert("users").Values(map[string]interface{}{"id": 1, "name": "alice"}).Execute()
//   Insert("users").Values({id: 1, name: "alice"}).Execute()
func evaluateInsert(db *Database, expr string, start time.Time, result QueryResult) (QueryResult, error) {
	// 提取表名: Insert("name")
	idx := strings.Index(expr, "Insert(")
	if idx < 0 {
		return result, fmt.Errorf("malformed INSERT")
	}
	tableName, rest, err := extractQuotedName(expr[idx+7:])
	if err != nil {
		return result, fmt.Errorf("malformed INSERT: %w", err)
	}

	// 提取 Values(...)
	row, err := extractOneRow(rest)
	if err != nil {
		return result, fmt.Errorf("failed to parse VALUES: %w", err)
	}

	n, err := db.Insert(tableName).Value(row).Execute()
	result.Duration = time.Since(start)
	if err != nil {
		return result, err
	}
	result.AffectedRows = n
	result.Rows = nil
	result.Value = fmt.Sprintf("INSERT OK (%d row(s))", n)
	return result, nil
}

// evaluateUpdate 处理 UPDATE("table").Set(col, val).Where(cond).Execute()
// 简化语法:
//   Update("users").Set("name", "bob").Where("id = 1").Execute()
//   Update("users").Sets({name: "bob", age: 30}).Where("id = 1").Execute()
func evaluateUpdate(db *Database, expr string, start time.Time, result QueryResult) (QueryResult, error) {
	idx := strings.Index(expr, "Update(")
	if idx < 0 {
		return result, fmt.Errorf("malformed UPDATE")
	}
	tableName, rest, err := extractQuotedName(expr[idx+7:])
	if err != nil {
		return result, fmt.Errorf("malformed UPDATE: %w", err)
	}

	ub := db.Update(tableName)

	// 解析 Set(col, val) 调用
	if err := parseSetCalls(rest, ub); err != nil {
		return result, fmt.Errorf("failed to parse SET: %w", err)
	}

	// 解析 .Where(cond)
	if w := extractWhere(rest); w != "" {
		ub.Where(w)
	}

	n, err := ub.Execute()
	result.Duration = time.Since(start)
	if err != nil {
		return result, err
	}
	result.AffectedRows = n
	result.Value = fmt.Sprintf("UPDATE OK")
	return result, nil
}

// evaluateDelete 处理 DELETE("table").Where(cond).Execute()
func evaluateDelete(db *Database, expr string, start time.Time, result QueryResult) (QueryResult, error) {
	idx := strings.Index(expr, "Delete(")
	if idx < 0 {
		return result, fmt.Errorf("malformed DELETE")
	}
	tableName, rest, err := extractQuotedName(expr[idx+7:])
	if err != nil {
		return result, fmt.Errorf("malformed DELETE: %w", err)
	}

	db_ := db.Delete(tableName)
	if w := extractWhere(rest); w != "" {
		db_.Where(w)
	}

	n, err := db_.Execute()
	result.Duration = time.Since(start)
	if err != nil {
		return result, err
	}
	result.AffectedRows = n
	result.Value = fmt.Sprintf("DELETE OK")
	return result, nil
}

// evaluateCreateTable 处理 CREATE TABLE
// 简化语法: CreateTable("users", {id: "INTEGER PK", name: "TEXT", age: "INTEGER"})
// 或:      CreateTable("users", ["id INTEGER PRIMARY KEY", "name TEXT", "age INTEGER"])
func evaluateCreateTable(db *Database, expr string, start time.Time, result QueryResult) (QueryResult, error) {
	// 跳过 "CreateTable(" 前缀
	rest := expr[12:]
	// 第一个参数: 表名（带引号）
	tableName, rest, err := extractQuotedName(rest)
	if err != nil {
		return result, fmt.Errorf("malformed CREATE TABLE: %w", err)
	}
	tableName = strings.TrimSpace(tableName)

	// 解析列定义（简化版：col1 TYPE, col2 TYPE, ...）
	// 找到第一个 ),然后去掉尾部的 )
	colDefs, err := parseColumnDefs(rest)
	if err != nil {
		return result, fmt.Errorf("failed to parse column definitions: %w", err)
	}

	schema := NewTableSchema(tableName, colDefs...)
	if err := db.CreateTable(schema); err != nil {
		return result, err
	}
	result.Duration = time.Since(start)
	result.Value = fmt.Sprintf("CREATE TABLE %s OK (%d columns)", tableName, len(colDefs))
	return result, nil
}

// evaluateDropTable 处理 DROP TABLE
func evaluateDropTable(db *Database, expr string, start time.Time, result QueryResult) (QueryResult, error) {
	rest := expr[10:]
	tableName, _, err := extractQuotedName(rest)
	if err != nil {
		return result, fmt.Errorf("malformed DROP TABLE: %w", err)
	}
	tableName = strings.TrimSpace(tableName)
	if err := db.DropTable(tableName); err != nil {
		return result, err
	}
	result.Duration = time.Since(start)
	result.Value = fmt.Sprintf("DROP TABLE %s OK", tableName)
	return result, nil
}

// ===== DML 解析辅助函数 =====

// extractQuotedName 提取括号内的带引号字符串
// 输入: `"users").Values(...)` 输出: `users`, `).Values(...)`
func extractQuotedName(s string) (string, string, error) {
	if len(s) == 0 {
		return "", "", fmt.Errorf("empty")
	}
	// 跳过前导空白
	i := 0
	for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	if i >= len(s) {
		return "", "", fmt.Errorf("empty after trim")
	}
	if s[i] != '"' && s[i] != '\'' {
		return "", "", fmt.Errorf("expected quoted string at position %d, got %q", i, s[i])
	}
	quote := s[i]
	end := strings.IndexByte(s[i+1:], quote)
	if end < 0 {
		return "", "", fmt.Errorf("unterminated string")
	}
	name := s[i+1 : i+1+end]
	rest := s[i+1+end+1:]
	return name, rest, nil
}

// extractOneRow 提取 Values(...) 中的单行数据
// 支持格式: Values({"key": value, "key2": value2})
//        或 Values({key: value, key2: value2})
func extractOneRow(rest string) (map[string]interface{}, error) {
	// 找 .Values(
	idx := strings.Index(strings.ToUpper(rest), ".VALUES(")
	if idx < 0 {
		return nil, fmt.Errorf("no .Values() found")
	}
	rest = rest[idx+8:]
	// 找匹配的 )
	depth := 0
	end := -1
	for i := 0; i < len(rest); i++ {
		switch rest[i] {
		case '{', '(':
			depth++
		case '}', ')':
			depth--
			if depth == 0 {
				end = i
				break
			}
		}
		if end >= 0 {
			break
		}
	}
	if end < 0 {
		return nil, fmt.Errorf("malformed .Values()")
	}
	content := rest[:end]
	return parseRowObject(content)
}

// parseRowObject 解析 {key: value, key2: value2}
// 简化实现：只支持字符串和数字值
func parseRowObject(s string) (map[string]interface{}, error) {
	out := make(map[string]interface{})
	// 去掉首尾的大括号
	s = strings.TrimSpace(s)
	if s == "" {
		return out, nil
	}
	if strings.HasPrefix(s, "{") {
		s = s[1:]
	}
	if strings.HasSuffix(s, "}") {
		s = s[:len(s)-1]
	}

	// 按逗号分割，但要处理嵌套
	depth := 0
	start := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '{', '(':
			depth++
		case '}', ')':
			depth--
		case ',':
			if depth == 0 {
				pair := strings.TrimSpace(s[start:i])
				if err := parseKV(pair, out); err != nil {
					return nil, err
				}
				start = i + 1
			}
		}
	}
	if start < len(s) {
		pair := strings.TrimSpace(s[start:])
		if pair != "" {
			if err := parseKV(pair, out); err != nil {
				return nil, err
			}
		}
	}
	return out, nil
}

// parseKV 解析 "key": value 或 key: value
func parseKV(s string, out map[string]interface{}) error {
	colon := -1
	depth := 0
	inStr := byte(0)
	for i := 0; i < len(s); i++ {
		c := s[i]
		if inStr != 0 {
			if c == inStr {
				inStr = 0
			}
			continue
		}
		switch c {
		case '"', '\'':
			inStr = c
		case '{', '(':
			depth++
		case '}', ')':
			depth--
		case ':':
			if depth == 0 {
				colon = i
				break
			}
		}
		if colon >= 0 {
			break
		}
	}
	if colon < 0 {
		return fmt.Errorf("missing colon in: %s", s)
	}
	key := strings.TrimSpace(s[:colon])
	// 去掉 key 的引号
	key = strings.Trim(key, `"'`)
	value := strings.TrimSpace(s[colon+1:])

	out[key] = parseValueSimple(value)
	return nil
}

// parseValueSimple 解析简单字面量
func parseValueSimple(s string) interface{} {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	// 字符串
	if (strings.HasPrefix(s, `"`) && strings.HasSuffix(s, `"`)) ||
		(strings.HasPrefix(s, `'`) && strings.HasSuffix(s, `'`)) {
		return s[1 : len(s)-1]
	}
	// null
	if strings.ToUpper(s) == "NULL" {
		return nil
	}
	// 布尔
	upper := strings.ToUpper(s)
	if upper == "TRUE" {
		return true
	}
	if upper == "FALSE" {
		return false
	}
	// 整数
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return i
	}
	// 浮点
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}
	// 默认字符串（去掉引号）
	return strings.Trim(s, `"'`)
}

// parseSetCalls 解析 .Set(col, val).Set(col2, val2) 或 .Sets({...})
func parseSetCalls(rest string, ub *UpdateBuilder) error {
	upper := strings.ToUpper(rest)
	// 找 .Sets({...})
	idx := strings.Index(upper, ".SETS(")
	if idx >= 0 {
		rest2 := rest[idx+6:]
		depth := 0
		end := -1
		for i := 0; i < len(rest2); i++ {
			switch rest2[i] {
			case '(', '{':
				depth++
			case ')', '}':
				depth--
				if depth == 0 {
					end = i
					break
				}
			}
			if end >= 0 {
				break
			}
		}
		if end >= 0 {
			row, err := parseRowObject(rest2[:end+1])
			if err != nil {
				return err
			}
			ub.Sets(row)
			return nil
		}
	}

	// 找所有 .Set(col, val)
	searchPos := 0
	for {
		idx := strings.Index(upper[searchPos:], ".SET(")
		if idx < 0 {
			break
		}
		idx += searchPos
		rest2 := rest[idx+5:]
		end := -1
		depth := 0
		inStr := byte(0)
		for i := 0; i < len(rest2); i++ {
			c := rest2[i]
			if inStr != 0 {
				if c == inStr {
					inStr = 0
				}
				continue
			}
			switch c {
			case '"', '\'':
				inStr = c
			case '(':
				depth++
			case ')':
				if depth == 0 {
					end = i
					break
				}
				depth--
			}
			if end >= 0 {
				break
			}
		}
		if end < 0 {
			return fmt.Errorf("malformed .Set()")
		}
		// 解析 col, val
		// rest2[:end] 排除了末尾的 )
		args := splitArgs(rest2[:end])
		if len(args) < 2 {
			return fmt.Errorf("Set() requires 2 arguments: col, value")
		}
		col := strings.Trim(args[0], `"'`)
		val := parseValueSimple(strings.Join(args[1:], ","))
		ub.Set(col, val)
		searchPos = idx + 5 + end + 1
	}
	return nil
}

// extractWhere 提取 .Where(cond) 中的条件
func extractWhere(rest string) string {
	upper := strings.ToUpper(rest)
	idx := strings.Index(upper, ".WHERE(")
	if idx < 0 {
		return ""
	}
	rest2 := rest[idx+7:]
	depth := 0
	inStr := byte(0)
	end := -1
	for i := 0; i < len(rest2); i++ {
		c := rest2[i]
		if inStr != 0 {
			if c == inStr {
				inStr = 0
			}
			continue
		}
		switch c {
		case '"', '\'':
			inStr = c
		case '(':
			depth++
		case ')':
			if depth == 0 {
				end = i
				break
			}
			depth--
		}
		if end >= 0 {
			break
		}
	}
	if end < 0 {
		return ""
	}
	// rest2[:end] 排除了末尾的 )
	// 同时去掉首尾的引号（CLI 语法使用 "col = val" 风格）
	cond := strings.TrimSpace(rest2[:end])
	cond = strings.Trim(cond, `"'`)
	return cond
}

// parseColumnDefs 解析 CREATE TABLE 的列定义
// 简化语法: "id INTEGER PRIMARY KEY", "name TEXT NOT NULL", "age INTEGER"
func parseColumnDefs(s string) ([]*ColumnDef, error) {
	// 找到第一个 ( 表示列定义开始
	start := strings.Index(s, "(")
	if start < 0 {
		return nil, fmt.Errorf("expected ( for column definitions")
	}
	s = s[start+1:]
	// 找到匹配的 )
	depth := 1
	end := -1
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				end = i
				break
			}
		}
		if end >= 0 {
			break
		}
	}
	if end < 0 {
		return nil, fmt.Errorf("unmatched parentheses")
	}
	content := s[:end]

	// 按逗号分割列定义
	parts := splitArgs(content)
	var cols []*ColumnDef
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		// 格式: "name TYPE [NOT NULL] [PRIMARY KEY]" 或  name TYPE
		p = strings.Trim(p, `"'`)
		tokens := strings.Fields(p)
		if len(tokens) < 2 {
			return nil, fmt.Errorf("invalid column definition: %s", p)
		}
		name := tokens[0]
		typ := strings.ToUpper(tokens[1])
		nullable := true
		for i := 2; i < len(tokens); i++ {
			if strings.ToUpper(tokens[i]) == "NOT" && i+1 < len(tokens) && strings.ToUpper(tokens[i+1]) == "NULL" {
				nullable = false
				break
			}
		}
		cols = append(cols, NewColumn(name, typ, nullable))
	}
	return cols, nil
}
