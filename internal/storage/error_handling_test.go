package storage

import (
	"os"
	"testing"

	"github.com/wedb/wedb/internal/api"
)

func TestErrorHandling_EmptyTableName(t *testing.T) {
	dbFile := "test_empty_table_name.db"
	defer os.Remove(dbFile)

	db, err := NewWeDBDatabase(dbFile, 4096)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// 测试空表名
	err = db.CreateTable(&api.TableSchema{
		TableName: "",
		Columns:   []api.ColumnSchema{{Name: "id", Type: api.TypeInteger}},
	})
	if err == nil {
		t.Error("Expected error for empty table name")
	}
}

func TestErrorHandling_DuplicateColumnNames(t *testing.T) {
	dbFile := "test_duplicate_columns.db"
	defer os.Remove(dbFile)

	db, err := NewWeDBDatabase(dbFile, 4096)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// 测试重复列名
	err = db.CreateTable(&api.TableSchema{
		TableName: "users",
		Columns: []api.ColumnSchema{
			{Name: "id", Type: api.TypeInteger},
			{Name: "id", Type: api.TypeInteger}, // 重复列名
		},
	})
	if err == nil {
		t.Error("Expected error for duplicate column names")
	}
}

func TestErrorHandling_EmptyRow(t *testing.T) {
	dbFile := "test_empty_row.db"
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

	// 测试空行
	err = db.InsertRow("users", nil)
	if err == nil {
		t.Error("Expected error for nil row")
	}

	// 测试空行数据
	err = db.InsertRow("users", map[string]interface{}{})
	if err == nil {
		t.Error("Expected error for empty row")
	}
}

func TestErrorHandling_InvalidColumns(t *testing.T) {
	dbFile := "test_invalid_columns.db"
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

	// 测试不存在的列
	err = db.InsertRow("users", map[string]interface{}{
		"id":   int64(1),
		"age":  int64(25), // age 列不存在
	})
	if err == nil {
		t.Error("Expected error for invalid column")
	}
}

func TestErrorHandling_MissingPrimaryKey(t *testing.T) {
	dbFile := "test_missing_pk.db"
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
		PrimaryKey: "id", // id 是主键
	}

	if err := db.CreateTable(schema); err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// 测试缺少主键
	err = db.InsertRow("users", map[string]interface{}{
		"name": "Alice", // 缺少 id 列
	})
	if err == nil {
		t.Error("Expected error for missing primary key")
	}
}

func TestErrorHandling_TableAlreadyExists(t *testing.T) {
	dbFile := "test_table_exists.db"
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

	// 测试表已存在
	err = db.CreateTable(schema)
	if err == nil {
		t.Error("Expected error for existing table")
	}
}

func TestErrorHandling_NoColumns(t *testing.T) {
	dbFile := "test_no_columns.db"
	defer os.Remove(dbFile)

	db, err := NewWeDBDatabase(dbFile, 4096)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// 测试没有列
	err = db.CreateTable(&api.TableSchema{
		TableName: "users",
		Columns:   []api.ColumnSchema{},
	})
	if err == nil {
		t.Error("Expected error for table with no columns")
	}
}

func TestErrorHandling_InvalidIndex(t *testing.T) {
	dbFile := "test_invalid_index.db"
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

	// 测试空索引名
	err = db.CreateIndex("users", &api.IndexInfo{
		IndexName: "",
		Columns:   []string{"name"},
	})
	if err == nil {
		t.Error("Expected error for empty index name")
	}

	// 测试没有列的索引
	err = db.CreateIndex("users", &api.IndexInfo{
		IndexName: "idx_name",
		Columns:   []string{},
	})
	if err == nil {
		t.Error("Expected error for index with no columns")
	}
}

func TestErrorHandling_TableNotFound(t *testing.T) {
	dbFile := "test_table_not_found.db"
	defer os.Remove(dbFile)

	db, err := NewWeDBDatabase(dbFile, 4096)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// 测试不存在的表
	_, err = db.ScanTable("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent table")
	}

	err = db.DropTable("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent table")
	}
}

func TestErrorHandling_DatabaseClosed(t *testing.T) {
	dbFile := "test_db_closed.db"
	defer os.Remove(dbFile)

	db, err := NewWeDBDatabase(dbFile, 4096)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}

	// 关闭数据库
	if err := db.Close(); err != nil {
		t.Fatalf("Failed to close database: %v", err)
	}

	// 测试在关闭的数据库上操作
	schema := &api.TableSchema{
		TableName: "users",
		Columns: []api.ColumnSchema{
			{Name: "id", Type: api.TypeInteger},
		},
		PrimaryKey: "id",
	}

	err = db.CreateTable(schema)
	if err == nil {
		t.Error("Expected error for operation on closed database")
	}

	err = db.Ping()
	if err == nil {
		t.Error("Expected error for ping on closed database")
	}
}

func TestErrorHandling_LongNames(t *testing.T) {
	dbFile := "test_long_names.db"
	defer os.Remove(dbFile)

	db, err := NewWeDBDatabase(dbFile, 4096)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// 测试过长的表名
	longName := string(make([]byte, 256))
	err = db.CreateTable(&api.TableSchema{
		TableName: longName,
		Columns: []api.ColumnSchema{
			{Name: "id", Type: api.TypeInteger},
		},
		PrimaryKey: "id",
	})
	if err == nil {
		t.Error("Expected error for table name too long")
	}
}

func TestErrorHandling_InvalidPrimaryKey(t *testing.T) {
	dbFile := "test_invalid_pk.db"
	defer os.Remove(dbFile)

	db, err := NewWeDBDatabase(dbFile, 4096)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// 测试不存在的主键
	err = db.CreateTable(&api.TableSchema{
		TableName: "users",
		Columns: []api.ColumnSchema{
			{Name: "id", Type: api.TypeInteger},
			{Name: "name", Type: api.TypeText},
		},
		PrimaryKey: "age", // age 列不存在
	})
	if err == nil {
		t.Error("Expected error for nonexistent primary key column")
	}
}