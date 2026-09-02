package wqlv3

import (
	"testing"
)

// TestNoQuotesDMLInsert 测试无双引号 DML Insert 解析
func TestNoQuotesDMLInsert(t *testing.T) {
	db, m := setupTestDB(t)

	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "Insert - 对象字面量形式",
			input: `db.Table(users).Insert({id: 1, name: "alice", age: 30}).Execute()`,
		},
		{
			name:  "Insert - 列值对形式",
			input: `db.Table(users).Insert(id, 2, name, "bob", age, 25).Execute()`,
		},
		{
			name:  "Insert - 省略 Execute() 终结",
			input: `db.Table(users).Insert({id: 3, name: "carol", age: 40})`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := EvaluateQueryNoQuotes(db, tt.input)
			if err != nil {
				t.Fatalf("EvaluateQueryNoQuotes(%q) error: %v", tt.input, err)
			}
		})
	}

	// 验证所有行都已插入
	rows, _ := m.ScanTable("users")
	if len(rows) != 3 {
		t.Errorf("expected 3 rows inserted, got %d", len(rows))
	}
}

// TestNoQuotesDMLUpdate 测试无双引号 DML Update 解析
func TestNoQuotesDMLUpdate(t *testing.T) {
	db, _ := setupTestDB(t)
	_, _ = db.Insert("users").Values(
		map[string]interface{}{"id": int64(1), "name": "alice", "age": int64(30)},
	).Execute()

	// Update 用 Set + Where
	_, err := EvaluateQueryNoQuotes(db, `db.Table(users).Set(age, 31).Where(id = 1).Execute()`)
	if err != nil {
		t.Fatalf("Update via no-quotes failed: %v", err)
	}

	row, _ := db.Table("users").First()
	if row["age"] != int64(31) {
		t.Errorf("Update not applied: age = %v, want 31", row["age"])
	}
}

// TestNoQuotesDMLDelete 测试无双引号 DML Delete 解析
func TestNoQuotesDMLDelete(t *testing.T) {
	db, _ := setupTestDB(t)
	_, _ = db.Insert("users").Values(
		map[string]interface{}{"id": int64(1), "name": "alice", "age": int64(30)},
		map[string]interface{}{"id": int64(2), "name": "bob", "age": int64(25)},
		map[string]interface{}{"id": int64(3), "name": "carol", "age": int64(40)},
	).Execute()

	_, err := EvaluateQueryNoQuotes(db, `db.Table(users).Where(age < 30).Delete().Execute()`)
	if err != nil {
		t.Fatalf("Delete via no-quotes failed: %v", err)
	}

	count, _ := db.Table("users").Count()
	if count != 2 {
		t.Errorf("After Delete, count = %d, want 2 (alice + carol)", count)
	}
}

// TestNoQuotesDDLCreateTable 测试无双引号 DDL CreateTable 解析
func TestNoQuotesDDLCreateTable(t *testing.T) {
	m := newMockAdapter()
	db := NewDatabase(m)

	_, err := EvaluateQueryNoQuotes(db, `db.Table(products).Create(id INTEGER PRIMARY KEY, name TEXT, price REAL).Execute()`)
	if err != nil {
		t.Fatalf("CreateTable via no-quotes failed: %v", err)
	}

	if !m.TableExists("products") {
		t.Error("Table products should be created")
	}
}

// TestNoQuotesDDLDropTable 测试无双引号 DDL DropTable 解析
func TestNoQuotesDDLDropTable(t *testing.T) {
	m := newMockAdapter()
	db := NewDatabase(m)
	_ = m.CreateTable(NewTableSchema("temp", NewColumn("id", "INTEGER", false)))

	_, err := EvaluateQueryNoQuotes(db, `db.Table(temp).Drop().Execute()`)
	if err != nil {
		t.Fatalf("DropTable via no-quotes failed: %v", err)
	}

	if m.TableExists("temp") {
		t.Error("Table temp should be dropped")
	}
}

// TestNoQuotesFullDMLCycle 完整 DML 周期测试
func TestNoQuotesFullDMLCycle(t *testing.T) {
	m := newMockAdapter()
	db := NewDatabase(m)

	// CREATE
	if _, err := EvaluateQueryNoQuotes(db, `db.Table(orders).Create(id INTEGER PRIMARY KEY, product TEXT, qty INTEGER).Execute()`); err != nil {
		t.Fatalf("CREATE failed: %v", err)
	}

	// INSERT
	if _, err := EvaluateQueryNoQuotes(db, `db.Table(orders).Insert({id: 1, product: "apple", qty: 10}).Execute()`); err != nil {
		t.Fatalf("INSERT failed: %v", err)
	}
	if _, err := EvaluateQueryNoQuotes(db, `db.Table(orders).Insert({id: 2, product: "banana", qty: 20}).Execute()`); err != nil {
		t.Fatalf("INSERT 2 failed: %v", err)
	}

	// SELECT 验证
	res, _ := EvaluateQueryNoQuotes(db, `db.Table(orders).All()`)
	if len(res.Rows) != 2 {
		t.Errorf("SELECT should return 2 rows, got %d", len(res.Rows))
	}

	// UPDATE
	if _, err := EvaluateQueryNoQuotes(db, `db.Table(orders).Set(qty, 100).Where(product = "apple").Execute()`); err != nil {
		t.Fatalf("UPDATE failed: %v", err)
	}
	row, _ := db.Table("orders").Where("product = 'apple'").First()
	if row["qty"] != int64(100) {
		t.Errorf("UPDATE not applied: qty = %v, want 100", row["qty"])
	}

	// DELETE
	if _, err := EvaluateQueryNoQuotes(db, `db.Table(orders).Where(qty < 50).Delete().Execute()`); err != nil {
		t.Fatalf("DELETE failed: %v", err)
	}
	count, _ := db.Table("orders").Count()
	if count != 1 {
		t.Errorf("After DELETE, count = %d, want 1", count)
	}

	// DROP
	if _, err := EvaluateQueryNoQuotes(db, `db.Table(orders).Drop().Execute()`); err != nil {
		t.Fatalf("DROP failed: %v", err)
	}
	if m.TableExists("orders") {
		t.Error("Table orders should be dropped")
	}
}

// TestParserASTInsert 测试 Insert 操作的 AST 生成
func TestParserASTInsert(t *testing.T) {
	// 通过 EvaluateQueryNoQuotes 间接验证（AST 在 buildQueryBuilder 中被消费）
	db, m := setupTestDB(t)

	input := `db.Table(users).Insert({id: 99, name: "tester", age: 18}).Execute()`
	_, err := EvaluateQueryNoQuotes(db, input)
	if err != nil {
		t.Fatalf("Parse/Execute failed: %v", err)
	}

	rows, _ := m.ScanTable("users")
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	row := rows[0]
	if row["id"] != int64(99) {
		t.Errorf("id = %v, want 99", row["id"])
	}
	if row["name"] != "tester" {
		t.Errorf("name = %v, want tester", row["name"])
	}
	if row["age"] != int64(18) {
		t.Errorf("age = %v, want 18", row["age"])
	}
}

// TestParserASTCreateTable 测试 CreateTable 的列定义
func TestParserASTCreateTable(t *testing.T) {
	// 验证不同列类型和约束
	m := newMockAdapter()
	db := NewDatabase(m)

	input := `db.Table(items).Create(
		id INTEGER PRIMARY KEY,
		name TEXT NOT NULL,
		price REAL,
		note BLOB NULL
	).Execute()`
	_, err := EvaluateQueryNoQuotes(db, input)
	if err != nil {
		t.Fatalf("CreateTable failed: %v", err)
	}
	if !m.TableExists("items") {
		t.Error("Table items should be created")
	}
}
