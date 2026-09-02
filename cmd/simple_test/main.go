package main

import (
	"fmt"
	"os"

	"github.com/wedb/wedb/internal/api"
	"github.com/wedb/wedb/internal/storage"
)

func main() {
	// 创建临时数据库文件
	dbFile := "simple_test.db"
	defer os.Remove(dbFile)

	// 创建数据库
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
			{
				Name: "email",
				Type: api.TypeText,
			},
		},
		PrimaryKey: "id",
	}

	fmt.Println("Creating table 'users'...")
	if err := db.CreateTable(schema); err != nil {
		fmt.Printf("Failed to create table: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ Table created successfully!")

	// 验证表存在
	if !db.TableExists("users") {
		fmt.Println("❌ Table should exist")
		os.Exit(1)
	}

	fmt.Println("✅ Table exists!")

	// 验证表结构
	gotSchema, err := db.GetTableSchema("users")
	if err != nil {
		fmt.Printf("Failed to get table schema: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Table schema: %s with %d columns\n", gotSchema.TableName, len(gotSchema.Columns))

	fmt.Println("\n✅ All tests passed successfully!")
}