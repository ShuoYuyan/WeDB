package main

import (
	"fmt"
	"log"

	"github.com/wedb/wedb/internal/api"
	"github.com/wedb/wedb/internal/storage"
)

func main() {
	// 创建数据库
	db, err := storage.NewWeDBDatabase("test_where.db", 4096)
	if err != nil {
		log.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// 创建表
	schema := &api.TableSchema{
		TableName: "users",
		Columns: []api.ColumnSchema{
			{Name: "id", Type: api.TypeInteger},
			{Name: "name", Type: api.TypeText},
			{Name: "age", Type: api.TypeInteger},
		},
		PrimaryKey: "id",
	}

	if err := db.CreateTable(schema); err != nil {
		log.Fatalf("Failed to create table: %v", err)
	}

	// 插入测试数据
	rows := []map[string]interface{}{
		{"id": 1, "name": "Alice", "age": 25},
		{"id": 2, "name": "Bob", "age": 30},
		{"id": 3, "name": "Charlie", "age": 35},
	}

	for _, row := range rows {
		if err := db.InsertRow("users", row); err != nil {
			log.Fatalf("Failed to insert row: %v", err)
		}
	}

	// 测试 1: 全表扫描
	fmt.Println("Test 1: 全表扫描")
	allRows, err := db.ScanTable("users")
	if err != nil {
		log.Fatalf("Failed to scan table: %v", err)
	}
	fmt.Printf("  Total rows: %d\n", len(allRows))

	// 测试 2: WHERE 条件为空字符串
	fmt.Println("\nTest 2: WHERE 条件为空字符串")
	opts := &api.QueryOptions{
		Where: "",
	}
	filteredRows, err := db.ScanTableWithOptions("users", opts)
	if err != nil {
		log.Fatalf("Failed to scan with options: %v", err)
	}
	fmt.Printf("  Filtered rows: %d\n", len(filteredRows))

	// 测试 3: DELETE with empty condition
	fmt.Println("\nTest 3: DELETE with empty condition")
	if err := db.DeleteRow("users", ""); err != nil {
		log.Fatalf("Failed to delete row: %v", err)
	}
	count, err := db.Count("users", "")
	if err != nil {
		log.Fatalf("Failed to count rows: %v", err)
	}
	fmt.Printf("  Rows after delete: %d\n", count)

	fmt.Println("\n=== 测试完成 ===")
}
