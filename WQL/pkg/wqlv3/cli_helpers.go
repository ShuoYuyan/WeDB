// CLI 辅助函数：为命令行工具提供简单的字符串接口。
// 这些函数不涉及 SQL，只是字符串解析和表格输出。
package wqlv3

import (
	"fmt"
	"strings"
	"time"
)

// QueryResult 统一的查询结果封装
type QueryResult struct {
	Rows      []map[string]interface{} // 查询行
	Value     interface{}              // 聚合值（Count/Sum/Avg/Min/Max）
	Columns   []string                 // 显式选择的列
	Duration  time.Duration             // 执行时间
	Statement string                   // 原始语句（用于显示）
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
// 接受的格式: db.Table("name").Where(cond).All() 或 Table("name").Where(cond).All()
//
// 由于完整的 WQL 解析需要复杂的语法分析（pkg/wql/lexer/parser），
// 这里采用简化的正则式解析。
func EvaluateQuery(db *Database, expr string) (QueryResult, error) {
	expr = strings.TrimSpace(expr)
	result := QueryResult{Statement: expr, Duration: 0}
	start := time.Now()

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
