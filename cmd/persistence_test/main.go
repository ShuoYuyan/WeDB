package main

import (
	"fmt"
	"log"
	"os"

	"github.com/wedb/wedb/internal/api"
	"github.com/wedb/wedb/internal/storage"
	"github.com/wedb/wedb/pkg/adapter"
)

func main() {
	dbFile := "persistence_test.db"

	// 删除旧的数据库文件（如果存在）
	os.Remove(dbFile)
	os.Remove(dbFile + "-journal")
	os.Remove(dbFile + ".metadata")

	fmt.Println("=== 数据持久化和恢复测试 ===")

	// 第一次会话：创建数据库并插入数据
	fmt.Println("Session 1: 创建数据库并插入数据")
	wedbDB1, err := storage.NewWeDBDatabase(dbFile, 4096)
	if err != nil {
		log.Fatalf("Failed to create database: %v", err)
	}
	db1 := adapter.NewWeDBAdapter(wedbDB1)

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
	if err := db1.CreateTable(schema); err != nil {
		log.Fatalf("Failed to create table: %v", err)
	}
	fmt.Println("  表创建成功")

	// 插入数据
	rows := []map[string]interface{}{
		{"id": 1, "name": "Alice", "age": 25},
		{"id": 2, "name": "Bob", "age": 30},
		{"id": 3, "name": "Charlie", "age": 35},
	}
	if err := db1.InsertRows("users", rows); err != nil {
		log.Fatalf("Failed to insert rows: %v", err)
	}
	fmt.Println("  数据插入成功")

	// 验证数据
	count, err := db1.Count("users", "")
	if err != nil {
		log.Fatalf("Failed to count rows: %v", err)
	}
	fmt.Printf("  当前行数: %d\n", count)

	// 关闭数据库
	if err := wedbDB1.Close(); err != nil {
		log.Fatalf("Failed to close database: %v", err)
	}
	fmt.Println("  数据库已关闭")

	// 第二次会话：重新打开数据库，验证数据是否持久化
	fmt.Println("Session 2: 重新打开数据库，验证数据持久化")
	wedbDB2, err := storage.NewWeDBDatabase(dbFile, 4096)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	db2 := adapter.NewWeDBAdapter(wedbDB2)

	// 验证表是否存在
	if !db2.TableExists("users") {
		log.Fatalf("Table should exist after persistence")
	}
	fmt.Println("  表存在")

	// 不要再创建表，因为表已经存在

	// 验证数据是否恢复
	rows2, err := db2.ScanTable("users")
	if err != nil {
		log.Fatalf("Failed to scan table: %v", err)
	}
	fmt.Printf("  恢复的行数: %d\n", len(rows2))

	// 验证数据内容
	for _, row := range rows2 {
		fmt.Printf("    - id: %v, name: %v, age: %v\n", row["id"], row["name"], row["age"])
	}

	if len(rows2) != 3 {
		log.Fatalf("Expected 3 rows after persistence, got %d", len(rows2))
	}
	fmt.Println("  数据持久化成功")

	// 第三次会话：修改数据并验证
	fmt.Println("Session 3: 修改数据并验证")
	// 修改一行数据
	updateRow := map[string]interface{}{"age": 26}
	if err := db2.UpdateRow("users", updateRow, "id = 1"); err != nil {
		log.Fatalf("Failed to update row: %v", err)
	}

	// 验证修改
	updatedRows, err := db2.ScanTable("users")
	if err != nil {
		log.Fatalf("Failed to scan table: %v", err)
	}

	aliceAge := int64(0)
	for _, row := range updatedRows {
		if row["name"] == "Alice" {
			aliceAge = row["age"].(int64)
		}
	}

	if aliceAge != 26 {
		log.Fatalf("Expected Alice's age to be 26, got %d", aliceAge)
	}
	fmt.Printf("  数据修改成功 (Alice's age: %d)\n", aliceAge)

	// 关闭数据库
	if err := wedbDB2.Close(); err != nil {
		log.Fatalf("Failed to close database: %v", err)
	}
	fmt.Println("  数据库已关闭")

	// 第四次会话：再次打开，验证修改是否持久化
	fmt.Println("Session 4: 再次打开，验证修改是否持久化")
	wedbDB3, err := storage.NewWeDBDatabase(dbFile, 4096)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	db3 := adapter.NewWeDBAdapter(wedbDB3)

	// 验证修改是否持久化
	rows3, err := db3.ScanTable("users")
	if err != nil {
		log.Fatalf("Failed to scan table: %v", err)
	}

	aliceAge = int64(0)
	for _, row := range rows3 {
		if row["name"] == "Alice" {
			aliceAge = row["age"].(int64)
		}
	}

	if aliceAge != 26 {
		log.Fatalf("Expected Alice's age to be 26 after persistence, got %d", aliceAge)
	}
	fmt.Printf("  修改持久化成功 (Alice's age: %d)\n", aliceAge)

	// 删除一行数据
	if err := db3.DeleteRow("users", "id = 2"); err != nil {
		log.Fatalf("Failed to delete row: %v", err)
	}

	// 验证删除
	count3, err := db3.Count("users", "")
	if err != nil {
		log.Fatalf("Failed to count rows: %v", err)
	}

	if count3 != 2 {
		log.Fatalf("Expected 2 rows after delete, got %d", count3)
	}
	fmt.Printf("  删除成功，剩余行数: %d\n", count3)

	// 关闭数据库
	if err := wedbDB3.Close(); err != nil {
		log.Fatalf("Failed to close database: %v", err)
	}
	fmt.Println("  数据库已关闭")

	// 第五次会话：验证删除是否持久化
	fmt.Println("Session 5: 验证删除是否持久化")
	wedbDB4, err := storage.NewWeDBDatabase(dbFile, 4096)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	db4 := adapter.NewWeDBAdapter(wedbDB4)

	// 验证删除是否持久化
	count4, err := db4.Count("users", "")
	if err != nil {
		log.Fatalf("Failed to count rows: %v", err)
	}

	if count4 != 2 {
		log.Fatalf("Expected 2 rows after persistence of delete, got %d", count4)
	}
	fmt.Printf("  删除持久化成功，剩余行数: %d\n", count4)

	// 验证剩余数据
	rows4, err := db4.ScanTable("users")
	if err != nil {
		log.Fatalf("Failed to scan table: %v", err)
	}
	fmt.Println("  剩余数据:")
	for _, row := range rows4 {
		fmt.Printf("    - id: %v, name: %v, age: %v\n", row["id"], row["name"], row["age"])
	}

	// 关闭数据库
	if err := wedbDB4.Close(); err != nil {
		log.Fatalf("Failed to close database: %v", err)
	}

	fmt.Println("\n=== 数据持久化和恢复测试完成 ===")
}
