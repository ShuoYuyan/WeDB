// Package wqlv3 是 WQL (WeDB Query Language) 的 v3 实现。
//
// WQL 是 WeDB 的原生查询语言，**完全自实现的查询计划器**：
//   - 自有词法器（pkg/wql/lexer）
//   - 自有语法分析器（pkg/wql/parser）
//   - 自有查询计划 IR（pkg/wql/planner）
//   - 自有优化器（pkg/wql/optimizer）
//   - 自有执行器（pkg/wql/executor）
//
// 本包（wqlv3）提供更高层的 Go Fluent API：
//   db.T("users").Where(age > 18).Select(name, age).All()
//
// **重要**：WQL 不生成 SQL 字符串。执行器直接调用 WeDB 的 Go API。
package wqlv3

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// Database 是 WQL 的入口
type Database struct {
	adapter Adapter
}

// NewDatabase 创建 WQL 数据库
func NewDatabase(a Adapter) *Database {
	return &Database{adapter: a}
}

// Adapter 是 WQL 与底层存储之间的接口
// 当前实现：WeDBAdapter（直接调用 WeDB 的 Go API）
// 未来可扩展：PostgreSQLAdapter、MySQLAdapter 等
type Adapter interface {
	// 表操作
	ScanTable(tableName string) ([]map[string]interface{}, error)
	ScanTableWithColumns(tableName string, columns []string) ([]map[string]interface{}, error)
	ListTables() []string
	TableExists(name string) bool
	CreateTable(schema *TableSchema) error
	DropTable(name string) error

	// 数据操作 (DML)
	InsertRow(tableName string, row map[string]interface{}) error
	InsertRows(tableName string, rows []map[string]interface{}) error
	UpdateRow(tableName string, row map[string]interface{}, condition string) error
	DeleteRow(tableName string, condition string) error
	Count(tableName, condition string) (int64, error)

	// 聚合
	Min(tableName, column, condition string) (interface{}, error)
	Max(tableName, column, condition string) (interface{}, error)
	Sum(tableName, column, condition string) (float64, error)
	Avg(tableName, column, condition string) (float64, error)
}

// TableSchema 简化的表结构（用于 CLI 显示）
type TableSchema struct {
	Name    string
	Columns []ColumnDef
}

// ColumnDef 列定义
type ColumnDef struct {
	Name     string
	Type     string
	Nullable bool
}

// Close 关闭数据库连接
func (d *Database) Close() error {
	return nil
}

// Ping 检查连接
func (d *Database) Ping() error {
	return nil
}

// ===== InsertBuilder =====

// InsertBuilder 插入数据构建器
// 用法: db.Insert("users").Values(row1, row2, ...).Execute()
type InsertBuilder struct {
	db        Adapter
	tableName string
	rows      []map[string]interface{}
}

// Values 设置要插入的行（单行或多行）
func (ib *InsertBuilder) Values(rows ...map[string]interface{}) *InsertBuilder {
	ib.rows = append(ib.rows, rows...)
	return ib
}

// Value 快捷方法：插入单行
func (ib *InsertBuilder) Value(row map[string]interface{}) *InsertBuilder {
	return ib.Values(row)
}

// Execute 执行插入，返回影响行数
func (ib *InsertBuilder) Execute() (int64, error) {
	if len(ib.rows) == 0 {
		return 0, fmt.Errorf("no rows to insert")
	}
	if len(ib.rows) == 1 {
		if err := ib.db.InsertRow(ib.tableName, ib.rows[0]); err != nil {
			return 0, err
		}
		return 1, nil
	}
	if err := ib.db.InsertRows(ib.tableName, ib.rows); err != nil {
		return 0, err
	}
	return int64(len(ib.rows)), nil
}

// ===== UpdateBuilder =====

// UpdateBuilder 更新数据构建器
// 用法: db.Update("users").Set("name", "bob").Where("id = 1").Execute()
type UpdateBuilder struct {
	db        Adapter
	tableName string
	updates   map[string]interface{}
	condition string
}

// Set 设置要更新的列值
func (ub *UpdateBuilder) Set(column string, value interface{}) *UpdateBuilder {
	ub.updates[column] = value
	return ub
}

// Sets 批量设置列值
func (ub *UpdateBuilder) Sets(values map[string]interface{}) *UpdateBuilder {
	for k, v := range values {
		ub.updates[k] = v
	}
	return ub
}

// Where 设置更新条件
func (ub *UpdateBuilder) Where(condition string) *UpdateBuilder {
	ub.condition = condition
	return ub
}

// Execute 执行更新，返回影响行数
func (ub *UpdateBuilder) Execute() (int64, error) {
	if len(ub.updates) == 0 {
		return 0, fmt.Errorf("no columns to update")
	}
	if ub.condition == "" {
		return 0, fmt.Errorf("UPDATE without WHERE is not allowed for safety")
	}
	if err := ub.db.UpdateRow(ub.tableName, ub.updates, ub.condition); err != nil {
		return 0, err
	}
	return -1, nil // WeDB Go API 不返回受影响行数
}

// ===== DeleteBuilder =====

// DeleteBuilder 删除数据构建器
// 用法: db.Delete("users").Where("age < 18").Execute()
type DeleteBuilder struct {
	db        Adapter
	tableName string
	condition string
}

// Where 设置删除条件
func (db *DeleteBuilder) Where(condition string) *DeleteBuilder {
	db.condition = condition
	return db
}

// Execute 执行删除，返回影响行数
func (db *DeleteBuilder) Execute() (int64, error) {
	if db.condition == "" {
		return 0, fmt.Errorf("DELETE without WHERE is not allowed for safety")
	}
	if err := db.db.DeleteRow(db.tableName, db.condition); err != nil {
		return 0, err
	}
	return -1, nil // WeDB Go API 不返回受影响行数
}

// ===== DDL 辅助函数 =====

// NewTableSchema 创建表结构定义
// 用法: wqlv3.NewTableSchema("users",
//
//	wqlv3.NewColumn("id", "INTEGER", false),
//	wqlv3.NewColumn("name", "TEXT", true),
//	wqlv3.NewColumn("age", "INTEGER", true),
//	wqlv3.NewPrimaryKey("id"))
func NewTableSchema(name string, columns ...*ColumnDef) *TableSchema {
	return &TableSchema{
		Name:    name,
		Columns: derefColumns(columns),
	}
}

// NewColumn 创建列定义
func NewColumn(name, typ string, nullable bool) *ColumnDef {
	return &ColumnDef{
		Name:     name,
		Type:     typ,
		Nullable: nullable,
	}
}

func derefColumns(cols []*ColumnDef) []ColumnDef {
	out := make([]ColumnDef, 0, len(cols))
	for _, c := range cols {
		if c != nil {
			out = append(out, *c)
		}
	}
	return out
}

// ===== Fluent API =====

// Table 创建表查询构建器
// 用法: db.Table("users").Select("id", "name").Where("age > 18").All()
func (d *Database) Table(name string) *QueryBuilder {
	return &QueryBuilder{
		db:        d.adapter,
		tableName: name,
		selects:   nil, // nil = SELECT *
	}
}

// Insert 创建插入构建器
// 用法: db.Insert("users").Values(map[string]interface{}{"id": 1, "name": "alice"}).Execute()
func (d *Database) Insert(tableName string) *InsertBuilder {
	return &InsertBuilder{
		db:        d.adapter,
		tableName: tableName,
	}
}

// Update 创建更新构建器
// 用法: db.Update("users").Set("name", "bob").Where("id = 1").Execute()
func (d *Database) Update(tableName string) *UpdateBuilder {
	return &UpdateBuilder{
		db:        d.adapter,
		tableName: tableName,
		updates:   make(map[string]interface{}),
	}
}

// Delete 创建删除构建器
// 用法: db.Delete("users").Where("age < 18").Execute()
func (d *Database) Delete(tableName string) *DeleteBuilder {
	return &DeleteBuilder{
		db:        d.adapter,
		tableName: tableName,
	}
}

// CreateTable 创建表（DDL）
// 用法: db.CreateTable(wqlv3.NewTableSchema("users", wqlv3.NewColumn("id", "INTEGER", false), ...))
func (d *Database) CreateTable(schema *TableSchema) error {
	return d.adapter.CreateTable(schema)
}

// DropTable 删除表（DDL）
func (d *Database) DropTable(tableName string) error {
	return d.adapter.DropTable(tableName)
}

// JoinSpec 内部 JOIN 规格
type JoinSpec struct {
	Type       string                 // "Join" / "LeftJoin" / "RightJoin"
	Table      string                 // 被连接的表名
	LeftKey    string                 // 旧语法: 左表键
	RightKey   string                 // 旧语法: 右表键
	OnExpr     string                 // ON 条件字符串（已格式化）
}

// QueryBuilder 链式查询构建器
type QueryBuilder struct {
	db        Adapter
	tableName string

	// 查询条件
	selects   []string    // nil = *
	where     string      // WHERE 子句（已格式化）
	orderCol  string
	orderDir  string // "ASC" 或 "DESC"
	skipN     int64
	takeN     int64
	joins     []JoinSpec // JOIN 链
	groupBy   []string   // GROUP BY 列
	having    string     // HAVING 条件（已格式化）
	aggFuncs  []AggSpec  // 聚合函数规格（仅在有 GROUP BY 时使用）
}

// AggSpec 聚合函数规格
type AggSpec struct {
	Function string // "Count" / "Sum" / "Avg" / "Min" / "Max"
	Column   string // 列名；Count() 为空表示 COUNT(*)
	Alias    string // 可选别名
}

// Select 指定要查询的列
func (qb *QueryBuilder) Select(cols ...string) *QueryBuilder {
	qb.selects = cols
	return qb
}

// Where 设置 WHERE 条件
// condition 是字符串形式的条件表达式，如: "age > 18 AND name = 'alice'"
func (qb *QueryBuilder) Where(condition string) *QueryBuilder {
	qb.where = condition
	return qb
}

// OrderBy 设置排序列
func (qb *QueryBuilder) OrderBy(col, dir string) *QueryBuilder {
	qb.orderCol = col
	qb.orderDir = strings.ToUpper(dir)
	return qb
}

// Skip 设置跳过的行数
func (qb *QueryBuilder) Skip(n int64) *QueryBuilder {
	qb.skipN = n
	return qb
}

// Take 设置最大返回行数
func (qb *QueryBuilder) Take(n int64) *QueryBuilder {
	qb.takeN = n
	return qb
}

// Join 链式添加 JOIN
// joinType: "Join" (INNER) / "LeftJoin" / "RightJoin"
// 旧语法: Join(orders, users.id, orders.user_id)
// 新语法: Join(orders, ON users.id = orders.user_id)
func (qb *QueryBuilder) Join(joinType, table, leftKey, rightKey, onExpr string) *QueryBuilder {
	qb.joins = append(qb.joins, JoinSpec{
		Type:     joinType,
		Table:    table,
		LeftKey:  leftKey,
		RightKey: rightKey,
		OnExpr:   onExpr,
	})
	return qb
}

// GroupBy 设置 GROUP BY 列
func (qb *QueryBuilder) GroupBy(cols ...string) *QueryBuilder {
	qb.groupBy = append(qb.groupBy, cols...)
	return qb
}

// Having 设置 HAVING 条件
func (qb *QueryBuilder) Having(condition string) *QueryBuilder {
	qb.having = condition
	return qb
}

// AddAggregate 显式注册聚合函数
func (qb *QueryBuilder) AddAggregate(fn, col, alias string) *QueryBuilder {
	qb.aggFuncs = append(qb.aggFuncs, AggSpec{
		Function: fn,
		Column:   col,
		Alias:    alias,
	})
	return qb
}

// All 执行查询并返回所有行
func (qb *QueryBuilder) All() ([]map[string]interface{}, error) {
	return qb.execute()
}

// First 返回第一行
func (qb *QueryBuilder) First() (map[string]interface{}, error) {
	qb.takeN = 1
	rows, err := qb.execute()
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}

// Count 统计行数
func (qb *QueryBuilder) Count() (int64, error) {
	return qb.db.Count(qb.tableName, qb.where)
}

// Sum 计算列总和
func (qb *QueryBuilder) Sum(col string) (float64, error) {
	return qb.db.Sum(qb.tableName, col, qb.where)
}

// Avg 计算列平均值
func (qb *QueryBuilder) Avg(col string) (float64, error) {
	return qb.db.Avg(qb.tableName, col, qb.where)
}

// Min 获取列最小值
func (qb *QueryBuilder) Min(col string) (interface{}, error) {
	return qb.db.Min(qb.tableName, col, qb.where)
}

// Max 获取列最大值
func (qb *QueryBuilder) Max(col string) (interface{}, error) {
	return qb.db.Max(qb.tableName, col, qb.where)
}

// execute 实际执行查询
func (qb *QueryBuilder) execute() ([]map[string]interface{}, error) {
	// 第1步: 扫描主表（无列过滤，先取全量）
	rows, err := qb.db.ScanTable(qb.tableName)
	if err != nil {
		return nil, fmt.Errorf("scan table %s: %w", qb.tableName, err)
	}

	// 第2步: 依次应用 JOIN（嵌套循环）
	for _, j := range qb.joins {
		rightRows, err := qb.db.ScanTable(j.Table)
		if err != nil {
			return nil, fmt.Errorf("scan join table %s: %w", j.Table, err)
		}
		rows = joinRows(rows, rightRows, j)
	}

	// 第3步: WHERE 过滤（在内存中）
	if qb.where != "" {
		rows = filterRows(rows, qb.where)
	}

	// 第4步: GROUP BY + 聚合
	if len(qb.groupBy) > 0 || len(qb.aggFuncs) > 0 {
		rows = groupAndAggregate(rows, qb.groupBy, qb.aggFuncs)
	}

	// 第5步: HAVING 过滤（聚合后）
	if qb.having != "" {
		// 将 HAVING 中的 "Func(col)" 形式重写为已合成的 key，便于 ParseWhere 识别
		havingRewritten := rewriteHavingAggregates(qb.having, qb.aggFuncs)
		rows = filterRows(rows, havingRewritten)
	}

	// 第6步: SELECT 列投影
	if len(qb.selects) > 0 {
		rows = projectColumns(rows, qb.selects)
	}

	// 第7步: ORDER BY 排序
	if qb.orderCol != "" {
		rows = sortRows(rows, qb.orderCol, qb.orderDir)
	}

	// 第8步: SKIP / TAKE
	if qb.skipN > 0 || qb.takeN > 0 {
		start := int(qb.skipN)
		if start > len(rows) {
			start = len(rows)
		}
		end := len(rows)
		if qb.takeN > 0 && int(qb.takeN) < len(rows)-start {
			end = start + int(qb.takeN)
		}
		rows = rows[start:end]
	}

	return rows, nil
}

// rewriteHavingAggregates 将 HAVING 字符串中的 "Func(col)" 替换为已合成的 key
// 例如: "Sum(amount) > 50" -> "__agg_Sum_amount > 50"
// 同时配合 evaluateWithAggKeys 在过滤时还原。
func rewriteHavingAggregates(s string, aggs []AggSpec) string {
	// 简单替换：遍历所有已注册的 agg，替换其输出 key 的出现位置
	// 为了安全，使用首尾边界确保是完整的 token
	out := s
	for _, agg := range aggs {
		key := aggOutputKey(agg)
		// 用一个特殊 key 替代，便于 ParseWhere 识别为列名
		placeholder := "__agg_" + strings.ReplaceAll(key, "(", "_") + "_"
		placeholder = strings.ReplaceAll(placeholder, ")", "_")
		// 替换时确保是 word 边界
		out = replaceToken(out, key, placeholder)
	}
	return out
}

// evaluateHavingWithAggs 评估带聚合重写后的 HAVING 表达式。
// 将 __agg_*_ 占位符还原为合成 key 的实际值。
func evaluateHavingWithAggs(expr Expression, row map[string]interface{}) bool {
	// 自定义求值：递归遍历，将 ColumnExpr 名字映射到 row
	val, err := evalExprForHaving(expr, row)
	if err != nil {
		return false
	}
	return toBoolForFilter(val)
}

func evalExprForHaving(expr Expression, row map[string]interface{}) (interface{}, error) {
	switch v := expr.(type) {
	case *ColumnExpr:
		if val, ok := row[v.Name]; ok {
			return val, nil
		}
		return nil, nil
	case *LiteralExpr:
		return v.Value, nil
	case *BinaryExpr:
		l, err := evalExprForHaving(v.Left, row)
		if err != nil {
			return nil, err
		}
		r, err := evalExprForHaving(v.Right, row)
		if err != nil {
			return nil, err
		}
		return evalBinaryForHaving(v.Op, l, r)
	case *UnaryExpr:
		inner, err := evalExprForHaving(v.Operand, row)
		if err != nil {
			return nil, err
		}
		b := toBoolForFilter(inner)
		if v.Op == "NOT" {
			return !b, nil
		}
		return b, nil
	}
	return nil, fmt.Errorf("unsupported expr type")
}

func evalBinaryForHaving(op BinaryOp, left, right interface{}) (interface{}, error) {
	switch op {
	case OpEq:
		return compareForFilter(left, right) == 0, nil
	case OpNe:
		return compareForFilter(left, right) != 0, nil
	case OpLt:
		return compareForFilter(left, right) < 0, nil
	case OpLe:
		return compareForFilter(left, right) <= 0, nil
	case OpGt:
		return compareForFilter(left, right) > 0, nil
	case OpGe:
		return compareForFilter(left, right) >= 0, nil
	case OpAnd:
		return toBoolForFilter(left) && toBoolForFilter(right), nil
	case OpOr:
		return toBoolForFilter(left) || toBoolForFilter(right), nil
	}
	return nil, fmt.Errorf("unsupported op: %s", op)
}

// replaceToken 替换整个 word 边界内的 token
func replaceToken(s, old, new string) string {
	// 找到所有 old 的位置，逐个检查是否为完整 token
	out := ""
	i := 0
	for i < len(s) {
		if i+len(old) <= len(s) && s[i:i+len(old)] == old {
			// 检查 word 边界
			before := i == 0 || !isWordByte(s[i-1])
			after := i+len(old) == len(s) || !isWordByte(s[i+len(old)])
			if before && after {
				out += new
				i += len(old)
				continue
			}
		}
		out += string(s[i])
		i++
	}
	return out
}

// groupAndAggregate 按 groupBy 列分组，对每组应用 aggFuncs 聚合。
// 如果无 groupBy 但有聚合：整个结果集当作一个组。
// 输出列：groupBy 列 + 聚合结果列（key 为 "Function(col)" 或 alias）
func groupAndAggregate(rows []map[string]interface{}, groupBy []string, aggFuncs []AggSpec) []map[string]interface{} {
	// 无聚合且无 groupBy：原样返回
	if len(aggFuncs) == 0 && len(groupBy) == 0 {
		return rows
	}

	// 分组：以 groupBy 值的元组作为 key
	groups := make(map[string][]map[string]interface{})
	order := make([]string, 0) // 保留插入顺序
	for _, row := range rows {
		key := groupKey(row, groupBy)
		if _, exists := groups[key]; !exists {
			order = append(order, key)
		}
		groups[key] = append(groups[key], row)
	}

	out := make([]map[string]interface{}, 0, len(groups))
	for _, key := range order {
		grp := groups[key]
		// 构造新行：取第一行的 groupBy 列值
		merged := make(map[string]interface{})
		for _, col := range groupBy {
			if len(grp) > 0 {
				merged[col] = grp[0][col]
			}
		}
		// 应用聚合
		for _, agg := range aggFuncs {
			val := computeAggregate(grp, agg.Function, agg.Column)
			outKey := agg.Alias
			if outKey == "" {
				outKey = aggOutputKey(agg)
			}
			merged[outKey] = val
			// 同时为 HAVING 重写提供占位 key（让 ParseWhere 视为合法列名）
			placeholder := "__agg_" + strings.ReplaceAll(aggOutputKey(agg), "(", "_") + "_"
			placeholder = strings.ReplaceAll(placeholder, ")", "_")
			merged[placeholder] = val
		}
		// 即使无 aggFuncs，也要为 groupBy-only 的情况输出一行
		if len(aggFuncs) == 0 && len(groupBy) > 0 {
			// 已经合并了 groupBy 字段；OK
		}
		out = append(out, merged)
	}
	return out
}

// aggOutputKey 返回聚合函数在结果行中的默认 key
func aggOutputKey(agg AggSpec) string {
	if agg.Column == "" {
		return fmt.Sprintf("%s()", agg.Function)
	}
	return fmt.Sprintf("%s(%s)", agg.Function, agg.Column)
}

// groupKey 为分组键构造字符串
func groupKey(row map[string]interface{}, groupBy []string) string {
	if len(groupBy) == 0 {
		return "_all_"
	}
	parts := make([]string, len(groupBy))
	for i, col := range groupBy {
		parts[i] = fmt.Sprintf("%v", row[col])
	}
	return strings.Join(parts, "\x00")
}

// computeAggregate 对一组行应用聚合函数
func computeAggregate(group []map[string]interface{}, fn, col string) interface{} {
	switch strings.ToLower(fn) {
	case "count":
		if col == "" {
			return int64(len(group))
		}
		// COUNT(col): 统计 col 非 null 的行数
		n := int64(0)
		for _, r := range group {
			if v, ok := r[col]; ok && v != nil {
				n++
			}
		}
		return n
	case "sum":
		var sum float64
		for _, r := range group {
			if v, ok := r[col]; ok && v != nil {
				if f, ok := toFloat(v); ok {
					sum += f
				}
			}
		}
		return sum
	case "avg":
		var sum float64
		var n int64
		for _, r := range group {
			if v, ok := r[col]; ok && v != nil {
				if f, ok := toFloat(v); ok {
					sum += f
					n++
				}
			}
		}
		if n == 0 {
			return nil
		}
		return sum / float64(n)
	case "min":
		var minVal interface{}
		for _, r := range group {
			v, ok := r[col]
			if !ok || v == nil {
				continue
			}
			if minVal == nil || compareValues(v, minVal) < 0 {
				minVal = v
			}
		}
		return minVal
	case "max":
		var maxVal interface{}
		for _, r := range group {
			v, ok := r[col]
			if !ok || v == nil {
				continue
			}
			if maxVal == nil || compareValues(v, maxVal) > 0 {
				maxVal = v
			}
		}
		return maxVal
	}
	return nil
}

// projectColumns 投影出指定列（若列不存在则填 nil）
func projectColumns(rows []map[string]interface{}, cols []string) []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(rows))
	for _, row := range rows {
		proj := make(map[string]interface{}, len(cols))
		for _, c := range cols {
			// 先按完整名找；找不到则按短名（去表前缀）找
			if v, ok := row[c]; ok {
				proj[c] = v
				continue
			}
			if idx := strings.LastIndex(c, "."); idx >= 0 {
				short := c[idx+1:]
				if v, ok := row[short]; ok {
					proj[c] = v
					continue
				}
			}
			proj[c] = nil
		}
		out = append(out, proj)
	}
	return out
}

// joinRows 嵌套循环连接。
// 返回的每行是 left+right 字段合并后的新行。
// 冲突策略：右表字段加表名前缀以避免覆盖；左表字段保持原名；
// 解析时如 ColumnExpr "table.col" 找不到，则退回到 "col" 短名。
// 行为：
//   - INNER Join: 仅保留匹配的行
//   - LeftJoin:   保留所有左行，右表不匹配时右表字段为 nil
//   - RightJoin:  保留所有右行，左表不匹配时左表字段为 nil
// 匹配规则：优先使用 OnExpr 解析的布尔表达式（如果非空）；
// 否则使用 LeftKey/RightKey 单列等值。
func joinRows(left, right []map[string]interface{}, j JoinSpec) []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(left))

	// 预解析 ON 表达式
	var onParsed Expression
	if j.OnExpr != "" {
		if parsed, err := ParseWhere(j.OnExpr); err == nil {
			onParsed = parsed
		}
	}

	leftMatched := make([]bool, len(left))
	rightMatched := make([]bool, len(right))

	for li, lr := range left {
		matched := false
		for ri, rr := range right {
			var ok bool
			if onParsed != nil {
				merged := mergeRows(lr, rr, j.Table)
				ok = EvalBoolExpr(onParsed, merged)
			} else {
				ok = valuesEqual(lr[j.LeftKey], rr[j.RightKey])
			}
			if ok {
				matched = true
				leftMatched[li] = true
				rightMatched[ri] = true
				if onParsed != nil {
					out = append(out, mergeRows(lr, rr, j.Table))
				} else {
					// 旧语法: 简单合并（不冲突）
					merged := make(map[string]interface{})
					for k, v := range lr {
						merged[k] = v
					}
					for k, v := range rr {
						if _, exists := merged[k]; !exists {
							merged[k] = v
						}
					}
					out = append(out, merged)
				}
			}
		}
		if !matched && (j.Type == "LeftJoin" || j.Type == "Left Join" || strings.EqualFold(j.Type, "leftjoin")) {
			out = append(out, mergeRows(lr, nil, j.Table))
		}
	}

	// RightJoin: 补上未匹配的右行
	if strings.EqualFold(strings.ReplaceAll(j.Type, " ", ""), "rightjoin") {
		for ri, rr := range right {
			if !rightMatched[ri] {
				out = append(out, mergeRows(nil, rr, j.Table))
			}
		}
	}

	return out
}

// mergeRows 合并左右行；nil 表示占位空行
// 为了支持 ON users.id = orders.user_id，右表的列会带 "table." 前缀
// 以避免与左表同名列冲突。
func mergeRows(left, right map[string]interface{}, rightTable string) map[string]interface{} {
	merged := make(map[string]interface{})
	for k, v := range left {
		merged[k] = v
	}
	for k, v := range right {
		if rightTable != "" {
			prefixed := rightTable + "." + k
			if _, exists := merged[k]; !exists {
				merged[k] = v
			}
			merged[prefixed] = v
		} else {
			merged[k] = v
		}
	}
	return merged
}

// valuesEqual 比较两个值是否相等（用于单列等值连接）
func valuesEqual(a, b interface{}) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	// 类型相同的直接比较
	if reflect.DeepEqual(a, b) {
		return true
	}
	// 数字之间的宽松比较
	af, aok := toFloat(a)
	bf, bok := toFloat(b)
	if aok && bok {
		return af == bf
	}
	return false
}

func toFloat(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case int:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	case float32:
		return float64(n), true
	case float64:
		return n, true
	}
	return 0, false
}

// filterRows 在内存中应用 WHERE 过滤
func filterRows(rows []map[string]interface{}, where string) []map[string]interface{} {
	expr, err := ParseWhere(where)
	if err != nil {
		// 解析失败: 返回所有行（保守策略）
		return rows
	}
	out := make([]map[string]interface{}, 0, len(rows))
	for _, row := range rows {
		if EvalBoolExpr(expr, row) {
			out = append(out, row)
		}
	}
	return out
}

// sortRows 在内存中排序
func sortRows(rows []map[string]interface{}, col, dir string) []map[string]interface{} {
	sort.SliceStable(rows, func(i, j int) bool {
		a, b := rows[i][col], rows[j][col]
		cmp := compareValues(a, b)
		if dir == "DESC" {
			return cmp > 0
		}
		return cmp < 0
	})
	return rows
}

// compareValues 比较两个值
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
	switch x := a.(type) {
	case int:
		return cmpInt(int64(x), b)
	case int32:
		return cmpInt(int64(x), b)
	case int64:
		return cmpInt(x, b)
	case float32:
		return cmpFloat(float64(x), b)
	case float64:
		return cmpFloat(x, b)
	case string:
		if y, ok := b.(string); ok {
			return strings.Compare(x, y)
		}
	}
	return 0
}

func cmpInt(a int64, b interface{}) int {
	switch y := b.(type) {
	case int:
		switch {
		case a < int64(y):
			return -1
		case a > int64(y):
			return 1
		}
	case int32:
		switch {
		case a < int64(y):
			return -1
		case a > int64(y):
			return 1
		}
	case int64:
		switch {
		case a < y:
			return -1
		case a > y:
			return 1
		}
	case float32:
		switch {
		case a < int64(y):
			return -1
		case a > int64(y):
			return 1
		}
	case float64:
		switch {
		case a < int64(y):
			return -1
		case a > int64(y):
			return 1
		}
	}
	return 0
}

func cmpFloat(a float64, b interface{}) int {
	switch y := b.(type) {
	case int:
		switch {
		case a < float64(y):
			return -1
		case a > float64(y):
			return 1
		}
	case int32:
		switch {
		case a < float64(y):
			return -1
		case a > float64(y):
			return 1
		}
	case int64:
		switch {
		case a < float64(y):
			return -1
		case a > float64(y):
			return 1
		}
	case float32:
		switch {
		case a < float64(y):
			return -1
		case a > float64(y):
			return 1
		}
	case float64:
		switch {
		case a < y:
			return -1
		case a > y:
			return 1
		}
	}
	return 0
}
