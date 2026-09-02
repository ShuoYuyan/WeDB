package adapter

import (
	"os"
	"testing"

	"github.com/wedb/wedb/internal/api"
	"github.com/wedb/wedb/internal/storage"
)

func TestWeDBAdapter_GetTableSchema(t *testing.T) {
	dbFile := "test_get_schema.db"
	defer os.Remove(dbFile)

	wedbDB, err := storage.NewWeDBDatabase(dbFile, 4096)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer wedbDB.Close()

	adapter := NewWeDBAdapter(wedbDB)

	schema := &api.TableSchema{
		TableName: "users",
		Columns: []api.ColumnSchema{
			{Name: "id", Type: api.TypeInteger, PrimaryKey: true},
			{Name: "name", Type: api.TypeText},
			{Name: "age", Type: api.TypeInteger},
		},
		PrimaryKey: "id",
	}

	if err := adapter.CreateTable(schema); err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	gotSchema, err := adapter.GetTableSchema("users")
	if err != nil {
		t.Fatalf("Failed to get table schema: %v", err)
	}

	if gotSchema.TableName != "users" {
		t.Errorf("Expected table name users, got %s", gotSchema.TableName)
	}

	if len(gotSchema.Columns) != 3 {
		t.Errorf("Expected 3 columns, got %d", len(gotSchema.Columns))
	}

	if gotSchema.PrimaryKey != "id" {
		t.Errorf("Expected primary key id, got %s", gotSchema.PrimaryKey)
	}
}

func TestWeDBAdapter_ScanTableWithColumns(t *testing.T) {
	dbFile := "test_scan_columns_adapter.db"
	defer os.Remove(dbFile)

	wedbDB, err := storage.NewWeDBDatabase(dbFile, 4096)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer wedbDB.Close()

	adapter := NewWeDBAdapter(wedbDB)

	schema := &api.TableSchema{
		TableName: "users",
		Columns: []api.ColumnSchema{
			{Name: "id", Type: api.TypeInteger},
			{Name: "name", Type: api.TypeText},
			{Name: "age", Type: api.TypeInteger},
		},
		PrimaryKey: "id",
	}

	if err := adapter.CreateTable(schema); err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	row := map[string]interface{}{
		"id":   int64(1),
		"name": "Alice",
		"age":  int64(25),
	}

	if err := adapter.InsertRow("users", row); err != nil {
		t.Fatalf("Failed to insert row: %v", err)
	}

	// 测试 ScanTableWithColumns
	rows, err := adapter.ScanTableWithColumns("users", []string{"id", "name"})
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

func TestWeDBAdapter_UpdateRows(t *testing.T) {
	dbFile := "test_update_rows_adapter.db"
	defer os.Remove(dbFile)

	wedbDB, err := storage.NewWeDBDatabase(dbFile, 4096)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer wedbDB.Close()

	adapter := NewWeDBAdapter(wedbDB)

	schema := &api.TableSchema{
		TableName: "users",
		Columns: []api.ColumnSchema{
			{Name: "id", Type: api.TypeInteger},
			{Name: "name", Type: api.TypeText},
			{Name: "age", Type: api.TypeInteger},
		},
		PrimaryKey: "id",
	}

	if err := adapter.CreateTable(schema); err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	rows := []map[string]interface{}{
		{"id": int64(1), "name": "Alice", "age": int64(25)},
		{"id": int64(2), "name": "Bob", "age": int64(30)},
	}

	for _, row := range rows {
		if err := adapter.InsertRow("users", row); err != nil {
			t.Fatalf("Failed to insert row: %v", err)
		}
	}

	// 测试批量更新
	updateRows := []map[string]interface{}{
		{"age": int64(26)},
		{"age": int64(31)},
	}

	if err := adapter.UpdateRows("users", updateRows, "*"); err != nil {
		t.Fatalf("Failed to update rows: %v", err)
	}

	// 验证更新
	scanRows, err := adapter.ScanTable("users")
	if err != nil {
		t.Fatalf("Failed to scan table: %v", err)
	}

	if len(scanRows) != 2 {
		t.Errorf("Expected 2 rows, got %d", len(scanRows))
	}
}

func TestWeDBAdapter_DeleteRows(t *testing.T) {
	dbFile := "test_delete_rows_adapter.db"
	defer os.Remove(dbFile)

	wedbDB, err := storage.NewWeDBDatabase(dbFile, 4096)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer wedbDB.Close()

	adapter := NewWeDBAdapter(wedbDB)

	schema := &api.TableSchema{
		TableName: "users",
		Columns: []api.ColumnSchema{
			{Name: "id", Type: api.TypeInteger},
			{Name: "name", Type: api.TypeText},
			{Name: "age", Type: api.TypeInteger},
		},
		PrimaryKey: "id",
	}

	if err := adapter.CreateTable(schema); err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	rows := []map[string]interface{}{
		{"id": int64(1), "name": "Alice", "age": int64(25)},
		{"id": int64(2), "name": "Bob", "age": int64(30)},
		{"id": int64(3), "name": "Charlie", "age": int64(35)},
	}

	for _, row := range rows {
		if err := adapter.InsertRow("users", row); err != nil {
			t.Fatalf("Failed to insert row: %v", err)
		}
	}

	// 测试批量删除
	conditions := []string{"id = 1", "id = 2"}

	if err := adapter.DeleteRows("users", conditions); err != nil {
		t.Fatalf("Failed to delete rows: %v", err)
	}

	// 验证删除
	count, err := adapter.Count("users", "")
	if err != nil {
		t.Fatalf("Failed to count rows: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 row after delete, got %d", count)
	}
}

func TestWeDBAdapter_BeginTx(t *testing.T) {
	dbFile := "test_begin_tx.db"
	defer os.Remove(dbFile)

	wedbDB, err := storage.NewWeDBDatabase(dbFile, 4096)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer wedbDB.Close()

	adapter := NewWeDBAdapter(wedbDB)

	schema := &api.TableSchema{
		TableName: "users",
		Columns: []api.ColumnSchema{
			{Name: "id", Type: api.TypeInteger},
			{Name: "name", Type: api.TypeText},
		},
		PrimaryKey: "id",
	}

	if err := adapter.CreateTable(schema); err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	tx, err := adapter.BeginTx(nil, &api.TxOptions{})
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	// 在事务中插入数据
	row := map[string]interface{}{
		"id":   int64(1),
		"name": "Alice",
	}

	if err := adapter.InsertRow("users", row); err != nil {
		t.Fatalf("Failed to insert row: %v", err)
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		t.Fatalf("Failed to commit transaction: %v", err)
	}

	// 验证数据
	count, err := adapter.Count("users", "")
	if err != nil {
		t.Fatalf("Failed to count rows: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 row after commit, got %d", count)
	}
}

func TestWeDBAdapter_GetTableStats(t *testing.T) {
	dbFile := "test_table_stats_adapter.db"
	defer os.Remove(dbFile)

	wedbDB, err := storage.NewWeDBDatabase(dbFile, 4096)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer wedbDB.Close()

	adapter := NewWeDBAdapter(wedbDB)

	schema := &api.TableSchema{
		TableName: "users",
		Columns: []api.ColumnSchema{
			{Name: "id", Type: api.TypeInteger},
		},
		PrimaryKey: "id",
	}

	if err := adapter.CreateTable(schema); err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	stats, err := adapter.GetTableStats("users")
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

func TestWeDBAdapter_DropTable(t *testing.T) {
	dbFile := "test_drop_table_adapter.db"
	defer os.Remove(dbFile)

	wedbDB, err := storage.NewWeDBDatabase(dbFile, 4096)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer wedbDB.Close()

	adapter := NewWeDBAdapter(wedbDB)

	schema := &api.TableSchema{
		TableName: "users",
		Columns: []api.ColumnSchema{
			{Name: "id", Type: api.TypeInteger},
		},
		PrimaryKey: "id",
	}

	if err := adapter.CreateTable(schema); err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	if !adapter.TableExists("users") {
		t.Error("Table should exist")
	}

	if err := adapter.DropTable("users"); err != nil {
		t.Fatalf("Failed to drop table: %v", err)
	}

	if adapter.TableExists("users") {
		t.Error("Table should not exist after drop")
	}
}

func TestWeDBAdapter_ListTables(t *testing.T) {
	dbFile := "test_list_tables_adapter.db"
	defer os.Remove(dbFile)

	wedbDB, err := storage.NewWeDBDatabase(dbFile, 4096)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer wedbDB.Close()

	adapter := NewWeDBAdapter(wedbDB)

	schema := &api.TableSchema{
		TableName: "users",
		Columns: []api.ColumnSchema{
			{Name: "id", Type: api.TypeInteger},
		},
		PrimaryKey: "id",
	}

	if err := adapter.CreateTable(schema); err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	tables := adapter.ListTables()
	if len(tables) != 1 {
		t.Errorf("Expected 1 table, got %d", len(tables))
	}

	if tables[0] != "users" {
		t.Errorf("Expected table name users, got %s", tables[0])
	}
}