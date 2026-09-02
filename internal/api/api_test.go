package api

import (
	"context"
	"testing"
	"time"
)

func TestIsolationLevel_String(t *testing.T) {
	tests := []struct {
		name   string
		level  IsolationLevel
		expect string
	}{
		{"Default", LevelDefault, "DEFAULT"},
		{"ReadUncommitted", LevelReadUncommitted, "READ UNCOMMITTED"},
		{"ReadCommitted", LevelReadCommitted, "READ COMMITTED"},
		{"RepeatableRead", LevelRepeatableRead, "REPEATABLE READ"},
		{"Snapshot", LevelSnapshot, "SNAPSHOT"},
		{"Serializable", LevelSerializable, "SERIALIZABLE"},
		{"Unknown", IsolationLevel(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.level.String(); got != tt.expect {
				t.Errorf("IsolationLevel.String() = %v, want %v", got, tt.expect)
			}
		})
	}
}

func TestTxOptions(t *testing.T) {
	opts := &TxOptions{
		Isolation: LevelSerializable,
		ReadOnly:  true,
		Timeout:   30 * time.Second,
	}

	if opts.Isolation != LevelSerializable {
		t.Errorf("Expected isolation LevelSerializable")
	}
	if !opts.ReadOnly {
		t.Errorf("Expected ReadOnly to be true")
	}
	if opts.Timeout != 30*time.Second {
		t.Errorf("Expected timeout 30s")
	}
}

func TestTableStats(t *testing.T) {
	stats := &TableStats{
		RowCount:     100,
		TableSize:    4096,
		IndexCount:   3,
		ColumnCount:  5,
		LastModified: time.Now(),
		Created:      time.Now(),
	}

	if stats.RowCount != 100 {
		t.Errorf("Expected RowCount 100")
	}
	if stats.IndexCount != 3 {
		t.Errorf("Expected IndexCount 3")
	}
}

func TestColumnStats(t *testing.T) {
	stats := &ColumnStats{
		ColumnName:  "age",
		Type:        TypeInteger,
		NullCount:   10,
		UniqueCount: 80,
		Min:         int64(0),
		Max:         int64(100),
		Average:     50.0,
		StdDev:      28.87,
	}

	if stats.ColumnName != "age" {
		t.Errorf("Expected ColumnName age")
	}
	if stats.Type != TypeInteger {
		t.Errorf("Expected TypeInteger")
	}
	if stats.NullCount != 10 {
		t.Errorf("Expected NullCount 10")
	}
}

func TestColumnSchema(t *testing.T) {
	col := &ColumnSchema{
		Name:          "id",
		Type:          TypeInteger,
		Nullable:      false,
		Default:       nil,
		PrimaryKey:    true,
		AutoIncrement: true,
		Unique:        true,
	}

	if col.Name != "id" {
		t.Errorf("Expected Name id")
	}
	if !col.PrimaryKey {
		t.Errorf("Expected PrimaryKey to be true")
	}
	if !col.AutoIncrement {
		t.Errorf("Expected AutoIncrement to be true")
	}
}

func TestTableSchema(t *testing.T) {
	schema := &TableSchema{
		TableName: "users",
		Columns: []ColumnSchema{
			{Name: "id", Type: TypeInteger, PrimaryKey: true},
			{Name: "name", Type: TypeText},
		},
		PrimaryKey:    "id",
		AutoIncrement: true,
	}

	if schema.TableName != "users" {
		t.Errorf("Expected TableName users")
	}
	if schema.PrimaryKey != "id" {
		t.Errorf("Expected PrimaryKey id")
	}
	if len(schema.Columns) != 2 {
		t.Errorf("Expected 2 columns")
	}
}

func TestColumnType_String(t *testing.T) {
	tests := []struct {
		name     string
		colType  ColumnType
		expected string
	}{
		{"Integer", TypeInteger, "INTEGER"},
		{"Real", TypeReal, "REAL"},
		{"Text", TypeText, "TEXT"},
		{"Blob", TypeBlob, "BLOB"},
		{"Null", TypeNull, "NULL"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := string(tt.colType)
			if got != tt.expected {
				t.Errorf("ColumnType.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIndexType(t *testing.T) {
	tests := []struct {
		name     string
		idxType  IndexType
		expected string
	}{
		{"BTree", TypeBTree, "BTREE"},
		{"Hash", TypeHash, "HASH"},
		{"FullText", TypeFullText, "FULLTEXT"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := string(tt.idxType)
			if got != tt.expected {
				t.Errorf("IndexType.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIndexInfo(t *testing.T) {
	idx := &IndexInfo{
		IndexName: "idx_age",
		Columns:   []string{"age"},
		Unique:    false,
		Type:      TypeBTree,
	}

	if idx.IndexName != "idx_age" {
		t.Errorf("Expected IndexName idx_age")
	}
	if len(idx.Columns) != 1 {
		t.Errorf("Expected 1 column")
	}
}

func TestSortOrder(t *testing.T) {
	if SortAsc != "ASC" {
		t.Errorf("Expected SortAsc to be ASC")
	}
	if SortDesc != "DESC" {
		t.Errorf("Expected SortDesc to be DESC")
	}
}

func TestSortBy(t *testing.T) {
	sort := SortBy{
		Column: "age",
		Order:  SortDesc,
	}

	if sort.Column != "age" {
		t.Errorf("Expected Column age")
	}
	if sort.Order != SortDesc {
		t.Errorf("Expected Order SortDesc")
	}
}

func TestQueryOptions(t *testing.T) {
	opts := &QueryOptions{
		Columns: []string{"id", "name"},
		Where:   "age > 18",
		OrderBy: []SortBy{{Column: "age", Order: SortAsc}},
		Limit:   10,
		Offset:  5,
	}

	if len(opts.Columns) != 2 {
		t.Errorf("Expected 2 columns")
	}
	if opts.Where != "age > 18" {
		t.Errorf("Expected Where 'age > 18'")
	}
	if opts.Limit != 10 {
		t.Errorf("Expected Limit 10")
	}
	if opts.Offset != 5 {
		t.Errorf("Expected Offset 5")
	}
}

func TestTransaction_Duration(t *testing.T) {
	// 这是一个接口测试，实际实现会在 storage 包中测试
	// 这里只测试接口的定义
	var _ Transaction = (*mockTx)(nil)
}

func TestDatabase_Ping(t *testing.T) {
	// 这是一个接口测试，实际实现会在 storage 包中测试
	// 这里只测试接口的定义
	var _ Database = (*mockDB)(nil)
}

// mockTx 实现 Transaction 接口用于测试
type mockTx struct{}
func (m *mockTx) Commit() error { return nil }
func (m *mockTx) Rollback() error { return nil }
func (m *mockTx) Savepoint(name string) error { return nil }
func (m *mockTx) RollbackTo(name string) error { return nil }
func (m *mockTx) ReleaseSavepoint(name string) error { return nil }
func (m *mockTx) IsActive() bool { return false }
func (m *mockTx) IsolationLevel() IsolationLevel { return LevelDefault }
func (m *mockTx) ID() string { return "test" }
func (m *mockTx) StartTime() time.Time { return time.Now() }
func (m *mockTx) Duration() time.Duration { return 0 }

// mockDB 实现 Database 接口用于测试
type mockDB struct{}
func (m *mockDB) Ping() error { return nil }
func (m *mockDB) Close() error { return nil }
func (m *mockDB) CreateTable(schema *TableSchema) error { return nil }
func (m *mockDB) DropTable(tableName string) error { return nil }
func (m *mockDB) GetTableSchema(tableName string) (*TableSchema, error) { return nil, nil }
func (m *mockDB) ListTables() []string { return nil }
func (m *mockDB) TableExists(tableName string) bool { return false }
func (m *mockDB) ScanTable(tableName string) ([]map[string]interface{}, error) { return nil, nil }
func (m *mockDB) ScanTableWithColumns(tableName string, columns []string) ([]map[string]interface{}, error) { return nil, nil }
func (m *mockDB) InsertRow(tableName string, row map[string]interface{}) error { return nil }
func (m *mockDB) InsertRows(tableName string, rows []map[string]interface{}) error { return nil }
func (m *mockDB) UpdateRow(tableName string, row map[string]interface{}, condition string) error { return nil }
func (m *mockDB) UpdateRows(tableName string, rows []map[string]interface{}, condition string) error { return nil }
func (m *mockDB) DeleteRow(tableName string, condition string) error { return nil }
func (m *mockDB) DeleteRows(tableName string, conditions []string) error { return nil }
func (m *mockDB) CreateIndex(tableName string, index *IndexInfo) error { return nil }
func (m *mockDB) DropIndex(tableName string, indexName string) error { return nil }
func (m *mockDB) GetIndexInfo(tableName string) ([]IndexInfo, error) { return nil, nil }
func (m *mockDB) IndexExists(tableName string, indexName string) bool { return false }
func (m *mockDB) Begin() (Transaction, error) { return nil, nil }
func (m *mockDB) BeginTx(ctx context.Context, opts *TxOptions) (Transaction, error) { return nil, nil }
func (m *mockDB) Count(tableName string, condition string) (int64, error) { return 0, nil }
func (m *mockDB) Min(tableName string, column string, condition string) (interface{}, error) { return nil, nil }
func (m *mockDB) Max(tableName string, column string, condition string) (interface{}, error) { return nil, nil }
func (m *mockDB) Sum(tableName string, column string, condition string) (float64, error) { return 0, nil }
func (m *mockDB) Avg(tableName string, column string, condition string) (float64, error) { return 0, nil }
func (m *mockDB) GetTableStats(tableName string) (*TableStats, error) { return nil, nil }
func (m *mockDB) GetColumnStats(tableName string, column string) (*ColumnStats, error) { return nil, nil }
func (m *mockDB) IsClosed() bool { return false }