package main

import (
	"fmt"
	"os"

	"github.com/wedb/wedb/internal/api"
	"github.com/wedb/wedb/pkg/adapter"
)

func main() {
	// 创建临时数据库文件
	dbFile := "quick_test.db"
	defer os.Remove(dbFile)

	// 创建数据库
	adapter := adapter.NewWeDBAdapter(nil)
	if err := adapter.OpenDatabase(dbFile); err != nil {
		fmt.Printf("Failed to open database: %v\n", err)
		os.Exit(1)
	}
	defer adapter.CloseDatabase()

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
	if err := adapter.CreateTable(schema); err != nil {
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
	if err := adapter.InsertRows("users", rows); err != nil {
		fmt.Printf("Failed to insert rows: %v\n", err)
		os.Exit(1)
	}

	// 查询数据
	fmt.Println("\nQuerying all rows:")
	allRows, err := adapter.ScanTable("users")
	if err != nil {
		fmt.Printf("Failed to scan table: %v\n", err)
		os.Exit(1)
	}

	for _, row := range allRows {
		fmt.Printf("  ID: %v, Name: %v, Age: %v\n", row["id"], row["name"], row["age"])
	}

	// 测试聚合函数
	fmt.Println("\nAggregation tests:")
	count, _ := adapter.Count("users", "")
	fmt.Printf("  Count: %d\n", count)

	min, _ := adapter.Min("users", "age", "")
	fmt.Printf("  Min Age: %v\n", min)

	max, _ := adapter.Max("users", "age", "")
	fmt.Printf("  Max Age: %v\n", max)

	sum, _ := adapter.Sum("users", "age", "")
	fmt.Printf("  Sum Age: %.1f\n", sum)

	avg, _ := adapter.Avg("users", "age", "")
	fmt.Printf("  Avg Age: %.1f\n", avg)

	// 创建索引
	fmt.Println("\nCreating index on 'age' column...")
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

	// 列出索引
	indexes, err := adapter.GetIndexInfo("users")
	if err != nil {
		fmt.Printf("Failed to get index info: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("  Indexes: %d\n", len(indexes))
	for _, idx := range indexes {
		fmt.Printf("    - %s on columns %v\n", idx.IndexName, idx.Columns)
	}

	// 测试事务
	fmt.Println("\nTesting transaction...")
	tx, err := adapter.Begin()
	if err != nil {
		fmt.Printf("Failed to begin transaction: %v\n", err)
		os.Exit(1)
	}

	// 插入新行
	newRow := map[string]interface{}{
		"id":   4,
		"name": "David",
		"age":  40,
	}

	if err := adapter.InsertRow("users", newRow); err != nil {
		fmt.Printf("Failed to insert row: %v\n", err)
		os.Exit(1)
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		fmt.Printf("Failed to commit transaction: %v", err)
		os.Exit(1)
	}

	// 验证事务后的数据
	countAfter, _ := adapter.Count("users", "")
	fmt.Printf("  Count after transaction: %d\n", countAfter)

	// 获取表统计
	fmt.Println("\nTable statistics:")
	stats, err := adapter.GetTableStats("users")
	if err != nil {
		fmt.Printf("Failed to get table stats: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("  Row Count: %d\n", stats.RowCount)
	fmt.Printf("  Index Count: %d\n", stats.IndexCount)
	fmt.Printf("  Column Count: %d\n", stats.ColumnCount)
	fmt.Printf("  Table Size: %d bytes\n", stats.TableSize)

	fmt.Println("\n✅ All tests passed successfully!")
}