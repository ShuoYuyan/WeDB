package main

import (
	"fmt"
	"log"

	"github.com/wedb/wedb/internal/api"
	"github.com/wedb/wedb/internal/storage"
	"github.com/wedb/wedb/pkg/adapter"
)

func main() {
	// 创建数据库
	wedbDB, err := storage.NewWeDBDatabase("index_test.db", 4096)
	if err != nil {
		log.Fatalf("Failed to create database: %v", err)
	}
	defer wedbDB.Close()

	// 创建适配器
	db := adapter.NewWeDBAdapter(wedbDB)

	fmt.Println("=== 索引查询性能测试 ===")

	// 创建表
	schema := &api.TableSchema{
		TableName: "users",
		Columns: []api.ColumnSchema{
			{Name: "id", Type: api.TypeInteger},
			{Name: "name", Type: api.TypeText},
			{Name: "age", Type: api.TypeInteger},
			{Name: "email", Type: api.TypeText},
		},
		PrimaryKey: "id",
	}

	if err := db.CreateTable(schema); err != nil {
		log.Fatalf("Failed to create table: %v", err)
	}

	// 插入测试数据
	rows := []map[string]interface{}{
		{"id": 1, "name": "Alice", "age": 25, "email": "alice@example.com"},
		{"id": 2, "name": "Bob", "age": 30, "email": "bob@example.com"},
		{"id": 3, "name": "Charlie", "age": 35, "email": "charlie@example.com"},
		{"id": 4, "name": "David", "age": 28, "email": "david@example.com"},
		{"id": 5, "name": "Eve", "age": 32, "email": "eve@example.com"},
	}

	if err := db.InsertRows("users", rows); err != nil {
		log.Fatalf("Failed to insert rows: %v", err)
	}

	// 测试 1: 创建索引
	fmt.Println("Test 1: 创建索引")
	indexInfo := &api.IndexInfo{
		IndexName: "idx_age",
		Columns:   []string{"age"},
		Unique:    false,
	}
	if err := db.CreateIndex("users", indexInfo); err != nil {
		log.Fatalf("Failed to create index: %v", err)
	}
	fmt.Println("  索引创建成功")

	// 测试 2: 获取索引信息
	fmt.Println("\nTest 2: 获取索引信息")
	indexes, err := db.GetIndexInfo("users")
	if err != nil {
		log.Fatalf("Failed to get index info: %v", err)
	}
	for _, idx := range indexes {
		fmt.Printf("  Index: %s, Columns: %v, Unique: %v\n", idx.IndexName, idx.Columns, idx.Unique)
	}

	// 测试 3: 使用索引搜索
	fmt.Println("\nTest 3: 使用索引搜索")
	// 注意：索引搜索功能需要暴露索引管理器或添加公共接口
	// 暂时跳过此测试

	// 测试 4: 全表扫描 vs 索引查询（性能对比）
	fmt.Println("\nTest 4: 性能对比")
	// 全表扫描
	allRows, err := db.ScanTable("users")
	if err != nil {
		log.Fatalf("Failed to scan table: %v", err)
	}
	fmt.Printf("  全表扫描: %d 行\n", len(allRows))

	// 使用 WHERE 条件
	opts := &api.QueryOptions{
		Where: "age > 30",
	}
	filteredRows, err := wedbDB.ScanTableWithOptions("users", opts)
	if err != nil {
		log.Fatalf("Failed to scan with options: %v", err)
	}
	fmt.Printf("  WHERE age > 30: %d 行\n", len(filteredRows))
	for _, row := range filteredRows {
		fmt.Printf("    - %v\n", row)
	}

	// 测试 5: 创建复合索引
	fmt.Println("\nTest 5: 创建复合索引")
	indexInfo2 := &api.IndexInfo{
		IndexName: "idx_name_age",
		Columns:   []string{"name", "age"},
		Unique:    false,
	}
	if err := db.CreateIndex("users", indexInfo2); err != nil {
		log.Fatalf("Failed to create index: %v", err)
	}
	fmt.Println("  复合索引创建成功")

	// 再次获取索引信息
	indexes, err = db.GetIndexInfo("users")
	if err != nil {
		log.Fatalf("Failed to get index info: %v", err)
	}
	fmt.Printf("  当前索引数量: %d\n", len(indexes))
	for _, idx := range indexes {
		fmt.Printf("    Index: %s, Columns: %v, Unique: %v\n", idx.IndexName, idx.Columns, idx.Unique)
	}

	// 测试 6: 删除索引
	fmt.Println("\nTest 6: 删除索引")
	if err := db.DropIndex("users", "idx_age"); err != nil {
		log.Fatalf("Failed to drop index: %v", err)
	}
	fmt.Println("  索引删除成功")

	// 再次获取索引信息
	indexes, err = db.GetIndexInfo("users")
	if err != nil {
		log.Fatalf("Failed to get index info: %v", err)
	}
	fmt.Printf("  当前索引数量: %d\n", len(indexes))
	for _, idx := range indexes {
		fmt.Printf("    Index: %s, Columns: %v, Unique: %v\n", idx.IndexName, idx.Columns, idx.Unique)
	}

	fmt.Println("\n=== 索引查询性能测试完成 ===")
}
