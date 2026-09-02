package types

import (
	"sync"
)

// 默认页面大小
const DefaultPageSize = 4096

// 数据类型
type DataType int

const (
	TypeInteger DataType = iota
	TypeReal
	TypeText
	TypeBlob
	TypeNull
)

// Schema 数据库 Schema
type Schema struct {
	Tables  map[string]*Table
	Indexes map[string]*Index
	mu      sync.RWMutex
}

// NewSchema 创建新 Schema
func NewSchema() *Schema {
	return &Schema{
		Tables:  make(map[string]*Table),
		Indexes: make(map[string]*Index),
	}
}

// GetTable 获取表
func (s *Schema) GetTable(name string) *Table {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Tables[name]
}

// AddTable 添加表
func (s *Schema) AddTable(table *Table) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Tables[table.TableName] = table
}

// RemoveTable 移除表
func (s *Schema) RemoveTable(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.Tables, name)
}

// TableExists 检查表是否存在
func (s *Schema) TableExists(name string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.Tables[name]
	return exists
}

// GetIndex 获取索引
func (s *Schema) GetIndex(name string) *Index {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Indexes[name]
}

// AddIndex 添加索引
func (s *Schema) AddIndex(index *Index) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Indexes[index.IndexName] = index
}

// RemoveIndex 移除索引
func (s *Schema) RemoveIndex(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.Indexes, name)
}

// IndexExists 检查索引是否存在
func (s *Schema) IndexExists(name string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.Indexes[name]
	return exists
}

// Table 表定义
type Table struct {
	TableName     string
	Columns       []*Column
	PrimaryIndex  *Index
}

// Column 列定义
type Column struct {
	Name          string
	Type          DataType
	PrimaryKey    bool
	AutoIncrement bool
}

// Index 索引定义
type Index struct {
	IndexName string
	TableName string
	Columns   []string
	Unique    bool
	Primary   bool // 内部主键索引，不出现在用户索引列表
}

// Database 表示数据库连接
// 对应 SQLite 中的 sqlite3 结构
type Database struct {
	mu          sync.RWMutex      // 读写锁
	Filename    string            // 数据库文件名
	Flags       int               // 打开标志
	IsInMemory  bool              // 是否为内存数据库
	Schema      *Schema           // 主数据库 Schema
	AttachedDBs map[string]*Database // 附加数据库（ATTACH DATABASE）
	LastRowid   int64             // 最近插入的 ROWID
	PageSize    int               // 页面大小
	CacheSize   int               // 缓存大小（页数）
	Encoding    string            // 编码（UTF-8, UTF-16le, UTF-16be）
	AutoCommit  bool              // 自动提交模式
	IsReadOnly  bool              // 是否只读
	Pager       interface{}       // Pager 实例（类型待定）
	Vdbe        interface{}       // 活动的 VDBE 列表（类型待定）
	nStmt       int               // 活动语句数
	MaxPage     int               // 最大页面号
	ChangeCount int               // 变更计数器
}

// NewDatabase 创建新数据库连接
func NewDatabase(filename string) *Database {
	isInMemory := filename == ":memory:"
	return &Database{
		Filename:    filename,
		IsInMemory:  isInMemory,
		Schema:      NewSchema(),
		AttachedDBs: make(map[string]*Database),
		PageSize:    DefaultPageSize,
		CacheSize:   2000, // 默认缓存 2000 页
		Encoding:    "UTF-8",
		AutoCommit:  true,
		IsReadOnly:  false,
	}
}

// Open 打开数据库
func (db *Database) Open(flags int) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	db.Flags = flags
	db.IsReadOnly = (flags & 0x01) != 0 // SQLITE_OPEN_READONLY

	// TODO: 初始化 Pager
	// TODO: 读取 Schema

	return nil
}

// Close 关闭数据库
func (db *Database) Close() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	// TODO: 清理资源
	// TODO: 关闭 Pager
	// TODO: 释放所有语句

	return nil
}

// GetSchema 获取 Schema
func (db *Database) GetSchema() *Schema {
	return db.Schema
}

// GetTable 获取表
func (db *Database) GetTable(name string) *Table {
	return db.Schema.GetTable(name)
}

// AttachDatabase 附加数据库
func (db *Database) AttachDatabase(name string, filename string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	// TODO: 实现附加数据库逻辑
	return nil
}

// DetachDatabase 分离数据库
func (db *Database) DetachDatabase(name string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	// TODO: 实现分离数据库逻辑
	return nil
}

// GetLastRowid 获取最近插入的 ROWID
func (db *Database) GetLastRowid() int64 {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.LastRowid
}

// SetLastRowid 设置最近插入的 ROWID
func (db *Database) SetLastRowid(rowid int64) {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.LastRowid = rowid
}

// IncrementChangeCount 增加变更计数器
func (db *Database) IncrementChangeCount() {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.ChangeCount++
}

// GetChangeCount 获取变更计数器
func (db *Database) GetChangeCount() int {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.ChangeCount
}

// IsAutoCommit 判断是否为自动提交模式
func (db *Database) IsAutoCommitMode() bool {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.AutoCommit
}

// SetAutoCommit 设置自动提交模式
func (db *Database) SetAutoCommit(autoCommit bool) {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.AutoCommit = autoCommit
}

// GetPageSize 获取页面大小
func (db *Database) GetPageSize() int {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.PageSize
}

// SetPageSize 设置页面大小
func (db *Database) SetPageSize(pageSize int) {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.PageSize = pageSize
}

// GetCacheSize 获取缓存大小
func (db *Database) GetCacheSize() int {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.CacheSize
}

// SetCacheSize 设置缓存大小
func (db *Database) SetCacheSize(cacheSize int) {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.CacheSize = cacheSize
}

// IncStmt 增加活动语句数
func (db *Database) IncStmt() {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.nStmt++
}

// DecStmt 减少活动语句数
func (db *Database) DecStmt() {
	db.mu.Lock()
	defer db.mu.Unlock()
	if db.nStmt > 0 {
		db.nStmt--
	}
}

// GetStmtCount 获取活动语句数
func (db *Database) GetStmtCount() int {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.nStmt
}