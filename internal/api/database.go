package api

import (
	"context"
	"time"
)

// Database 是 WeDB 存储引擎的核心接口
// 这个接口实现了 WQL/pkg/wql/database.Database 接口
// WeDB 不生成 SQL，直接操作数据（B-Tree、页面、事务）
type Database interface {
	// ===== 表操作 =====

	// CreateTable 创建表
	// 创建新表，定义表结构
	CreateTable(schema *TableSchema) error

	// DropTable 删除表
	// 删除指定表及其所有数据
	DropTable(tableName string) error

	// GetTableSchema 获取表结构
	// 获取表的完整结构信息
	GetTableSchema(tableName string) (*TableSchema, error)

	// ListTables 列出所有表
	// 返回数据库中所有表的名称
	ListTables() []string

	// TableExists 检查表是否存在
	// 检查指定表是否存在
	TableExists(tableName string) bool

	// ===== 数据操作 =====

	// ScanTable 扫描表
	// 直接扫描表，返回所有数据（不使用SQL）
	// 这是WQL的核心方法，支持直接表扫描
	ScanTable(tableName string) ([]map[string]interface{}, error)

	// ScanTableWithColumns 扫描表（指定列）
	// 扫描表，只返回指定列的数据
	ScanTableWithColumns(tableName string, columns []string) ([]map[string]interface{}, error)

	// InsertRow 插入单行
	// 向表中插入一行数据
	InsertRow(tableName string, row map[string]interface{}) error

	// InsertRows 批量插入
	// 向表中批量插入多行数据
	InsertRows(tableName string, rows []map[string]interface{}) error

	// UpdateRow 更新行
	// 更新满足条件的行
	UpdateRow(tableName string, row map[string]interface{}, condition string) error

	// UpdateRows 批量更新
	// 批量更新满足条件的行
	UpdateRows(tableName string, rows []map[string]interface{}, condition string) error

	// DeleteRow 删除行
	// 删除满足条件的行
	DeleteRow(tableName string, condition string) error

	// DeleteRows 批量删除
	// 批量删除满足条件的行
	DeleteRows(tableName string, conditions []string) error

	// ===== 索引操作 =====

	// CreateIndex 创建索引
	// 在指定列上创建索引
	CreateIndex(tableName string, index *IndexInfo) error

	// DropIndex 删除索引
	// 删除指定索引
	DropIndex(tableName string, indexName string) error

	// GetIndexInfo 获取索引信息
	// 获取表的所有索引信息
	GetIndexInfo(tableName string) ([]IndexInfo, error)

	// IndexExists 检查索引是否存在
	// 检查指定索引是否存在
	IndexExists(tableName string, indexName string) bool

	// ===== 事务操作 =====

	// Begin 开始事务
	// 开始一个新事务
	Begin() (Transaction, error)

	// BeginTx 开始事务（带上下文）
	// 带上下文和超时控制的事务
	BeginTx(ctx context.Context, opts *TxOptions) (Transaction, error)

	// ===== 查询操作 =====

	// Count 计数
	// 统计满足条件的行数
	Count(tableName string, condition string) (int64, error)

	// Min 获取最小值
	// 获取指定列的最小值
	Min(tableName string, column string, condition string) (interface{}, error)

	// Max 获取最大值
	// 获取指定列的最大值
	Max(tableName string, column string, condition string) (interface{}, error)

	// Sum 求和
	// 计算指定列的总和
	Sum(tableName string, column string, condition string) (float64, error)

	// Avg 计算平均值
	// 计算指定列的平均值
	Avg(tableName string, column string, condition string) (float64, error)

	// ===== 元数据操作 =====

	// GetTableStats 获取表统计信息
	// 获取表的行数、大小等统计信息
	GetTableStats(tableName string) (*TableStats, error)

	// GetColumnStats 获取列统计信息
	// 获取列的统计信息（最小值、最大值、唯一值数量等）
	GetColumnStats(tableName string, column string) (*ColumnStats, error)

	// ===== 数据库操作 =====

	// Close 关闭数据库
	// 关闭数据库连接，释放资源
	Close() error

	// Ping 检查连接
	// 检查数据库连接是否正常
	Ping() error

	// IsClosed 检查是否已关闭
	// 检查数据库连接是否已关闭
	IsClosed() bool
}

// Transaction 事务接口
// 支持ACID特性的事务操作
type Transaction interface {
	// ===== 基本操作 =====

	// Commit 提交事务
	// 提交事务的所有更改
	Commit() error

	// Rollback 回滚事务
	// 回滚事务的所有更改
	Rollback() error

	// ===== 嵌套事务 =====

	// Savepoint 创建保存点
	// 创建一个保存点，可以回滚到此点
	Savepoint(name string) error

	// RollbackTo 回滚到保存点
	// 回滚到指定的保存点
	RollbackTo(name string) error

	// ReleaseSavepoint 释放保存点
	// 释放指定的保存点
	ReleaseSavepoint(name string) error

	// ===== 状态查询 =====

	// IsActive 检查事务是否活跃
	// 检查事务是否仍然活跃
	IsActive() bool

	// IsolationLevel 获取隔离级别
	// 获取事务的隔离级别
	IsolationLevel() IsolationLevel

	// ===== 事务属性 =====

	// ID 获取事务ID
	// 获取事务的唯一标识符
	ID() string

	// StartTime 获取开始时间
	// 获取事务开始的时间
	StartTime() time.Time

	// Duration 获取持续时间
	// 获取事务持续的时长
	Duration() time.Duration
}

// TxOptions 事务选项
type TxOptions struct {
	// Isolation 隔离级别
	Isolation IsolationLevel
	// ReadOnly 是否只读
	ReadOnly bool
	// Timeout 超时时间
	Timeout time.Duration
}

// IsolationLevel 事务隔离级别
type IsolationLevel int

const (
	// LevelDefault 默认隔离级别
	LevelDefault IsolationLevel = iota
	// LevelReadUncommitted 读未提交
	LevelReadUncommitted
	// LevelReadCommitted 读已提交
	LevelReadCommitted
	// LevelRepeatableRead 可重复读
	LevelRepeatableRead
	// LevelSnapshot 快照隔离
	LevelSnapshot
	// LevelSerializable 可串行化
	LevelSerializable
)

// String 返回隔离级别的字符串表示
func (il IsolationLevel) String() string {
	switch il {
	case LevelDefault:
		return "DEFAULT"
	case LevelReadUncommitted:
		return "READ UNCOMMITTED"
	case LevelReadCommitted:
		return "READ COMMITTED"
	case LevelRepeatableRead:
		return "REPEATABLE READ"
	case LevelSnapshot:
		return "SNAPSHOT"
	case LevelSerializable:
		return "SERIALIZABLE"
	default:
		return "UNKNOWN"
	}
}