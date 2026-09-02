// WeDB 适配器：将 WQL Adapter 接口映射到 WeDB 的 Go API。
// 这是 WQL 与 WeDB 之间的胶水层，**不生成任何 SQL 字符串**。
package wqlv3

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/wedb/wedb/internal/api"
	"github.com/wedb/wedb/internal/storage"
)

// WeDBAdapter 实现 wqlv3.Adapter 接口
// 直接调用 WeDB 的 Go API，不经过 SQL
type WeDBAdapter struct {
	db *storage.WeDBDatabase
}

// NewWeDBAdapter 创建 WeDB 适配器
func NewWeDBAdapter(db *storage.WeDBDatabase) *WeDBAdapter {
	return &WeDBAdapter{db: db}
}

// ===== 表操作 =====

// ScanTable 扫描整张表
func (w *WeDBAdapter) ScanTable(tableName string) ([]map[string]interface{}, error) {
	return w.db.ScanTable(tableName)
}

// ScanTableWithColumns 扫描指定列
func (w *WeDBAdapter) ScanTableWithColumns(tableName string, columns []string) ([]map[string]interface{}, error) {
	if len(columns) == 0 {
		return w.db.ScanTable(tableName)
	}
	// 使用 API 的 ScanTableWithColumns（如果存在）
	// 否则回退到全表扫描后过滤
	rows, err := w.db.ScanTable(tableName)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]interface{}, 0, len(rows))
	for _, row := range rows {
		filtered := make(map[string]interface{})
		for _, col := range columns {
			if v, ok := row[col]; ok {
				filtered[col] = v
			}
		}
		out = append(out, filtered)
	}
	return out, nil
}

// ScanTableWithOptions 把 WHERE/ORDER BY/LIMIT/OFFSET 下推到 WeDB 存储引擎
func (w *WeDBAdapter) ScanTableWithOptions(tableName string, opts *QueryOptions) ([]map[string]interface{}, error) {
	if w.db == nil {
		return nil, fmt.Errorf("database not opened")
	}
	apiOpts := &api.QueryOptions{
		Columns: opts.Columns,
		Where:   opts.Where,
		Limit:   opts.Limit,
		Offset:  opts.Offset,
	}
	for _, ob := range opts.OrderBy {
		// 解析 "col [ASC|DESC]" 形式
		parts := strings.Fields(ob)
		if len(parts) == 0 {
			continue
		}
		col := parts[0]
		order := api.SortAsc
		if len(parts) > 1 && strings.EqualFold(parts[1], "DESC") {
			order = api.SortDesc
		}
		apiOpts.OrderBy = append(apiOpts.OrderBy, api.SortBy{Column: col, Order: order})
	}
	return w.db.ScanTableWithOptions(tableName, apiOpts)
}

// CreateTable 创建表
func (w *WeDBAdapter) CreateTable(schema *TableSchema) error {
	if w.db == nil {
		return fmt.Errorf("database not opened")
	}
	apiSchema := w.convertSchema(schema)
	return w.db.CreateTable(apiSchema)
}

// DropTable 删除表
func (w *WeDBAdapter) DropTable(tableName string) error {
	if w.db == nil {
		return fmt.Errorf("database not opened")
	}
	return w.db.DropTable(tableName)
}

// ListTables 列出所有表
func (w *WeDBAdapter) ListTables() []string {
	if w.db == nil {
		return nil
	}
	return w.db.ListTables()
}

// TableExists 检查表是否存在
func (w *WeDBAdapter) TableExists(name string) bool {
	if w.db == nil {
		return false
	}
	return w.db.TableExists(name)
}

// GetTableSchema 获取表结构
func (w *WeDBAdapter) GetTableSchema(name string) (*TableSchema, error) {
	if w.db == nil {
		return nil, fmt.Errorf("database not opened")
	}
	apiSchema, err := w.db.GetTableSchema(name)
	if err != nil {
		return nil, err
	}
	return w.convertFromAPISchema(apiSchema), nil
}

// ===== DML 操作 =====

// InsertRow 插入单行
func (w *WeDBAdapter) InsertRow(tableName string, row map[string]interface{}) error {
	if w.db == nil {
		return fmt.Errorf("database not opened")
	}
	if !w.TableExists(tableName) {
		return fmt.Errorf("table not found: %s", tableName)
	}
	return w.db.InsertRow(tableName, row)
}

// InsertRows 批量插入
func (w *WeDBAdapter) InsertRows(tableName string, rows []map[string]interface{}) error {
	if w.db == nil {
		return fmt.Errorf("database not opened")
	}
	if !w.TableExists(tableName) {
		return fmt.Errorf("table not found: %s", tableName)
	}
	return w.db.InsertRows(tableName, rows)
}

// UpdateRow 更新满足条件的行
func (w *WeDBAdapter) UpdateRow(tableName string, row map[string]interface{}, condition string) error {
	if w.db == nil {
		return fmt.Errorf("database not opened")
	}
	if !w.TableExists(tableName) {
		return fmt.Errorf("table not found: %s", tableName)
	}
	if condition == "" {
		return fmt.Errorf("UPDATE without WHERE is not allowed for safety; use UpdateRows for unconditional updates")
	}
	return w.db.UpdateRow(tableName, row, condition)
}

// DeleteRow 删除满足条件的行
func (w *WeDBAdapter) DeleteRow(tableName string, condition string) error {
	if w.db == nil {
		return fmt.Errorf("database not opened")
	}
	if !w.TableExists(tableName) {
		return fmt.Errorf("table not found: %s", tableName)
	}
	if condition == "" {
		return fmt.Errorf("DELETE without WHERE is not allowed for safety")
	}
	return w.db.DeleteRow(tableName, condition)
}

// ===== 聚合操作 =====

// Count 统计行数
func (w *WeDBAdapter) Count(tableName, condition string) (int64, error) {
	if w.db == nil {
		return 0, fmt.Errorf("database not opened")
	}
	return w.db.Count(tableName, condition)
}

// Min 最小值
func (w *WeDBAdapter) Min(tableName, column, condition string) (interface{}, error) {
	if w.db == nil {
		return nil, fmt.Errorf("database not opened")
	}
	return w.db.Min(tableName, column, condition)
}

// Max 最大值
func (w *WeDBAdapter) Max(tableName, column, condition string) (interface{}, error) {
	if w.db == nil {
		return nil, fmt.Errorf("database not opened")
	}
	return w.db.Max(tableName, column, condition)
}

// Sum 求和
func (w *WeDBAdapter) Sum(tableName, column, condition string) (float64, error) {
	if w.db == nil {
		return 0, fmt.Errorf("database not opened")
	}
	return w.db.Sum(tableName, column, condition)
}

// Avg 平均值
func (w *WeDBAdapter) Avg(tableName, column, condition string) (float64, error) {
	if w.db == nil {
		return 0, fmt.Errorf("database not opened")
	}
	return w.db.Avg(tableName, column, condition)
}

// ===== 模式转换 =====

func (w *WeDBAdapter) convertSchema(s *TableSchema) *api.TableSchema {
	cols := make([]api.ColumnSchema, len(s.Columns))
	for i, c := range s.Columns {
		cols[i] = api.ColumnSchema{
			Name:     c.Name,
			Type:     api.ColumnType(c.Type),
			Nullable: c.Nullable,
		}
	}
	return &api.TableSchema{
		TableName: s.Name,
		Columns:   cols,
	}
}

func (w *WeDBAdapter) convertFromAPISchema(s *api.TableSchema) *TableSchema {
	cols := make([]ColumnDef, len(s.Columns))
	for i, c := range s.Columns {
		cols[i] = ColumnDef{
			Name:     c.Name,
			Type:     string(c.Type),
			Nullable: c.Nullable,
		}
	}
	return &TableSchema{Name: s.TableName, Columns: cols}
}

// ===== 事务支持（为未来预留） =====

// BeginTx 开始事务
func (w *WeDBAdapter) BeginTx(ctx context.Context, opts *api.TxOptions) (Transaction, error) {
	tx, err := w.db.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return &weDBTx{tx: tx}, nil
}

// Transaction 事务接口
type Transaction interface {
	Commit() error
	Rollback() error
}

type weDBTx struct {
	tx api.Transaction
}

func (t *weDBTx) Commit() error   { return t.tx.Commit() }
func (t *weDBTx) Rollback() error { return t.tx.Rollback() }

// ===== 表达式引擎（内部 WHERE 解析辅助） =====

// evalExprRef 内部辅助：与 expression.go 配合
// 在内存中评估 WHERE 条件
// 注意：这是 WQL 的内存过滤器，不涉及 SQL
// 当数据量小时适用；大数据量时应下推到存储引擎

func parseAndEvalWhere(where string, row map[string]interface{}) bool {
	expr, err := ParseWhere(where)
	if err != nil {
		return true // 保守：解析失败视为通过
	}
	return EvalBoolExpr(expr, row)
}

// 简单的辅助函数：从字符串解析数字
func mustParseFloat(s string) (float64, bool) {
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f, true
	}
	return 0, false
}

// 简单的辅助函数：比较两个 interface{} 值
func compareForFilter(a, b interface{}) int {
	// 尝试数字比较
	if af, aok := toFloat64ForFilter(a); aok {
		if bf, bok := toFloat64ForFilter(b); bok {
			switch {
			case af < bf:
				return -1
			case af > bf:
				return 1
			}
			return 0
		}
	}
	// 字符串比较
	as, _ := a.(string)
	bs, _ := b.(string)
	return strings.Compare(as, bs)
}

func toFloat64ForFilter(v interface{}) (float64, bool) {
	switch x := v.(type) {
	case int:
		return float64(x), true
	case int32:
		return float64(x), true
	case int64:
		return float64(x), true
	case float32:
		return float64(x), true
	case float64:
		return x, true
	case string:
		if f, err := strconv.ParseFloat(x, 64); err == nil {
			return f, true
		}
	}
	return 0, false
}
