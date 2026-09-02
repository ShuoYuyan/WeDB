package storage

import (
	"fmt"

	"github.com/wedb/wedb/internal/api"
)

// CreateIndex 创建索引
func (db *WeDBDatabase) CreateIndex(tableName string, index *api.IndexInfo) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.closed {
		return fmt.Errorf("database is closed")
	}

	// 验证表名
	if tableName == "" {
		return fmt.Errorf("table name cannot be empty")
	}

	// 验证索引
	if index == nil {
		return fmt.Errorf("index cannot be nil")
	}
	if index.IndexName == "" {
		return fmt.Errorf("index name cannot be empty")
	}
	if len(index.IndexName) > 255 {
		return fmt.Errorf("index name too long: maximum 255 characters")
	}
	if len(index.Columns) == 0 {
		return fmt.Errorf("index must have at least one column")
	}
	if len(index.Columns) > 16 {
		return fmt.Errorf("index cannot have more than 16 columns")
	}

	// 检查表是否存在（已持有写锁，使用无锁版本）
	if !db.tableExistsLocked(tableName) {
		return fmt.Errorf("table not found: %s", tableName)
	}

	// 检查索引名是否已存在
	if db.indexManager.IndexExists(index.IndexName) {
		return fmt.Errorf("index already exists: %s", index.IndexName)
	}

	// 使用索引管理器创建索引
	return db.indexManager.CreateIndex(index.IndexName, tableName, index.Columns, index.Unique)
}

// DropIndex 删除索引
func (db *WeDBDatabase) DropIndex(tableName string, indexName string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.closed {
		return fmt.Errorf("database is closed")
	}

	// 验证表名
	if tableName == "" {
		return fmt.Errorf("table name cannot be empty")
	}

	// 验证索引名
	if indexName == "" {
		return fmt.Errorf("index name cannot be empty")
	}

	// 检查表是否存在（已持有写锁，使用无锁版本）
	if !db.tableExistsLocked(tableName) {
		return fmt.Errorf("table not found: %s", tableName)
	}

	// 检查索引是否存在
	if !db.indexManager.IndexExists(indexName) {
		return fmt.Errorf("index not found: %s", indexName)
	}

	return db.indexManager.DropIndex(indexName)
}

// GetIndexInfo 获取索引信息
func (db *WeDBDatabase) GetIndexInfo(tableName string) ([]api.IndexInfo, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.closed {
		return nil, fmt.Errorf("database is closed")
	}

	// 获取表的所有索引
	indexes := db.indexManager.GetTableIndexes(tableName)

	// 转换为 api.IndexInfo
	result := make([]api.IndexInfo, 0, len(indexes))
	for _, idx := range indexes {
		result = append(result, api.IndexInfo{
			IndexName: idx.IndexName,
			Columns:   idx.Columns,
			Unique:    idx.Unique,
			Type:      api.TypeBTree, // 默认为 B-Tree 索引
		})
	}

	return result, nil
}

// IndexExists 检查索引是否存在
func (db *WeDBDatabase) IndexExists(tableName string, indexName string) bool {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.closed {
		return false
	}

	// 使用索引管理器检查索引是否存在
	return db.indexManager.IndexExists(indexName)
}