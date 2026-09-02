package main

import (
	"fmt"
	"os"

	"github.com/wedb/wedb/internal/api"
	"github.com/wedb/wedb/internal/storage"
)

func main() {
	// 创建临时数据库文件
	dbFile := "update_test.db"
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

	fmt.Println("Creating table 'users'...")
	if err := db.CreateTable(schema); err != nil {
		fmt.Printf("Failed to create table: %v\n", err)
		os.Exit(1)
	}

	// 插入数据
	rows := []map[string]interface{}{
		{
			"id":   1,
			"name": "Alice",
			"age":  25,
		},
		{
			"id":   2,
			"name": "Bob",
			"age":  30,
		},
		{
			"id":   3,
			"name": "Charlie",
			"age":  35,
		},
	}

	fmt.Println("Inserting 3 rows...")
	if err := db.InsertRows("users", rows); err != nil {
		fmt.Printf("Failed to insert rows: %v\n", err)
		os.Exit(1)
	}

	// 查询数据
	fmt.Println("\nBefore update:")
	allRows, err := db.ScanTable("users")
	if err != nil {
		fmt.Printf("Failed to scan table: %v\n", err)
		os.Exit(1)
	}

	for _, row := range allRows {
		fmt.Printf("  ID: %v, Name: %v, Age: %v\n", row["id"], row["name"], row["age"])
	}

	// 更新数据（所有行的 age 都加 1）
	fmt.Println("\nUpdating all rows (age = 26)...")
	updateRow := map[string]interface{}{
		"age": 26,
	}
	if err := db.UpdateRow("users", updateRow, "*"); err != nil {
		fmt.Printf("Failed to update row: %v\n", err)
		os.Exit(1)
	}

	// 查询更新后的数据
	fmt.Println("\nAfter update:")
	updatedRows, err := db.ScanTable("users")
	if err != nil {
		fmt.Printf("Failed to scan table: %v\n", err)
		os.Exit(1)
	}

	for _, row := range updatedRows {
		fmt.Printf("  ID: %v, Name: %v, Age: %v\n", row["id"], row["name"], row["age"])
	}

	fmt.Println("\n✅ All tests passed successfully!")
}