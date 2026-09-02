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
	wedbDB, err := storage.NewWeDBDatabase("where_test.db", 4096)
	if err != nil {
		log.Fatalf("Failed to create database: %v", err)
	}
	defer wedbDB.Close()

	// 创建适配器
	db := adapter.NewWeDBAdapter(wedbDB)

	fmt.Println("=== WHERE 子句测试 ===")

	// 创建表
	schema := &api.TableSchema{
		TableName: "users",
		Columns: []api.ColumnSchema{
			{Name: "id", Type: api.TypeInteger},
			{Name: "name", Type: api.TypeText},
			{Name: "age", Type: api.TypeInteger},
			{Name: "score", Type: api.TypeReal},
		},
		PrimaryKey: "id",
	}

	if err := db.CreateTable(schema); err != nil {
		log.Fatalf("Failed to create table: %v", err)
	}

	// 插入测试数据
	rows := []map[string]interface{}{
		{"id": 1, "name": "Alice", "age": 25, "score": 85.5},
		{"id": 2, "name": "Bob", "age": 30, "score": 92.0},
		{"id": 3, "name": "Charlie", "age": 35, "score": 78.5},
		{"id": 4, "name": "David", "age": 28, "score": 88.0},
		{"id": 5, "name": "Eve", "age": 32, "score": 95.5},
	}

	if err := db.InsertRows("users", rows); err != nil {
		log.Fatalf("Failed to insert rows: %v", err)
	}

	// 测试 1: 全表扫描
	fmt.Println("Test 1: 全表扫描")
	allRows, err := db.ScanTable("users")
	if err != nil {
		log.Fatalf("Failed to scan table: %v", err)
	}
	fmt.Printf("  Total rows: %d\n", len(allRows))
	for _, row := range allRows {
		fmt.Printf("  - %v\n", row)
	}

	// 测试 2: WHERE age > 30
	fmt.Println("\nTest 2: WHERE age > 30")
	updateRow := map[string]interface{}{"score": 100.0}
	if err := db.UpdateRow("users", updateRow, "age > 30"); err != nil {
		log.Fatalf("Failed to update rows: %v", err)
	}
	updatedRows, err := db.ScanTable("users")
	if err != nil {
		log.Fatalf("Failed to scan table: %v", err)
	}
	for _, row := range updatedRows {
		if row["age"].(int64) > 30 {
			fmt.Printf("  - %v\n", row)
		}
	}

	// 测试 3: WHERE age = 25
	fmt.Println("\nTest 3: WHERE age = 25")
	rows25, err := db.ScanTableWithColumns("users", []string{"name", "age"})
	if err != nil {
		log.Fatalf("Failed to scan table: %v", err)
	}
	for _, row := range rows25 {
		if row["age"].(int64) == 25 {
			fmt.Printf("  - %v\n", row)
		}
	}

	// 测试 4: WHERE age < 30 AND score > 80
	fmt.Println("\nTest 4: WHERE age < 30 AND score > 80")
	updateRow2 := map[string]interface{}{"score": 90.0}
	if err := db.UpdateRow("users", updateRow2, "age < 30 AND score > 80"); err != nil {
		log.Fatalf("Failed to update rows: %v", err)
	}
	updatedRows2, err := db.ScanTable("users")
	if err != nil {
		log.Fatalf("Failed to scan table: %v", err)
	}
	for _, row := range updatedRows2 {
		if row["age"].(int64) < 30 && row["score"].(float64) > 80 {
			fmt.Printf("  - %v\n", row)
		}
	}

	// 测试 5: WHERE name = 'Alice'
	fmt.Println("\nTest 5: WHERE name = 'Alice'")
	updateRow3 := map[string]interface{}{"age": 26}
	if err := db.UpdateRow("users", updateRow3, "name = 'Alice'"); err != nil {
		log.Fatalf("Failed to update rows: %v", err)
	}
	updatedRows3, err := db.ScanTable("users")
	if err != nil {
		log.Fatalf("Failed to scan table: %v", err)
	}
	for _, row := range updatedRows3 {
		if row["name"] == "Alice" {
			fmt.Printf("  - %v\n", row)
		}
	}

	// 测试 6: WHERE age IN (25, 30, 35)
	fmt.Println("\nTest 6: WHERE age IN (25, 30, 35)")
	updateRow4 := map[string]interface{}{"score": 95.0}
	if err := db.UpdateRow("users", updateRow4, "age IN (25, 30, 35)"); err != nil {
		log.Fatalf("Failed to update rows: %v", err)
	}
	updatedRows4, err := db.ScanTable("users")
	if err != nil {
		log.Fatalf("Failed to scan table: %v", err)
	}
	for _, row := range updatedRows4 {
		age := row["age"].(int64)
		if age == 25 || age == 30 || age == 35 {
			fmt.Printf("  - %v\n", row)
		}
	}

	// 测试 7: WHERE name LIKE 'A%'
	fmt.Println("\nTest 7: WHERE name LIKE 'A%'")
	updateRow5 := map[string]interface{}{"score": 99.0}
	if err := db.UpdateRow("users", updateRow5, "name LIKE 'A%'"); err != nil {
		log.Fatalf("Failed to update rows: %v", err)
	}
	updatedRows5, err := db.ScanTable("users")
	if err != nil {
		log.Fatalf("Failed to scan table: %v", err)
	}
	for _, row := range updatedRows5 {
		if name, ok := row["name"].(string); ok && len(name) > 0 && name[0] == 'A' {
			fmt.Printf("  - %v\n", row)
		}
	}

	// 测试 8: WHERE age >= 30 OR score >= 90
	fmt.Println("\nTest 8: WHERE age >= 30 OR score >= 90")
	updateRow6 := map[string]interface{}{"score": 97.0}
	if err := db.UpdateRow("users", updateRow6, "age >= 30 OR score >= 90"); err != nil {
		log.Fatalf("Failed to update rows: %v", err)
	}
	updatedRows6, err := db.ScanTable("users")
	if err != nil {
		log.Fatalf("Failed to scan table: %v", err)
	}
	for _, row := range updatedRows6 {
		age := row["age"].(int64)
		score := row["score"].(float64)
		if age >= 30 || score >= 90 {
			fmt.Printf("  - %v\n", row)
		}
	}

	// 测试 9: DELETE WHERE age < 30
	fmt.Println("\nTest 9: DELETE WHERE age < 30")
	if err := db.DeleteRow("users", "age < 30"); err != nil {
		log.Fatalf("Failed to delete rows: %v", err)
	}
	finalRows, err := db.ScanTable("users")
	if err != nil {
		log.Fatalf("Failed to scan table: %v", err)
	}
	fmt.Printf("  Remaining rows: %d\n", len(finalRows))
	for _, row := range finalRows {
		fmt.Printf("  - %v\n", row)
	}

	// 测试 10: Count with WHERE
	fmt.Println("\nTest 10: Count with WHERE")
	count, err := db.Count("users", "age >= 30")
	if err != nil {
		log.Fatalf("Failed to count: %v", err)
	}
	fmt.Printf("  Count (age >= 30): %d\n", count)

	// 测试 11: Min/Max/Avg/Sum with WHERE
	fmt.Println("\nTest 11: Min/Max/Avg/Sum with WHERE")
	min, err := db.Min("users", "age", "age >= 30")
	if err != nil {
		log.Fatalf("Failed to get min: %v", err)
	}
	fmt.Printf("  Min age (age >= 30): %v\n", min)

	max, err := db.Max("users", "age", "age >= 30")
	if err != nil {
		log.Fatalf("Failed to get max: %v", err)
	}
	fmt.Printf("  Max age (age >= 30): %v\n", max)

	avg, err := db.Avg("users", "age", "age >= 30")
	if err != nil {
		log.Fatalf("Failed to get avg: %v", err)
	}
	fmt.Printf("  Avg age (age >= 30): %.2f\n", avg)

	sum, err := db.Sum("users", "age", "age >= 30")
	if err != nil {
		log.Fatalf("Failed to get sum: %v", err)
	}
	fmt.Printf("  Sum age (age >= 30): %.2f\n", sum)

	fmt.Println("\n=== WHERE 子句测试完成 ===")
}
