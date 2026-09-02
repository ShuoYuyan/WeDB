package api

// IndexInfo 表示索引信息
type IndexInfo struct {
	// IndexName 索引名
	IndexName string
	// Columns 索引列
	Columns []string
	// Unique 是否唯一索引
	Unique bool
	// Type 索引类型
	Type IndexType
}

// IndexType 索引类型
type IndexType string

const (
	// TypeBTree B-Tree 索引
	TypeBTree IndexType = "BTREE"
	// TypeHash 哈希索引
	TypeHash IndexType = "HASH"
	// TypeFullText 全文索引
	TypeFullText IndexType = "FULLTEXT"
)