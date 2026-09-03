package wqlv3

import "testing"

// TestTerminalCount: Count() as a terminal operation
func TestTerminalCount(t *testing.T) {
	m := newMockAdapter()
	db := NewDatabase(m)
	_ = m.CreateTable(NewTableSchema("users",
		NewColumn("id", "INTEGER", false),
		NewColumn("age", "INTEGER", true),
	))
	_ = m.InsertRows("users", []map[string]interface{}{
		{"id": int64(1), "age": int64(20)},
		{"id": int64(2), "age": int64(30)},
		{"id": int64(3), "age": int64(40)},
	})

	res, err := EvaluateQueryNoQuotes(db, `db.Table(users).Count()`)
	if err != nil {
		t.Fatalf("Count() failed: %v", err)
	}
	if res.Value == nil {
		t.Fatalf("expected non-nil Value for Count()")
	}
	if got, ok := res.Value.(int64); !ok || got != 3 {
		t.Errorf("Count() = %v (type %T), want int64(3)", res.Value, res.Value)
	}
}

func TestTerminalCountWithWhere(t *testing.T) {
	m := newMockAdapter()
	db := NewDatabase(m)
	_ = m.CreateTable(NewTableSchema("users",
		NewColumn("id", "INTEGER", false),
		NewColumn("age", "INTEGER", true),
	))
	_ = m.InsertRows("users", []map[string]interface{}{
		{"id": int64(1), "age": int64(20)},
		{"id": int64(2), "age": int64(30)},
		{"id": int64(3), "age": int64(40)},
	})

	res, err := EvaluateQueryNoQuotes(db, `db.Table(users).Where(age > 25).Count()`)
	if err != nil {
		t.Fatalf("Count() with WHERE failed: %v", err)
	}
	if got, ok := res.Value.(int64); !ok || got != 2 {
		t.Errorf("Count(age>25) = %v, want 2", res.Value)
	}
}

func TestTerminalSum(t *testing.T) {
	m := newMockAdapter()
	db := NewDatabase(m)
	_ = m.CreateTable(NewTableSchema("orders",
		NewColumn("id", "INTEGER", false),
		NewColumn("amount", "REAL", true),
	))
	_ = m.InsertRows("orders", []map[string]interface{}{
		{"id": int64(1), "amount": 10.0},
		{"id": int64(2), "amount": 20.0},
		{"id": int64(3), "amount": 30.0},
	})

	res, err := EvaluateQueryNoQuotes(db, `db.Table(orders).Sum(amount)`)
	if err != nil {
		t.Fatalf("Sum() failed: %v", err)
	}
	if got, ok := res.Value.(float64); !ok || got != 60.0 {
		t.Errorf("Sum(amount) = %v (type %T), want 60.0", res.Value, res.Value)
	}
}

func TestTerminalAvg(t *testing.T) {
	m := newMockAdapter()
	db := NewDatabase(m)
	_ = m.CreateTable(NewTableSchema("orders",
		NewColumn("id", "INTEGER", false),
		NewColumn("amount", "REAL", true),
	))
	_ = m.InsertRows("orders", []map[string]interface{}{
		{"id": int64(1), "amount": 10.0},
		{"id": int64(2), "amount": 20.0},
		{"id": int64(3), "amount": 30.0},
	})

	res, err := EvaluateQueryNoQuotes(db, `db.Table(orders).Avg(amount)`)
	if err != nil {
		t.Fatalf("Avg() failed: %v", err)
	}
	if got, ok := res.Value.(float64); !ok || got != 20.0 {
		t.Errorf("Avg(amount) = %v (type %T), want 20.0", res.Value, res.Value)
	}
}

func TestTerminalMin(t *testing.T) {
	m := newMockAdapter()
	db := NewDatabase(m)
	_ = m.CreateTable(NewTableSchema("orders",
		NewColumn("id", "INTEGER", false),
		NewColumn("amount", "REAL", true),
	))
	_ = m.InsertRows("orders", []map[string]interface{}{
		{"id": int64(1), "amount": 50.0},
		{"id": int64(2), "amount": 10.0},
		{"id": int64(3), "amount": 30.0},
	})

	res, err := EvaluateQueryNoQuotes(db, `db.Table(orders).Min(amount)`)
	if err != nil {
		t.Fatalf("Min() failed: %v", err)
	}
	if res.Value == nil {
		t.Errorf("expected non-nil Min value")
	}
}

func TestTerminalMax(t *testing.T) {
	m := newMockAdapter()
	db := NewDatabase(m)
	_ = m.CreateTable(NewTableSchema("orders",
		NewColumn("id", "INTEGER", false),
		NewColumn("amount", "REAL", true),
	))
	_ = m.InsertRows("orders", []map[string]interface{}{
		{"id": int64(1), "amount": 50.0},
		{"id": int64(2), "amount": 10.0},
		{"id": int64(3), "amount": 30.0},
	})

	res, err := EvaluateQueryNoQuotes(db, `db.Table(orders).Max(amount)`)
	if err != nil {
		t.Fatalf("Max() failed: %v", err)
	}
	if res.Value == nil {
		t.Errorf("expected non-nil Max value")
	}
}

// TestTerminalCountWithAS: Count() AS alias should work
func TestTerminalCountWithAS(t *testing.T) {
	m := newMockAdapter()
	db := NewDatabase(m)
	_ = m.CreateTable(NewTableSchema("users",
		NewColumn("id", "INTEGER", false),
	))
	_ = m.InsertRows("users", []map[string]interface{}{{"id": int64(1)}})

	res, err := EvaluateQueryNoQuotes(db, `db.Table(users).Count() AS total`)
	if err != nil {
		t.Fatalf("Count() AS alias failed: %v", err)
	}
	_ = res
}

// TestUpdateSetWhere: Set().Where() roundtrip
func TestUpdateSetWhere(t *testing.T) {
	m := newMockAdapter()
	db := NewDatabase(m)
	_ = m.CreateTable(NewTableSchema("users",
		NewColumn("id", "INTEGER", false),
		NewColumn("name", "TEXT", false),
		NewColumn("age", "INTEGER", true),
	))
	_ = m.InsertRows("users", []map[string]interface{}{
		{"id": int64(1), "name": "alice", "age": int64(30)},
	})

	_, err := EvaluateQueryNoQuotes(db, `db.Table(users).Set(age, 35).Where(name = alice).Execute()`)
	if err != nil {
		t.Fatalf("Set().Where() failed: %v", err)
	}
	rows, _ := m.ScanTable("users")
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0]["age"].(int64) != 35 {
		t.Errorf("expected age=35 after Set, got %v", rows[0]["age"])
	}
}
