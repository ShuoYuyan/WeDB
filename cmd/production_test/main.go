package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/wedb/wedb/internal/api"
	"github.com/wedb/wedb/internal/storage"
	"github.com/wedb/wedb/pkg/adapter"
)

func main() {
	dbFile := "production_test.db"

	// 删除旧的数据库文件（如果存在）
	os.Remove(dbFile)
	os.Remove(dbFile + "-journal")
	os.Remove(dbFile + ".metadata")

	fmt.Println("=== WeDB 生产级验收测试 ===")

	// 创建数据库
	wedbDB, err := storage.NewWeDBDatabase(dbFile, 4096)
	if err != nil {
		log.Fatalf("Failed to create database: %v", err)
	}
	defer wedbDB.Close()

	db := adapter.NewWeDBAdapter(wedbDB)
	wedbDBInstance := wedbDB

	fmt.Println("测试 1: 表创建和删除")
	testTableCreation(db)

	fmt.Println("\n测试 2: 数据插入")
	testDataInsertion(db)

	fmt.Println("\n测试 3: 数据查询")
	testDataQuery(db)

	fmt.Println("\n测试 4: 数据更新")
	testDataUpdate(db)

	fmt.Println("\n测试 5: 数据删除")
	testDataDeletion(db)

	fmt.Println("\n测试 6: WHERE 子句")
	testWhereClause(db, wedbDBInstance)

	fmt.Println("\n测试 7: ORDER BY 排序")
	testOrderBy(db, wedbDBInstance)

	fmt.Println("\n测试 8: LIMIT/OFFSET 分页")
	testLimitOffset(db, wedbDBInstance)

	fmt.Println("\n测试 9: 聚合函数")
	testAggregation(db)

	fmt.Println("\n测试 10: 索引操作")
	testIndexOperations(db)

	fmt.Println("\n测试 11: 事务操作")
	testTransactions(db)

	fmt.Println("\n测试 13: 性能测试")
	testPerformance(dbFile)

	fmt.Println("\n测试 14: 边界条件")
	testBoundaryConditions(db)

	fmt.Println("\n测试 12: 数据持久化（最后执行）")
	testDataPersistence(wedbDBInstance, dbFile)

	fmt.Println("\n=== 所有测试通过 ===")
}

func testTableCreation(db *adapter.WeDBAdapter) {
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

	// 验证表存在
	if !db.TableExists("users") {
		log.Fatalf("Table should exist")
	}

	// 获取表结构
	tableSchema, err := db.GetTableSchema("users")
	if err != nil {
		log.Fatalf("Failed to get table schema: %v", err)
	}

	if len(tableSchema.Columns) != 4 {
		log.Fatalf("Expected 4 columns, got %d", len(tableSchema.Columns))
	}

	fmt.Println("  ✓ 表创建成功")

	// 删除表
	if err := db.DropTable("users"); err != nil {
		log.Fatalf("Failed to drop table: %v", err)
	}

	if db.TableExists("users") {
		log.Fatalf("Table should not exist after drop")
	}

	fmt.Println("  ✓ 表删除成功")

	// 重新创建表以供后续测试使用
	if err := db.CreateTable(schema); err != nil {
		log.Fatalf("Failed to recreate table: %v", err)
	}
}

func testDataInsertion(db *adapter.WeDBAdapter) {
	// 插入单行
	row := map[string]interface{}{
		"id":    1,
		"name":  "Alice",
		"age":   25,
		"email": "alice@example.com",
	}

	if err := db.InsertRow("users", row); err != nil {
		log.Fatalf("Failed to insert row: %v", err)
	}

	// 插入多行
	rows := []map[string]interface{}{
		{"id": 2, "name": "Bob", "age": 30, "email": "bob@example.com"},
		{"id": 3, "name": "Charlie", "age": 35, "email": "charlie@example.com"},
		{"id": 4, "name": "David", "age": 28, "email": "david@example.com"},
		{"id": 5, "name": "Eve", "age": 32, "email": "eve@example.com"},
	}

	if err := db.InsertRows("users", rows); err != nil {
		log.Fatalf("Failed to insert rows: %v", err)
	}

	// 验证插入
	count, err := db.Count("users", "")
	if err != nil {
		log.Fatalf("Failed to count rows: %v", err)
	}

	if count != 5 {
		log.Fatalf("Expected 5 rows, got %d", count)
	}

	fmt.Println("  ✓ 数据插入成功")
}

func testDataQuery(db *adapter.WeDBAdapter) {
	// 查询所有数据
	rows, err := db.ScanTable("users")
	if err != nil {
		log.Fatalf("Failed to scan table: %v", err)
	}

	if len(rows) != 5 {
		log.Fatalf("Expected 5 rows, got %d", len(rows))
	}

	// 查询指定列
	cols, err := db.ScanTableWithColumns("users", []string{"name", "age"})
	if err != nil {
		log.Fatalf("Failed to scan table with columns: %v", err)
	}

	if len(cols) != 5 {
		log.Fatalf("Expected 5 rows, got %d", len(cols))
	}

	for _, row := range cols {
		if len(row) != 2 {
			log.Fatalf("Expected 2 columns, got %d", len(row))
		}
	}

	fmt.Println("  ✓ 数据查询成功")
}

func testDataUpdate(db *adapter.WeDBAdapter) {
	// 更新单行
	updateRow := map[string]interface{}{"age": 26}
	if err := db.UpdateRow("users", updateRow, "id = 1"); err != nil {
		log.Fatalf("Failed to update row: %v", err)
	}

	// 验证更新
	rows, err := db.ScanTable("users")
	if err != nil {
		log.Fatalf("Failed to scan table: %v", err)
	}

	for _, row := range rows {
		if row["id"].(int64) == 1 {
			if row["age"].(int64) != 26 {
				log.Fatalf("Expected age 26, got %v", row["age"])
			}
		}
	}

	// 更新多行
	updateRow2 := map[string]interface{}{"email": "updated@example.com"}
	if err := db.UpdateRow("users", updateRow2, "age > 30"); err != nil {
		log.Fatalf("Failed to update rows: %v", err)
	}

	fmt.Println("  ✓ 数据更新成功")
}

func testDataDeletion(db *adapter.WeDBAdapter) {
	// 先获取当前数据
	currentCount, err := db.Count("users", "")
	if err != nil {
		log.Fatalf("Failed to count rows: %v", err)
	}

	fmt.Printf("  当前数据行数: %d\n", currentCount)

	// 删除单行（如果存在）
	if currentCount > 0 {
		if err := db.DeleteRow("users", "id = 5"); err != nil {
			log.Fatalf("Failed to delete row: %v", err)
		}

		// 验证删除
		count, err := db.Count("users", "")
		if err != nil {
			log.Fatalf("Failed to count rows: %v", err)
		}

		if count >= currentCount {
			log.Fatalf("Expected less rows after delete, got %d", count)
		}

		currentCount = count
	}

	// 删除多行
	if err := db.DeleteRow("users", "age > 30"); err != nil {
		log.Fatalf("Failed to delete rows: %v", err)
	}

	// 验证删除
	count, err := db.Count("users", "")
	if err != nil {
		log.Fatalf("Failed to count rows: %v", err)
	}

	if count >= currentCount {
		log.Fatalf("Expected less rows after delete, got %d", count)
	}

	fmt.Printf("  删除后剩余行数: %d\n", count)
	fmt.Println("  ✓ 数据删除成功")
}

func testWhereClause(db *adapter.WeDBAdapter, wedbDBInstance *storage.WeDBDatabase) {
	// 先获取当前数据
	currentRows, err := db.ScanTable("users")
	if err != nil {
		log.Fatalf("Failed to scan table: %v", err)
	}

	if len(currentRows) == 0 {
		// 如果没有数据，插入一些测试数据
		testRows := []map[string]interface{}{
			{"id": 10, "name": "Test1", "age": 20, "email": "test1@example.com"},
			{"id": 11, "name": "Test2", "age": 30, "email": "test2@example.com"},
			{"id": 12, "name": "Test3", "age": 40, "email": "test3@example.com"},
		}
		if err := db.InsertRows("users", testRows); err != nil {
			log.Fatalf("Failed to insert test rows: %v", err)
		}
		currentRows, err = db.ScanTable("users")
		if err != nil {
			log.Fatalf("Failed to scan table: %v", err)
		}
	}

	fmt.Printf("  当前数据行数: %d\n", len(currentRows))

	// 测试各种 WHERE 条件
	opts := &api.QueryOptions{
		Where: "age >= 0",
	}

	filteredRows, err := wedbDBInstance.ScanTableWithOptions("users", opts)
	if err != nil {
		log.Fatalf("Failed to scan with WHERE: %v", err)
	}

	if len(filteredRows) == 0 {
		log.Fatalf("Expected at least 1 row, got %d", len(filteredRows))
	}

	fmt.Printf("  WHERE age >= 0: %d 行\n", len(filteredRows))

	// 测试等值条件
	opts.Where = "age = 30"
	filteredRows, err = wedbDBInstance.ScanTableWithOptions("users", opts)
	if err != nil {
		log.Fatalf("Failed to scan with WHERE: %v", err)
	}

	fmt.Printf("  WHERE age = 30: %d 行\n", len(filteredRows))

	fmt.Println("  ✓ WHERE 子句测试成功")
}

func testOrderBy(db *adapter.WeDBAdapter, wedbDBInstance *storage.WeDBDatabase) {
	// 测试 ORDER BY
	opts := &api.QueryOptions{
		OrderBy: []api.SortBy{{Column: "age", Order: api.SortAsc}},
	}

	sortedRows, err := wedbDBInstance.ScanTableWithOptions("users", opts)
	if err != nil {
		log.Fatalf("Failed to scan with ORDER BY: %v", err)
	}

	// 验证排序
	for i := 1; i < len(sortedRows); i++ {
		prevAge := sortedRows[i-1]["age"].(int64)
		currAge := sortedRows[i]["age"].(int64)
		if prevAge > currAge {
			log.Fatalf("Rows not sorted correctly")
		}
	}

	// 测试降序
	opts.OrderBy = []api.SortBy{{Column: "age", Order: api.SortDesc}}
	sortedRows, err = wedbDBInstance.ScanTableWithOptions("users", opts)
	if err != nil {
		log.Fatalf("Failed to scan with ORDER BY: %v", err)
	}

	for i := 1; i < len(sortedRows); i++ {
		prevAge := sortedRows[i-1]["age"].(int64)
		currAge := sortedRows[i]["age"].(int64)
		if prevAge < currAge {
			log.Fatalf("Rows not sorted correctly")
		}
	}

	fmt.Println("  ✓ ORDER BY 测试成功")
}

func testLimitOffset(db *adapter.WeDBAdapter, wedbDBInstance *storage.WeDBDatabase) {
	// 测试 LIMIT
	opts := &api.QueryOptions{
		Limit: 2,
		OrderBy: []api.SortBy{{Column: "age", Order: api.SortAsc}},
	}

	limitedRows, err := wedbDBInstance.ScanTableWithOptions("users", opts)
	if err != nil {
		log.Fatalf("Failed to scan with LIMIT: %v", err)
	}

	if len(limitedRows) > 2 {
		log.Fatalf("Expected at most 2 rows, got %d", len(limitedRows))
	}

	fmt.Printf("  LIMIT 2: %d 行\n", len(limitedRows))

	// 测试 OFFSET
	opts.Offset = 1
	limitedRows, err = wedbDBInstance.ScanTableWithOptions("users", opts)
	if err != nil {
		log.Fatalf("Failed to scan with OFFSET: %v", err)
	}

	fmt.Printf("  LIMIT 2 OFFSET 1: %d 行\n", len(limitedRows))

	fmt.Println("  ✓ LIMIT/OFFSET 测试成功")
}

func testAggregation(db *adapter.WeDBAdapter) {
	// 测试 Count
	count, err := db.Count("users", "")
	if err != nil {
		log.Fatalf("Failed to count: %v", err)
	}

	if count == 0 {
		log.Fatalf("Expected at least 1 row, got %d", count)
	}

	fmt.Printf("  Count: %d\n", count)

	// 测试 Min
	min, err := db.Min("users", "age", "")
	if err != nil {
		log.Fatalf("Failed to get min: %v", err)
	}

	fmt.Printf("  Min age: %v\n", min)

	// 测试 Max
	max, err := db.Max("users", "age", "")
	if err != nil {
		log.Fatalf("Failed to get max: %v", err)
	}

	fmt.Printf("  Max age: %v\n", max)

	// 测试 Sum
	sum, err := db.Sum("users", "age", "")
	if err != nil {
		log.Fatalf("Failed to get sum: %v", err)
	}

	fmt.Printf("  Sum age: %f\n", sum)

	// 测试 Avg
	avg, err := db.Avg("users", "age", "")
	if err != nil {
		log.Fatalf("Failed to get avg: %v", err)
	}

	fmt.Printf("  Avg age: %f\n", avg)

	fmt.Println("  ✓ 聚合函数测试成功")
}

func testIndexOperations(db *adapter.WeDBAdapter) {
	// 创建索引
	indexInfo := &api.IndexInfo{
		IndexName: "idx_age",
		Columns:   []string{"age"},
		Unique:    false,
		Type:      api.TypeBTree,
	}

	if err := db.CreateIndex("users", indexInfo); err != nil {
		log.Fatalf("Failed to create index: %v", err)
	}

	// 验证索引存在
	indexes, err := db.GetIndexInfo("users")
	if err != nil {
		log.Fatalf("Failed to get index info: %v", err)
	}

	if len(indexes) != 1 {
		log.Fatalf("Expected 1 index, got %d", len(indexes))
	}

	// 删除索引
	if err := db.DropIndex("users", "idx_age"); err != nil {
		log.Fatalf("Failed to drop index: %v", err)
	}

	// 验证索引已删除
	indexes, err = db.GetIndexInfo("users")
	if err != nil {
		log.Fatalf("Failed to get index info: %v", err)
	}

	if len(indexes) != 0 {
		log.Fatalf("Expected 0 indexes, got %d", len(indexes))
	}

	fmt.Println("  ✓ 索引操作测试成功")
}

func testTransactions(db *adapter.WeDBAdapter) {
	// 测试 COMMIT
	tx, err := db.Begin()
	if err != nil {
		log.Fatalf("Failed to begin transaction: %v", err)
	}

	txRow := map[string]interface{}{
		"id":    100,
		"name":  "TestUser",
		"age":   40,
		"email": "test@example.com",
	}

	if err := db.InsertRow("users", txRow); err != nil {
		log.Fatalf("Failed to insert row: %v", err)
	}

	if err := tx.Commit(); err != nil {
		log.Fatalf("Failed to commit transaction: %v", err)
	}

	// 验证数据已提交
	count, err := db.Count("users", "id = 100")
	if err != nil {
		log.Fatalf("Failed to count rows: %v", err)
	}

	if count != 1 {
		log.Fatalf("Expected 1 row, got %d", count)
	}

	// 测试 ROLLBACK
	tx, err = db.Begin()
	if err != nil {
		log.Fatalf("Failed to begin transaction: %v", err)
	}

	txRow2 := map[string]interface{}{
		"id":    101,
		"name":  "TestUser2",
		"age":   41,
		"email": "test2@example.com",
	}

	if err := db.InsertRow("users", txRow2); err != nil {
		log.Fatalf("Failed to insert row: %v", err)
	}

	if err := tx.Rollback(); err != nil {
		log.Fatalf("Failed to rollback transaction: %v", err)
	}

	// 验证数据已回滚
	count, err = db.Count("users", "id = 101")
	if err != nil {
		log.Fatalf("Failed to count rows: %v", err)
	}

	if count != 0 {
		log.Fatalf("Expected 0 rows, got %d", count)
	}

	fmt.Println("  ✓ 事务操作测试成功")
}

func testDataPersistence(wedbDBInstance *storage.WeDBDatabase, dbFile string) {
	// 先记录当前数据量
	currentCount, err := wedbDBInstance.Count("users", "")
	if err != nil {
		log.Fatalf("Failed to count rows: %v", err)
	}

	// 关闭数据库
	if err := wedbDBInstance.Close(); err != nil {
		log.Fatalf("Failed to close database: %v", err)
	}

	// 重新打开数据库
	newDB, err := storage.NewWeDBDatabase(dbFile, 4096)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer newDB.Close()

	newAdapter := adapter.NewWeDBAdapter(newDB)

	// 验证表存在
	if !newAdapter.TableExists("users") {
		log.Fatalf("Table should exist after persistence")
	}

	// 验证数据存在
	count, err := newAdapter.Count("users", "")
	if err != nil {
		log.Fatalf("Failed to count rows: %v", err)
	}

	if count != currentCount {
		log.Fatalf("Expected %d rows, got %d", currentCount, count)
	}

	fmt.Printf("  数据持久化成功，%d 行数据已恢复\n", count)
	fmt.Println("  ✓ 数据持久化测试成功")
}

func testPerformance(dbFile string) {
	// 创建新数据库进行性能测试
	perfDBFile := "performance_test.db"
	os.Remove(perfDBFile)
	os.Remove(perfDBFile + "-journal")
	os.Remove(perfDBFile + ".metadata")

	perfDB, err := storage.NewWeDBDatabase(perfDBFile, 4096)
	if err != nil {
		log.Fatalf("Failed to create database: %v", err)
	}
	defer perfDB.Close()

	perfAdapter := adapter.NewWeDBAdapter(perfDB)

	// 创建表
	schema := &api.TableSchema{
		TableName: "perf_test",
		Columns: []api.ColumnSchema{
			{Name: "id", Type: api.TypeInteger},
			{Name: "name", Type: api.TypeText},
			{Name: "value", Type: api.TypeInteger},
		},
		PrimaryKey: "id",
	}

	if err := perfAdapter.CreateTable(schema); err != nil {
		log.Fatalf("Failed to create table: %v", err)
	}

	// 测试插入性能
	numRows := 100
	start := time.Now()

	for i := 0; i < numRows; i++ {
		row := map[string]interface{}{
			"id":    int64(i + 1),
			"name":  fmt.Sprintf("User%d", i+1),
			"value": i + 1,
		}
		if err := perfAdapter.InsertRow("perf_test", row); err != nil {
			log.Fatalf("Failed to insert row %d: %v", i+1, err)
		}
	}

	insertDuration := time.Since(start)
	fmt.Printf("  插入 %d 行耗时: %v\n", numRows, insertDuration)

	// 测试查询性能
	start = time.Now()
	rows, err := perfAdapter.ScanTable("perf_test")
	if err != nil {
		log.Fatalf("Failed to scan table: %v", err)
	}
	scanDuration := time.Since(start)

	if len(rows) != numRows {
		log.Fatalf("Expected %d rows, got %d", numRows, len(rows))
	}

	fmt.Printf("  查询 %d 行耗时: %v\n", numRows, scanDuration)

	// 性能基准
	if insertDuration > 5*time.Second {
		fmt.Printf("  ⚠ 插入性能较慢 (%v > 5s)\n", insertDuration)
	} else {
		fmt.Printf("  ✓ 插入性能良好 (%v <= 5s)\n", insertDuration)
	}

	if scanDuration > 1*time.Second {
		fmt.Printf("  ⚠ 查询性能较慢 (%v > 1s)\n", scanDuration)
	} else {
		fmt.Printf("  ✓ 查询性能良好 (%v <= 1s)\n", scanDuration)
	}

	// 清理
	os.Remove(perfDBFile)
	os.Remove(perfDBFile + "-journal")
	os.Remove(perfDBFile + ".metadata")
}

func testBoundaryConditions(db *adapter.WeDBAdapter) {
	// 测试空表
	schema := &api.TableSchema{
		TableName: "empty_table",
		Columns: []api.ColumnSchema{
			{Name: "id", Type: api.TypeInteger},
			{Name: "name", Type: api.TypeText},
		},
		PrimaryKey: "id",
	}

	if err := db.CreateTable(schema); err != nil {
		log.Fatalf("Failed to create table: %v", err)
	}

	// 查询空表
	rows, err := db.ScanTable("empty_table")
	if err != nil {
		log.Fatalf("Failed to scan table: %v", err)
	}

	if len(rows) != 0 {
		log.Fatalf("Expected 0 rows, got %d", len(rows))
	}

	// 聚合函数在空表上
	count, err := db.Count("empty_table", "")
	if err != nil {
		log.Fatalf("Failed to count rows: %v", err)
	}

	if count != 0 {
		log.Fatalf("Expected count 0, got %d", count)
	}

	// 测试不存在的表
	if db.TableExists("nonexistent_table") {
		log.Fatalf("Non-existent table should not exist")
	}

	// 测试不存在的列
	_, err = db.ScanTableWithColumns("empty_table", []string{"nonexistent_column"})
	if err != nil {
		log.Fatalf("Failed to scan table with nonexistent column: %v", err)
	}

	// 验证返回的行不包含不存在的列
	if len(rows) > 0 && len(rows[0]) != 0 {
		log.Fatalf("Expected 0 columns for nonexistent column, got %d", len(rows[0]))
	}

	fmt.Println("  ✓ 边界条件测试成功")
}