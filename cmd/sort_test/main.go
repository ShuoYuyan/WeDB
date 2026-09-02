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
	wedbDB, err := storage.NewWeDBDatabase("sort_test.db", 4096)
	if err != nil {
		log.Fatalf("Failed to create database: %v", err)
	}
	defer wedbDB.Close()

	// 创建适配器
	db := adapter.NewWeDBAdapter(wedbDB)

	fmt.Println("=== 排序和分页测试 ===")

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
		{"id": 6, "name": "Frank", "age": 22, "score": 82.0},
		{"id": 7, "name": "Grace", "age": 29, "score": 90.0},
		{"id": 8, "name": "Henry", "age": 31, "score": 87.5},
		{"id": 9, "name": "Ivy", "age": 27, "score": 93.0},
		{"id": 10, "name": "Jack", "age": 33, "score": 89.5},
	}

	if err := db.InsertRows("users", rows); err != nil {
		log.Fatalf("Failed to insert rows: %v", err)
	}

	// 测试 1: 按 age 升序排序
	fmt.Println("Test 1: ORDER BY age ASC")
	opts1 := &api.QueryOptions{
		OrderBy: []api.SortBy{{Column: "age", Order: api.SortAsc}},
	}
	result1, err := wedbDB.ScanTableWithOptions("users", opts1)
	if err != nil {
		log.Fatalf("Failed to scan table: %v", err)
	}
	for _, row := range result1 {
		fmt.Printf("  - name: %s, age: %d, score: %.1f\n", row["name"], row["age"], row["score"])
	}

	// 测试 2: 按 age 降序排序
	fmt.Println("\nTest 2: ORDER BY age DESC")
	opts2 := &api.QueryOptions{
		OrderBy: []api.SortBy{{Column: "age", Order: api.SortDesc}},
	}
	result2, err := wedbDB.ScanTableWithOptions("users", opts2)
	if err != nil {
		log.Fatalf("Failed to scan table: %v", err)
	}
	for _, row := range result2 {
		fmt.Printf("  - name: %s, age: %d, score: %.1f\n", row["name"], row["age"], row["score"])
	}

	// 测试 3: 按 score 降序排序
	fmt.Println("\nTest 3: ORDER BY score DESC")
	opts3 := &api.QueryOptions{
		OrderBy: []api.SortBy{{Column: "score", Order: api.SortDesc}},
	}
	result3, err := wedbDB.ScanTableWithOptions("users", opts3)
	if err != nil {
		log.Fatalf("Failed to scan table: %v", err)
	}
	for _, row := range result3 {
		fmt.Printf("  - name: %s, age: %d, score: %.1f\n", row["name"], row["age"], row["score"])
	}

	// 测试 4: 按 name 升序排序
	fmt.Println("\nTest 4: ORDER BY name ASC")
	opts4 := &api.QueryOptions{
		OrderBy: []api.SortBy{{Column: "name", Order: api.SortAsc}},
	}
	result4, err := wedbDB.ScanTableWithOptions("users", opts4)
	if err != nil {
		log.Fatalf("Failed to scan table: %v", err)
	}
	for _, row := range result4 {
		fmt.Printf("  - name: %s, age: %d, score: %.1f\n", row["name"], row["age"], row["score"])
	}

	// 测试 5: 按 age 升序，然后按 score 降序排序
	fmt.Println("\nTest 5: ORDER BY age ASC, score DESC")
	opts5 := &api.QueryOptions{
		OrderBy: []api.SortBy{
			{Column: "age", Order: api.SortAsc},
			{Column: "score", Order: api.SortDesc},
		},
	}
	result5, err := wedbDB.ScanTableWithOptions("users", opts5)
	if err != nil {
		log.Fatalf("Failed to scan table: %v", err)
	}
	for _, row := range result5 {
		fmt.Printf("  - name: %s, age: %d, score: %.1f\n", row["name"], row["age"], row["score"])
	}

	// 测试 6: WHERE + ORDER BY
	fmt.Println("\nTest 6: WHERE age > 25 ORDER BY score DESC")
	opts6 := &api.QueryOptions{
		Where: "age > 25",
		OrderBy: []api.SortBy{{Column: "score", Order: api.SortDesc}},
	}
	result6, err := wedbDB.ScanTableWithOptions("users", opts6)
	if err != nil {
		log.Fatalf("Failed to scan table: %v", err)
	}
	for _, row := range result6 {
		fmt.Printf("  - name: %s, age: %d, score: %.1f\n", row["name"], row["age"], row["score"])
	}

	// 测试 7: LIMIT
	fmt.Println("\nTest 7: LIMIT 5")
	opts7 := &api.QueryOptions{
		Limit: 5,
		OrderBy: []api.SortBy{{Column: "age", Order: api.SortAsc}},
	}
	result7, err := wedbDB.ScanTableWithOptions("users", opts7)
	if err != nil {
		log.Fatalf("Failed to scan table: %v", err)
	}
	for _, row := range result7 {
		fmt.Printf("  - name: %s, age: %d, score: %.1f\n", row["name"], row["age"], row["score"])
	}

	// 测试 8: OFFSET
	fmt.Println("\nTest 8: OFFSET 5 LIMIT 3")
	opts8 := &api.QueryOptions{
		Offset: 5,
		Limit:  3,
		OrderBy: []api.SortBy{{Column: "age", Order: api.SortAsc}},
	}
	result8, err := wedbDB.ScanTableWithOptions("users", opts8)
	if err != nil {
		log.Fatalf("Failed to scan table: %v", err)
	}
	for _, row := range result8 {
		fmt.Printf("  - name: %s, age: %d, score: %.1f\n", row["name"], row["age"], row["score"])
	}

	// 测试 9: WHERE + ORDER BY + LIMIT + OFFSET
	fmt.Println("\nTest 9: WHERE age > 25 ORDER BY score DESC LIMIT 3 OFFSET 2")
	opts9 := &api.QueryOptions{
		Where: "age > 25",
		OrderBy: []api.SortBy{{Column: "score", Order: api.SortDesc}},
		Limit:   3,
		Offset:  2,
	}
	result9, err := wedbDB.ScanTableWithOptions("users", opts9)
	if err != nil {
		log.Fatalf("Failed to scan table: %v", err)
	}
	for _, row := range result9 {
		fmt.Printf("  - name: %s, age: %d, score: %.1f\n", row["name"], row["age"], row["score"])
	}

	// 测试 10: 指定列查询
	fmt.Println("\nTest 10: SELECT name, score ORDER BY score DESC LIMIT 5")
	opts10 := &api.QueryOptions{
		Columns: []string{"name", "score"},
		OrderBy: []api.SortBy{{Column: "score", Order: api.SortDesc}},
		Limit:   5,
	}
	result10, err := wedbDB.ScanTableWithOptions("users", opts10)
	if err != nil {
		log.Fatalf("Failed to scan table: %v", err)
	}
	for _, row := range result10 {
		fmt.Printf("  - name: %s, score: %.1f\n", row["name"], row["score"])
	}

	fmt.Println("\n=== 排序和分页测试完成 ===")
}
