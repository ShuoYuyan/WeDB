package storage

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wedb/wedb/internal/api"
	"github.com/wedb/wedb/internal/types"
	"github.com/wedb/wedb/internal/util"
)

// WeDBDatabase WeDB Database Implementation
type WeDBDatabase struct {
	filePath     string
	pageSize     int
	pager        *Pager
	cache        *PageCache
	btree        *BTree
	schema       *types.Schema
	tables       map[string]int
	indexManager *IndexManager
	nextRowID    int64 // 递增的行 ID
	tableRowIDs  map[string]int64 // 表级别的自增ID计数器
	batchBuffer  *BatchWriteBuffer // 批量写入缓冲
	mu           sync.RWMutex
	closed       bool
	// 指标收集
	metrics      *DatabaseMetrics
	// 熔断器
	circuitBreaker *CircuitBreaker
	// 事务管理器（MVCC）
	txManager    *TransactionManager
	// 写事务闸门：同一时刻只允许一个手动写事务（容量1）。
	// 后来的写事务在此排队，而不是立即失败 —— 与 SQLite 单写者模型一致。
	writeGate   chan struct{}
	tlMu        sync.RWMutex             // 表级锁注册表保护
	tableLocks  map[string]*sync.RWMutex // 每表一把读写锁（跨表并行写）
	curWriteTx  *WeDBTransaction         // 当前活跃写事务（持有闸门者）
	activeReadTx *WeDBTransaction // 最近开启且未结束的读事务（环境语义）
}

// DatabaseMetrics 数据库性能指标
// 用于收集和统计数据库的各种性能数据，包括查询次数、操作计数、缓存命中率等
type DatabaseMetrics struct {
	QueryCount      atomic.Int64 // 查询总数
	QueryTime       atomic.Int64 // 查询总时间（纳秒）
	InsertCount     atomic.Int64 // 插入操作计数
	UpdateCount     atomic.Int64 // 更新操作计数
	DeleteCount     atomic.Int64 // 删除操作计数
	SelectCount     atomic.Int64 // 查询操作计数
	CacheHitCount   atomic.Int64 // 缓存命中次数
	CacheMissCount  atomic.Int64 // 缓存未命中次数
	StartTime       time.Time    // 指标收集开始时间
}

// NewDatabaseMetrics 创建新的指标收集器
// 初始化一个数据库指标收集器，记录开始时间
func NewDatabaseMetrics() *DatabaseMetrics {
	return &DatabaseMetrics{
		StartTime: time.Now(),
	}
}

// GetCacheHitRate 计算并返回缓存命中率
// 返回值范围：0.0 - 100.0
func (dm *DatabaseMetrics) GetCacheHitRate() float64 {
	hits := dm.CacheHitCount.Load()
	misses := dm.CacheMissCount.Load()
	total := hits + misses
	if total == 0 {
		return 0.0
	}
	return float64(hits) / float64(total) * 100.0
}

// GetAvgQueryTime 计算并返回平均查询时间（毫秒）
// 返回所有查询的平均耗时
func (dm *DatabaseMetrics) GetAvgQueryTime() float64 {
	count := dm.QueryCount.Load()
	if count == 0 {
		return 0.0
	}
	return float64(dm.QueryTime.Load()) / float64(count) / 1000000.0
}

// CircuitBreakerState 熔断器状态
type CircuitBreakerState int

const (
	StateClosed CircuitBreakerState = iota // 关闭状态：正常工作，允许所有请求
	StateOpen                               // 打开状态：熔断触发，拒绝所有请求
	StateHalfOpen                           // 半开状态：尝试恢复，允许部分请求测试
)

// CircuitBreaker 熔断器
// 用于在错误累积时自动断开连接，防止系统过载
// 三态机制：关闭 -> 打开 -> 半开 -> 关闭
type CircuitBreaker struct {
	state        CircuitBreakerState  // 当前状态
	failureCount atomic.Int64         // 失败计数
	successCount atomic.Int64         // 成功计数（半开状态使用）
	threshold    int64                // 失败阈值（达到此值后打开熔断器）
	timeout      time.Duration        // 熔断超时时间（打开状态后等待多久尝试恢复）
	lastFailure  atomic.Value         // 最后一次失败时间（time.Time）
	mu           sync.RWMutex         // 读写锁
}

// NewCircuitBreaker 创建新的熔断器
// 参数：
//   - threshold: 失败阈值，达到此值后熔断器打开
//   - timeout: 熔断超时时间，打开后等待此时间才尝试恢复
func NewCircuitBreaker(threshold int64, timeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		state:     StateClosed,
		threshold: threshold,
		timeout:   timeout,
	}
}

// Allow 检查是否允许执行操作
// 返回true表示允许操作，false表示熔断器阻止操作
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.RLock()
	state := cb.state
	cb.mu.RUnlock()

	switch state {
	case StateClosed:
		return true
	case StateOpen:
		// 检查是否可以尝试恢复
		if time.Since(cb.lastFailure.Load().(time.Time)) > cb.timeout {
			cb.mu.Lock()
			cb.state = StateHalfOpen
			cb.successCount.Store(0)
			cb.mu.Unlock()
			return true
		}
		return false
	case StateHalfOpen:
		return true
	default:
		return false
	}
}

// RecordSuccess 记录操作成功
// 在半开状态下，连续成功3次后会恢复到关闭状态
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.successCount.Add(1)

	if cb.state == StateHalfOpen {
		// 半开状态下，如果有足够的成功，恢复到关闭状态
		if cb.successCount.Load() >= 3 {
			cb.state = StateClosed
			cb.failureCount.Store(0)
		}
	}
}

// RecordFailure 记录操作失败
// 失败计数增加，达到阈值后打开熔断器
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureCount.Add(1)
	cb.lastFailure.Store(time.Now())

	if cb.state == StateClosed {
		// 达到失败阈值，打开熔断器
		if cb.failureCount.Load() >= cb.threshold {
			cb.state = StateOpen
			log.Printf("[CircuitBreaker] Circuit opened due to %d failures", cb.threshold)
		}
	} else if cb.state == StateHalfOpen {
		// 半开状态下失败，重新打开熔断器
		cb.state = StateOpen
		log.Printf("[CircuitBreaker] Circuit reopened during half-open state")
	}
}

// GetState 获取当前熔断器状态
// 返回当前状态（关闭/打开/半开）
func (cb *CircuitBreaker) GetState() CircuitBreakerState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// dbSnapshot 数据库快照（用于MVCC）
type dbSnapshot struct {
	timestamp time.Time          // 快照时间戳
	txID      int64             // 事务ID
	pages     map[int]*PageSnapshot // 页面快照映射
	valid     bool              // 快照是否有效
	mu        sync.RWMutex
}

// createSnapshot 创建数据库快照
func (db *WeDBDatabase) createSnapshot() *dbSnapshot {
	db.mu.RLock()
	defer db.mu.RUnlock()

	snapshot := &dbSnapshot{
		timestamp: time.Now(),
		pages:     make(map[int]*PageSnapshot),
		valid:     true,
	}

	// 为每个活跃页面创建快照
	for pageNum := range db.cache.pages {
		snapshot.pages[pageNum] = &PageSnapshot{
			pageNum:   pageNum,
			timestamp: time.Now(),
		}
	}

	return snapshot
}

// NewWeDBDatabase creates a new database with default configuration
func NewWeDBDatabase(filePath string, pageSize int) (*WeDBDatabase, error) {
	return newWeDBDatabase(filePath, pageSize, nil)
}

// NewWeDBDatabaseSecure 打开/创建加密数据库（AES-256-XTS 静态加密）。
// passphrase 为空且目标库已加密时返回 ErrWrongPassphrase。
func NewWeDBDatabaseSecure(filePath string, pageSize int, passphrase []byte) (*WeDBDatabase, error) {
	return newWeDBDatabase(filePath, pageSize, passphrase)
}

func newWeDBDatabase(filePath string, pageSize int, passphrase []byte) (*WeDBDatabase, error) {
	if pageSize <= 0 {
		pageSize = 4096
	}

	// 初始化日志系统
	logFile := filepath.Join(filepath.Dir(filePath), "wedb.log")
	// Skip the structured logger when called from the ODBC driver
	// path; the manager process can deadlock on the logger's mutex
	// or file handle when the same file is being held by a
	// previous process. The legacy log.Printf output still appears.
	if os.Getenv("WEDB_DISABLE_STRUCTLOG") == "" {
		if err := util.InitLogging(util.INFO, logFile); err != nil {
			// 如果日志初始化失败，使用标准log作为后备
			log.Printf("[WeDB] Failed to initialize logging, using std log: %v", err)
		} else {
			util.GetLogger().Info("WeDB initializing", map[string]interface{}{
				"database": filePath,
				"pageSize": pageSize,
			})
		}
	}

	var pager *Pager
	{
		var perr error
		if len(passphrase) > 0 {
			pager, perr = NewPagerSecure(filePath, pageSize, passphrase)
		} else if fileExists(filePath+xkeySuffix) {
			// 已加密库：无口令打开或口令错误都会立即失败
			pager, perr = NewPagerSecure(filePath, pageSize, nil)
		} else {
			pager, perr = NewPager(filePath, pageSize)
		}
		if perr != nil {
			if perr == ErrWrongPassphrase {
				return nil, perr // 哨兵直传，调用方可用 == 判定（兼容 Go1.10 镜像无 %w）
			}
			util.GetLogger().Error("Failed to create pager", map[string]interface{}{"error": perr})
			return nil, fmt.Errorf("failed to create pager: %w", perr)
		}
	}

	cache := NewPageCache(1000, pager)

	db := &WeDBDatabase{
			filePath:       filePath,
			pageSize:       pageSize,
			pager:          pager,
			cache:          cache,
			schema:         types.NewSchema(),
			tables:         make(map[string]int),
			indexManager:   NewIndexManager(nil),
			tableRowIDs:    make(map[string]int64),
			closed:         false,
			metrics:        NewDatabaseMetrics(),
			circuitBreaker: NewCircuitBreaker(10, 30*time.Second),
			txManager:      NewTransactionManager(),
			writeGate:      make(chan struct{}, 1),
			tableLocks:     make(map[string]*sync.RWMutex),
		}
	
	// 创建批量写入缓冲
	db.batchBuffer = NewBatchWriteBuffer(DefaultBatchConfig(), func(pageNum uint32, data []byte, offset int64) error {
		// 写入页面到磁盘
		return db.pager.WritePage(int(pageNum), data)
	})
	db.indexManager.db = db

	if os.Getenv("WEDB_DISABLE_STRUCTLOG") == "" {
		util.GetLogger().Info("Database components created", map[string]interface{}{
			"cacheSize": 1000,
			"circuitBreaker": "enabled",
		})
	}

	// 尝试加载元数据
	hasMetadata := false
	if err := db.loadMetadata(); err != nil {
		// 如果元数据文件不存在，这是一个新数据库，忽略错误
		if err != ErrNoMetadata {
			util.GetLogger().Error("Failed to load metadata", map[string]interface{}{"error": err})
			return nil, fmt.Errorf("failed to load metadata: %v", err)
		}
		util.GetLogger().Info("No existing metadata, creating new database", nil)
	} else {
		// 元数据加载成功，这是一个已存在的数据库
		hasMetadata = true
		log.Printf("[WeDB] Loaded existing database metadata")
	}

	// 如果没有元数据，创建新的根页面
	if !hasMetadata {
		// 初始化根页面（叶子页面）
		rootPage := NewPage(1, pageSize, true)
		rootPage.Dirty = true
		if err := cache.Put(rootPage); err != nil {
			log.Printf("[WeDB] Failed to initialize root page: %v", err)
			return nil, fmt.Errorf("failed to initialize root page: %v", err)
		}

		db.btree = NewBTree(1, pageSize, true, pager, cache)
		if os.Getenv("WEDB_DISABLE_STRUCTLOG") == "" {
			util.GetLogger().Info("Creating index", map[string]interface{}{
				"table_name": "t",
				"columns":    []string{"id"},
				"unique":     true,
				"index_name": "t_pk",
			})
		}
	} else {
		// 使用 page 1 作为默认 B-Tree（用于内部操作）
		db.btree = NewBTree(1, pageSize, true, pager, cache)
	}

	log.Printf("[WeDB] Database opened successfully: %s", filePath)
	return db, nil
}

// NewWeDBDatabaseWithConfig creates a new database with custom configuration
func NewWeDBDatabaseWithConfig(filePath string, config *api.Config) (*WeDBDatabase, error) {
	if config == nil {
		config = api.DefaultConfig()
	}

	pageSize := config.PageSize
	if pageSize <= 0 {
		pageSize = 4096
	}

	pager, err := NewPager(filePath, pageSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create pager: %v", err)
	}

	// 使用配置中的缓存大小
		cacheSize := config.CacheSize
		if cacheSize <= 0 {
			cacheSize = 1000
		}
		cache := NewPageCache(cacheSize, pager)
	
		db := &WeDBDatabase{
				filePath:     filePath,
				pageSize:     pageSize,
				pager:        pager,
				cache:        cache,
				schema:       types.NewSchema(),
				tables:       make(map[string]int),
				indexManager: NewIndexManager(nil),
				tableRowIDs:  make(map[string]int64),
				closed:       false,
			}
		
		// 根据配置创建批量写入缓冲
		batchConfig := DefaultBatchConfig()
		if config.PageSize > 0 {
			// 可以根据配置调整缓冲区大小
			batchConfig.BufferSize = int64(config.PageSize) * 100 // 页面大小 * 100
		}
		db.batchBuffer = NewBatchWriteBuffer(batchConfig, func(pageNum uint32, data []byte, offset int64) error {
			// 写入页面到磁盘
			return db.pager.WritePage(int(pageNum), data)
		})
		db.indexManager.db = db
	
		// 尝试加载元数据
		hasMetadata := false
		if err := db.loadMetadata(); err != nil {
			// 如果元数据文件不存在，这是一个新数据库，忽略错误
			if err != ErrNoMetadata {
				return nil, fmt.Errorf("failed to load metadata: %v", err)
			}
		} else {
			// 元数据加载成功，这是一个已存在的数据库
			hasMetadata = true
		}
	
		// 如果没有元数据，创建新的根页面
		if !hasMetadata {
			// 初始化根页面（叶子页面）
			rootPage := NewPage(1, pageSize, true)
			rootPage.Dirty = true
			if err := cache.Put(rootPage); err != nil {
				return nil, fmt.Errorf("failed to initialize root page: %v", err)
			}
	
			db.btree = NewBTree(1, pageSize, true, pager, cache)
		} else {
			// 使用 page 1 作为默认 B-Tree（用于内部操作）
			db.btree = NewBTree(1, pageSize, true, pager, cache)
		}
	
		return db, nil
	}
// loadMetadata 加载元数据
func (db *WeDBDatabase) loadMetadata() error {
	metadata, err := LoadMetadata(db.filePath)
	if err != nil {
		return err
	}

	// 恢复表信息
	for tableName, tableMeta := range metadata.Tables {
		// 恢复表结构
		table := &types.Table{
			TableName: tableMeta.TableName,
			Columns:   make([]*types.Column, len(tableMeta.Columns)),
		}

		for i, colMeta := range tableMeta.Columns {
			var colType types.DataType
			switch colMeta.Type {
			case "INTEGER":
				colType = types.TypeInteger
			case "REAL":
				colType = types.TypeReal
			case "TEXT":
				colType = types.TypeText
			case "BLOB":
				colType = types.TypeBlob
			default:
				colType = types.TypeNull
			}

			table.Columns[i] = &types.Column{
				Name:          colMeta.Name,
				Type:          colType,
				PrimaryKey:    colMeta.PrimaryKey,
				AutoIncrement: colMeta.AutoIncrement,
			}
		}

		// 如果有主键，恢复 PrimaryIndex
		if tableMeta.PrimaryKey != "" {
			indexName := tableName + "_pk"
			table.PrimaryIndex = &types.Index{
				IndexName: indexName,
				TableName: tableName,
				Columns:   []string{tableMeta.PrimaryKey},
				Unique:    true,
			}
		}

		db.schema.AddTable(table)
		db.tables[tableName] = tableMeta.RootPage
	}

	// 恢复表级别的ID计数器
	for tableName, rowID := range metadata.TableRowIDs {
		db.tableRowIDs[tableName] = rowID
	}

	// 恢复pager的nextPage，确保新创建的表不会使用已存在的页面号
	maxRootPage := 0
	for _, tableMeta := range metadata.Tables {
		if tableMeta.RootPage > maxRootPage {
			maxRootPage = tableMeta.RootPage
		}
	}
	if maxRootPage > 0 {
		db.pager.SetNextPage(maxRootPage + 1)
	}

	return nil
}

// validateRow 验证行数据（检查约束）
func (db *WeDBDatabase) validateRow(tableName string, row map[string]interface{}) error {
	table, exists := db.schema.Tables[tableName]
	if !exists {
		return fmt.Errorf("table not found: %s", tableName)
	}

	// 检查NOT NULL约束
	for _, col := range table.Columns {
		// 检查列是否必须提供值
		value, exists := row[col.Name]
		if !exists && !col.AutoIncrement {
			return fmt.Errorf("column '%s' cannot be NULL", col.Name)
		}

		// 检查值是否为nil
		if exists && value == nil {
			return fmt.Errorf("column '%s' cannot be NULL", col.Name)
		}
	}

	// 检查UNIQUE约束
	indexes := db.indexManager.GetTableIndexes(tableName)
	for _, index := range indexes {
		if index.Unique {
			// 生成索引键
			key, err := db.indexManager.generateIndexKey(row, index.Columns)
			if err != nil {
				return fmt.Errorf("failed to generate index key: %v", err)
			}

			// 检查索引中是否已存在该键
			rows, err := db.indexManager.SearchIndex(index.IndexName, key)
			if err != nil {
				return fmt.Errorf("failed to check unique constraint: %v", err)
			}

			// 如果找到了匹配的行，说明违反了UNIQUE约束
			if len(rows) > 0 {
				return fmt.Errorf("unique constraint violation on columns: %v", index.Columns)
			}
		}
	}

	// 检查数据类型
	for _, col := range table.Columns {
		if value, exists := row[col.Name]; exists && value != nil {
			// 类型检查：接受所有 Go 整数/浮点变体（int、int64 等都常见）
			switch col.Type {
			case types.TypeInteger:
				if !isIntegerValue(value) {
					return fmt.Errorf("column '%s' expects INTEGER, got %T", col.Name, value)
				}
			case types.TypeReal:
				if !isFloatValue(value) && !isIntegerValue(value) {
					return fmt.Errorf("column '%s' expects REAL, got %T", col.Name, value)
				}
			case types.TypeText:
				if _, ok := value.(string); !ok {
					// 也接受其他类型，转换为字符串
					// 这里不做严格检查
				}
			case types.TypeBlob:
				if _, ok := value.([]byte); !ok {
					// 也接受字符串，转换为字节
					if _, ok := value.(string); !ok {
						return fmt.Errorf("column '%s' expects BLOB, got %T", col.Name, value)
					}
				}
			}
		}
	}

	return nil
}

// isIntegerValue reports whether v is interface{} Go integer kind.
func isIntegerValue(v interface{}) bool {
	switch v.(type) {
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64:
		return true
	}
	return false
}

// isFloatValue reports whether v is interface{} Go float kind.
func isFloatValue(v interface{}) bool {
	switch v.(type) {
	case float32, float64:
		return true
	}
	return false
}

// getTableNextRowID 获取表的下一个自增ID
func (db *WeDBDatabase) getTableNextRowID(tableName string) int64 {
	// 扫描表中的所有数据，找到自增列的最大值
	rootPage, exists := db.tables[tableName]
	if exists && rootPage > 0 {
		table, tableExists := db.schema.Tables[tableName]
		if tableExists {
			tempBTree := db.tableBTreeByRoot(rootPage, tableName)
			cursor := tempBTree.NewCursor()
			if cursor != nil {
				defer cursor.Close()
				if err := cursor.First(); err == nil {
					maxID := int64(0)
					for !cursor.EOF() {
						data, err := cursor.Data()
						if err == nil {
							row := DeserializeRow(data, table)
							// 查找自增列的值
							for _, col := range table.Columns {
								if col.AutoIncrement {
									if val, ok := row[col.Name].(int64); ok && val > maxID {
										maxID = val
									}
									break
								}
							}
						}
						if err := cursor.Next(); err != nil {
							break
						}
					}
					// 返回最大ID+1
					nextID := maxID + 1
					// 更新tableRowIDs
					if nextID > db.tableRowIDs[tableName] {
						db.tableRowIDs[tableName] = nextID
					}
					return nextID
				}
			}
		}
	}

	// 如果表中没有数据，使用tableRowIDs计数器
	nextID := db.tableRowIDs[tableName] + 1
	db.tableRowIDs[tableName] = nextID
	return nextID
}

// Close closes the database
func (db *WeDBDatabase) Close() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.closed {
		return nil
	}

	log.Printf("[WeDB] Closing database: %s", db.filePath)

	if err := db.btree.Flush(); err != nil {
		log.Printf("[WeDB] Failed to flush B-Tree: %v", err)
		return err
	}

	if err := db.btree.Close(); err != nil {
		log.Printf("[WeDB] Failed to close B-Tree: %v", err)
		return err
	}

	if err := db.pager.Close(); err != nil {
		log.Printf("[WeDB] Failed to close pager: %v", err)
		return err
	}

	// 保存元数据
	if err := SaveMetadata(db.filePath, db.tables, db.schema, db.tableRowIDs); err != nil {
		log.Printf("[WeDB] Failed to save metadata: %v", err)
		return fmt.Errorf("failed to save metadata: %v", err)
	}

	db.closed = true
	log.Printf("[WeDB] Database closed successfully: %s", db.filePath)
	return nil
}

// Ping checks if database is alive
func (db *WeDBDatabase) Ping() error {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.closed {
		return fmt.Errorf("database is closed")
	}

	return nil
}

// IsClosed checks if database is closed
func (db *WeDBDatabase) IsClosed() bool {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.closed
}

// CreateTable creates a table
func (db *WeDBDatabase) CreateTable(schema *api.TableSchema) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.closed {
		return fmt.Errorf("database is closed")
	}

	// 验证表名
	if schema == nil {
		return fmt.Errorf("schema cannot be nil")
	}
	if schema.TableName == "" {
		return fmt.Errorf("table name cannot be empty")
	}

	// 验证表名格式（防止注入）
	if err := util.ValidateTableName(schema.TableName); err != nil {
		return fmt.Errorf("invalid table name: %v", err)
	}

	// 检查表是否已存在
	if _, exists := db.tables[schema.TableName]; exists {
		return fmt.Errorf("table already exists: %s", schema.TableName)
	}

	// 验证列
	if len(schema.Columns) == 0 {
		return fmt.Errorf("table must have at least one column")
	}

	// 检查列名重复并验证列名格式
	columnNames := make(map[string]bool)
	for _, col := range schema.Columns {
		if col.Name == "" {
			return fmt.Errorf("column name cannot be empty")
		}

		// 验证列名格式（防止注入）
		if err := util.ValidateColumnName(col.Name); err != nil {
			return fmt.Errorf("invalid column name '%s': %v", col.Name, err)
		}

		if columnNames[col.Name] {
			return fmt.Errorf("duplicate column name: %s", col.Name)
		}
		columnNames[col.Name] = true
	}

	// 验证主键
	if schema.PrimaryKey != "" {
		if !columnNames[schema.PrimaryKey] {
			return fmt.Errorf("primary key column '%s' does not exist", schema.PrimaryKey)
		}
	}

	table := &types.Table{
		TableName:    schema.TableName,
		Columns:      make([]*types.Column, len(schema.Columns)),
		PrimaryIndex: nil,
	}

	// 主键判定：优先 schema.PrimaryKey；否则取列上标记 PrimaryKey 的首列
	pkName := schema.PrimaryKey
	if pkName == "" {
		for _, col := range schema.Columns {
			if col.PrimaryKey {
				pkName = col.Name
				break
			}
		}
	}

	for i, col := range schema.Columns {
		table.Columns[i] = &types.Column{
			Name:          col.Name,
			Type:          convertDataType(col.Type),
			PrimaryKey:    col.Name == pkName,
			AutoIncrement: col.AutoIncrement,
		}
	}

	rootPage := db.pager.AllocPage()
	
	if rootPage <= 0 {
		return fmt.Errorf("invalid root page number: %d", rootPage)
	}
	
	// 创建新的根页面并立即写入磁盘，覆盖任何旧的数据
	newPage := NewPage(rootPage, db.pageSize, true)
	pageData, err := newPage.Serialize()
	if err != nil {
		return fmt.Errorf("failed to serialize new page: %v", err)
	}
	if len(pageData) != db.pageSize {
		return fmt.Errorf("page data size mismatch: expected %d, got %d", db.pageSize, len(pageData))
	}
	if err := db.pager.WritePage(rootPage, pageData); err != nil {
		return fmt.Errorf("failed to write new page: %v", err)
	}
	
	db.tables[schema.TableName] = rootPage
	db.schema.Tables[schema.TableName] = table

	// 如果指定了主键，创建主键索引
	if pkName != "" {
		indexName := schema.TableName + "_pk"
		if err := db.indexManager.CreateIndex(indexName, schema.TableName, []string{pkName}, true); err != nil {
			return fmt.Errorf("failed to create primary key index: %v", err)
		}
		db.indexManager.MarkPrimary(indexName)

		// 更新表的 PrimaryIndex 引用
		table.PrimaryIndex = &types.Index{
			IndexName: indexName,
			TableName: schema.TableName,
			Columns:   []string{pkName},
			Unique:    true,
			Primary:   true,
		}
	}

	return nil
}

// DropTable drops a table
func (db *WeDBDatabase) DropTable(tableName string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.closed {
		return fmt.Errorf("database is closed")
	}

	if _, exists := db.tables[tableName]; !exists {
		return fmt.Errorf("table not found: %s", tableName)
	}

	// 收集表的所有页面
	rootPage := db.tables[tableName]
	pageNums := db.collectPageNumbers(rootPage)

	// 释放所有页面
	for _, pageNum := range pageNums {
		if err := db.pager.FreePage(pageNum); err != nil {
			// 记录错误，但继续释放其他页面
			continue
		}
	}

	// 删除表引用
	delete(db.tables, tableName)
	delete(db.schema.Tables, tableName)
	delete(db.tableRowIDs, tableName)

	// 删除索引（用户索引 + 内部主键索引）
	indexes := db.indexManager.GetTableIndexes(tableName)
	for _, index := range indexes {
		_ = db.indexManager.DropIndex(index.IndexName)
	}
	_ = db.indexManager.DropIndex(tableName + "_pk")

	return nil
}

// collectPageNumbers 收集B-Tree中的所有页面号
func (db *WeDBDatabase) collectPageNumbers(rootPage int) []int {
	visited := make(map[int]bool)
	var result []int

	var traverse func(pageNum int)
	traverse = func(pageNum int) {
		if pageNum <= 0 {
			return
		}

		// 避免重复访问
		if visited[pageNum] {
			return
		}
		visited[pageNum] = true

		// 添加到结果
		result = append(result, pageNum)

		// 读取页面
		page, err := db.cache.Get(pageNum)
		if err != nil {
			return
		}

		// 如果不是叶子页面，递归遍历子页面
		if !page.Header.IsLeaf() {
			// 遍历所有子页面
			for _, cell := range page.Cells {
				if cell.LeftChild > 0 {
					traverse(int(cell.LeftChild))
				}
			}
			if page.Header.RightChild > 0 {
				traverse(int(page.Header.RightChild))
			}
		}
	}

	// 从根页面开始遍历
	traverse(rootPage)

	return result
}

// GetTableSchema gets table schema
func (db *WeDBDatabase) GetTableSchema(tableName string) (*api.TableSchema, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.closed {
		return nil, fmt.Errorf("database is closed")
	}

	table, exists := db.schema.Tables[tableName]
	if !exists {
		return nil, fmt.Errorf("table not found: %s", tableName)
	}

	resultSchema := &api.TableSchema{
		TableName:     table.TableName,
		Columns:       make([]api.ColumnSchema, len(table.Columns)),
		PrimaryKey:    "",
		AutoIncrement: false,
	}

	for i, col := range table.Columns {
		resultSchema.Columns[i] = api.ColumnSchema{
			Name:          col.Name,
			Type:          convertColumnType(col.Type),
			AutoIncrement: col.AutoIncrement,
		}

		if col.PrimaryKey {
			resultSchema.PrimaryKey = col.Name
		}

		if col.AutoIncrement {
			resultSchema.AutoIncrement = true
		}
	}

	return resultSchema, nil
}

// ListTables lists all tables
func (db *WeDBDatabase) ListTables() []string {
	db.mu.RLock()
	defer db.mu.RUnlock()

	tables := make([]string, 0, len(db.tables))
	for name := range db.tables {
		tables = append(tables, name)
	}
	return tables
}

// TableExists checks if table exists
func (db *WeDBDatabase) TableExists(tableName string) bool {
	db.mu.RLock()
	defer db.mu.RUnlock()

	return db.tableExistsLocked(tableName)
}

// tableExistsLocked is TableExists without locking; caller must hold at
// least a read lock (Go RWMutex is not reentrant).
func (db *WeDBDatabase) tableExistsLocked(tableName string) bool {
	_, exists := db.tables[tableName]
	return exists
}

// ScanTable scans a table
func (db *WeDBDatabase) ScanTable(tableName string) ([]map[string]interface{}, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	return db.scanTableLocked(tableName)
}

// tableLock 返回指定表的行级读写锁（跨表并行写的基石）。
// 锁序约定：先短暂获取 db.mu（读校验）再取表锁；持表锁期间不得再获取 db.mu 写锁。
func (db *WeDBDatabase) tableLock(tableName string) *sync.RWMutex {
	db.tlMu.Lock()
	defer db.tlMu.Unlock()
	l, ok := db.tableLocks[tableName]
	if !ok {
		l = &sync.RWMutex{}
		db.tableLocks[tableName] = l
	}
	return l
}

// tableBTreeByRoot 构造表 B-Tree 并挂接根页迁移回调：
// 分裂产生新根时立即更新 db.tables[tableName]，避免后续扫描从旧根开始。
func (db *WeDBDatabase) tableBTreeByRoot(rootPage int, tableName string) *BTree {
	bt := NewBTree(rootPage, db.pageSize, true, db.pager, db.cache)
	if bt != nil && tableName != "" {
		bt.SetRootChangeCallback(func(newRoot int) {
			db.tables[tableName] = newRoot
		})
	}
	return bt
}

// scanTableLocked is ScanTable without locking; caller must hold at least
// a read lock.
func (db *WeDBDatabase) scanTableLocked(tableName string) ([]map[string]interface{}, error) {
	if db.closed {
		return nil, fmt.Errorf("database is closed")
	}

	// 验证表名
	if tableName == "" {
		return nil, fmt.Errorf("table name cannot be empty")
	}

	// 获取表的根页面
	rootPage, exists := db.tables[tableName]
	if !exists {
		return nil, fmt.Errorf("table not found: %s", tableName)
	}

	table, exists := db.schema.Tables[tableName]
	if !exists {
		return nil, fmt.Errorf("table not found: %s", tableName)
	}

	if rootPage <= 0 {
		return nil, fmt.Errorf("invalid root page number: %d", rootPage)
	}

	// 创建临时的 B-Tree 用于扫描
	tempBTree := db.tableBTreeByRoot(rootPage, tableName)
	if tempBTree == nil {
		return nil, fmt.Errorf("failed to create B-Tree")
	}

	cursor := tempBTree.NewCursor()
	if cursor == nil {
		return nil, fmt.Errorf("failed to create cursor")
	}
	
	if err := cursor.First(); err != nil {
		return nil, fmt.Errorf("failed to create cursor: %v", err)
	}
	defer cursor.Close()

	rows := make([]map[string]interface{}, 0)
	maxRows := 1000000 // 防止内存溢出
	for !cursor.EOF() {
		data, err := cursor.Data()
		if err != nil {
			return nil, fmt.Errorf("failed to get data: %v", err)
		}

		row := DeserializeRow(data, table)
		rows = append(rows, row)

		// 检查行数限制
		if len(rows) >= maxRows {
			return nil, fmt.Errorf("too many rows, maximum %d rows allowed", maxRows)
		}

		if err := cursor.Next(); err != nil {
			return nil, fmt.Errorf("failed to move cursor: %v", err)
		}
	}

	// 应用读事务语义（快照复用 / 脏读叠加）
	rows = db.applyReadViewLocked(tableName, rows)
	return rows, nil
}
// ScanTableWithColumns scans a table with specified columns
func (db *WeDBDatabase) ScanTableWithColumns(tableName string, columns []string) ([]map[string]interface{}, error) {
	allRows, err := db.ScanTable(tableName)
	if err != nil {
		return nil, err
	}

	rows := make([]map[string]interface{}, 0, len(allRows))
	for _, row := range allRows {
		newRow := make(map[string]interface{})
		for _, col := range columns {
			if val, exists := row[col]; exists {
				newRow[col] = val
			}
		}
		rows = append(rows, newRow)
	}

	return rows, nil
}

// ScanTableWithOptions scans a table with query options (WHERE, ORDER BY, LIMIT, OFFSET)
func (db *WeDBDatabase) ScanTableWithOptions(tableName string, opts *api.QueryOptions) ([]map[string]interface{}, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.closed {
		return nil, fmt.Errorf("database is closed")
	}

	return db.scanTableWithOptionsLocked(tableName, opts)
}

// scanTableWithOptionsLocked is ScanTableWithOptions without locking;
// caller must hold at least a read lock.
func (db *WeDBDatabase) scanTableWithOptionsLocked(tableName string, opts *api.QueryOptions) ([]map[string]interface{}, error) {
	if opts == nil {
		opts = &api.QueryOptions{}
	}

	// 获取表的列名列表（用于验证）
	table, tableExists := db.schema.Tables[tableName]
	var validColumns []string
	if tableExists {
		validColumns = make([]string, 0, len(table.Columns))
		for _, col := range table.Columns {
			validColumns = append(validColumns, col.Name)
		}
	}

	// 解析 WHERE 子句（带列名验证）
	whereClause, err := ParseWhereClauseWithValidation(opts.Where, validColumns)
	if err != nil {
		return nil, fmt.Errorf("failed to parse WHERE clause: %v", err)
	}

	// 尝试使用索引优化查询
	var filteredRows []map[string]interface{}
	if indexInfo := db.selectIndexForQuery(tableName, whereClause); indexInfo != nil {
		// 使用索引查询
		filteredRows, err = db.queryUsingIndex(tableName, indexInfo, whereClause)
		if err != nil {
			return nil, fmt.Errorf("failed to query using index: %v", err)
		}
	} else {
		// 全表扫描（调用方可能已持有读锁，使用无锁内部版本）
		allRows, err := db.scanTableLocked(tableName)
		if err != nil {
			return nil, err
		}

		// 应用 WHERE 过滤
		filteredRows = make([]map[string]interface{}, 0, len(allRows))
		for _, row := range allRows {
			matched, err := whereClause.Evaluate(row)
			if err != nil {
				return nil, fmt.Errorf("failed to evaluate WHERE clause: %v", err)
			}
			if matched {
				filteredRows = append(filteredRows, row)
			}
		}
	}

	// 应用排序
	if len(opts.OrderBy) > 0 {
		if err := SortRows(filteredRows, opts.OrderBy); err != nil {
			return nil, fmt.Errorf("failed to sort rows: %v", err)
		}
	}

	// 应用 OFFSET
	if opts.Offset > 0 && opts.Offset < len(filteredRows) {
		filteredRows = filteredRows[opts.Offset:]
	}

	// 应用 LIMIT
	if opts.Limit > 0 && opts.Limit < len(filteredRows) {
		filteredRows = filteredRows[:opts.Limit]
	}

	// 应用列过滤
	if len(opts.Columns) > 0 {
		resultRows := make([]map[string]interface{}, 0, len(filteredRows))
		for _, row := range filteredRows {
			newRow := make(map[string]interface{})
			for _, col := range opts.Columns {
				if val, exists := row[col]; exists {
					newRow[col] = val
				}
			}
			resultRows = append(resultRows, newRow)
		}
		return resultRows, nil
	}

	return filteredRows, nil
}

// selectIndexForQuery 为查询选择合适的索引
func (db *WeDBDatabase) selectIndexForQuery(tableName string, whereClause *WhereClause) *types.Index {
	// 如果WHERE条件为空，不需要索引
	if len(whereClause.Conditions) == 0 {
		return nil
	}

	// 获取表的所有索引
	indexes := db.indexManager.GetTableIndexes(tableName)
	if len(indexes) == 0 {
		return nil
	}

	// 简化实现：选择第一个匹配的索引
	// 优先选择主键索引
	for _, index := range indexes {
		// 检查索引列是否在WHERE条件中
		if db.indexMatchesWhere(index, whereClause) {
			// 优先使用唯一索引或主键索引
			if index.Unique {
				return index
			}
			// 暂时返回第一个匹配的索引
			return index
		}
	}

	return nil
}

// indexMatchesWhere 检查索引是否匹配WHERE条件
func (db *WeDBDatabase) indexMatchesWhere(index *types.Index, whereClause *WhereClause) bool {
	// 获取索引的第一列（简化实现，只考虑单列索引）
	if len(index.Columns) == 0 {
		return false
	}
	indexColumn := index.Columns[0]

	// 检查WHERE条件中是否包含该列
	for _, cond := range whereClause.Conditions {
		if cond.Column == indexColumn {
			return true
		}
	}

	return false
}

// queryUsingIndex 使用索引执行查询
func (db *WeDBDatabase) queryUsingIndex(tableName string, index *types.Index, whereClause *WhereClause) ([]map[string]interface{}, error) {
	// 获取WHERE条件的值
	indexValue, err := db.getIndexValue(whereClause, index.Columns[0])
	if err != nil {
		// 如果无法获取索引值，回退到全表扫描
		return db.fallbackToFullScan(tableName, whereClause)
	}

	// 生成索引键
	indexKey, err := db.indexManager.generateIndexKey(map[string]interface{}{
		index.Columns[0]: indexValue,
	}, index.Columns)
	if err != nil {
		return nil, fmt.Errorf("failed to generate index key: %v", err)
	}

	// 使用索引查询
	rows, err := db.indexManager.SearchIndex(index.IndexName, indexKey)
	if err != nil {
		// 索引查询失败，回退到全表扫描
		return db.fallbackToFullScan(tableName, whereClause)
	}

	// 再次应用WHERE条件过滤（确保完全匹配）
	filteredRows := make([]map[string]interface{}, 0, len(rows))
	for _, row := range rows {
		matched, err := whereClause.Evaluate(row)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate WHERE clause: %v", err)
		}
		if matched {
			filteredRows = append(filteredRows, row)
		}
	}

	return filteredRows, nil
}

// getIndexValue 从WHERE条件中获取索引列的值
func (db *WeDBDatabase) getIndexValue(whereClause *WhereClause, columnName string) (interface{}, error) {
	for _, cond := range whereClause.Conditions {
		if cond.Column == columnName {
			// 简化实现：只处理等于操作
			if cond.Operator == OpEqual {
				return cond.Value, nil
			}
		}
	}
	return nil, fmt.Errorf("no matching condition for column: %s", columnName)
}

// fallbackToFullScan 回退到全表扫描（调用方持有读锁时使用）
func (db *WeDBDatabase) fallbackToFullScan(tableName string, whereClause *WhereClause) ([]map[string]interface{}, error) {
	allRows, err := db.scanTableLocked(tableName)
	if err != nil {
		return nil, err
	}

	filteredRows := make([]map[string]interface{}, 0, len(allRows))
	for _, row := range allRows {
		matched, err := whereClause.Evaluate(row)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate WHERE clause: %v", err)
		}
		if matched {
			filteredRows = append(filteredRows, row)
		}
	}

	return filteredRows, nil
}

// InsertRow inserts a row
func (db *WeDBDatabase) InsertRow(tableName string, row map[string]interface{}) error {
	// ---- 高并发写路径：全局锁仅用于快速校验，行数据落盘使用表级锁 ----
	// 不同表之间完全并行；同表写串行（等价于 SQLite 的库级单写被细化到表级）。
	db.mu.RLock()
	if db.closed {
		db.mu.RUnlock()
		return fmt.Errorf("database is closed")
	}
	if tableName == "" {
		db.mu.RUnlock()
		return fmt.Errorf("table name cannot be empty")
	}
	if row == nil || len(row) == 0 {
		db.mu.RUnlock()
		return fmt.Errorf("row cannot be nil or empty")
	}
	_, exists := db.schema.Tables[tableName]
	curWrite := db.curWriteTx
	db.mu.RUnlock()

	if !exists {
		return fmt.Errorf("table not found: %s", tableName)
	}

	// 验证行数据（检查约束）
	if err := db.validateRow(tableName, row); err != nil {
		return fmt.Errorf("row validation failed: %v", err)
	}

	// 会话级写事务：暂存而非直接落盘（提交时重放）。
	if curWrite != nil && !curWrite.closed {
		tl := db.tableLock(tableName)
		tl.Lock()
		err := curWrite.stageInsert(tableName, row)
		tl.Unlock()
		return err
	}

	// 表级写锁：同表串行、跨表并行
	tl := db.tableLock(tableName)
	tl.Lock()
	defer tl.Unlock()

	// 二次确认（DDL 可能在间隙发生）
	db.mu.RLock()
	stillExists := db.tables[tableName] > 0
	db.mu.RUnlock()
	if !stillExists {
		return fmt.Errorf("table not found: %s", tableName)
	}

	return db.insertRowCommitted(tableName, row)
}


// InsertRows inserts multiple rows
// 使用批量优化：减少锁持有时间、批量索引更新
func (db *WeDBDatabase) InsertRows(tableName string, rows []map[string]interface{}) error {
	if db.closed {
		return fmt.Errorf("database is closed")
	}

	if len(rows) == 0 {
		return nil
	}

	// 使用事务优化批量插入
	tx, err := db.BeginTx(nil, &api.TxOptions{Isolation: api.LevelReadCommitted})
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	// 类型断言获取WeDBTransaction
	wedbTx, ok := tx.(*WeDBTransaction)
	if !ok {
		return fmt.Errorf("failed to cast transaction to WeDBTransaction")
	}

	// 批量插入数据
	for _, row := range rows {
		if err := db.InsertRowInTx(wedbTx, tableName, row); err != nil {
			return fmt.Errorf("failed to insert row: %v", err)
		}
	}

	// 提交事务
	return tx.Commit()
}

// InsertRowInTx 在事务中插入一行
func (db *WeDBDatabase) InsertRowInTx(tx *WeDBTransaction, tableName string, row map[string]interface{}) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.closed {
		return fmt.Errorf("database is closed")
	}

	// 获取表信息
	table, exists := db.schema.Tables[tableName]
	if !exists {
		return fmt.Errorf("table not found: %s", tableName)
	}

	rootPage, exists := db.tables[tableName]
	if !exists {
		return fmt.Errorf("table root page not found: %s", tableName)
	}

	// 找出自增列
	autoIncrementCol := ""
	for _, col := range table.Columns {
		if col.AutoIncrement {
			autoIncrementCol = col.Name
			break
		}
	}

	// 验证必填列
	for _, col := range table.Columns {
		if col.PrimaryKey {
			if _, ok := row[col.Name]; !ok {
				if !col.AutoIncrement {
					return fmt.Errorf("missing required primary key column: %s", col.Name)
				}
			}
		}
	}

	// 处理自增列
	if autoIncrementCol != "" {
		if _, ok := row[autoIncrementCol]; !ok {
			// 获取当前表的下一个自增ID
			nextID := db.getTableNextRowID(tableName)
			row[autoIncrementCol] = nextID
		} else {
			// 如果用户提供了自增列的值，更新tableRowIDs
			if val, ok := row[autoIncrementCol].(int64); ok {
				if val > db.tableRowIDs[tableName] {
					db.tableRowIDs[tableName] = val
				}
			}
		}
	}

	// 检查 rowID 溢出
	if db.nextRowID >= 0x7FFFFFFFFFFFFFFF {
		return fmt.Errorf("row ID overflow, cannot insert more rows")
	}

	// 使用递增的 ID 作为键
	db.nextRowID++
	rowID := db.nextRowID

	// 序列化数据
	data, err := serializeRow(row, table)
	if err != nil {
		return fmt.Errorf("failed to serialize row: %v", err)
	}

	if len(data) == 0 {
		return fmt.Errorf("serialized row data is empty")
	}
	if len(data) > db.pageSize-20 {
		return fmt.Errorf("row data too large: %d bytes exceeds page limit", len(data))
	}

	// 插入到B-Tree
	tempBTree := db.tableBTreeByRoot(rootPage, tableName)
	if err := tempBTree.Insert(rowID, data); err != nil {
		return fmt.Errorf("failed to insert row: %v", err)
	}

	// 更新索引
	if err := db.indexManager.UpdateIndex(tableName, row, "insert"); err != nil {
		// 回滚：删除已插入的数据
		tempBTree.Delete(rowID)
		return fmt.Errorf("failed to update index: %v", err)
	}

	return nil
}

// UpdateRow updates a row
func (db *WeDBDatabase) UpdateRow(tableName string, row map[string]interface{}, condition string) error {
	if db.closed {
		return fmt.Errorf("database is closed")
	}

	// 首先获取表结构（需要 db.mu.Lock()）
	db.mu.RLock()
	table, exists := db.schema.Tables[tableName]
	if !exists {
		db.mu.RUnlock()
		return fmt.Errorf("table not found: %s", tableName)
	}

	rootPage, exists := db.tables[tableName]
	if !exists {
		db.mu.RUnlock()
		return fmt.Errorf("table root page not found: %s", tableName)
	}

	// 获取表的列名列表（用于验证）
	validColumns := make([]string, 0, len(table.Columns))
	for _, col := range table.Columns {
		validColumns = append(validColumns, col.Name)
	}
	db.mu.RUnlock()

	// 解析 WHERE 子句（带列名验证）
	whereClause, err := ParseWhereClauseWithValidation(condition, validColumns)
	if err != nil {
		return fmt.Errorf("failed to parse WHERE clause: %v", err)
	}

	// 收集所有需要更新的键和数据对
	updates := make([]struct {
		key     int64
		oldData []byte
		newData []byte
		oldRow  map[string]interface{}
		newRow  map[string]interface{}
	}, 0)

	// 创建临时的 B-Tree 用于遍历
	tempBTree := db.tableBTreeByRoot(rootPage, tableName)
	cursor := tempBTree.NewCursor()
	if err := cursor.First(); err != nil {
		cursor.Close()
		return fmt.Errorf("failed to create cursor: %v", err)
	}

	for !cursor.EOF() {
		data, err := cursor.Data()
		if err != nil {
			cursor.Close()
			return fmt.Errorf("failed to get data: %v", err)
		}

		existingRow := DeserializeRow(data, table)

		// 使用 WHERE 子句评估是否匹配
		matched, err := whereClause.Evaluate(existingRow)
		if err != nil {
			cursor.Close()
			return fmt.Errorf("failed to evaluate WHERE clause: %v", err)
		}

		if matched {
			key, err := cursor.Key()
			if err != nil {
				cursor.Close()
				return fmt.Errorf("failed to get key: %v", err)
			}

			// 保存旧数据用于索引更新
			oldRow := make(map[string]interface{})
			for k, v := range existingRow {
				oldRow[k] = v
			}

			// 更新数据（合并为新的行数据）
			for key, val := range row {
				existingRow[key] = val
			}

			// 保存新行数据用于索引更新
			newRow := make(map[string]interface{})
			for k, v := range existingRow {
				newRow[k] = v
			}

			newData, err := serializeRow(existingRow, table)
			if err != nil {
				cursor.Close()
				return fmt.Errorf("failed to serialize row: %v", err)
			}

			// 保存到更新列表
			updates = append(updates, struct {
				key     int64
				oldData []byte
				newData []byte
				oldRow  map[string]interface{}
				newRow  map[string]interface{}
			}{
				key:     key,
				oldData: data,
				newData: newData,
				oldRow:  oldRow,
				newRow:  newRow,
			})
		}

		if err := cursor.Next(); err != nil {
			cursor.Close()
			return fmt.Errorf("failed to move cursor: %v", err)
		}
	}

	cursor.Close()

	// 如果没有需要更新的行，返回错误
	if len(updates) == 0 {
		return fmt.Errorf("no rows matched the condition")
	}

	// 会话级写事务：暂存预计算产物（提交时重放）
	if wtx := db.curWriteTx; wtx != nil && !wtx.closed {
		pk, _ := db.pkOfTableLocked(tableName)
		ups := make([]rowUpdate, len(updates))
		for i, u := range updates {
			newRowCp := make(map[string]interface{}, len(u.oldRow)+len(row))
			for k, v := range u.oldRow {
				newRowCp[k] = v
			}
			for k, v := range row {
				newRowCp[k] = v
			}
			ups[i] = rowUpdate{
				key:     u.key,
				oldData: u.oldData,
				newData: u.newData,
				oldRow:  u.oldRow,
				newRow:  newRowCp,
			}
		}
		return wtx.stageUpdates(tableName, pk, ups)
	}

	// 执行更新：删除旧数据、插入新数据，然后同步索引
	ups := make([]rowUpdate, len(updates))
	for i, u := range updates {
		ups[i] = rowUpdate{key: u.key, oldData: u.oldData, newData: u.newData, oldRow: u.oldRow}
	}
	if err := db.applyUpdateArtifacts(tableName, ups); err != nil {
		return err
	}

	return nil
}

// UpdateRows updates multiple rows
func (db *WeDBDatabase) UpdateRows(tableName string, rows []map[string]interface{}, condition string) error {
	if db.closed {
		return fmt.Errorf("database is closed")
	}

	if len(rows) == 0 {
		return nil
	}

	// 获取表信息
	db.mu.RLock()
	table, exists := db.schema.Tables[tableName]
	if !exists {
		db.mu.RUnlock()
		return fmt.Errorf("table not found: %s", tableName)
	}

	rootPage, exists := db.tables[tableName]
	if !exists {
		db.mu.RUnlock()
		return fmt.Errorf("table root page not found: %s", tableName)
	}
	db.mu.RUnlock()

	// 解析 WHERE 子句
	whereClause, err := ParseWhereClause(condition)
	if err != nil {
		return fmt.Errorf("failed to parse WHERE clause: %v", err)
	}

	// 创建临时的 B-Tree 用于遍历
	tempBTree := db.tableBTreeByRoot(rootPage, tableName)
	cursor := tempBTree.NewCursor()
	if err := cursor.First(); err != nil {
		cursor.Close()
		return fmt.Errorf("failed to create cursor: %v", err)
	}
	defer cursor.Close()

	// 收集所有需要更新的键和数据对
	updates := make([]struct {
		key     int64
		oldData []byte
		newData []byte
		oldRow  map[string]interface{}
	}, 0)

	for !cursor.EOF() {
		data, err := cursor.Data()
		if err != nil {
			return fmt.Errorf("failed to get data: %v", err)
		}

		existingRow := DeserializeRow(data, table)

		// 使用 WHERE 子句评估是否匹配
		matched, err := whereClause.Evaluate(existingRow)
		if err != nil {
			return fmt.Errorf("failed to evaluate WHERE clause: %v", err)
		}

		if matched {
			rowKey, err := cursor.Key()
			if err != nil {
				return fmt.Errorf("failed to get key: %v", err)
			}

			// 保存旧数据用于索引更新
			oldRow := make(map[string]interface{})
			for k, v := range existingRow {
				oldRow[k] = v
			}

			// 批量更新：对所有匹配的行应用所有更新
			for _, updateRow := range rows {
				for updateKey, val := range updateRow {
					existingRow[updateKey] = val
				}
			}

			newData, err := serializeRow(existingRow, table)
			if err != nil {
				return fmt.Errorf("failed to serialize row: %v", err)
			}

			// 保存到更新列表
			updates = append(updates, struct {
				key     int64
				oldData []byte
				newData []byte
				oldRow  map[string]interface{}
			}{
				key:     rowKey,
				oldData: data,
				newData: newData,
				oldRow:  oldRow,
			})
		}

		if err := cursor.Next(); err != nil {
			return fmt.Errorf("failed to move cursor: %v", err)
		}
	}

	// 如果没有需要更新的行，返回错误
	if len(updates) == 0 {
		return fmt.Errorf("no rows matched the condition")
	}

	// 会话级写事务：暂存预计算产物（提交时重放）
	if wtx := db.curWriteTx; wtx != nil && !wtx.closed {
		pk, _ := db.pkOfTableLocked(tableName)
		ups := make([]rowUpdate, len(updates))
		for i, u := range updates {
			newRow := DeserializeRow(u.newData, table)
			ups[i] = rowUpdate{key: u.key, oldData: u.oldData, newData: u.newData, oldRow: u.oldRow, newRow: newRow}
		}
		return wtx.stageUpdates(tableName, pk, ups)
	}

	// 执行更新：删除旧数据、插入新数据，然后同步索引
	ups := make([]rowUpdate, len(updates))
	for i, update := range updates {
		newRow := DeserializeRow(update.newData, table)
		ups[i] = rowUpdate{key: update.key, oldData: update.oldData, newData: update.newData, oldRow: update.oldRow, newRow: newRow}
	}
	if err := db.applyUpdateArtifacts(tableName, ups); err != nil {
		return err
	}

	return nil
}

// DeleteRow deletes a row
func (db *WeDBDatabase) DeleteRow(tableName string, condition string) error {
	if db.closed {
		return fmt.Errorf("database is closed")
	}

	// 首先获取表结构（不需要 db.mu.Lock()）
	db.mu.RLock()
	table, exists := db.schema.Tables[tableName]
	if !exists {
		db.mu.RUnlock()
		return fmt.Errorf("table not found: %s", tableName)
	}

	rootPage, exists := db.tables[tableName]
	if !exists {
		db.mu.RUnlock()
		return fmt.Errorf("table root page not found: %s", tableName)
	}

	// 获取表的列名列表（用于验证）
	validColumns := make([]string, 0, len(table.Columns))
	for _, col := range table.Columns {
		validColumns = append(validColumns, col.Name)
	}
	db.mu.RUnlock()

	// 解析 WHERE 子句（带列名验证）
	whereClause, err := ParseWhereClauseWithValidation(condition, validColumns)
	if err != nil {
		return fmt.Errorf("failed to parse WHERE clause: %v", err)
	}

	// 收集所有需要删除的键和行数据
	deletes := make([]struct {
		key  int64
		row  map[string]interface{}
	}, 0)

	// 创建临时的 B-Tree 用于遍历
	tempBTree := db.tableBTreeByRoot(rootPage, tableName)
	cursor := tempBTree.NewCursor()
	if err := cursor.First(); err != nil {
		cursor.Close()
		return fmt.Errorf("failed to create cursor: %v", err)
	}

	for !cursor.EOF() {
		data, err := cursor.Data()
		if err != nil {
			cursor.Close()
			return fmt.Errorf("failed to get data: %v", err)
		}

		row := DeserializeRow(data, table)

		// 使用 WHERE 子句评估是否匹配
		matched, err := whereClause.Evaluate(row)
		if err != nil {
			cursor.Close()
			return fmt.Errorf("failed to evaluate WHERE clause: %v", err)
		}

		if matched {
			key, err := cursor.Key()
			if err != nil {
				cursor.Close()
				return fmt.Errorf("failed to get key: %v", err)
			}

			deletes = append(deletes, struct {
				key  int64
				row  map[string]interface{}
			}{
				key: key,
				row: row,
			})
		}

		if err := cursor.Next(); err != nil {
			cursor.Close()
			return fmt.Errorf("failed to move cursor: %v", err)
		}
	}

	cursor.Close()

	// 如果没有需要删除的行，返回错误
	if len(deletes) == 0 {
		return fmt.Errorf("no rows matched the condition")
	}

	// 会话级写事务：暂存（提交时重放）
	if wtx := db.curWriteTx; wtx != nil && !wtx.closed {
		pk, _ := db.pkOfTableLocked(tableName)
		dels := make([]rowDelete, len(deletes))
		for i, d := range deletes {
			dels[i] = rowDelete{key: d.key, row: d.row}
		}
		return wtx.stageDeletes(tableName, pk, dels)
	}

	return db.applyDeleteArtifacts(tableName, func() []rowDelete {
		out := make([]rowDelete, len(deletes))
		for i, d := range deletes {
			out[i] = rowDelete{key: d.key, row: d.row}
		}
		return out
	}())
}

// DeleteRows deletes multiple rows
func (db *WeDBDatabase) DeleteRows(tableName string, conditions []string) error {
	if len(conditions) == 0 {
		return nil
	}

	if db.closed {
		return fmt.Errorf("database is closed")
	}

	// 获取表信息
	db.mu.RLock()
	table, exists := db.schema.Tables[tableName]
	if !exists {
		db.mu.RUnlock()
		return fmt.Errorf("table not found: %s", tableName)
	}

	rootPage, exists := db.tables[tableName]
	if !exists {
		db.mu.RUnlock()
		return fmt.Errorf("table root page not found: %s", tableName)
	}
	db.mu.RUnlock()

	// 创建临时的 B-Tree 用于遍历
	tempBTree := db.tableBTreeByRoot(rootPage, tableName)

	// 批量处理所有删除条件
	for _, condition := range conditions {
		// 解析 WHERE 子句
		whereClause, err := ParseWhereClause(condition)
		if err != nil {
			return fmt.Errorf("failed to parse WHERE clause: %v", err)
		}

		// 收集所有需要删除的键和行数据
		deletes := make([]struct {
			key  int64
			row  map[string]interface{}
		}, 0)

		// 遍历表
		cursor := tempBTree.NewCursor()
		if err := cursor.First(); err != nil {
			cursor.Close()
			return fmt.Errorf("failed to create cursor: %v", err)
		}

		for !cursor.EOF() {
			data, err := cursor.Data()
			if err != nil {
				cursor.Close()
				return fmt.Errorf("failed to get data: %v", err)
			}

			row := DeserializeRow(data, table)

			// 使用 WHERE 子句评估是否匹配
			matched, err := whereClause.Evaluate(row)
			if err != nil {
				cursor.Close()
				return fmt.Errorf("failed to evaluate WHERE clause: %v", err)
			}

			if matched {
				key, err := cursor.Key()
				if err != nil {
					cursor.Close()
					return fmt.Errorf("failed to get key: %v", err)
				}

				deletes = append(deletes, struct {
					key  int64
					row  map[string]interface{}
				}{
					key: key,
					row: row,
				})
			}

			if err := cursor.Next(); err != nil {
				cursor.Close()
				return fmt.Errorf("failed to move cursor: %v", err)
			}
		}

		cursor.Close()

		// 会话级写事务：暂存本条件的删除（提交时重放）
		if wtx := db.curWriteTx; wtx != nil && !wtx.closed {
			pk, _ := db.pkOfTableLocked(tableName)
			dels := make([]rowDelete, len(deletes))
			for i, d := range deletes {
				dels[i] = rowDelete{key: d.key, row: d.row}
			}
			if err := wtx.stageDeletes(tableName, pk, dels); err != nil {
				return err
			}
			continue
		}

		// 批量删除
		dels := make([]rowDelete, len(deletes))
		for i, d := range deletes {
			dels[i] = rowDelete{key: d.key, row: d.row}
		}
		if err := db.applyDeleteArtifacts(tableName, dels); err != nil {
			return err
		}
	}

	return nil
}

// Begin begins a transaction
func (db *WeDBDatabase) Begin() (api.Transaction, error) {
	return db.BeginTx(context.Background(), &api.TxOptions{})
}

// BeginTx begins a transaction with context and MVCC support.
// Write transactions serialize on db.writeGate: concurrent writers queue
// (bounded by ctx) instead of failing immediately.
func (db *WeDBDatabase) BeginTx(ctx context.Context, opts *api.TxOptions) (api.Transaction, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	// 处理默认隔离级别
	if opts == nil {
		opts = &api.TxOptions{
			Isolation: api.LevelDefault,
		}
	} else if opts.Isolation == api.LevelDefault {
		// 默认使用读已提交
		opts.Isolation = api.LevelReadCommitted
	}

	readOnly := opts.ReadOnly

	// 写事务先排队拿闸门（不持 db.mu，避免阻塞提交方）
	var gated bool
	if !readOnly {
		select {
		case db.writeGate <- struct{}{}:
			gated = true
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	db.mu.Lock()
	defer db.mu.Unlock()

	if db.closed {
		if gated {
			<-db.writeGate
		}
		return nil, fmt.Errorf("database is closed")
	}

	if err := db.pager.BeginTransaction(readOnly); err != nil {
		if gated {
			<-db.writeGate
		}
		return nil, fmt.Errorf("failed to begin transaction: %v", err)
	}

	// 在事务管理器中开始事务
	txID := db.txManager.BeginTransaction(opts.Isolation, readOnly)

	// 创建快照（如果使用快照隔离或可重复读）
	var snapshot *dbSnapshot
	if opts.Isolation == api.LevelSnapshot || opts.Isolation == api.LevelRepeatableRead {
		snapshot = db.txManager.createSnapshot(txID)
	}

	tx := &WeDBTransaction{
		db:          db,
		ctx:         ctx,
		opts:        opts,
		begin:       time.Now(),
		txID:        txID,
		snapshot:    snapshot,
		gated:       gated,
		isoLevel:    opts.Isolation,
		snapshotIso: opts.Isolation == api.LevelRepeatableRead || opts.Isolation == api.LevelSnapshot,
	}

	// 环境事务登记：后续 DML/扫描据此路由
	if gated {
		db.curWriteTx = tx
	} else {
		db.activeReadTx = tx
	}

	return tx, nil
}

// GetTableStats gets table statistics
func (db *WeDBDatabase) GetTableStats(tableName string) (*api.TableStats, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.closed {
		return nil, fmt.Errorf("database is closed")
	}

	if _, exists := db.tables[tableName]; !exists {
		return nil, fmt.Errorf("table not found: %s", tableName)
	}

	rows, err := db.scanTableLocked(tableName)
	if err != nil {
		return nil, err
	}

	return &api.TableStats{
		RowCount:     int64(len(rows)),
		IndexCount:   len(db.indexManager.GetTableIndexes(tableName)),
		ColumnCount:  0,
		LastModified: time.Now(),
		Created:      time.Now(),
		TableSize:    0,
	}, nil
}

// Count counts rows
func (db *WeDBDatabase) Count(tableName string, condition string) (int64, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.closed {
		return 0, fmt.Errorf("database is closed")
	}

	if _, exists := db.tables[tableName]; !exists {
		return 0, fmt.Errorf("table not found: %s", tableName)
	}

	// 获取表的列名列表（用于验证）
	table, exists := db.schema.Tables[tableName]
	if !exists {
		return 0, fmt.Errorf("table not found: %s", tableName)
	}
	validColumns := make([]string, 0, len(table.Columns))
	for _, col := range table.Columns {
		validColumns = append(validColumns, col.Name)
	}

	// 解析 WHERE 子句（带列名验证）
	whereClause, err := ParseWhereClauseWithValidation(condition, validColumns)
	if err != nil {
		return 0, fmt.Errorf("failed to parse WHERE clause: %v", err)
	}

	rows, err := db.scanTableLocked(tableName)
	if err != nil {
		return 0, err
	}

	// 过滤数据
	count := int64(0)
	for _, row := range rows {
		matched, err := whereClause.Evaluate(row)
		if err != nil {
			return 0, fmt.Errorf("failed to evaluate WHERE clause: %v", err)
		}
		if matched {
			count++
		}
	}

	return count, nil
}

// Min gets minimum value
func (db *WeDBDatabase) Min(tableName string, column string, condition string) (interface{}, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.closed {
		return nil, fmt.Errorf("database is closed")
	}

	if _, exists := db.tables[tableName]; !exists {
		return nil, fmt.Errorf("table not found: %s", tableName)
	}

	// 获取表的列名列表（用于验证）
	table, exists := db.schema.Tables[tableName]
	if !exists {
		return nil, fmt.Errorf("table not found: %s", tableName)
	}
	validColumns := make([]string, 0, len(table.Columns))
	for _, col := range table.Columns {
		validColumns = append(validColumns, col.Name)
	}

	// 解析 WHERE 子句（带列名验证）
	whereClause, err := ParseWhereClauseWithValidation(condition, validColumns)
	if err != nil {
		return nil, fmt.Errorf("failed to parse WHERE clause: %v", err)
	}

	rows, err := db.scanTableLocked(tableName)
	if err != nil {
		return nil, err
	}

	var minVal interface{}
	found := false

	for _, row := range rows {
		// 过滤数据
		matched, err := whereClause.Evaluate(row)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate WHERE clause: %v", err)
		}
		if !matched {
			continue
		}

		if val, exists := row[column]; exists {
			if !found {
				minVal = val
				found = true
			} else {
				switch v := val.(type) {
				case int:
					if min, ok := minVal.(int); ok && v < min {
						minVal = val
					}
				case int64:
					if min, ok := minVal.(int64); ok && v < min {
						minVal = val
					}
				case float64:
					if min, ok := minVal.(float64); ok && v < min {
						minVal = val
					}
				}
			}
		}
	}

	if !found {
		return nil, fmt.Errorf("column %s not found or no values", column)
	}

	return minVal, nil
}

// Max gets maximum value
func (db *WeDBDatabase) Max(tableName string, column string, condition string) (interface{}, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.closed {
		return nil, fmt.Errorf("database is closed")
	}

	if _, exists := db.tables[tableName]; !exists {
		return nil, fmt.Errorf("table not found: %s", tableName)
	}

	// 获取表的列名列表（用于验证）
	table, exists := db.schema.Tables[tableName]
	if !exists {
		return nil, fmt.Errorf("table not found: %s", tableName)
	}
	validColumns := make([]string, 0, len(table.Columns))
	for _, col := range table.Columns {
		validColumns = append(validColumns, col.Name)
	}

	// 解析 WHERE 子句（带列名验证）
	whereClause, err := ParseWhereClauseWithValidation(condition, validColumns)
	if err != nil {
		return nil, fmt.Errorf("failed to parse WHERE clause: %v", err)
	}

	rows, err := db.scanTableLocked(tableName)
	if err != nil {
		return nil, err
	}

	var maxVal interface{}
	found := false

	for _, row := range rows {
		// 过滤数据
		matched, err := whereClause.Evaluate(row)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate WHERE clause: %v", err)
		}
		if !matched {
			continue
		}

		if val, exists := row[column]; exists {
			if !found {
				maxVal = val
				found = true
			} else {
				switch v := val.(type) {
				case int:
					if max, ok := maxVal.(int); ok && v > max {
						maxVal = val
					}
				case int64:
					if max, ok := maxVal.(int64); ok && v > max {
						maxVal = val
					}
				case float64:
					if max, ok := maxVal.(float64); ok && v > max {
						maxVal = val
					}
				}
			}
		}
	}

	if !found {
		return nil, fmt.Errorf("column %s not found or no values", column)
	}

	return maxVal, nil
}

// Sum calculates sum
func (db *WeDBDatabase) Sum(tableName string, column string, condition string) (float64, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.closed {
		return 0, fmt.Errorf("database is closed")
	}

	if _, exists := db.tables[tableName]; !exists {
		return 0, fmt.Errorf("table not found: %s", tableName)
	}

	// 获取表的列名列表（用于验证）
	table, exists := db.schema.Tables[tableName]
	if !exists {
		return 0, fmt.Errorf("table not found: %s", tableName)
	}
	validColumns := make([]string, 0, len(table.Columns))
	for _, col := range table.Columns {
		validColumns = append(validColumns, col.Name)
	}

	// 解析 WHERE 子句（带列名验证）
	whereClause, err := ParseWhereClauseWithValidation(condition, validColumns)
	if err != nil {
		return 0, fmt.Errorf("failed to parse WHERE clause: %v", err)
	}

	rows, err := db.scanTableLocked(tableName)
	if err != nil {
		return 0, err
	}

	sum := 0.0
	found := false

	for _, row := range rows {
		// 过滤数据
		matched, err := whereClause.Evaluate(row)
		if err != nil {
			return 0, fmt.Errorf("failed to evaluate WHERE clause: %v", err)
		}
		if !matched {
			continue
		}

		if val, exists := row[column]; exists {
			found = true
			switch v := val.(type) {
			case int:
				sum += float64(v)
			case int64:
				sum += float64(v)
			case float64:
				sum += v
			}
		}
	}

	if !found {
		return 0, fmt.Errorf("column %s not found or no values", column)
	}

	return sum, nil
}

// Avg calculates average
func (db *WeDBDatabase) Avg(tableName string, column string, condition string) (float64, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.closed {
		return 0, fmt.Errorf("database is closed")
	}

	if _, exists := db.tables[tableName]; !exists {
		return 0, fmt.Errorf("table not found: %s", tableName)
	}

	// 获取表的列名列表（用于验证）
	table, exists := db.schema.Tables[tableName]
	if !exists {
		return 0, fmt.Errorf("table not found: %s", tableName)
	}
	validColumns := make([]string, 0, len(table.Columns))
	for _, col := range table.Columns {
		validColumns = append(validColumns, col.Name)
	}

	// 解析 WHERE 子句（带列名验证）
	whereClause, err := ParseWhereClauseWithValidation(condition, validColumns)
	if err != nil {
		return 0, fmt.Errorf("failed to parse WHERE clause: %v", err)
	}

	rows, err := db.scanTableLocked(tableName)
	if err != nil {
		return 0, err
	}

	sum := 0.0
	count := 0

	for _, row := range rows {
		// 过滤数据
		matched, err := whereClause.Evaluate(row)
		if err != nil {
			return 0, fmt.Errorf("failed to evaluate WHERE clause: %v", err)
		}
		if !matched {
			continue
		}

		if val, exists := row[column]; exists {
			count++
			switch v := val.(type) {
			case int:
				sum += float64(v)
			case int64:
				sum += float64(v)
			case float64:
				sum += v
			}
		}
	}

	if count == 0 {
		return 0, fmt.Errorf("column %s not found or no values", column)
	}

	return sum / float64(count), nil
}

// GetColumnStats gets column statistics
func (db *WeDBDatabase) GetColumnStats(tableName string, column string) (*api.ColumnStats, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.closed {
		return nil, fmt.Errorf("database is closed")
	}

	if _, exists := db.tables[tableName]; !exists {
		return nil, fmt.Errorf("table not found: %s", tableName)
	}

	rows, err := db.scanTableLocked(tableName)
	if err != nil {
		return nil, err
	}

	stats := &api.ColumnStats{
		ColumnName:  column,
		NullCount:   0,
		UniqueCount: 0,
		Min:         nil,
		Max:         nil,
		Average:     0,
		StdDev:      0,
	}

	values := make([]float64, 0)
	for _, row := range rows {
		if val, exists := row[column]; exists {
			if val == nil {
				stats.NullCount++
				continue
			}

			var num float64
			switch v := val.(type) {
			case int:
				num = float64(v)
			case int64:
				num = float64(v)
			case float64:
				num = v
			default:
				continue
			}

			values = append(values, num)

			if stats.Min == nil {
				stats.Min = num
			} else if min, ok := stats.Min.(float64); ok && num < min {
				stats.Min = num
			}

			if stats.Max == nil {
				stats.Max = num
			} else if max, ok := stats.Max.(float64); ok && num > max {
				stats.Max = num
			}
		}
	}

	if len(values) > 0 {
		var sum float64
		for _, val := range values {
			sum += val
		}
		stats.Average = sum / float64(len(values))

		variance := 0.0
		for _, val := range values {
			diff := val - stats.Average
			variance += diff * diff
		}
		variance /= float64(len(values))
		stats.StdDev = variance
	}

	return stats, nil
}

// WeDBTransaction transaction implementation
type WeDBTransaction struct {
	db       *WeDBDatabase
	ctx      context.Context
	opts     *api.TxOptions
	begin    time.Time
	closed   bool
	txID     int64           // 事务ID
	snapshot *dbSnapshot     // 快照（用于MVCC）
	gated    bool            // 是否持有写闸门（负责释放）

	// 会话级暂存（见 tx_staging.go）
	staged       []stagedOp
	stagedPKs    map[string]map[string]bool
	isoLevel     api.IsolationLevel
	snapshotIso  bool
	snapMu       sync.RWMutex
	snapCache    map[string][]map[string]interface{}
}

// Commit commits the transaction
func (tx *WeDBTransaction) Commit() error {
	tx.db.mu.Lock()
	defer tx.db.mu.Unlock()

	if tx.closed {
		return fmt.Errorf("transaction already closed")
	}

	// 无论提交成功与否都结束事务并归还写闸门，避免泄漏阻塞后续写者
	defer func() {
		tx.closed = true
		tx.releaseGateLocked()
		tx.detachLocked()
	}()

	// 重放暂存的 DML（写事务语义）
	if len(tx.staged) > 0 {
		if err := tx.replayStaged(); err != nil {
			return fmt.Errorf("commit aborted: %v", err)
		}
	}

	// 提交到事务管理器
	if err := tx.db.txManager.Commit(tx.txID); err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	// 刷新所有脏页面到磁盘
	if err := tx.db.btree.Flush(); err != nil {
		return fmt.Errorf("failed to flush btree: %v", err)
	}

	// 提交到pager
	if err := tx.db.pager.Commit(); err != nil {
		return fmt.Errorf("failed to commit: %v", err)
	}

	return nil
}

// Rollback rolls back the transaction
func (tx *WeDBTransaction) Rollback() error {
	tx.db.mu.Lock()
	defer tx.db.mu.Unlock()

	if tx.closed {
		return fmt.Errorf("transaction already closed")
	}

	// 无论回滚成功与否都结束事务并归还写闸门
	defer func() {
		tx.closed = true
		tx.staged = nil // 丢弃全部暂存变更
		tx.releaseGateLocked()
		tx.detachLocked()
	}()

	// 回滚到事务管理器
	if err := tx.db.txManager.Rollback(tx.txID); err != nil {
		return fmt.Errorf("failed to rollback transaction: %v", err)
	}

	// 回滚事务
	if err := tx.db.pager.Rollback(); err != nil {
		return fmt.Errorf("failed to rollback: %v", err)
	}

	// 清空缓存，重新加载数据
	if err := tx.db.cache.Clear(); err != nil {
		return fmt.Errorf("failed to clear cache: %v", err)
	}

	return nil
}

// releaseGateLocked returns the write gate; caller holds db.mu.
func (tx *WeDBTransaction) releaseGateLocked() {
	if tx.gated {
		select {
		case <-tx.db.writeGate:
		default:
		}
		tx.gated = false
	}
}

// detachLocked 清理数据库上的环境事务登记；调用方持有 db.mu。
func (tx *WeDBTransaction) detachLocked() {
	if tx.db.curWriteTx == tx {
		tx.db.curWriteTx = nil
	}
	if tx.db.activeReadTx == tx {
		tx.db.activeReadTx = nil
	}
}

// IsActive checks if transaction is active
func (tx *WeDBTransaction) IsActive() bool {
	return !tx.closed
}

// IsolationLevel gets isolation level
func (tx *WeDBTransaction) IsolationLevel() api.IsolationLevel {
	if tx.opts != nil {
		return tx.opts.Isolation
	}
	return api.LevelDefault
}

// ID gets transaction ID
func (tx *WeDBTransaction) ID() string {
	return fmt.Sprintf("%d", tx.begin.UnixNano())
}

// StartTime gets start time
func (tx *WeDBTransaction) StartTime() time.Time {
	return tx.begin
}

// Duration gets duration
func (tx *WeDBTransaction) Duration() time.Duration {
	return time.Since(tx.begin)
}

// Savepoint creates a savepoint
func (tx *WeDBTransaction) Savepoint(name string) error {
	tx.db.mu.Lock()
	defer tx.db.mu.Unlock()

	if tx.closed {
		return fmt.Errorf("transaction already closed")
	}

	return fmt.Errorf("not implemented")
}

// RollbackTo rolls back to a savepoint
func (tx *WeDBTransaction) RollbackTo(name string) error {
	tx.db.mu.Lock()
	defer tx.db.mu.Unlock()

	if tx.closed {
		return fmt.Errorf("transaction already closed")
	}

	return fmt.Errorf("not implemented")
}

// ReleaseSavepoint releases a savepoint
func (tx *WeDBTransaction) ReleaseSavepoint(name string) error {
	tx.db.mu.Lock()
	defer tx.db.mu.Unlock()

	if tx.closed {
		return fmt.Errorf("transaction already closed")
	}

	return fmt.Errorf("not implemented")
}

// convertDataType converts column type
func convertDataType(colType api.ColumnType) types.DataType {
	switch colType {
	case api.TypeInteger:
		return types.TypeInteger
	case api.TypeReal:
		return types.TypeReal
	case api.TypeText:
		return types.TypeText
	case api.TypeBlob:
		return types.TypeBlob
	default:
		return types.TypeNull
	}
}

// convertColumnType converts column type
func convertColumnType(dataType types.DataType) api.ColumnType {
	switch dataType {
	case types.TypeInteger:
		return api.TypeInteger
	case types.TypeReal:
		return api.TypeReal
	case types.TypeText:
		return api.TypeText
	case types.TypeBlob:
		return api.TypeBlob
	default:
		return api.TypeNull
	}
}

// serializeRow serializes a row
func serializeRow(row map[string]interface{}, table *types.Table) ([]byte, error) {
	data := make([]byte, 0)
	for _, col := range table.Columns {
		val := row[col.Name]
		data = append(data, fmt.Sprintf("%v", val)...)
		data = append(data, '|')
	}
	return data, nil
}

// DeserializeRow deserializes a row
func DeserializeRow(data []byte, table *types.Table) map[string]interface{} {
	row := make(map[string]interface{})
	parts := SplitData(data, '|')
	for i, col := range table.Columns {
		if i < len(parts) {
			strVal := parts[i]
			// 根据列类型转换值
			switch col.Type {
			case types.TypeInteger:
				var val int64
				if strVal != "" {
					fmt.Sscanf(strVal, "%d", &val)
				}
				row[col.Name] = val
			case types.TypeReal:
				var val float64
				if strVal != "" {
					fmt.Sscanf(strVal, "%f", &val)
				}
				row[col.Name] = val
			default:
				row[col.Name] = strVal
			}
		}
	}
	return row
}

// SplitData splits data
func SplitData(data []byte, sep byte) []string {
	parts := make([]string, 0)
	start := 0
	for i, b := range data {
		if b == sep {
			parts = append(parts, string(data[start:i]))
			start = i + 1
		}
	}
	if start < len(data) {
		parts = append(parts, string(data[start:]))
	}
	return parts
}

// HealthCheck 检查数据库健康状态
func (db *WeDBDatabase) HealthCheck() error {
	db.mu.RLock()
	defer db.mu.RUnlock()

	// 检查数据库是否已关闭
	if db.closed {
		return fmt.Errorf("database is closed")
	}

	// 检查文件完整性
	if err := db.pager.CheckIntegrity(); err != nil {
		return fmt.Errorf("file integrity check failed: %v", err)
	}

	// 检查缓存状态
	cacheStats := db.cache.GetStats()
	if cacheStats.UsedMemory > cacheStats.MaxMemory {
		return fmt.Errorf("cache memory usage too high: %d/%d bytes", cacheStats.UsedMemory, cacheStats.MaxMemory)
	}

	// 检查脏页面数量
	if cacheStats.DirtyCount > 100 {
		return fmt.Errorf("too many dirty pages: %d", cacheStats.DirtyCount)
	}

	return nil
}

// GetHealthStatus 获取详细健康状态信息
func (db *WeDBDatabase) GetHealthStatus() (map[string]interface{}, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	status := make(map[string]interface{})

	// 基本信息
	status["status"] = "healthy"
	if db.closed {
		status["status"] = "closed"
	}
	status["timestamp"] = time.Now().Unix()
	status["file_path"] = db.filePath

	// 表信息
	status["table_count"] = len(db.tables)
	tables := make([]string, 0, len(db.tables))
	for name := range db.tables {
		tables = append(tables, name)
	}
	status["tables"] = tables

	// 索引信息
	totalIndexes := 0
	for range db.indexManager.GetTableIndexes("") {
		totalIndexes++
	}
	status["total_indexes"] = totalIndexes

	// 缓存信息
	cacheStats := db.cache.GetStats()
	status["cache"] = map[string]interface{}{
		"size":        cacheStats.Size,
		"max_size":     cacheStats.MaxSize,
		"dirty_count":  cacheStats.DirtyCount,
		"used_memory":  cacheStats.UsedMemory,
		"max_memory":  cacheStats.MaxMemory,
		"memory_usage": float64(cacheStats.UsedMemory) / float64(cacheStats.MaxMemory) * 100,
	}

	// 存储信息
	fileInfo, err := os.Stat(db.filePath)
	if err == nil {
		status["file_size"] = fileInfo.Size()
	} else {
		status["file_size"] = 0
	}

	// 熔断器状态
	if db.circuitBreaker != nil {
		cbState := db.circuitBreaker.GetState()
		stateStr := "closed"
		switch cbState {
		case StateOpen:
			stateStr = "open"
		case StateHalfOpen:
			stateStr = "half-open"
		}
		status["circuit_breaker"] = map[string]interface{}{
			"state": stateStr,
			"failure_count": db.circuitBreaker.failureCount.Load(),
			"threshold": db.circuitBreaker.threshold,
		}
	}

	return status, nil
}

// GetMetrics 获取数据库性能指标
func (db *WeDBDatabase) GetMetrics() map[string]interface{} {
	if db.metrics == nil {
		return map[string]interface{}{
			"error": "metrics not initialized",
		}
	}

	cacheStats := db.cache.GetStats()

	return map[string]interface{}{
		"query": map[string]interface{}{
			"total_count":   db.metrics.QueryCount.Load(),
			"avg_time_ms":   db.metrics.GetAvgQueryTime(),
		},
		"operations": map[string]interface{}{
			"insert_count": db.metrics.InsertCount.Load(),
			"update_count": db.metrics.UpdateCount.Load(),
			"delete_count": db.metrics.DeleteCount.Load(),
			"select_count": db.metrics.SelectCount.Load(),
		},
		"cache": map[string]interface{}{
			"hit_count":    db.metrics.CacheHitCount.Load(),
			"miss_count":   db.metrics.CacheMissCount.Load(),
			"hit_rate":     db.metrics.GetCacheHitRate(),
			"size":         cacheStats.Size,
			"dirty_count":  cacheStats.DirtyCount,
		},
		"uptime": time.Since(db.metrics.StartTime).String(),
	}
}
