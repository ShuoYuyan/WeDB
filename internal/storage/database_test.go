package storage

import (
	"os"
	"testing"
	"time"

	"github.com/wedb/wedb/internal/api"
)

// TestCreateTable 测试创建表
func TestCreateTable(t *testing.T) {
	// 创建临时数据库文件
	dbFile := "test_wedb.db"
	defer os.Remove(dbFile)

	// 打开数据库
	db, err := NewWeDBDatabase(dbFile, 4096)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// 创建表
	schema := &api.TableSchema{
		TableName: "users",
		Columns: []api.ColumnSchema{
			{
				Name: "id",
				Type: api.TypeInteger,
			},
			{
				Name: "name",
				Type: api.TypeText,
			},
			{
				Name: "age",
				Type: api.TypeInteger,
			},
		},
		PrimaryKey: "id",
	}

	if err := db.CreateTable(schema); err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// 验证表存在
	if !db.TableExists("users") {
		t.Error("Table should exist")
	}

	// 获取表结构
	gotSchema, err := db.GetTableSchema("users")
	if err != nil {
		t.Fatalf("Failed to get table schema: %v", err)
	}

	if gotSchema.TableName != "users" {
		t.Errorf("Expected table name 'users', got '%s'", gotSchema.TableName)
	}

	if len(gotSchema.Columns) != 3 {
		t.Errorf("Expected 3 columns, got %d", len(gotSchema.Columns))
	}
}

// TestInsertRow 测试插入行
func TestInsertRow(t *testing.T) {
	// 创建临时数据库文件
	dbFile := "test_wedb_insert.db"
	defer os.Remove(dbFile)

	// 打开数据库
	db, err := NewWeDBDatabase(dbFile, 4096)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// 创建表
	schema := &api.TableSchema{
		TableName: "users",
		Columns: []api.ColumnSchema{
			{
				Name: "id",
				Type: api.TypeInteger,
			},
			{
				Name: "name",
				Type: api.TypeText,
			},
			{
				Name: "age",
				Type: api.TypeInteger,
			},
		},
		PrimaryKey: "id",
	}

	if err := db.CreateTable(schema); err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// 插入行
	row := map[string]interface{}{
		"id":   1,
		"name": "Alice",
		"age":  25,
	}

	if err := db.InsertRow("users", row); err != nil {
		t.Fatalf("Failed to insert row: %v", err)
	}

	// 扫描表验证
	rows, err := db.ScanTable("users")
	if err != nil {
		t.Fatalf("Failed to scan table: %v", err)
	}

	if len(rows) != 1 {
		t.Errorf("Expected 1 row, got %d", len(rows))
	}
}

// TestScanTable 测试扫描表
func TestScanTable(t *testing.T) {
	// 创建临时数据库文件
	dbFile := "test_wedb_scan.db"
	defer os.Remove(dbFile)

	// 打开数据库
	db, err := NewWeDBDatabase(dbFile, 4096)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// 创建表
	schema := &api.TableSchema{
		TableName: "users",
		Columns: []api.ColumnSchema{
			{
				Name: "id",
				Type: api.TypeInteger,
			},
			{
				Name: "name",
				Type: api.TypeText,
			},
			{
				Name: "age",
				Type: api.TypeInteger,
			},
		},
		PrimaryKey: "id",
	}

	if err := db.CreateTable(schema); err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// 插入多行
	rows := []map[string]interface{}{
		{"id": 1, "name": "Alice", "age": 25},
		{"id": 2, "name": "Bob", "age": 30},
		{"id": 3, "name": "Charlie", "age": 35},
	}

	if err := db.InsertRows("users", rows); err != nil {
		t.Fatalf("Failed to insert rows: %v", err)
	}

	// 扫描表
	scannedRows, err := db.ScanTable("users")
	if err != nil {
		t.Fatalf("Failed to scan table: %v", err)
	}

	if len(scannedRows) != 3 {
		t.Errorf("Expected 3 rows, got %d", len(scannedRows))
	}
}

// TestCount 测试计数
func TestCount(t *testing.T) {
	// 创建临时数据库文件
	dbFile := "test_wedb_count.db"
	defer os.Remove(dbFile)

	// 打开数据库
	db, err := NewWeDBDatabase(dbFile, 4096)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// 创建表
	schema := &api.TableSchema{
		TableName: "users",
		Columns: []api.ColumnSchema{
			{
				Name: "id",
				Type: api.TypeInteger,
			},
			{
				Name: "name",
				Type: api.TypeText,
			},
		},
		PrimaryKey: "id",
	}

	if err := db.CreateTable(schema); err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// 插入多行
	rows := []map[string]interface{}{
		{"id": 1, "name": "Alice"},
		{"id": 2, "name": "Bob"},
		{"id": 3, "name": "Charlie"},
	}

	if err := db.InsertRows("users", rows); err != nil {
		t.Fatalf("Failed to insert rows: %v", err)
	}

	// 计数
	count, err := db.Count("users", "")
	if err != nil {
		t.Fatalf("Failed to count: %v", err)
	}

	if count != 3 {
		t.Errorf("Expected count 3, got %d", count)
	}
}

// TestMinAndMax 测试最小值和最大值
func TestMinAndMax(t *testing.T) {
	// 创建临时数据库文件
	dbFile := "test_wedb_minmax.db"
	defer os.Remove(dbFile)

	// 打开数据库
	db, err := NewWeDBDatabase(dbFile, 4096)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// 创建表
	schema := &api.TableSchema{
		TableName: "users",
		Columns: []api.ColumnSchema{
			{
				Name: "id",
				Type: api.TypeInteger,
			},
			{
				Name: "age",
				Type: api.TypeInteger,
			},
		},
		PrimaryKey: "id",
	}

	if err := db.CreateTable(schema); err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// 插入多行
	rows := []map[string]interface{}{
		{"id": 1, "age": 25},
		{"id": 2, "age": 30},
		{"id": 3, "age": 35},
	}

	if err := db.InsertRows("users", rows); err != nil {
		t.Fatalf("Failed to insert rows: %v", err)
	}

	// 测试最小值
	min, err := db.Min("users", "age", "")
	if err != nil {
		t.Fatalf("Failed to get min: %v", err)
	}

	minInt, ok := min.(int64)
	if !ok || minInt != 25 {
		t.Errorf("Expected min 25, got %v", min)
	}

	// 测试最大值
	max, err := db.Max("users", "age", "")
	if err != nil {
		t.Fatalf("Failed to get max: %v", err)
	}

	maxInt, ok := max.(int64)
	if !ok || maxInt != 35 {
		t.Errorf("Expected max 35, got %v", max)
	}
}

// TestSumAndAvg 测试求和和平均值
func TestSumAndAvg(t *testing.T) {
	// 创建临时数据库文件
	dbFile := "test_wedb_sumavg.db"
	defer os.Remove(dbFile)

	// 打开数据库
	db, err := NewWeDBDatabase(dbFile, 4096)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// 创建表
	schema := &api.TableSchema{
		TableName: "users",
		Columns: []api.ColumnSchema{
			{
				Name: "id",
				Type: api.TypeInteger,
			},
			{
				Name: "age",
				Type: api.TypeInteger,
			},
		},
		PrimaryKey: "id",
	}

	if err := db.CreateTable(schema); err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// 插入多行
	rows := []map[string]interface{}{
		{"id": 1, "age": 25},
		{"id": 2, "age": 30},
		{"id": 3, "age": 35},
	}

	if err := db.InsertRows("users", rows); err != nil {
		t.Fatalf("Failed to insert rows: %v", err)
	}

	// 测试求和
	sum, err := db.Sum("users", "age", "")
	if err != nil {
		t.Fatalf("Failed to get sum: %v", err)
	}

	expectedSum := 25.0 + 30.0 + 35.0
	if sum != expectedSum {
		t.Errorf("Expected sum %f, got %f", expectedSum, sum)
	}

	// 测试平均值
	avg, err := db.Avg("users", "age", "")
	if err != nil {
		t.Fatalf("Failed to get avg: %v", err)
	}

	expectedAvg := expectedSum / 3.0
	if avg != expectedAvg {
		t.Errorf("Expected avg %f, got %f", expectedAvg, avg)
	}
}

// TestTransaction 测试事务
func TestTransaction(t *testing.T) {
	// 创建临时数据库文件
	dbFile := "test_wedb_transaction.db"
	defer os.Remove(dbFile)

	// 打开数据库
	db, err := NewWeDBDatabase(dbFile, 4096)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// 创建表
	schema := &api.TableSchema{
		TableName: "users",
		Columns: []api.ColumnSchema{
			{
				Name: "id",
				Type: api.TypeInteger,
			},
			{
				Name: "name",
				Type: api.TypeText,
			},
		},
		PrimaryKey: "id",
	}

	if err := db.CreateTable(schema); err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// 开始事务
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	// 插入数据
	row := map[string]interface{}{
		"id":   1,
		"name": "Alice",
	}

	if err := db.InsertRow("users", row); err != nil {
		t.Fatalf("Failed to insert row: %v", err)
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		t.Fatalf("Failed to commit transaction: %v", err)
	}

	// 验证数据
	rows, err := db.ScanTable("users")
	if err != nil {
		t.Fatalf("Failed to scan table: %v", err)
	}

	if len(rows) != 1 {
		t.Errorf("Expected 1 row, got %d", len(rows))
	}
}

// TestCircuitBreaker 测试熔断器功能
func TestCircuitBreaker(t *testing.T) {
	// 创建熔断器，失败阈值为3，超时时间为1秒
	cb := NewCircuitBreaker(3, 1*time.Second)

	// 初始状态应该是关闭的
	if cb.GetState() != StateClosed {
		t.Errorf("Expected initial state to be Closed, got %v", cb.GetState())
	}

	// 应该允许操作
	if !cb.Allow() {
		t.Error("Should allow operations when circuit is closed")
	}

	// 记录3次失败
	cb.RecordFailure()
	cb.RecordFailure()
	cb.RecordFailure()

	// 应该打开熔断器
	if cb.GetState() != StateOpen {
		t.Errorf("Expected circuit to be Open after 3 failures, got %v", cb.GetState())
	}

	// 不应该允许操作
	if cb.Allow() {
		t.Error("Should not allow operations when circuit is open")
	}

	// 记录成功（熔断器打开时不影响）
	cb.RecordSuccess()

	// 等待超时
	time.Sleep(2 * time.Second)

	// 应该尝试恢复，进入半开状态
	if cb.Allow() {
		// 第一次允许，状态变为半开
		if cb.GetState() != StateHalfOpen {
			t.Errorf("Expected state to be HalfOpen after timeout, got %v", cb.GetState())
		}
	}

	// 记录3次成功
	cb.RecordSuccess()
	cb.RecordSuccess()
	cb.RecordSuccess()

	// 应该恢复到关闭状态
	if cb.GetState() != StateClosed {
		t.Errorf("Expected circuit to be Closed after successful recovery, got %v", cb.GetState())
	}
}

// TestCircuitBreakerHalfOpenFailure 测试半开状态下失败
func TestCircuitBreakerHalfOpenFailure(t *testing.T) {
	cb := NewCircuitBreaker(3, 1*time.Second)

	// 触发熔断器打开
	cb.RecordFailure()
	cb.RecordFailure()
	cb.RecordFailure()

	// 等待超时
	time.Sleep(2 * time.Second)

	// 进入半开状态
	cb.Allow()

	// 记录一次成功
	cb.RecordSuccess()

	// 记录一次失败，应该重新打开
	cb.RecordFailure()

	if cb.GetState() != StateOpen {
		t.Errorf("Expected circuit to reopen on HalfOpen failure, got %v", cb.GetState())
	}
}

// TestDatabaseMetrics 测试数据库指标收集
func TestDatabaseMetrics(t *testing.T) {
	metrics := NewDatabaseMetrics()

	// 测试初始状态
	if metrics.QueryCount.Load() != 0 {
		t.Errorf("Expected initial query count to be 0, got %d", metrics.QueryCount.Load())
	}

	// 模拟一些查询
	metrics.QueryCount.Add(10)
	metrics.QueryTime.Add(1_000_000_000) // 1秒
	metrics.CacheHitCount.Add(8)
	metrics.CacheMissCount.Add(2)

	// 测试缓存命中率
	hitRate := metrics.GetCacheHitRate()
	expectedHitRate := 80.0 // 8/(8+2)*100
	if hitRate != expectedHitRate {
		t.Errorf("Expected hit rate %.1f, got %.1f", expectedHitRate, hitRate)
	}

	// 测试平均查询时间
	avgTime := metrics.GetAvgQueryTime()
	expectedAvgTime := 100.0 // 1秒 / 10查询
	if avgTime != expectedAvgTime {
		t.Errorf("Expected avg query time %.1fms, got %.1fms", expectedAvgTime, avgTime)
	}

	// 测试各种操作计数
	metrics.InsertCount.Add(5)
	if metrics.InsertCount.Load() != 5 {
		t.Errorf("Expected insert count 5, got %d", metrics.InsertCount.Load())
	}
}

// TestDatabaseMetricsZeroQueries 测试零查询情况
func TestDatabaseMetricsZeroQueries(t *testing.T) {
	metrics := NewDatabaseMetrics()

	// 没有查询时，平均时间应该是0
	avgTime := metrics.GetAvgQueryTime()
	if avgTime != 0.0 {
		t.Errorf("Expected avg query time 0.0 for no queries, got %.1f", avgTime)
	}

	// 没有缓存操作时，命中率应该是0
	hitRate := metrics.GetCacheHitRate()
	if hitRate != 0.0 {
		t.Errorf("Expected hit rate 0.0 for no cache operations, got %.1f", hitRate)
	}
}

// TestIntegration_CRUDOperations 测试完整的CRUD操作集成
func TestIntegration_CRUDOperations(t *testing.T) {
	// 创建临时数据库文件
	dbFile := "test_integration_crud.db"
	defer os.Remove(dbFile)

	// 打开数据库
	db, err := NewWeDBDatabase(dbFile, 4096)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// 1. 创建表
	schema := &api.TableSchema{
		TableName: "products",
		Columns: []api.ColumnSchema{
			{Name: "id", Type: api.TypeInteger},
			{Name: "name", Type: api.TypeText},
			{Name: "price", Type: api.TypeInteger},
		},
		PrimaryKey: "id",
	}

	if err := db.CreateTable(schema); err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// 2. 插入多条记录
	products := []map[string]interface{}{
		{"id": 1, "name": "Product A", "price": 100},
		{"id": 2, "name": "Product B", "price": 200},
		{"id": 3, "name": "Product C", "price": 300},
	}

	if err := db.InsertRows("products", products); err != nil {
		t.Fatalf("Failed to insert rows: %v", err)
	}

	// 3. 验证插入
	rows, err := db.ScanTable("products")
	if err != nil {
		t.Fatalf("Failed to scan table: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("Expected 3 rows, got %d", len(rows))
	}

	// 4. 更新记录
	updateRow := map[string]interface{}{
		"id":    2,
		"name":  "Product B Updated",
		"price": 250,
	}
	if err := db.UpdateRow("products", updateRow, "id = 2"); err != nil {
		t.Fatalf("Failed to update row: %v", err)
	}

	// 5. 验证更新
	updatedRows, err := db.ScanTable("products")
	if err != nil {
		t.Fatalf("Failed to scan table after update: %v", err)
	}
	for _, row := range updatedRows {
		if row["id"].(int64) == 2 {
			if row["name"].(string) != "Product B Updated" {
				t.Errorf("Expected name 'Product B Updated', got '%s'", row["name"])
			}
			if row["price"].(int64) != 250 {
				t.Errorf("Expected price 250, got %d", row["price"])
			}
		}
	}

	// 6. 删除记录
	if err := db.DeleteRow("products", "id = 3"); err != nil {
		t.Fatalf("Failed to delete row: %v", err)
	}

	// 7. 验证删除
	remainingRows, err := db.ScanTable("products")
	if err != nil {
		t.Fatalf("Failed to scan table after delete: %v", err)
	}
	if len(remainingRows) != 2 {
		t.Fatalf("Expected 2 rows after delete, got %d", len(remainingRows))
	}

	// 8. 验证健康状态
	if err := db.HealthCheck(); err != nil {
		t.Errorf("Health check failed: %v", err)
	}

	// 9. 获取指标
	metrics := db.GetMetrics()
	if _, ok := metrics["query"]; !ok {
		t.Error("Expected metrics to contain query information")
	}
}

// TestIntegration_WithIndex 测试带索引的集成操作
func TestIntegration_WithIndex(t *testing.T) {
	// 创建临时数据库文件
	dbFile := "test_integration_index.db"
	defer os.Remove(dbFile)

	// 打开数据库
	db, err := NewWeDBDatabase(dbFile, 4096)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// 创建表
	schema := &api.TableSchema{
		TableName: "orders",
		Columns: []api.ColumnSchema{
			{Name: "id", Type: api.TypeInteger},
			{Name: "customer_id", Type: api.TypeInteger},
			{Name: "amount", Type: api.TypeInteger},
		},
		PrimaryKey: "id",
	}

	if err := db.CreateTable(schema); err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// 创建索引
	if err := db.CreateIndex("orders", &api.IndexInfo{
		IndexName: "idx_customer",
		Columns:   []string{"customer_id"},
		Unique:    false,
	}); err != nil {
		t.Fatalf("Failed to create index: %v", err)
	}

	// 插入数据
	orders := []map[string]interface{}{
		{"id": 1, "customer_id": 100, "amount": 500},
		{"id": 2, "customer_id": 100, "amount": 300},
		{"id": 3, "customer_id": 200, "amount": 700},
	}

	if err := db.InsertRows("orders", orders); err != nil {
		t.Fatalf("Failed to insert orders: %v", err)
	}

	// 查询特定客户的订单
	customerOrders, err := db.ScanTable("orders")
	if err != nil {
		t.Fatalf("Failed to scan orders: %v", err)
	}

	customer100Count := 0
	for _, order := range customerOrders {
		if order["customer_id"].(int64) == 100 {
			customer100Count++
		}
	}

	if customer100Count != 2 {
		t.Errorf("Expected 2 orders for customer 100, got %d", customer100Count)
	}
}

// BenchmarkInsertRow 插入行性能测试
func BenchmarkInsertRow(b *testing.B) {
	// 创建临时数据库文件
	dbFile := "bench_insert.db"
	defer os.Remove(dbFile)

	// 打开数据库
	db, err := NewWeDBDatabase(dbFile, 4096)
	if err != nil {
		b.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// 创建表
	schema := &api.TableSchema{
		TableName: "bench_table",
		Columns: []api.ColumnSchema{
			{Name: "id", Type: api.TypeInteger},
			{Name: "data", Type: api.TypeText},
		},
		PrimaryKey: "id",
	}

	if err := db.CreateTable(schema); err != nil {
		b.Fatalf("Failed to create table: %v", err)
	}

	// 重置计时器
	b.ResetTimer()

	// 执行基准测试
	for i := 0; i < b.N; i++ {
		row := map[string]interface{}{
			"id":   int64(i),
			"data": "benchmark data",
		}
		if err := db.InsertRow("bench_table", row); err != nil {
			b.Fatalf("Failed to insert row: %v", err)
		}
	}
}

// BenchmarkScanTable 扫描表性能测试
func BenchmarkScanTable(b *testing.B) {
	// 创建临时数据库文件
	dbFile := "bench_scan.db"
	defer os.Remove(dbFile)

	// 打开数据库
	db, err := NewWeDBDatabase(dbFile, 4096)
	if err != nil {
		b.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// 创建表
	schema := &api.TableSchema{
		TableName: "bench_table",
		Columns: []api.ColumnSchema{
			{Name: "id", Type: api.TypeInteger},
			{Name: "data", Type: api.TypeText},
		},
		PrimaryKey: "id",
	}

	if err := db.CreateTable(schema); err != nil {
		b.Fatalf("Failed to create table: %v", err)
	}

	// 插入1000条数据
	for i := 0; i < 1000; i++ {
		row := map[string]interface{}{
			"id":   int64(i),
			"data": "benchmark data",
		}
		if err := db.InsertRow("bench_table", row); err != nil {
			b.Fatalf("Failed to insert row: %v", err)
		}
	}

	// 重置计时器
	b.ResetTimer()

	// 执行基准测试
	for i := 0; i < b.N; i++ {
		_, err := db.ScanTable("bench_table")
		if err != nil {
			b.Fatalf("Failed to scan table: %v", err)
		}
	}
}

// BenchmarkUpdateRow 更新行性能测试
func BenchmarkUpdateRow(b *testing.B) {
	// 创建临时数据库文件
	dbFile := "bench_update.db"
	defer os.Remove(dbFile)

	// 打开数据库
	db, err := NewWeDBDatabase(dbFile, 4096)
	if err != nil {
		b.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// 创建表
	schema := &api.TableSchema{
		TableName: "bench_table",
		Columns: []api.ColumnSchema{
			{Name: "id", Type: api.TypeInteger},
			{Name: "data", Type: api.TypeText},
		},
		PrimaryKey: "id",
	}

	if err := db.CreateTable(schema); err != nil {
		b.Fatalf("Failed to create table: %v", err)
	}

	// 插入初始数据
	row := map[string]interface{}{
		"id":   1,
		"data": "initial data",
	}
	if err := db.InsertRow("bench_table", row); err != nil {
		b.Fatalf("Failed to insert row: %v", err)
	}

	// 重置计时器
	b.ResetTimer()

	// 执行基准测试
	for i := 0; i < b.N; i++ {
		updateRow := map[string]interface{}{
			"id":   1,
			"data": "updated data",
		}
		if err := db.UpdateRow("bench_table", updateRow, "id = 1"); err != nil {
			b.Fatalf("Failed to update row: %v", err)
		}
	}
}

// BenchmarkCircuitBreaker 熔断器性能测试
func BenchmarkCircuitBreaker(b *testing.B) {
	cb := NewCircuitBreaker(100, 30*time.Second)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if cb.Allow() {
			cb.RecordSuccess()
		} else {
			cb.RecordFailure()
		}
	}
}

// BenchmarkDatabaseMetrics 指标收集性能测试
func BenchmarkDatabaseMetrics(b *testing.B) {
	metrics := NewDatabaseMetrics()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		metrics.QueryCount.Add(1)
		metrics.QueryTime.Add(1000000) // 1ms
		metrics.CacheHitCount.Add(1)
		metrics.CacheMissCount.Add(1)
		_ = metrics.GetCacheHitRate()
		_ = metrics.GetAvgQueryTime()
	}
}