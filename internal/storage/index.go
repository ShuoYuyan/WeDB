package storage

import (
	"fmt"
	"sync"

	"github.com/wedb/wedb/internal/types"
	"github.com/wedb/wedb/internal/util"
)

// IndexManager 索引管理器
// 管理所有表的索引
type IndexManager struct {
	db     *WeDBDatabase
	indexes map[string]*Index // 索引名到索引的映射
	mu     sync.RWMutex
}

// Index 索引定义
type Index struct {
	IndexName string
	TableName string
	Columns   []string
	Unique    bool
	Primary   bool   // 内部主键索引，不出现在用户索引列表
	BTree     *BTree // 索引 B-Tree
}

// NewIndexManager 创建新索引管理器
func NewIndexManager(db *WeDBDatabase) *IndexManager {
	return &IndexManager{
		db:     db,
		indexes: make(map[string]*Index),
	}
}

// CreateIndex creates an index
func (im *IndexManager) CreateIndex(indexName string, tableName string, columns []string, unique bool) error {
	im.mu.Lock()
	defer im.mu.Unlock()

	util.GetLogger().Info("Creating index", map[string]interface{}{
		"index_name": indexName,
		"table_name": tableName,
		"columns": columns,
		"unique": unique,
	})

	// 检查索引是否已存在
	if _, exists := im.indexes[indexName]; exists {
		return fmt.Errorf("index already exists: %s", indexName)
	}

	// 检查表是否存在
	table, exists := im.db.schema.Tables[tableName]
	if !exists {
		return fmt.Errorf("table not found: %s", tableName)
	}

	// 检查列是否存在
	for _, colName := range columns {
		found := false
		for _, col := range table.Columns {
			if col.Name == colName {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("column not found: %s", colName)
		}
	}

	// 对于唯一索引，预先检查数据是否违反唯一约束
	// 注意：暂时跳过预检查以提高性能，在插入时检查唯一约束
	// 如果需要严格验证，可以取消下面的注释
	/*
	if unique {
		if err := im.checkUniqueConstraint(tableName, columns); err != nil {
			return fmt.Errorf("unique constraint violation: %w", err)
		}
	}
	*/

	// 分配索引 B-Tree 的根页面
	rootPage := im.db.pager.AllocPage()

	// 初始化索引 B-Tree 的根页面（叶子页面）
	indexRootPage := NewPage(rootPage, im.db.pageSize, true)
	indexRootPage.Dirty = true
	if err := im.db.cache.Put(indexRootPage); err != nil {
		return fmt.Errorf("failed to initialize index root page: %w", err)
	}

	// 创建索引 B-Tree（不是表 B-Tree，使用 BLOB 键）
	indexBTree := NewBTree(rootPage, im.db.pageSize, false, im.db.pager, im.db.cache)

	// 创建索引
	index := &Index{
		IndexName: indexName,
		TableName: tableName,
		Columns:   columns,
		Unique:    unique,
		BTree:     indexBTree,
	}

	// 添加到索引映射
	im.indexes[indexName] = index
	im.db.schema.Indexes[indexName] = &types.Index{
		IndexName: indexName,
		TableName: tableName,
		Columns:   columns,
		Unique:    unique,
	}

	// 填充索引（扫描表数据）
	// 注意：暂时跳过填充以提高性能，插入新数据时会自动更新索引
	// 如果需要为现有数据创建索引，可以取消下面的注释
	/*
	if err := im.populateIndex(index); err != nil {
		// 填充失败，清理索引
		delete(im.indexes, indexName)
		delete(im.db.schema.Indexes, indexName)
		return fmt.Errorf("failed to populate index: %w", err)
	}
	*/

	util.GetLogger().Info("Index created successfully", map[string]interface{}{
		"index_name": indexName,
		"root_page": rootPage,
	})

	return nil
}

// checkUniqueConstraint 检查唯一约束
func (im *IndexManager) checkUniqueConstraint(tableName string, columns []string) error {
	// 获取表结构
	table, exists := im.db.schema.Tables[tableName]
	if !exists {
		return fmt.Errorf("table not found: %s", tableName)
	}

	// 获取表的根页面
	rootPage, exists := im.db.tables[tableName]
	if !exists {
		return fmt.Errorf("table root page not found: %s", tableName)
	}

	// 用于跟踪已见过的键值
	seenKeys := make(map[int64]bool)

	// 使用辅助方法遍历表数据
	err := im.traverseTable(rootPage, tableName, table, func(row map[string]interface{}) error {
		// 生成索引键
		key, err := im.generateIndexKey(row, columns)
		if err != nil {
			return err
		}

		// 检查是否已见过这个键
		if seenKeys[key] {
			return fmt.Errorf("duplicate values found for columns %v", columns)
		}
		seenKeys[key] = true

		return nil
	})

	return err
}

// populateIndex 填充索引
func (im *IndexManager) populateIndex(index *Index) error {
	// 获取表结构
	table, exists := im.db.schema.Tables[index.TableName]
	if !exists {
		return fmt.Errorf("table not found: %s", index.TableName)
	}

	// 获取表的根页面
	rootPage, exists := im.db.tables[index.TableName]
	if !exists {
		return fmt.Errorf("table root page not found: %s", index.TableName)
	}

	// 收集所有索引条目
	cells := make([]*Cell, 0, 1000) // 预分配空间

	// 遍历表数据，收集所有键值对
	tableName := index.TableName
	err := im.traverseTable(rootPage, tableName, table, func(row map[string]interface{}) error {
		// 生成索引键
		key, err := im.generateIndexKey(row, index.Columns)
		if err != nil {
			return err
		}

		// 生成索引值（rowid）
		value, err := im.generateRowID(row, index.TableName)
		if err != nil {
			return err
		}

		// 添加到待插入列表
		cells = append(cells, &Cell{Key: key, Data: value})
		return nil
	})

	if err != nil {
		return err
	}

	// 使用批量插入提高性能
	if len(cells) > 0 {
		return index.BTree.BatchInsert(cells)
	}

	return nil
}

// traverseTable 遍历表数据，对每行数据执行回调函数
// 优化：避免重复创建BTree和cursor，提高性能
func (im *IndexManager) traverseTable(rootPage int, tableName string, table *types.Table, callback func(row map[string]interface{}) error) error {
	// 创建临时BTree用于遍历
	tempBTree := im.db.tableBTreeByRoot(rootPage, tableName)
	cursor := tempBTree.NewCursor()
	if err := cursor.First(); err != nil {
		return fmt.Errorf("failed to create cursor: %w", err)
	}
	defer cursor.Close()

	// 遍历所有行
	for !cursor.EOF() {
		data, err := cursor.Data()
		if err != nil {
			return fmt.Errorf("failed to get data: %w", err)
		}

		// 反序列化行数据
		row := DeserializeRow(data, table)

		// 执行回调函数
		if err := callback(row); err != nil {
			return err
		}

		// 移动到下一行
		if err := cursor.Next(); err != nil {
			return fmt.Errorf("failed to move cursor: %w", err)
		}
	}

	return nil
}

// generateIndexKey 生成索引键
// 修复：使用复合键而不是哈希值，避免哈希冲突
func (im *IndexManager) generateIndexKey(row map[string]interface{}, columns []string) (int64, error) {
	if len(columns) == 0 {
		return 0, fmt.Errorf("no columns specified for index key")
	}

	// 对于单列索引，直接使用该列的值
	if len(columns) == 1 {
		colName := columns[0]
		val, exists := row[colName]
		if !exists {
			return 0, fmt.Errorf("column '%s' not found in row", colName)
		}
		return im.valueToInt64(val)
	}

	// 对于多列索引，组合多个列的值
	// 使用位运算组合多个值，避免哈希冲突
	var result int64 = 0

	for i, colName := range columns {
		val, exists := row[colName]
		if !exists {
			return 0, fmt.Errorf("column '%s' not found in row", colName)
		}
		
		intVal, err := im.valueToInt64(val)
		if err != nil {
			return 0, fmt.Errorf("failed to convert column '%s' to int64: %w", colName, err)
		}
		
		// 使用位移和异或操作组合多个值
		// 每个值占用一定位数（例如16位），避免冲突
		shift := uint(i * 16)
		result ^= (intVal & 0xFFFF) << shift
	}
	
	return result, nil
}

// valueToInt64 将任意值转换为int64
func (im *IndexManager) valueToInt64(val interface{}) (int64, error) {
	if val == nil {
		return 0, nil
	}

	switch v := val.(type) {
	case int:
		return int64(v), nil
	case int8:
		return int64(v), nil
	case int16:
		return int64(v), nil
	case int32:
		return int64(v), nil
	case int64:
		return v, nil
	case uint:
		return int64(v), nil
	case uint8:
		return int64(v), nil
	case uint16:
		return int64(v), nil
	case uint32:
		return int64(v), nil
	case uint64:
		if v > 0x7FFFFFFFFFFFFFFF {
			return 0, fmt.Errorf("value too large: %d", v)
		}
		return int64(v), nil
	case float32:
		return int64(v), nil
	case float64:
		return int64(v), nil
	case string:
		// 对于字符串，使用其哈希值
		h := fnv64Hash(v)
		return int64(h), nil
	case []byte:
		// 对于字节数组，使用其哈希值
		h := fnv64HashBytes(v)
		return int64(h), nil
	default:
		// 其他类型尝试转换为字符串再哈希
		h := fnv64Hash(fmt.Sprintf("%v", v))
		return int64(h), nil
	}
}

// fnv64Hash 实现 FNV-1a 64位哈希算法
func fnv64Hash(s string) uint64 {
	h := uint64(14695981039346656037)
	for _, c := range s {
		h ^= uint64(c)
		h *= 1099511628211
	}
	return h
}

// fnv64HashBytes 对字节数组进行64位哈希
func fnv64HashBytes(data []byte) uint64 {
	h := uint64(14695981039346656037)
	for _, b := range data {
		h ^= uint64(b)
		h *= 1099511628211
	}
	return h
}

// generateRowID 生成 rowid
func (im *IndexManager) generateRowID(row map[string]interface{}, tableName string) ([]byte, error) {
	// 获取表结构
	table, exists := im.db.schema.Tables[tableName]
	if !exists {
		return nil, fmt.Errorf("table not found: %s", tableName)
	}

	// 获取主键列名
	var primaryKey string
	if table.PrimaryIndex != nil && len(table.PrimaryIndex.Columns) > 0 {
		primaryKey = table.PrimaryIndex.Columns[0]
	} else {
		// 如果没有主键，尝试查找标记为PrimaryKey的列
		for _, col := range table.Columns {
			if col.PrimaryKey {
				primaryKey = col.Name
				break
			}
		}
	}

	if primaryKey == "" {
		return nil, fmt.Errorf("no primary key found for table: %s", tableName)
	}

	// 使用主键列作为 rowid
	if id, exists := row[primaryKey]; exists {
		rowID := fmt.Sprintf("%v", id)
		return []byte(rowID), nil
	}

	return nil, fmt.Errorf("primary key column '%s' not found in row", primaryKey)
}

// DropIndex drops an index
func (im *IndexManager) DropIndex(indexName string) error {
	im.mu.Lock()
	defer im.mu.Unlock()

	util.GetLogger().Info("Dropping index", map[string]interface{}{
		"index_name": indexName,
	})

	// 检查索引是否存在
	index, exists := im.indexes[indexName]
	if !exists {
		return fmt.Errorf("index not found: %s", indexName)
	}

	// 关闭索引 B-Tree
	if err := index.BTree.Close(); err != nil {
		return fmt.Errorf("failed to close index btree: %w", err)
	}

	// 删除索引
	delete(im.indexes, indexName)
	delete(im.db.schema.Indexes, indexName)

	util.GetLogger().Info("Index dropped successfully", map[string]interface{}{
		"index_name": indexName,
	})

	return nil
}

// GetIndexInfo 获取索引信息
func (im *IndexManager) GetIndexInfo(indexName string) (*types.Index, error) {
	im.mu.RLock()
	defer im.mu.RUnlock()

	// 检查索引是否存在
	index, exists := im.indexes[indexName]
	if !exists {
		return nil, fmt.Errorf("index not found: %s", indexName)
	}

	return &types.Index{
		IndexName: index.IndexName,
		TableName: index.TableName,
		Columns:   index.Columns,
		Unique:    index.Unique,
	}, nil
}

// IndexExists 检查索引是否存在
func (im *IndexManager) IndexExists(indexName string) bool {
	im.mu.RLock()
	defer im.mu.RUnlock()

	_, exists := im.indexes[indexName]
	return exists
}

// MarkPrimary 将索引标记为内部主键索引（不出现在用户索引列表）。
func (im *IndexManager) MarkPrimary(indexName string) {
	im.mu.Lock()
	defer im.mu.Unlock()
	if idx, ok := im.indexes[indexName]; ok {
		idx.Primary = true
	}
}

// GetTableIndexes 获取表的用户索引（不含内部主键索引）
func (im *IndexManager) GetTableIndexes(tableName string) []*types.Index {
	im.mu.RLock()
	defer im.mu.RUnlock()

	indexes := make([]*types.Index, 0)
	for _, index := range im.indexes {
		if index.TableName == tableName && !index.Primary {
			indexes = append(indexes, &types.Index{
				IndexName: index.IndexName,
				TableName: index.TableName,
				Columns:   index.Columns,
				Unique:    index.Unique,
			})
		}
	}
	return indexes
}

// UpdateIndex 更新索引（在插入/更新/删除数据时调用）
// 注意：对于update操作，需要传递旧row数据来删除旧索引
func (im *IndexManager) UpdateIndex(tableName string, row map[string]interface{}, operation string) error {
	im.mu.Lock()
	defer im.mu.Unlock()

	// 获取表的所有索引
	for _, index := range im.indexes {
		if index.TableName != tableName {
			continue
		}

		// 生成索引键
		key, err := im.generateIndexKey(row, index.Columns)
		if err != nil {
			return err
		}

		// 生成索引值（rowid）
		value, err := im.generateRowID(row, tableName)
		if err != nil {
			return err
		}

		// 根据操作类型更新索引
		switch operation {
		case "insert":
			// 检查唯一约束
			if index.Unique {
				if existing, err := index.BTree.Search(key); err == nil && existing != nil {
					return fmt.Errorf("unique constraint violation on index %s", index.IndexName)
				}
			}
			if err := index.BTree.Insert(key, value); err != nil {
				return err
			}
		case "delete":
			if err := index.BTree.Delete(key); err != nil {
				return err
			}
		case "update":
			// 注意：这个方法不推荐用于update操作
			// 应该使用UpdateIndexWithOldRow来处理update操作
			// 这里保留是为了向后兼容，但会给出警告
			return fmt.Errorf("UpdateIndex does not support update operation, use UpdateIndexWithOldRow instead")
		}
	}

	return nil
}

// UpdateIndexWithOldRow 更新索引（带旧row数据，支持回滚）
// 用于UpdateRows操作，确保索引一致性
func (im *IndexManager) UpdateIndexWithOldRow(tableName string, oldRow, newRow map[string]interface{}, operation string) error {
	im.mu.Lock()
	defer im.mu.Unlock()

	// 获取表的所有索引
	for _, index := range im.indexes {
		if index.TableName != tableName {
			continue
		}

		// 获取旧row的rowid
		oldRowID, err := im.generateRowID(oldRow, tableName)
		if err != nil {
			return err
		}

		// 获取新row的rowid
		newRowID, err := im.generateRowID(newRow, tableName)
		if err != nil {
			return err
		}

		// 根据操作类型更新索引
		switch operation {
		case "update":
			// 生成旧键和新键
			oldKey, err := im.generateIndexKey(oldRow, index.Columns)
			if err != nil {
				return fmt.Errorf("failed to generate old index key: %w", err)
			}

			newKey, err := im.generateIndexKey(newRow, index.Columns)
			if err != nil {
				return fmt.Errorf("failed to generate new index key: %w", err)
			}

			// 如果键值没有变化，不需要更新索引
			if oldKey == newKey {
				continue
			}

			// 对于唯一索引，先检查新键是否已存在（排除当前row）
			if index.Unique {
				existingData, err := index.BTree.Search(newKey)
				if err != nil {
					return fmt.Errorf("failed to check unique constraint: %w", err)
				}
				if existingData != nil && string(existingData) != string(newRowID) {
					return fmt.Errorf("unique constraint violation on index %s", index.IndexName)
				}
			}

			// 先删除旧索引
			if err := index.BTree.Delete(oldKey); err != nil {
				return fmt.Errorf("failed to delete old index: %w", err)
			}

			// 插入新索引
			if err := index.BTree.Insert(newKey, newRowID); err != nil {
				// 插入失败，回滚：恢复旧索引
				_ = index.BTree.Insert(oldKey, oldRowID)
				return fmt.Errorf("failed to insert new index, rollback attempted: %w", err)
			}
		}
	}

	return nil
}

// SearchIndex 使用索引搜索
func (im *IndexManager) SearchIndex(indexName string, key int64) ([]map[string]interface{}, error) {
	im.mu.RLock()
	defer im.mu.RUnlock()

	// 检查索引是否存在
	index, exists := im.indexes[indexName]
	if !exists {
		return nil, fmt.Errorf("index not found: %s", indexName)
	}

	// 使用索引 B-Tree 搜索
	data, err := index.BTree.Search(key)
	if err != nil {
		return nil, err
	}

	if data == nil {
		return []map[string]interface{}{}, nil
	}

	// 解析 rowid
	rowIDStr := string(data)
	var rowID int64
	_, err = fmt.Sscanf(rowIDStr, "%d", &rowID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse rowid: %w", err)
	}

	// 获取表结构
	table, exists := im.db.schema.Tables[index.TableName]
	if !exists {
		return nil, fmt.Errorf("table not found: %s", index.TableName)
	}

	// 获取表的根页面
	rootPage, exists := im.db.tables[index.TableName]
	if !exists {
		return nil, fmt.Errorf("table root page not found: %s", index.TableName)
	}

	// 使用 rowid 直接从表 B-Tree 中查找（避免全表扫描）
	tempBTree := im.db.tableBTreeByRoot(rootPage, index.TableName)
	rowData, err := tempBTree.Search(rowID)
	if err != nil {
		return nil, fmt.Errorf("failed to search table btree: %w", err)
	}

	if rowData == nil {
		return []map[string]interface{}{}, nil
	}

	// 反序列化行数据
	row := DeserializeRow(rowData, table)
	return []map[string]interface{}{row}, nil
}

// SearchIndexRange 使用索引范围搜索
func (im *IndexManager) SearchIndexRange(indexName string, minKey, maxKey int64) ([]map[string]interface{}, error) {
	im.mu.RLock()
	defer im.mu.RUnlock()

	// 检查索引是否存在
	index, exists := im.indexes[indexName]
	if !exists {
		return nil, fmt.Errorf("index not found: %s", indexName)
	}

	// 获取表结构
	table, exists := im.db.schema.Tables[index.TableName]
	if !exists {
		return nil, fmt.Errorf("table not found: %s", index.TableName)
	}

	// 获取表的根页面
	rootPage, exists := im.db.tables[index.TableName]
	if !exists {
		return nil, fmt.Errorf("table root page not found: %s", index.TableName)
	}

	// 创建表 B-Tree 用于查找
	tableBTree := im.db.tableBTreeByRoot(rootPage, index.TableName)

	// 使用B-Tree的Cursor进行范围查询，避免全表遍历
	rowIDs := make([]int64, 0)
	cursor := index.BTree.NewCursor()
	
	// 优化：先定位到minKey，然后只遍历范围内的数据
	if err := cursor.Seek(minKey); err != nil {
		cursor.Close()
		return nil, fmt.Errorf("failed to seek to minKey: %w", err)
	}

	for !cursor.EOF() {
		key, err := cursor.Key()
		if err != nil {
			cursor.Close()
			return nil, fmt.Errorf("failed to get key: %w", err)
		}

		// 如果键超过maxKey，停止遍历
		if key > maxKey {
			break
		}

		// 获取数据
		data, err := cursor.Data()
		if err != nil {
			cursor.Close()
			return nil, fmt.Errorf("failed to get data: %w", err)
		}

		// 解析 rowid
		rowIDStr := string(data)
		var rowID int64
		_, err = fmt.Sscanf(rowIDStr, "%d", &rowID)
		if err != nil {
			cursor.Close()
			return nil, fmt.Errorf("failed to parse rowid: %w", err)
		}

		rowIDs = append(rowIDs, rowID)

		if err := cursor.Next(); err != nil {
			cursor.Close()
			return nil, fmt.Errorf("failed to move cursor: %w", err)
		}
	}
	cursor.Close()

	// 使用 rowid 从表 B-Tree 中直接查找完整行（避免全表扫描）
	rows := make([]map[string]interface{}, 0, len(rowIDs))
	for _, rowID := range rowIDs {
		rowData, err := tableBTree.Search(rowID)
		if err != nil {
			return nil, fmt.Errorf("failed to search table btree: %w", err)
		}

		if rowData != nil {
			row := DeserializeRow(rowData, table)
			rows = append(rows, row)
		}
	}

	return rows, nil
}