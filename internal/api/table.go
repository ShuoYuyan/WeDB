package api

import "time"

// TableSchema 表示表的结构定义
type TableSchema struct {
	// TableName 表名
	TableName string
	// Columns 列定义
	Columns []ColumnSchema
	// PrimaryKey 主键列名
	PrimaryKey string
	// AutoIncrement 是否自增
	AutoIncrement bool
}

// ColumnSchema 表示列的定义
type ColumnSchema struct {
	// Name 列名
	Name string
	// Type 数据类型
	Type ColumnType
	// Nullable 是否可为 NULL
	Nullable bool
	// Default 默认值
	Default interface{}
	// PrimaryKey 是否主键
	PrimaryKey bool
	// AutoIncrement 是否自增
	AutoIncrement bool
	// Unique 是否唯一
	Unique bool
}

// ColumnType 列数据类型
type ColumnType string

const (
	// TypeInteger 整数类型
	TypeInteger ColumnType = "INTEGER"
	// TypeReal 浮点数类型
	TypeReal ColumnType = "REAL"
	// TypeText 文本类型
	TypeText ColumnType = "TEXT"
	// TypeBlob 二进制类型
	TypeBlob ColumnType = "BLOB"
	// TypeNull NULL 类型
	TypeNull ColumnType = "NULL"
)

// SortOrder 排序方向
type SortOrder string

const (
	// SortAsc 升序
	SortAsc SortOrder = "ASC"
	// SortDesc 降序
	SortDesc SortOrder = "DESC"
)

// SortBy 排序条件
type SortBy struct {
	// Column 列名
	Column string
	// Order 排序方向
	Order SortOrder
}

// QueryOptions 查询选项
type QueryOptions struct {
	// Columns 要查询的列（空表示所有列）
	Columns []string
	// WHERE 条件
	Where string
	// 排序条件
	OrderBy []SortBy
	// LIMIT 数量
	Limit int
	// OFFSET 偏移量
	Offset int
}

// TableStats 表统计信息
type TableStats struct {
	// RowCount 行数
	RowCount int64
	// TableSize 表大小（字节）
	TableSize int64
	// IndexCount 索引数量
	IndexCount int
	// ColumnCount 列数量
	ColumnCount int
	// LastModified 最后修改时间
	LastModified time.Time
	// Created 创建时间
	Created time.Time
}

// ColumnStats 列统计信息
type ColumnStats struct {
	// ColumnName 列名
	ColumnName string
	// Type 数据类型
	Type ColumnType
	// Min 最小值
	Min interface{}
	// Max 最大值
	Max interface{}
	// UniqueCount 唯一值数量
	UniqueCount int64
	// NullCount NULL 值数量
	NullCount int64
	// Average 平均值（仅数值类型）
	Average float64
	// StdDev 标准差（仅数值类型）
	StdDev float64
}