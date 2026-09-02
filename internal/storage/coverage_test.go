package storage

import (
	"os"
	"testing"

	"github.com/wedb/wedb/internal/api"
)

func TestScanTableWithColumns(t *testing.T) {
	dbFile := "test_scan_columns.db"
	defer os.Remove(dbFile)

	db, err := NewWeDBDatabase(dbFile, 4096)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

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
		t.Fatalf("Failed to create table: %v", err)
	}

	row := map[string]interface{}{
		"id":   int64(1),
		"name": "Alice",
		"age":  int64(25),
	}

	if err := db.InsertRow("users", row); err != nil {
		t.Fatalf("Failed to insert row: %v", err)
	}

	// 测试 ScanTableWithColumns
	rows, err := db.ScanTableWithColumns("users", []string{"id", "name"})
	if err != nil {
		t.Fatalf("Failed to scan with columns: %v", err)
	}

	if len(rows) != 1 {
		t.Errorf("Expected 1 row, got %d", len(rows))
	}

	if rows[0]["id"] == nil || rows[0]["name"] == nil {
		t.Error("Expected id and name columns")
	}

	if rows[0]["age"] != nil {
		t.Error("Should not have age column")
	}
}

func TestScanTableWithOptions(t *testing.T) {
	dbFile := "test_scan_options.db"
	defer os.Remove(dbFile)

	db, err := NewWeDBDatabase(dbFile, 4096)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

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
		t.Fatalf("Failed to create table: %v", err)
	}

	rows := []map[string]interface{}{
		{"id": int64(1), "name": "Alice", "age": int64(25)},
		{"id": int64(2), "name": "Bob", "age": int64(30)},
		{"id": int64(3), "name": "Charlie", "age": int64(35)},
	}

	for _, row := range rows {
		if err := db.InsertRow("users", row); err != nil {
			t.Fatalf("Failed to insert row: %v", err)
		}
	}

	// 测试 WHERE 条件
	whereRows, err := db.ScanTableWithOptions("users", &api.QueryOptions{
		Where: "age > 30",
	})
	if err != nil {
		t.Fatalf("Failed to scan with WHERE: %v", err)
	}

	if len(whereRows) != 1 {
		t.Errorf("Expected 1 row with age > 30, got %d", len(whereRows))
	}

	// 测试 ORDER BY
	orderRows, err := db.ScanTableWithOptions("users", &api.QueryOptions{
		OrderBy: []api.SortBy{{Column: "age", Order: api.SortDesc}},
	})
	if err != nil {
		t.Fatalf("Failed to scan with ORDER BY: %v", err)
	}

	if len(orderRows) != 3 {
		t.Errorf("Expected 3 rows with ORDER BY, got %d", len(orderRows))
	}

	// 测试 LIMIT/OFFSET
	limitRows, err := db.ScanTableWithOptions("users", &api.QueryOptions{
		Limit:  2,
		Offset: 1,
	})
	if err != nil {
		t.Fatalf("Failed to scan with LIMIT/OFFSET: %v", err)
	}

	if len(limitRows) != 2 {
		t.Errorf("Expected 2 rows with LIMIT 2 OFFSET 1, got %d", len(limitRows))
	}
}

func TestUpdateRows(t *testing.T) {
	dbFile := "test_update_rows.db"
	defer os.Remove(dbFile)

	db, err := NewWeDBDatabase(dbFile, 4096)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

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
		t.Fatalf("Failed to create table: %v", err)
	}

	rows := []map[string]interface{}{
		{"id": int64(1), "name": "Alice", "age": int64(25)},
		{"id": int64(2), "name": "Bob", "age": int64(30)},
	}

	for _, row := range rows {
		if err := db.InsertRow("users", row); err != nil {
			t.Fatalf("Failed to insert row: %v", err)
		}
	}

	// 测试批量更新
	updateRows := []map[string]interface{}{
		{"age": int64(26)},
		{"age": int64(31)},
	}

	if err := db.UpdateRows("users", updateRows, "*"); err != nil {
		t.Fatalf("Failed to update rows: %v", err)
	}

	// 验证更新
	scanRows, err := db.ScanTable("users")
	if err != nil {
		t.Fatalf("Failed to scan table: %v", err)
	}

	if len(scanRows) != 2 {
		t.Errorf("Expected 2 rows, got %d", len(scanRows))
	}
}

func TestDeleteRows(t *testing.T) {
	dbFile := "test_delete_rows.db"
	defer os.Remove(dbFile)

	db, err := NewWeDBDatabase(dbFile, 4096)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

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
		t.Fatalf("Failed to create table: %v", err)
	}

	rows := []map[string]interface{}{
		{"id": int64(1), "name": "Alice", "age": int64(25)},
		{"id": int64(2), "name": "Bob", "age": int64(30)},
		{"id": int64(3), "name": "Charlie", "age": int64(35)},
	}

	for _, row := range rows {
		if err := db.InsertRow("users", row); err != nil {
			t.Fatalf("Failed to insert row: %v", err)
		}
	}

	// 测试批量删除
	conditions := []string{"id = 1", "id = 2"}

	if err := db.DeleteRows("users", conditions); err != nil {
		t.Fatalf("Failed to delete rows: %v", err)
	}

	// 验证删除
	count, err := db.Count("users", "")
	if err != nil {
		t.Fatalf("Failed to count rows: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 row after delete, got %d", count)
	}
}

func TestGetTableStats(t *testing.T) {
	dbFile := "test_table_stats.db"
	defer os.Remove(dbFile)

	db, err := NewWeDBDatabase(dbFile, 4096)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	schema := &api.TableSchema{
		TableName: "users",
		Columns: []api.ColumnSchema{
			{Name: "id", Type: api.TypeInteger},
			{Name: "name", Type: api.TypeText},
		},
		PrimaryKey: "id",
	}

	if err := db.CreateTable(schema); err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	stats, err := db.GetTableStats("users")
	if err != nil {
		t.Fatalf("Failed to get table stats: %v", err)
	}

	if stats == nil {
		t.Fatal("Table stats should not be nil")
	}

	if stats.RowCount != 0 {
		t.Errorf("Expected row count 0, got %d", stats.RowCount)
	}
}

func TestGetColumnStats(t *testing.T) {
	dbFile := "test_column_stats.db"
	defer os.Remove(dbFile)

	db, err := NewWeDBDatabase(dbFile, 4096)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	schema := &api.TableSchema{
		TableName: "users",
		Columns: []api.ColumnSchema{
			{Name: "id", Type: api.TypeInteger},
			{Name: "age", Type: api.TypeInteger},
		},
		PrimaryKey: "id",
	}

	if err := db.CreateTable(schema); err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	rows := []map[string]interface{}{
		{"id": int64(1), "age": int64(25)},
		{"id": int64(2), "age": int64(30)},
		{"id": int64(3), "age": int64(35)},
	}

	for _, row := range rows {
		if err := db.InsertRow("users", row); err != nil {
			t.Fatalf("Failed to insert row: %v", err)
		}
	}

	stats, err := db.GetColumnStats("users", "age")
	if err != nil {
		t.Fatalf("Failed to get column stats: %v", err)
	}

	if stats == nil {
		t.Fatal("Column stats should not be nil")
	}

	if stats.ColumnName != "age" {
		t.Errorf("Expected column name age, got %s", stats.ColumnName)
	}
}

func TestDropTable(t *testing.T) {
	dbFile := "test_drop_table.db"
	defer os.Remove(dbFile)

	db, err := NewWeDBDatabase(dbFile, 4096)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}

	schema := &api.TableSchema{
		TableName: "users",
		Columns: []api.ColumnSchema{
			{Name: "id", Type: api.TypeInteger},
		},
		PrimaryKey: "id",
	}

	if err := db.CreateTable(schema); err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	if !db.TableExists("users") {
		t.Error("Table should exist")
	}

	if err := db.DropTable("users"); err != nil {
		t.Fatalf("Failed to drop table: %v", err)
	}

	if db.TableExists("users") {
		t.Error("Table should not exist after drop")
	}
}

func TestListTables(t *testing.T) {
	dbFile := "test_list_tables.db"
	defer os.Remove(dbFile)

	db, err := NewWeDBDatabase(dbFile, 4096)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	schema := &api.TableSchema{
		TableName: "users",
		Columns: []api.ColumnSchema{
			{Name: "id", Type: api.TypeInteger},
		},
		PrimaryKey: "id",
	}

	if err := db.CreateTable(schema); err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	tables := db.ListTables()
	if len(tables) != 1 {
		t.Errorf("Expected 1 table, got %d", len(tables))
	}

	if tables[0] != "users" {
		t.Errorf("Expected table name users, got %s", tables[0])
	}
}