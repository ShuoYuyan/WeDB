package main

import (
	"fmt"
	"os"

	"github.com/wedb/wedb/internal/api"
	"github.com/wedb/wedb/internal/storage"
)

func main() {
	// 创建临时数据库文件
	dbFile := "transaction_test.db"
	defer os.Remove(dbFile)

	// 创建数据库
	fmt.Println("Opening database...")
	db, err := storage.NewWeDBDatabase(dbFile, 4096)
	if err != nil {
		fmt.Printf("Failed to create database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// 创建表
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

	fmt.Println("Creating table 'test_transaction'...")
	if err := db.CreateTable(schema); err != nil {
		fmt.Printf("Failed to create table: %v\n", err)
		os.Exit(1)
	}

	// 开始事务 1
	fmt.Println("\n=== Testing COMMIT ===")
	tx1, err := db.Begin()
	if err != nil {
		fmt.Printf("Failed to begin transaction: %v\n", err)
		os.Exit(1)
	}

	// 插入数据
	row1 := map[string]interface{}{
		"id":    1,
		"value": 100,
	}

	fmt.Println("Inserting row (id=1, value=100)...")
	if err := db.InsertRow("test_transaction", row1); err != nil {
		fmt.Printf("Failed to insert row: %v\n", err)
		os.Exit(1)
	}

	// 提交事务
	fmt.Println("Committing transaction...")
	if err := tx1.Commit(); err != nil {
		fmt.Printf("Failed to commit transaction: %v\n", err)
		os.Exit(1)
	}

	// 验证数据已提交
	count1, err := db.Count("test_transaction", "")
	if err != nil {
		fmt.Printf("Failed to count rows: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("After commit: %d rows\n", count1)
	if count1 != 1 {
		fmt.Printf("ERROR: Expected 1 row after commit, got %d\n", count1)
		os.Exit(1)
	}

	// 查询数据
	rows1, err := db.ScanTable("test_transaction")
	if err != nil {
		fmt.Printf("Failed to scan table: %v\n", err)
		os.Exit(1)
	}

	for _, row := range rows1 {
		fmt.Printf("  ID: %v, Value: %v\n", row["id"], row["value"])
	}

	// 测试回滚
	fmt.Println("\n=== Testing ROLLBACK ===")
	tx2, err := db.Begin()
	if err != nil {
		fmt.Printf("Failed to begin transaction: %v\n", err)
		os.Exit(1)
	}

	// 插入另一行数据
	row2 := map[string]interface{}{
		"id":    2,
		"value": 200,
	}

	fmt.Println("Inserting row (id=2, value=200)...")
	if err := db.InsertRow("test_transaction", row2); err != nil {
		fmt.Printf("Failed to insert row: %v\n", err)
		os.Exit(1)
	}

	// 回滚事务
	fmt.Println("Rolling back transaction...")
	if err := tx2.Rollback(); err != nil {
		fmt.Printf("Failed to rollback transaction: %v\n", err)
		os.Exit(1)
	}

	// 验证数据已回滚
	count2, err := db.Count("test_transaction", "")
	if err != nil {
		fmt.Printf("Failed to count rows: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("After rollback: %d rows\n", count2)
	if count2 != 1 {
		fmt.Printf("ERROR: Expected 1 row after rollback, got %d\n", count2)
		os.Exit(1)
	}

	// 查询数据
	rows2, err := db.ScanTable("test_transaction")
	if err != nil {
		fmt.Printf("Failed to scan table: %v\n", err)
		os.Exit(1)
	}

	for _, row := range rows2 {
		fmt.Printf("  ID: %v, Value: %v\n", row["id"], row["value"])
	}

	fmt.Println("\n✅ All tests passed successfully!")
}
