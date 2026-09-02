package adapter

import (
	"context"
	"fmt"

	"github.com/wedb/wedb/internal/api"
	"github.com/wedb/wedb/internal/storage"
)

// WeDBAdapter 适配 WeDB 存储引擎到 WQL 接口
// 这是 WQL 查询语言与 WeDB 存储引擎之间的桥梁
type WeDBAdapter struct {
	db *storage.WeDBDatabase
}

// NewWeDBAdapter 创建新的 WeDB 适配器
func NewWeDBAdapter(db *storage.WeDBDatabase) *WeDBAdapter {
	return &WeDBAdapter{
		db: db,
	}
}

// OpenDatabase 打开数据库
func (wa *WeDBAdapter) OpenDatabase(filePath string) error {
	db, err := storage.NewWeDBDatabase(filePath, 4096)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	wa.db = db
	return nil
}

// CloseDatabase 关闭数据库
func (wa *WeDBAdapter) CloseDatabase() error {
	if wa.db == nil {
		return nil
	}
	return wa.db.Close()
}

// CreateTable 创建表
func (wa *WeDBAdapter) CreateTable(schema *api.TableSchema) error {
	if wa.db == nil {
		return fmt.Errorf("database not opened")
	}
	return wa.db.CreateTable(schema)
}

// DropTable 删除表
func (wa *WeDBAdapter) DropTable(tableName string) error {
	if wa.db == nil {
		return fmt.Errorf("database not opened")
	}
	return wa.db.DropTable(tableName)
}

// GetTableSchema 获取表结构
func (wa *WeDBAdapter) GetTableSchema(tableName string) (*api.TableSchema, error) {
	if wa.db == nil {
		return nil, fmt.Errorf("database not opened")
	}
	return wa.db.GetTableSchema(tableName)
}

// ListTables 列出所有表
func (wa *WeDBAdapter) ListTables() []string {
	if wa.db == nil {
		return []string{}
	}
	return wa.db.ListTables()
}

// TableExists 检查表是否存在
func (wa *WeDBAdapter) TableExists(tableName string) bool {
	if wa.db == nil {
		return false
	}
	return wa.db.TableExists(tableName)
}

// ScanTable 扫描表
func (wa *WeDBAdapter) ScanTable(tableName string) ([]map[string]interface{}, error) {
	if wa.db == nil {
		return nil, fmt.Errorf("database not opened")
	}
	return wa.db.ScanTable(tableName)
}

// ScanTableWithColumns 扫描表（指定列）
func (wa *WeDBAdapter) ScanTableWithColumns(tableName string, columns []string) ([]map[string]interface{}, error) {
	if wa.db == nil {
		return nil, fmt.Errorf("database not opened")
	}
	return wa.db.ScanTableWithColumns(tableName, columns)
}

// InsertRow 插入单行
func (wa *WeDBAdapter) InsertRow(tableName string, row map[string]interface{}) error {
	if wa.db == nil {
		return fmt.Errorf("database not opened")
	}
	return wa.db.InsertRow(tableName, row)
}

// InsertRows 批量插入
func (wa *WeDBAdapter) InsertRows(tableName string, rows []map[string]interface{}) error {
	if wa.db == nil {
		return fmt.Errorf("database not opened")
	}
	return wa.db.InsertRows(tableName, rows)
}

// UpdateRow 更新行
func (wa *WeDBAdapter) UpdateRow(tableName string, row map[string]interface{}, condition string) error {
	if wa.db == nil {
		return fmt.Errorf("database not opened")
	}
	return wa.db.UpdateRow(tableName, row, condition)
}

// UpdateRows 批量更新
func (wa *WeDBAdapter) UpdateRows(tableName string, rows []map[string]interface{}, condition string) error {
	if wa.db == nil {
		return fmt.Errorf("database not opened")
	}
	return wa.db.UpdateRows(tableName, rows, condition)
}

// DeleteRow 删除行
func (wa *WeDBAdapter) DeleteRow(tableName string, condition string) error {
	if wa.db == nil {
		return fmt.Errorf("database not opened")
	}
	return wa.db.DeleteRow(tableName, condition)
}

// DeleteRows 批量删除
func (wa *WeDBAdapter) DeleteRows(tableName string, conditions []string) error {
	if wa.db == nil {
		return fmt.Errorf("database not opened")
	}
	return wa.db.DeleteRows(tableName, conditions)
}

// CreateIndex 创建索引
func (wa *WeDBAdapter) CreateIndex(tableName string, index *api.IndexInfo) error {
	if wa.db == nil {
		return fmt.Errorf("database not opened")
	}
	return wa.db.CreateIndex(tableName, index)
}

// DropIndex 删除索引
func (wa *WeDBAdapter) DropIndex(tableName string, indexName string) error {
	if wa.db == nil {
		return fmt.Errorf("database not opened")
	}
	return wa.db.DropIndex(tableName, indexName)
}

// GetIndexInfo 获取索引信息
func (wa *WeDBAdapter) GetIndexInfo(tableName string) ([]api.IndexInfo, error) {
	if wa.db == nil {
		return nil, fmt.Errorf("database not opened")
	}
	return wa.db.GetIndexInfo(tableName)
}

// IndexExists 检查索引是否存在
func (wa *WeDBAdapter) IndexExists(tableName string, indexName string) bool {
	if wa.db == nil {
		return false
	}
	return wa.db.IndexExists(tableName, indexName)
}

// Count 计数
func (wa *WeDBAdapter) Count(tableName string, condition string) (int64, error) {
	if wa.db == nil {
		return 0, fmt.Errorf("database not opened")
	}
	return wa.db.Count(tableName, condition)
}

// Min 获取最小值
func (wa *WeDBAdapter) Min(tableName string, column string, condition string) (interface{}, error) {
	if wa.db == nil {
		return nil, fmt.Errorf("database not opened")
	}
	return wa.db.Min(tableName, column, condition)
}

// Max 获取最大值
func (wa *WeDBAdapter) Max(tableName string, column string, condition string) (interface{}, error) {
	if wa.db == nil {
		return nil, fmt.Errorf("database not opened")
	}
	return wa.db.Max(tableName, column, condition)
}

// Sum 求和
func (wa *WeDBAdapter) Sum(tableName string, column string, condition string) (float64, error) {
	if wa.db == nil {
		return 0, fmt.Errorf("database not opened")
	}
	return wa.db.Sum(tableName, column, condition)
}

// Avg 计算平均值
func (wa *WeDBAdapter) Avg(tableName string, column string, condition string) (float64, error) {
	if wa.db == nil {
		return 0, fmt.Errorf("database not opened")
	}
	return wa.db.Avg(tableName, column, condition)
}

// GetTableStats 获取表统计信息
func (wa *WeDBAdapter) GetTableStats(tableName string) (*api.TableStats, error) {
	if wa.db == nil {
		return nil, fmt.Errorf("database not opened")
	}
	return wa.db.GetTableStats(tableName)
}

// GetColumnStats 获取列统计信息
func (wa *WeDBAdapter) GetColumnStats(tableName string, column string) (*api.ColumnStats, error) {
	if wa.db == nil {
		return nil, fmt.Errorf("database not opened")
	}
	return wa.db.GetColumnStats(tableName, column)
}

// Begin 开始事务
func (wa *WeDBAdapter) Begin() (api.Transaction, error) {
	if wa.db == nil {
		return nil, fmt.Errorf("database not opened")
	}
	return wa.db.Begin()
}

// BeginTx 开始事务（带上下文）
func (wa *WeDBAdapter) BeginTx(ctx context.Context, opts *api.TxOptions) (api.Transaction, error) {
	if wa.db == nil {
		return nil, fmt.Errorf("database not opened")
	}
	return wa.db.BeginTx(ctx, opts)
}

// Ping 检查连接
func (wa *WeDBAdapter) Ping() error {
	if wa.db == nil {
		return fmt.Errorf("database not opened")
	}
	return wa.db.Ping()
}

// IsClosed 检查是否已关闭
func (wa *WeDBAdapter) IsClosed() bool {
	if wa.db == nil {
		return true
	}
	return wa.db.IsClosed()
}

// GetDatabase 获取底层数据库实例
func (wa *WeDBAdapter) GetDatabase() *storage.WeDBDatabase {
	return wa.db
}