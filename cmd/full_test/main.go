package main

import (
	"fmt"
	"os"

	"github.com/wedb/wedb/internal/api"
	"github.com/wedb/wedb/internal/storage"
	"github.com/wedb/wedb/pkg/adapter"
)

func main() {
	// 创建临时数据库文件
	dbFile := "full_test.db"
	defer os.Remove(dbFile)

	// 创建数据库
	fmt.Println("Opening database...")
	db, err := storage.NewWeDBDatabase(dbFile, 4096)
	if err != nil {
		fmt.Printf("Failed to create database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// 创建适配器
	adapter := adapter.NewWeDBAdapter(db)

	// 测试 1: 创建表
	fmt.Println("\n=== Test 1: CreateTable ===")
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
		fmt.Printf("Failed to create table: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✅ Table created successfully!")

	// 测试 2: 插入数据
	fmt.Println("\n=== Test 2: InsertData ===")
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
		fmt.Printf("Failed to insert rows: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ Inserted %d rows successfully!\n", len(rows))

	// 测试 3: 查询数据
	fmt.Println("\n=== Test 3: SelectData ===")
	queryRows, err := adapter.ScanTable("users")
	if err != nil {
		fmt.Printf("Failed to scan table: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ Found %d rows:\n", len(queryRows))
	for _, row := range queryRows {
		fmt.Printf("  ID: %v, Name: %v, Age: %v\n", row["id"], row["name"], row["age"])
	}

	// 测试 4: 更新数据
	fmt.Println("\n=== Test 4: UpdateData ===")
	updateRow := map[string]interface{}{
		"age": 26,
	}

	if err := adapter.UpdateRow("users", updateRow, "*"); err != nil {
		fmt.Printf("Failed to update row: %v\n", err)
		os.Exit(1)
	}

	// 验证更新
	updatedRows, err := adapter.ScanTable("users")
	if err != nil {
		fmt.Printf("Failed to scan table: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ Updated %d rows:\n", len(updatedRows))
	allAge26 := true
	for _, row := range updatedRows {
		if age, ok := row["age"]; ok && age != int64(26) {
			fmt.Printf("  ERROR: Expected age 26, got %v\n", age)
			allAge26 = false
		} else {
			fmt.Printf("  ID: %v, Name: %v, Age: %v\n", row["id"], row["name"], row["age"])
		}
	}
	if allAge26 {
		fmt.Println("✅ All ages updated to 26!")
	} else {
		fmt.Println("❌ Not all ages are 26!")
		os.Exit(1)
	}

	// 测试 5: 删除数据
	fmt.Println("\n=== Test 5: DeleteData ===")
	if err := adapter.DeleteRow("users", ""); err != nil {
		fmt.Printf("Failed to delete row: %v\n", err)
		os.Exit(1)
	}

	// 验证删除
	deleteCount, err := adapter.Count("users", "")
	if err != nil {
		fmt.Printf("Failed to count rows: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ After delete: %d rows (expected 2)\n", deleteCount)
	if deleteCount != 2 {
		fmt.Printf("❌ Expected 2 rows after delete, got %d\n", deleteCount)
		os.Exit(1)
	}

	// 测试 6: 聚合函数
	fmt.Println("\n=== Test 6: Aggregation ===")
	count, err := adapter.Count("users", "")
	if err != nil {
		fmt.Printf("Failed to count: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("  Count: %d (expected 2)\n", count)

	minAge, err := adapter.Min("users", "age", "")
	if err != nil {
		fmt.Printf("Failed to get min: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("  Min Age: %v (expected 26)\n", minAge)

	maxAge, err := adapter.Max("users", "age", "")
	if err != nil {
		fmt.Printf("Failed to get max: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("  Max Age: %v (expected 26)\n", maxAge)

	sumAge, err := adapter.Sum("users", "age", "")
	if err != nil {
		fmt.Printf("Failed to get sum: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("  Sum Age: %v (expected 52.0)\n", sumAge)

	avgAge, err := adapter.Avg("users", "age", "")
	if err != nil {
		fmt.Printf("Failed to get avg: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("  Avg Age: %v (expected 26.0)\n", avgAge)

	// 验证聚合结果
	if count != 2 {
		fmt.Printf("❌ Count mismatch: expected 2, got %d\n", count)
		os.Exit(1)
	}
	if minAge != int64(26) {
		fmt.Printf("❌ Min Age mismatch: expected 26, got %v\n", minAge)
		os.Exit(1)
	}
	if maxAge != int64(26) {
		fmt.Printf("❌ Max Age mismatch: expected 26, got %v\n", maxAge)
		os.Exit(1)
	}
	if sumAge != 52.0 {
		fmt.Printf("❌ Sum Age mismatch: expected 52.0, got %v\n", sumAge)
		os.Exit(1)
	}
	if avgAge != 26.0 {
		fmt.Printf("❌ Avg Age mismatch: expected 26.0, got %v\n", avgAge)
		os.Exit(1)
	}
	fmt.Println("✅ All aggregation tests passed!")

	// 测试 7: 事务
	fmt.Println("\n=== Test 7: Transaction ===")
	txSchema := &api.TableSchema{
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

	// 检查表是否已存在
	if adapter.TableExists("test_transaction") {
		fmt.Println("Dropping existing test_transaction table...")
		adapter.DropTable("test_transaction")
	}

	if err := adapter.CreateTable(txSchema); err != nil {
		fmt.Printf("Failed to create table: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✅ test_transaction table created")
	
	// 检查创建后表的状态
	afterCreateCount, err := adapter.Count("test_transaction", "")
	if err != nil {
		fmt.Printf("Failed to count rows after create: %v\n", err)
	} else {
		fmt.Printf("  After create (before insert): %d rows in test_transaction\n", afterCreateCount)
		
		if afterCreateCount > 0 {
			// 查询表数据以调试
			txRows, err := adapter.ScanTable("test_transaction")
			if err != nil {
				fmt.Printf("Failed to scan test_transaction: %v\n", err)
			} else {
				fmt.Printf("  Unexpected rows in test_transaction after create:\n")
				for _, row := range txRows {
					fmt.Printf("    ID: %v, Value: %v\n", row["id"], row["value"])
				}
			}
		}
	}

	// 测试提交
	fmt.Println("Testing COMMIT...")
	tx1, err := adapter.Begin()
	if err != nil {
		fmt.Printf("Failed to begin transaction: %v\n", err)
		os.Exit(1)
	}

	txRow1 := map[string]interface{}{
		"id":    1,
		"value": 100,
	}

	if err := adapter.InsertRow("test_transaction", txRow1); err != nil {
		fmt.Printf("Failed to insert row: %v\n", err)
		os.Exit(1)
	}

	// 在提交前检查行数
	beforeCommitCount, err := adapter.Count("test_transaction", "")
	if err != nil {
		fmt.Printf("Failed to count rows before commit: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("  Before commit: %d rows in test_transaction\n", beforeCommitCount)

	if err := tx1.Commit(); err != nil {
		fmt.Printf("Failed to commit transaction: %v\n", err)
		os.Exit(1)
	}

	commitCount, err := adapter.Count("test_transaction", "")
	if err != nil {
		fmt.Printf("Failed to count rows: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ After commit: %d rows (expected 1)\n", commitCount)
	if commitCount != 1 {
		fmt.Printf("❌ Expected 1 row after commit, got %d\n", commitCount)
		
		// 查询表数据以调试
		txRows, err := adapter.ScanTable("test_transaction")
		if err != nil {
			fmt.Printf("Failed to scan test_transaction: %v\n", err)
		} else {
			fmt.Printf("  Rows in test_transaction:\n")
			for _, row := range txRows {
				fmt.Printf("    ID: %v, Value: %v\n", row["id"], row["value"])
			}
		}
		os.Exit(1)
	}

	// 测试回滚
	tx2, err := adapter.Begin()
	if err != nil {
		fmt.Printf("Failed to begin transaction: %v\n", err)
		os.Exit(1)
	}

	txRow2 := map[string]interface{}{
		"id":    2,
		"value": 200,
	}

	if err := adapter.InsertRow("test_transaction", txRow2); err != nil {
		fmt.Printf("Failed to insert row: %v\n", err)
		os.Exit(1)
	}

	if err := tx2.Rollback(); err != nil {
		fmt.Printf("Failed to rollback transaction: %v\n", err)
		os.Exit(1)
	}

	rollbackCount, err := adapter.Count("test_transaction", "")
	if err != nil {
		fmt.Printf("Failed to count rows: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ After rollback: %d rows (expected 1)\n", rollbackCount)
	if rollbackCount != 1 {
		fmt.Printf("❌ Expected 1 row after rollback, got %d\n", rollbackCount)
		os.Exit(1)
	}

	// 测试 8: 索引操作
	fmt.Println("\n=== Test 8: IndexOperations ===")
	index := &api.IndexInfo{
		IndexName: "idx_age",
		Columns:   []string{"age"},
		Unique:    false,
		Type:      api.TypeBTree,
	}

	if err := adapter.CreateIndex("users", index); err != nil {
		fmt.Printf("Failed to create index: %v\n", err)
		os.Exit(1)
	}

	if !adapter.IndexExists("users", "idx_age") {
		fmt.Println("❌ Index should exist")
		os.Exit(1)
	}

	indexes, err := adapter.GetIndexInfo("users")
	if err != nil {
		fmt.Printf("Failed to get index info: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ Found %d indexes (expected 1)\n", len(indexes))
	if len(indexes) != 1 {
		fmt.Printf("❌ Expected 1 index, got %d\n", len(indexes))
		os.Exit(1)
	}
	fmt.Printf("  Index: %s on columns %v\n", indexes[0].IndexName, indexes[0].Columns)

	if err := adapter.DropIndex("users", "idx_age"); err != nil {
		fmt.Printf("Failed to drop index: %v\n", err)
		os.Exit(1)
	}

	if adapter.IndexExists("users", "idx_age") {
		fmt.Println("❌ Index should not exist after drop")
		os.Exit(1)
	}
	fmt.Println("✅ Index operations passed!")

	fmt.Println("\n✅ All tests passed successfully!")
}
