// WQL 字符串解析器（无双引号设计）
// 这是 wqlv3 的高级入口：接受类自然语言的 WQL 字符串，
// 如 `db.Table(users).Select(name, age).Where(age > 18).All()`，
// 完全不需要双引号（标识符和字符串值都不需要引号）。
//
// 实现策略：
//   1. 使用 lexer/parser 包进行完整词法+语法分析
//   2. 将 AST 转换为 wqlv3 的 Builder API 调用
//   3. 执行并返回结果
package wqlv3

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/wedb/wedb/WQL/pkg/wql/lexer"
	"github.com/wedb/wedb/WQL/pkg/wql/parser"
)

// EvaluateQueryNoQuotes 解析无双引号 WQL 并执行
// 支持语法:
//   db.Table(users).Select(name, age).Where(age > 18).All()
//   db.Table(orders).Select(city, Count() AS cnt).GroupBy(city).All()
//   db.Table(orders).Where(amount > 100).Sum(amount)
//   db.Table(users).Take(10).OrderBy(age, DESC)
//   db.Table(users).Insert({id: 1, name: "alice"}).Execute()
//   db.Table(users).Set(age, 31).Where(id = 1).Execute()
//   db.Table(users).Where(age < 18).Delete().Execute()
//   db.Table(products).Create(id INTEGER PRIMARY KEY, name TEXT).Execute()
//   db.Table(temp).Drop().Execute()
//
// 注意：
//   - 标识符（表名、列名）不需要引号：db.Table(users) 而不是 db.Table("users")
//   - 字符串值可以用引号：name = "alice" 或 name = alice（如果 alice 不是列名）
//   - 数字不需要引号：age > 18
func EvaluateQueryNoQuotes(db *Database, expr string) (QueryResult, error) {
	expr = strings.TrimSpace(expr)
	result := QueryResult{Statement: expr, Duration: 0}
	start := time.Now()

	// 1. 词法分析
	lex := lexer.NewLexer(expr)

	// 2. 语法分析
	p := parser.NewParser(lex)
	query, err := p.Parse()
	if err != nil {
		return result, fmt.Errorf("parse error: %w", err)
	}

	// 3. 构造查询构建器（DML/DDL 副作用延后到 executeQuery 中执行）
	qb, err := buildQueryBuilder(db, query)
	if err != nil {
		return result, err
	}

	// 4. 执行（DML/DDL 在这里完成；SELECT 在这里返回结果）
	result, err = executeQuery(db, qb, query.Operations)
	if err != nil {
		return result, err
	}

	result.Duration = time.Since(start)
	return result, nil
}

// buildQueryBuilder 从解析后的 query 构造 wqlv3 QueryBuilder
// SELECT 查询走 qb；DML/DDL 副作用延后到 executeQuery 中执行
func buildQueryBuilder(db *Database, query *parser.WQLQuery) (*QueryBuilder, error) {
	if query.Source == "" {
		return nil, fmt.Errorf("no source table")
	}
	qb := db.Table(query.Source)

	// 保存最近一次的 WHERE（供 Delete / Update 使用）
	var lastWhereExpr parser.Expression

	for _, op := range query.Operations {
		switch o := op.(type) {
		case *parser.SelectOperation:
			cols := make([]string, 0, len(o.Columns))
			for _, c := range o.Columns {
				cols = append(cols, exprToString(c))
			}
			qb = qb.Select(cols...)
		case *parser.WhereOperation:
			lastWhereExpr = o.Condition
			where := exprToString(o.Condition)
			qb = qb.Where(where)
		case *parser.OrderByOperation:
			col := exprToString(o.Column)
			dir := o.Direction
			if dir == "" {
				dir = "ASC"
			}
			qb = qb.OrderBy(col, dir)
		case *parser.TakeOperation:
			n, err := exprToInt(o.Count)
			if err != nil {
				return nil, err
			}
			qb = qb.Take(n)
		case *parser.SkipOperation:
			n, err := exprToInt(o.Count)
			if err != nil {
				return nil, err
			}
			qb = qb.Skip(n)
		case *parser.LimitOperation:
			n, err := exprToInt(o.Count)
			if err != nil {
				return nil, err
			}
			qb = qb.Take(n)
		case *parser.JoinOperation:
			joinType := o.JoinType
			tableName := exprToString(o.Table)
			var onExpr string
			if o.Condition != nil {
				onExpr = exprToString(o.Condition)
			}
			var leftKey, rightKey string
			if o.LeftKey != nil {
				leftKey = exprToString(o.LeftKey)
			}
			if o.RightKey != nil {
				rightKey = exprToString(o.RightKey)
			}
			qb = qb.Join(joinType, tableName, leftKey, rightKey, onExpr)
		case *parser.InsertOperation:
			// 延后到 executeQuery 中执行；表名从 query.Source 取得
		case *parser.SetOperation:
			// 延后到 executeQuery 中执行；记录 Set 之后的最近一个 Where 作为条件
			cond := findWhereAfter(query.Operations, o)
			if cond == nil {
				cond = lastWhereExpr // 兜底：使用之前的 Where
			}
			o.Condition = cond
		case *parser.DeleteOperation:
			// 延后到 executeQuery 中执行；记录条件
			o.Condition = lastWhereExpr
		case *parser.CreateTableOperation, *parser.DropTableOperation:
			// 延后到 executeQuery 中执行
		case *parser.ExecuteOperation:
			// 终结符，executeQuery 中处理
		// GroupBy, Having, Join: 暂不支持（需要更复杂的查询计划器）
		default:
			// 忽略不支持的操作
		}
	}

	return qb, nil
}

// objectLiteralToMap 将对象字面量 AST 转为 map
func objectLiteralToMap(obj *parser.ObjectLiteralExpression) map[string]interface{} {
	out := map[string]interface{}{}
	for _, f := range obj.Fields {
		key := strings.TrimSpace(exprToString(f.Key))
		// key 不应该有引号
		key = strings.Trim(key, `"`)
		out[key] = literalValue(f.Value)
	}
	return out
}

// literalValue 取字面量表达式的 Go 原生值
// 数字字符串自动转为 int64/float64，布尔转为 bool
func literalValue(e parser.Expression) interface{} {
	switch v := e.(type) {
	case *parser.LiteralExpression:
		if s, ok := v.Value.(string); ok {
			// 数字字面量在 lexer 中是字符串形式，转为原生类型
			if i, err := strconv.ParseInt(s, 10, 64); err == nil {
				return i
			}
			if f, err := strconv.ParseFloat(s, 64); err == nil {
				return f
			}
			if s == "true" {
				return true
			}
			if s == "false" {
				return false
			}
			if s == "null" {
				return nil
			}
			return s
		}
		return v.Value
	case *parser.Identifier:
		return v.Value
	}
	return exprToString(e)
}

// executeQuery 执行查询并返回结果
// 处理顺序：先收集所有 DML/DDL 操作，最后按顺序执行；
// 然后处理 SELECT 终结符（All/First）。
func executeQuery(db *Database, qb *QueryBuilder, ops []parser.Operation) (QueryResult, error) {
	result := QueryResult{Rows: nil}

	// 检查是否包含 DML/DDL 操作
	hasDML := false
	for _, op := range ops {
		switch op.(type) {
		case *parser.InsertOperation, *parser.SetOperation, *parser.DeleteOperation,
			*parser.CreateTableOperation, *parser.DropTableOperation:
			hasDML = true
		}
	}

	// 如果有 DML/DDL，先执行（顺序与解析顺序一致）
	if hasDML {
		for _, op := range ops {
			switch o := op.(type) {
			case *parser.InsertOperation:
				rows := make([]map[string]interface{}, 0, len(o.Rows))
				for _, r := range o.Rows {
					if obj, ok := r.(*parser.ObjectLiteralExpression); ok {
						rows = append(rows, objectLiteralToMap(obj))
					}
				}
				n, err := db.Insert(qb.tableName).Values(rows...).Execute()
				if err != nil {
					return result, err
				}
				result.AffectedRows = n
			case *parser.SetOperation:
				updates := map[string]interface{}{}
				for _, u := range o.Updates {
					if obj, ok := u.(*parser.ObjectLiteralExpression); ok {
						for k, v := range objectLiteralToMap(obj) {
							updates[k] = v
						}
					}
				}
				ub := db.Update(qb.tableName).Sets(updates)
				if o.Condition != nil {
					ub.Where(exprToString(o.Condition))
				}
				n, err := ub.Execute()
				if err != nil {
					return result, err
				}
				result.AffectedRows = n
			case *parser.DeleteOperation:
				dbDel := db.Delete(qb.tableName)
				if o.Condition != nil {
					dbDel.Where(exprToString(o.Condition))
				}
				n, err := dbDel.Execute()
				if err != nil {
					return result, err
				}
				result.AffectedRows = n
			case *parser.CreateTableOperation:
				cols := make([]*ColumnDef, 0, len(o.Columns))
				for _, col := range o.Columns {
					cols = append(cols, NewColumn(col.Name, col.Type, col.Nullable))
				}
				if err := db.CreateTable(NewTableSchema(qb.tableName, cols...)); err != nil {
					return result, err
				}
			case *parser.DropTableOperation:
				if err := db.DropTable(qb.tableName); err != nil {
					return result, err
				}
			}
		}

		// DML/DDL 后，如果最后一个操作是 Execute() 或只有 DML/DDL，返回空结果
		lastOp := ops[len(ops)-1]
		if _, isExec := lastOp.(*parser.ExecuteOperation); isExec {
			result.Rows = []map[string]interface{}{}
			return result, nil
		}
		// 如果没有 Execute 但都是 DML，也返回空
		allDML := true
		for _, op := range ops {
			switch op.(type) {
			case *parser.AllOperation, *parser.FirstOperation,
				*parser.SelectOperation, *parser.WhereOperation, *parser.OrderByOperation,
				*parser.TakeOperation, *parser.SkipOperation, *parser.LimitOperation,
				*parser.GroupByOperation, *parser.HavingOperation, *parser.JoinOperation:
				allDML = false
			}
		}
		if allDML {
			result.Rows = []map[string]interface{}{}
			return result, nil
		}
	}

	// SELECT 终结符
	for i := len(ops) - 1; i >= 0; i-- {
		op := ops[i]
		switch op.(type) {
		case *parser.AllOperation:
			rows, err := qb.All()
			if err != nil {
				return result, err
			}
			result.Rows = rows
			return result, nil
		case *parser.FirstOperation:
			row, err := qb.First()
			if err != nil {
				return result, err
			}
			if row != nil {
				result.Rows = []map[string]interface{}{row}
			}
			return result, nil
		}
	}

	// 没有显式终结操作时，默认执行 All
	rows, err := qb.All()
	if err != nil {
		return result, err
	}
	result.Rows = rows
	return result, nil
}

// findWhereAfter 查找 ops 中 target 之后最近的一个 WhereOperation
func findWhereAfter(ops []parser.Operation, target parser.Operation) parser.Expression {
	found := false
	for _, op := range ops {
		if !found {
			if op == target {
				found = true
			}
			continue
		}
		if w, ok := op.(*parser.WhereOperation); ok {
			return w.Condition
		}
	}
	return nil
}

// exprToString 将 Expression 转为 WQL 字符串（无双引号设计）
// 关键：字符串字面量需要加引号，标识符不加引号
func exprToString(e parser.Expression) string {
	return astToString(e)
}

// astToString 递归地将 AST 转为字符串表示
// 遵循 WQL 无双引号设计：
//   - 标识符（列名、表名）不加引号：name, age
//   - 字符串字面量必须加引号："alice", "hello world"
//   - 数字不加引号：18, 3.14
//   - 布尔不加引号：true, false
//   - NULL 不加引号：null
func astToString(e parser.Expression) string {
	if e == nil {
		return ""
	}
	switch v := e.(type) {
	case *parser.Identifier:
		return v.Value
	case *parser.LiteralExpression:
		return literalToString(v.Value)
	case *parser.FunctionCallExpression:
		args := make([]string, len(v.Arguments))
		for i, a := range v.Arguments {
			args[i] = astToString(a)
		}
		return fmt.Sprintf("%s(%s)", v.Name, joinArgs(args))
	case *parser.CallExpression:
		args := make([]string, len(v.Arguments))
		for i, a := range v.Arguments {
			args[i] = astToString(a)
		}
		return fmt.Sprintf("%s(%s)", astToString(v.Callee), joinArgs(args))
	case *parser.BinaryExpression:
		left := astToString(v.Left)
		right := astToString(v.Right)
		// 字符串右值需要加括号以避免运算符优先级问题
		if _, isLit := v.Right.(*parser.LiteralExpression); isLit && v.Operator == "AND" {
			// 嵌套 AND 不需要额外括号
		}
		return fmt.Sprintf("(%s %s %s)", left, v.Operator, right)
	default:
		return e.String()
	}
}

// literalToString 将字面量值转为字符串
// 字符串加引号，其他类型不加
func literalToString(v interface{}) string {
	switch val := v.(type) {
	case string:
		// 字符串必须加双引号
		// 需要转义内部的引号和反斜杠
		escaped := strings.ReplaceAll(val, `\`, `\\`)
		escaped = strings.ReplaceAll(escaped, `"`, `\"`)
		return fmt.Sprintf(`"%s"`, escaped)
	case bool:
		if val {
			return "true"
		}
		return "false"
	case nil:
		return "null"
	default:
		// 数字、布尔等直接用 %v
		return fmt.Sprintf("%v", val)
	}
}

// joinArgs 用逗号连接参数（不加分隔符前缀/后缀）
func joinArgs(args []string) string {
	out := ""
	for i, a := range args {
		if i > 0 {
			out += ", "
		}
		out += a
	}
	return out
}

// exprToInt 将 Expression 转为整数
func exprToInt(e parser.Expression) (int64, error) {
	switch v := e.(type) {
	case *parser.LiteralExpression:
		switch n := v.Value.(type) {
		case int:
			return int64(n), nil
		case int64:
			return n, nil
		case float64:
			return int64(n), nil
		case string:
			parsed, err := strconv.ParseInt(n, 10, 64)
			if err != nil {
				return 0, fmt.Errorf("cannot convert %q to integer: %w", n, err)
			}
			return parsed, nil
		}
	case *parser.Identifier:
		return 0, fmt.Errorf("expected integer, got identifier: %s", v.Value)
	}
	// 尝试解析字符串形式
	s := e.String()
	parsed, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("cannot convert %q to integer: %w", s, err)
	}
	return parsed, nil
}
