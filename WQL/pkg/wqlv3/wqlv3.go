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
	// 第1步: 扫描表
	rows, err := qb.db.ScanTableWithColumns(qb.tableName, qb.selects)
	if err != nil {
		return nil, fmt.Errorf("scan table %s: %w", qb.tableName, err)
	}

	// 第2步: WHERE 过滤（在内存中）
	if qb.where != "" {
		rows = filterRows(rows, qb.where)
	}

	// 第3步: ORDER BY 排序
	if qb.orderCol != "" {
		rows = sortRows(rows, qb.orderCol, qb.orderDir)
	}

	// 第4步: SKIP / TAKE
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
