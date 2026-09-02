package wqlv3

import (
	"sync"
	"testing"
)

// mockAdapter 是一个内存中模拟的 Adapter
// 用于测试 DML 操作而不需要真实的 WeDB 数据库
type mockAdapter struct {
	mu        sync.Mutex
	tables    map[string][]map[string]interface{}
	autoID    map[string]int64
}

func newMockAdapter() *mockAdapter {
	return &mockAdapter{
		tables: make(map[string][]map[string]interface{}),
		autoID: make(map[string]int64),
	}
}

func (m *mockAdapter) ScanTable(tableName string) ([]map[string]interface{}, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	rows, ok := m.tables[tableName]
	if !ok {
		return []map[string]interface{}{}, nil
	}
	out := make([]map[string]interface{}, len(rows))
	for i, r := range rows {
		cp := make(map[string]interface{}, len(r))
		for k, v := range r {
			cp[k] = v
		}
		out[i] = cp
	}
	return out, nil
}

func (m *mockAdapter) ScanTableWithColumns(tableName string, columns []string) ([]map[string]interface{}, error) {
	rows, err := m.ScanTable(tableName)
	if err != nil {
		return nil, err
	}
	if len(columns) == 0 {
		return rows, nil
	}
	out := make([]map[string]interface{}, 0, len(rows))
	for _, row := range rows {
		filtered := make(map[string]interface{})
		for _, c := range columns {
			if v, ok := row[c]; ok {
				filtered[c] = v
			}
		}
		out = append(out, filtered)
	}
	return out, nil
}

func (m *mockAdapter) ListTables() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]string, 0, len(m.tables))
	for t := range m.tables {
		out = append(out, t)
	}
	return out
}

func (m *mockAdapter) TableExists(name string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.tables[name]
	return ok
}

func (m *mockAdapter) CreateTable(schema *TableSchema) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.tables[schema.Name]; ok {
		return nil // 简化：已存在则不报错
	}
	m.tables[schema.Name] = []map[string]interface{}{}
	return nil
}

func (m *mockAdapter) DropTable(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.tables, name)
	return nil
}

func (m *mockAdapter) InsertRow(tableName string, row map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.tables[tableName]; !ok {
		return errMock("table not found: " + tableName)
	}
	// 深拷贝
	cp := make(map[string]interface{}, len(row))
	for k, v := range row {
		cp[k] = v
	}
	m.tables[tableName] = append(m.tables[tableName], cp)
	return nil
}

func (m *mockAdapter) InsertRows(tableName string, rows []map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.tables[tableName]; !ok {
		return errMock("table not found: " + tableName)
	}
	for _, r := range rows {
		cp := make(map[string]interface{}, len(r))
		for k, v := range r {
			cp[k] = v
		}
		m.tables[tableName] = append(m.tables[tableName], cp)
	}
	return nil
}

func (m *mockAdapter) UpdateRow(tableName string, row map[string]interface{}, condition string) (retErr error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	rows, ok := m.tables[tableName]
	if !ok {
		return errMock("table not found: " + tableName)
	}
	if condition == "" {
		return errMock("empty condition")
	}
	expr, err := ParseWhere(condition)
	if err != nil {
		return err
	}
	for _, r := range rows {
		if EvalBoolExpr(expr, r) {
			for k, v := range row {
				r[k] = v
			}
		}
	}
	return nil
}

func (m *mockAdapter) DeleteRow(tableName string, condition string) (retErr error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	rows, ok := m.tables[tableName]
	if !ok {
		return errMock("table not found: " + tableName)
	}
	if condition == "" {
		return errMock("empty condition")
	}
	expr, err := ParseWhere(condition)
	if err != nil {
		return err
	}
	defer func() {
		if r := recover(); r != nil {
			retErr = errMock("delete panic: " + asString(r))
		}
	}()
	kept := make([]map[string]interface{}, 0, len(rows))
	for _, r := range rows {
		if !EvalBoolExpr(expr, r) {
			kept = append(kept, r)
		}
	}
	m.tables[tableName] = kept
	return nil
}

func (m *mockAdapter) Count(tableName, condition string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	rows, ok := m.tables[tableName]
	if !ok {
		return 0, errMock("table not found: " + tableName)
	}
	if condition == "" {
		return int64(len(rows)), nil
	}
	expr, err := ParseWhere(condition)
	if err != nil {
		return 0, err
	}
	n := int64(0)
	for _, r := range rows {
		if EvalBoolExpr(expr, r) {
			n++
		}
	}
	return n, nil
}

func (m *mockAdapter) Min(tableName, column, condition string) (interface{}, error) {
	rows, err := m.ScanTable(tableName)
	if err != nil {
		return nil, err
	}
	filtered, err := filterByCondition(rows, condition)
	if err != nil {
		return nil, err
	}
	if len(filtered) == 0 {
		return nil, nil
	}
	var min interface{}
	for _, r := range filtered {
		v, ok := r[column]
		if !ok {
			continue
		}
		if min == nil || compareValues(v, min) < 0 {
			min = v
		}
	}
	return min, nil
}

func (m *mockAdapter) Max(tableName, column, condition string) (interface{}, error) {
	rows, err := m.ScanTable(tableName)
	if err != nil {
		return nil, err
	}
	filtered, err := filterByCondition(rows, condition)
	if err != nil {
		return nil, err
	}
	if len(filtered) == 0 {
		return nil, nil
	}
	var max interface{}
	for _, r := range filtered {
		v, ok := r[column]
		if !ok {
			continue
		}
		if max == nil || compareValues(v, max) > 0 {
			max = v
		}
	}
	return max, nil
}

func (m *mockAdapter) Sum(tableName, column, condition string) (float64, error) {
	rows, err := m.ScanTable(tableName)
	if err != nil {
		return 0, err
	}
	filtered, err := filterByCondition(rows, condition)
	if err != nil {
		return 0, err
	}
	var sum float64
	for _, r := range filtered {
		if v, ok := r[column]; ok {
			if f, _ := toFloat64ForFilter(v); true {
				sum += f
			}
		}
	}
	return sum, nil
}

func (m *mockAdapter) Avg(tableName, column, condition string) (float64, error) {
	rows, err := m.ScanTable(tableName)
	if err != nil {
		return 0, err
	}
	filtered, err := filterByCondition(rows, condition)
	if err != nil {
		return 0, err
	}
	if len(filtered) == 0 {
		return 0, nil
	}
	var sum float64
	for _, r := range filtered {
		if v, ok := r[column]; ok {
			if f, _ := toFloat64ForFilter(v); true {
				sum += f
			}
		}
	}
	return sum / float64(len(filtered)), nil
}

func filterByCondition(rows []map[string]interface{}, condition string) ([]map[string]interface{}, error) {
	if condition == "" {
		return rows, nil
	}
	return filterRows(rows, condition), nil
}

type mockError struct{ msg string }

func (e *mockError) Error() string { return e.msg }
func errMock(msg string) error     { return &mockError{msg: msg} }
func asString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// ===== DML 测试 =====

func setupTestDB(t *testing.T) (*Database, *mockAdapter) {
	m := newMockAdapter()
	_ = m.CreateTable(NewTableSchema("users",
		NewColumn("id", "INTEGER", false),
		NewColumn("name", "TEXT", false),
		NewColumn("age", "INTEGER", true),
	))
	db := NewDatabase(m)
	return db, m
}

func TestInsert(t *testing.T) {
	db, m := setupTestDB(t)

	// 插入单行
	n, err := db.Insert("users").Value(map[string]interface{}{
		"id": int64(1), "name": "alice", "age": int64(30),
	}).Execute()
	if err != nil {
		t.Fatalf("Insert failed: %v", err)
	}
	if n != 1 {
		t.Errorf("Insert affected = %d, want 1", n)
	}

	// 验证
	rows, _ := db.Table("users").All()
	if len(rows) != 1 || rows[0]["name"] != "alice" {
		t.Errorf("Insert not persisted: %v", rows)
	}
	_ = m
}

func TestInsertMulti(t *testing.T) {
	db, _ := setupTestDB(t)

	rows := []map[string]interface{}{
		{"id": int64(1), "name": "alice", "age": int64(30)},
		{"id": int64(2), "name": "bob", "age": int64(25)},
		{"id": int64(3), "name": "carol", "age": int64(40)},
	}
	n, err := db.Insert("users").Values(rows...).Execute()
	if err != nil {
		t.Fatalf("Insert failed: %v", err)
	}
	if n != 3 {
		t.Errorf("Insert affected = %d, want 3", n)
	}
}

func TestUpdate(t *testing.T) {
	db, _ := setupTestDB(t)
	_, _ = db.Insert("users").Values(
		map[string]interface{}{"id": int64(1), "name": "alice", "age": int64(30)},
		map[string]interface{}{"id": int64(2), "name": "bob", "age": int64(25)},
	).Execute()

	// 更新
	_, err := db.Update("users").Set("age", int64(31)).Where("id = 1").Execute()
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// 验证
	row, _ := db.Table("users").Where("id = 1").First()
	if row["age"] != int64(31) {
		t.Errorf("Update not applied: age = %v, want 31", row["age"])
	}
	if row["name"] != "alice" {
		t.Errorf("Name changed: %v", row["name"])
	}
}

func TestDelete(t *testing.T) {
	db, _ := setupTestDB(t)
	_, _ = db.Insert("users").Values(
		map[string]interface{}{"id": int64(1), "name": "alice", "age": int64(30)},
		map[string]interface{}{"id": int64(2), "name": "bob", "age": int64(25)},
		map[string]interface{}{"id": int64(3), "name": "carol", "age": int64(40)},
	).Execute()

	// 删除
	_, err := db.Delete("users").Where("age < 30").Execute()
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// 验证：应该只剩 alice (30) 和 carol (40)
	rows, _ := db.Table("users").All()
	if len(rows) != 2 {
		t.Errorf("After delete, got %d rows, want 2", len(rows))
	}
	for _, r := range rows {
		if r["age"] == int64(25) {
			t.Errorf("bob (age=25) should have been deleted")
		}
	}
}

func TestSafetyCheckNoWhere(t *testing.T) {
	db, _ := setupTestDB(t)
	_, _ = db.Insert("users").Value(
		map[string]interface{}{"id": int64(1), "name": "alice", "age": int64(30)},
	).Execute()

	// 无 WHERE 条件的 UPDATE 应该被拒绝
	_, err := db.Update("users").Set("age", 0).Execute()
	if err == nil {
		t.Error("UPDATE without WHERE should fail for safety")
	}

	// 无 WHERE 条件的 DELETE 应该被拒绝
	_, err = db.Delete("users").Execute()
	if err == nil {
		t.Error("DELETE without WHERE should fail for safety")
	}
}

func TestCreateTableViaFluent(t *testing.T) {
	m := newMockAdapter()
	db := NewDatabase(m)

	err := db.CreateTable(NewTableSchema("products",
		NewColumn("id", "INTEGER", false),
		NewColumn("name", "TEXT", false),
		NewColumn("price", "REAL", true),
	))
	if err != nil {
		t.Fatalf("CreateTable failed: %v", err)
	}
	if !m.TableExists("products") {
		t.Error("Table products was not created")
	}
}

func TestFullDMLCycle(t *testing.T) {
	// 完整生命周期: CREATE -> INSERT -> UPDATE -> SELECT -> DELETE -> DROP
	m := newMockAdapter()
	db := NewDatabase(m)

	// 1. CREATE
	err := db.CreateTable(NewTableSchema("orders",
		NewColumn("id", "INTEGER", false),
		NewColumn("product", "TEXT", false),
		NewColumn("qty", "INTEGER", true),
	))
	if err != nil {
		t.Fatalf("CREATE failed: %v", err)
	}

	// 2. INSERT
	_, err = db.Insert("orders").Values(
		map[string]interface{}{"id": int64(1), "product": "apple", "qty": int64(10)},
		map[string]interface{}{"id": int64(2), "product": "banana", "qty": int64(20)},
		map[string]interface{}{"id": int64(3), "product": "cherry", "qty": int64(30)},
	).Execute()
	if err != nil {
		t.Fatalf("INSERT failed: %v", err)
	}

	// 3. SELECT
	rows, _ := db.Table("orders").Where("qty > 15").All()
	if len(rows) != 2 {
		t.Errorf("SELECT should return 2 rows, got %d", len(rows))
	}

	// 4. UPDATE
	_, err = db.Update("orders").Set("qty", int64(100)).Where("product = 'apple'").Execute()
	if err != nil {
		t.Fatalf("UPDATE failed: %v", err)
	}
	row, _ := db.Table("orders").Where("product = 'apple'").First()
	if row["qty"] != int64(100) {
		t.Errorf("UPDATE not applied: qty = %v, want 100", row["qty"])
	}

	// 5. COUNT
	count, _ := db.Table("orders").Count()
	if count != 3 {
		t.Errorf("COUNT = %d, want 3", count)
	}

	// 6. DELETE
	_, err = db.Delete("orders").Where("qty < 50").Execute()
	if err != nil {
		t.Fatalf("DELETE failed: %v", err)
	}
	count, _ = db.Table("orders").Count()
	if count != 1 {
		t.Errorf("After DELETE, COUNT = %d, want 1", count)
	}

	// 7. DROP
	err = db.DropTable("orders")
	if err != nil {
		t.Fatalf("DROP failed: %v", err)
	}
	if m.TableExists("orders") {
		t.Error("Table orders should be dropped")
	}
}

func TestEvaluateQueryInsertString(t *testing.T) {
	db, _ := setupTestDB(t)

	result, err := EvaluateQuery(db, `Insert("users").Values({"id": 1, "name": "alice", "age": 30}).Execute()`)
	if err != nil {
		t.Fatalf("EvaluateQuery INSERT failed: %v", err)
	}
	if result.AffectedRows != 1 {
		t.Errorf("AffectedRows = %d, want 1", result.AffectedRows)
	}

	// 验证
	row, _ := db.Table("users").First()
	if row["name"] != "alice" {
		t.Errorf("Insert via string failed: %v", row)
	}
}

func TestEvaluateQueryUpdateString(t *testing.T) {
	db, _ := setupTestDB(t)
	_, _ = db.Insert("users").Value(
		map[string]interface{}{"id": int64(1), "name": "alice", "age": int64(30)},
	).Execute()

	_, err := EvaluateQuery(db, `Update("users").Set("age", 31).Where("id = 1").Execute()`)
	if err != nil {
		t.Fatalf("EvaluateQuery UPDATE failed: %v", err)
	}

	row, _ := db.Table("users").First()
	if row["age"] != int64(31) {
		t.Errorf("Update via string failed: age = %v, want 31", row["age"])
	}
}

func TestEvaluateQueryDeleteString(t *testing.T) {
	db, _ := setupTestDB(t)
	_, _ = db.Insert("users").Values(
		map[string]interface{}{"id": int64(1), "name": "alice", "age": int64(30)},
		map[string]interface{}{"id": int64(2), "name": "bob", "age": int64(25)},
	).Execute()

	_, err := EvaluateQuery(db, `Delete("users").Where("age < 30").Execute()`)
	if err != nil {
		t.Fatalf("EvaluateQuery DELETE failed: %v", err)
	}

	count, _ := db.Table("users").Count()
	if count != 1 {
		t.Errorf("After string DELETE, count = %d, want 1", count)
	}
}

func TestEvaluateQueryCreateTableString(t *testing.T) {
	m := newMockAdapter()
	db := NewDatabase(m)

	_, err := EvaluateQuery(db, `CreateTable("orders", ("id INTEGER NOT NULL", "product TEXT", "qty INTEGER")).Execute()`)
	if err != nil {
		t.Fatalf("EvaluateQuery CREATE TABLE failed: %v", err)
	}
	if !m.TableExists("orders") {
		t.Error("Table orders was not created")
	}
}
