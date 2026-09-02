package adapter

import (
	"os"
	"testing"

	"github.com/wedb/wedb/internal/api"
	"github.com/wedb/wedb/internal/storage"
)

// TestIntegration 完整的集成测试
func TestIntegration(t *testing.T) {
	// 创建临时数据库文件
	dbFile := "test_integration.db"
	defer os.Remove(dbFile)
	defer os.Remove(dbFile + "-journal")
	defer os.Remove(dbFile + ".metadata")

	// 创建数据库
	db, err := storage.NewWeDBDatabase(dbFile, 4096)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// 创建适配器
	adapter := NewWeDBAdapter(db)

	// 测试创建表
	t.Run("CreateTable", testCreateTable(adapter))
	// 测试插入数据
	t.Run("InsertData", testInsertData(adapter))
	// 测试查询数据
	t.Run("SelectData", testSelectData(adapter))
	// 测试更新数据
	t.Run("UpdateData", testUpdateData(adapter))
	// 测试删除数据
	t.Run("DeleteData", testDeleteData(adapter))
	// 测试索引
	t.Run("IndexOperations", testIndexOperations(adapter))
	// 测试事务
	t.Run("Transaction", testTransaction(adapter))
	// 测试聚合函数
	t.Run("Aggregation", testAggregation(adapter))
}

func testCreateTable(adapter *WeDBAdapter) func(*testing.T) {
	return func(t *testing.T) {
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
				{
					Name: "email",
					Type: api.TypeText,
				},
			},
			PrimaryKey: "id",
		}

		if err := adapter.CreateTable(schema); err != nil {
			t.Fatalf("Failed to create table: %v", err)
		}

		// 验证表存在
		if !adapter.TableExists("users") {
			t.Error("Table should exist")
		}

		// 验证表结构
		gotSchema, err := adapter.GetTableSchema("users")
		if err != nil {
			t.Fatalf("Failed to get table schema: %v", err)
		}

		if gotSchema.TableName != "users" {
			t.Errorf("Expected table name 'users', got '%s'", gotSchema.TableName)
		}

		if len(gotSchema.Columns) != 4 {
			t.Errorf("Expected 4 columns, got %d", len(gotSchema.Columns))
		}

		if gotSchema.PrimaryKey != "id" {
			t.Errorf("Expected primary key 'id', got '%s'", gotSchema.PrimaryKey)
		}
	}
}

func testInsertData(adapter *WeDBAdapter) func(*testing.T) {
	return func(t *testing.T) {
		// 插入多行数据
		rows := []map[string]interface{}{
			{
				"id":    1,
				"name":  "Alice",
				"age":   25,
				"email": "alice@example.com",
			},
			{
				"id":    2,
				"name":  "Bob",
				"age":   30,
				"email": "bob@example.com",
			},
			{
				"id":    3,
				"name":  "Charlie",
				"age":   35,
				"email": "charlie@example.com",
			},
		}

		if err := adapter.InsertRows("users", rows); err != nil {
			t.Fatalf("Failed to insert rows: %v", err)
		}

		// 验证数据数量
		count, err := adapter.Count("users", "")
		if err != nil {
			t.Fatalf("Failed to count rows: %v", err)
		}

		if count != 3 {
			t.Errorf("Expected 3 rows, got %d", count)
		}
	}
}

func testSelectData(adapter *WeDBAdapter) func(*testing.T) {
	return func(t *testing.T) {
		// 查询所有数据
		rows, err := adapter.ScanTable("users")
		if err != nil {
			t.Fatalf("Failed to scan table: %v", err)
		}

		if len(rows) != 3 {
			t.Errorf("Expected 3 rows, got %d", len(rows))
		}

		// 验证数据内容
		for _, row := range rows {
			if _, ok := row["id"]; !ok {
				t.Error("Row should have 'id' column")
			}
			if _, ok := row["name"]; !ok {
				t.Error("Row should have 'name' column")
			}
		}
	}
}

func testUpdateData(adapter *WeDBAdapter) func(*testing.T) {
	return func(t *testing.T) {
		// 更新数据
		row := map[string]interface{}{
			"age": 26,
		}

		if err := adapter.UpdateRow("users", row, "*"); err != nil {
			t.Fatalf("Failed to update row: %v", err)
		}

		// 验证更新
		rows, err := adapter.ScanTable("users")
		if err != nil {
			t.Fatalf("Failed to scan table: %v", err)
		}

		// 检查是否所有 age 都被更新为 26
		for _, row := range rows {
			if age, ok := row["age"]; ok && age != int64(26) {
				t.Errorf("Expected age to be 26, got %v", age)
			}
		}
	}
}

func testDeleteData(adapter *WeDBAdapter) func(*testing.T) {
	return func(t *testing.T) {
		// 删除一行数据（使用 WHERE 子句只删除 id=1 的行）
		if err := adapter.DeleteRow("users", "id = 1"); err != nil {
			t.Fatalf("Failed to delete row: %v", err)
		}

		// 验证删除后的数据数量
		count, err := adapter.Count("users", "")
		if err != nil {
			t.Fatalf("Failed to count rows: %v", err)
		}

		if count != 2 {
			t.Errorf("Expected 2 rows after delete, got %d", count)
		}
	}
}

func testIndexOperations(adapter *WeDBAdapter) func(*testing.T) {
	return func(t *testing.T) {
		// 创建索引
		index := &api.IndexInfo{
			IndexName: "idx_age",
			Columns:   []string{"age"},
			Unique:    false,
			Type:      api.TypeBTree,
		}

		if err := adapter.CreateIndex("users", index); err != nil {
			t.Fatalf("Failed to create index: %v", err)
		}

		// 验证索引存在
		if !adapter.IndexExists("users", "idx_age") {
			t.Error("Index should exist")
		}

		// 获取索引信息
		indexes, err := adapter.GetIndexInfo("users")
		if err != nil {
			t.Fatalf("Failed to get index info: %v", err)
		}

		if len(indexes) != 1 {
			t.Errorf("Expected 1 index, got %d", len(indexes))
		}

		// 删除索引
		if err := adapter.DropIndex("users", "idx_age"); err != nil {
			t.Fatalf("Failed to drop index: %v", err)
		}

		// 验证索引已删除
		if adapter.IndexExists("users", "idx_age") {
			t.Error("Index should not exist after drop")
		}
	}
}

func testTransaction(adapter *WeDBAdapter) func(*testing.T) {
	return func(t *testing.T) {
		// 创建测试表
		schema := &api.TableSchema{
			TableName: "test_transaction",
			Columns: []api.ColumnSchema{
				{
					Name: "id",
					Type: api.TypeInteger,
				},
				{
					Name: "value",
					Type: api.TypeInteger,
				},
			},
			PrimaryKey: "id",
		}

		if err := adapter.CreateTable(schema); err != nil {
			t.Fatalf("Failed to create table: %v", err)
		}
		defer adapter.DropTable("test_transaction")

		// 开始事务
		tx, err := adapter.Begin()
		if err != nil {
			t.Fatalf("Failed to begin transaction: %v", err)
		}

		// 插入数据
		row := map[string]interface{}{
			"id":    1,
			"value": 100,
		}

		if err := adapter.InsertRow("test_transaction", row); err != nil {
			t.Fatalf("Failed to insert row: %v", err)
		}

		// 提交事务
		if err := tx.Commit(); err != nil {
			t.Fatalf("Failed to commit transaction: %v", err)
		}

		// 验证数据已提交
		count, err := adapter.Count("test_transaction", "")
		if err != nil {
			t.Fatalf("Failed to count rows: %v", err)
		}

		if count != 1 {
			t.Errorf("Expected 1 row after commit, got %d", count)
		}

		// 测试回滚
		tx2, err := adapter.Begin()
		if err != nil {
			t.Fatalf("Failed to begin transaction: %v", err)
		}

		row2 := map[string]interface{}{
			"id":    2,
			"value": 200,
		}

		if err := adapter.InsertRow("test_transaction", row2); err != nil {
			t.Fatalf("Failed to insert row: %v", err)
		}

		// 回滚事务
		if err := tx2.Rollback(); err != nil {
			t.Fatalf("Failed to rollback transaction: %v", err)
		}

		// 验证数据未回滚
		count2, err := adapter.Count("test_transaction", "")
		if err != nil {
			t.Fatalf("Failed to count rows: %v", err)
		}

		if count2 != 1 {
			t.Errorf("Expected 1 row after rollback, got %d", count2)
		}
	}
}

func testAggregation(adapter *WeDBAdapter) func(*testing.T) {
	return func(t *testing.T) {
		// 测试聚合函数
		// 注意：此时数据已经过 UpdateData（所有age=26）和DeleteData（删除1行）
		// 所以剩余2行的age都应该是26

		// 测试 Count
		count, err := adapter.Count("users", "")
		if err != nil {
			t.Fatalf("Failed to count: %v", err)
		}
		if count != 2 {
			t.Errorf("Expected count 2, got %d", count)
		}

		// 测试 Min
		min, err := adapter.Min("users", "age", "")
		if err != nil {
			t.Fatalf("Failed to get min: %v", err)
		}
		minInt, ok := min.(int64)
		if !ok || minInt != 26 {
			t.Errorf("Expected min 26, got %v", min)
		}

		// 测试 Max
		maxVal, err := adapter.Max("users", "age", "")
		if err != nil {
			t.Fatalf("Failed to get max: %v", err)
		}
		maxInt, ok := maxVal.(int64)
		if !ok || maxInt != 26 {
			t.Errorf("Expected max 26, got %v", maxVal)
		}

		// 测试 Sum
		sum, err := adapter.Sum("users", "age", "")
		if err != nil {
			t.Fatalf("Failed to get sum: %v", err)
		}
		if sum != 52.0 { // 26 + 26
			t.Errorf("Expected sum 52.0, got %f", sum)
		}

		// 测试 Avg
		avg, err := adapter.Avg("users", "age", "")
		if err != nil {
			t.Fatalf("Failed to get avg: %v", err)
		}
		if avg != 26.0 { // (26 + 26) / 2
			t.Errorf("Expected average 26.0, got %f", avg)
		}

		// 测试表统计
		stats, err := adapter.GetTableStats("users")
		if err != nil {
			t.Fatalf("Failed to get table stats: %v", err)
		}
		if stats.RowCount != 2 {
			t.Errorf("Expected row count 2, got %d", stats.RowCount)
		}
		if stats.IndexCount != 0 {
			t.Errorf("Expected index count 0, got %d", stats.IndexCount)
		}

// 测试 GetColumnStats
		colStats, err := adapter.GetColumnStats("users", "age")
		if err != nil {
			t.Fatalf("Failed to get column stats: %v", err)
		}
		minFloat, ok := colStats.Min.(float64)
		if !ok || minFloat != 26.0 {
			t.Errorf("Expected min 26.0, got %v", colStats.Min)
		}
		maxFloat, ok := colStats.Max.(float64)
		if !ok || maxFloat != 26.0 {
			t.Errorf("Expected max 26.0, got %v", colStats.Max)
		}
		if colStats.Average != 26.0 {
			t.Errorf("Expected average 26.0, got %f", colStats.Average)
		}
	}
}