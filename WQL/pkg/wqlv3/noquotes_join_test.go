package wqlv3

import (
	"testing"
)

// joinTestDB creates a test DB with users + orders tables
func joinTestDB(t *testing.T) (*Database, *mockAdapter) {
	m := newMockAdapter()
	_ = m.CreateTable(NewTableSchema("users",
		NewColumn("id", "INTEGER", false),
		NewColumn("name", "TEXT", false),
	))
	_ = m.CreateTable(NewTableSchema("orders",
		NewColumn("id", "INTEGER", false),
		NewColumn("user_id", "INTEGER", false),
		NewColumn("product", "TEXT", false),
		NewColumn("amount", "INTEGER", true),
	))
	_ = m.InsertRows("users", []map[string]interface{}{
		{"id": int64(1), "name": "alice"},
		{"id": int64(2), "name": "bob"},
		{"id": int64(3), "name": "carol"},
	})
	_ = m.InsertRows("orders", []map[string]interface{}{
		{"id": int64(100), "user_id": int64(1), "product": "apple", "amount": int64(10)},
		{"id": int64(101), "user_id": int64(1), "product": "banana", "amount": int64(5)},
		{"id": int64(102), "user_id": int64(2), "product": "cherry", "amount": int64(20)},
		// user_id=99 does not exist in users (for LEFT JOIN testing)
	})
	db := NewDatabase(m)
	return db, m
}

// TestJoinInnerON tests INNER JOIN with ON condition
func TestJoinInnerON(t *testing.T) {
	db, _ := joinTestDB(t)

	res, err := EvaluateQueryNoQuotes(db, `
		db.Table(users).Join(orders, ON users.id = orders.user_id).All()
	`)
	if err != nil {
		t.Fatalf("INNER JOIN failed: %v", err)
	}
	if len(res.Rows) == 0 {
		t.Errorf("expected 3 rows, got 0")
	}

	// 3 matches: alice x2 + bob x1
	if len(res.Rows) != 3 {
		t.Errorf("expected 3 rows, got %d (%+v)", len(res.Rows), res.Rows)
	}
	for _, row := range res.Rows {
		if _, hasName := row["name"]; !hasName {
			t.Errorf("missing 'name' in joined row: %+v", row)
		}
		if _, hasProduct := row["product"]; !hasProduct {
			t.Errorf("missing 'product' in joined row: %+v", row)
		}
	}
}

// TestJoinLeftKeepUnmatched tests LEFT JOIN keeps unmatched left rows
func TestJoinLeftKeepUnmatched(t *testing.T) {
	db, _ := joinTestDB(t)

	res, err := EvaluateQueryNoQuotes(db, `
		db.Table(users).LeftJoin(orders, ON users.id = orders.user_id).All()
	`)
	if err != nil {
		t.Fatalf("LEFT JOIN failed: %v", err)
	}

	// 4 rows: alice x2 + bob x1 + carol x1 (unmatched, product/amount are nil)
	if len(res.Rows) != 4 {
		t.Errorf("expected 4 rows, got %d", len(res.Rows))
	}
	foundCarol := false
	for _, row := range res.Rows {
		if row["name"] == "carol" {
			foundCarol = true
			if row["product"] != nil {
				t.Errorf("carol should have nil product, got %v", row["product"])
			}
		}
	}
	if !foundCarol {
		t.Error("carol (unmatched) missing from LEFT JOIN result")
	}
}

// TestJoinWithWhere tests JOIN + WHERE combination
func TestJoinWithWhere(t *testing.T) {
	db, _ := joinTestDB(t)

	res, err := EvaluateQueryNoQuotes(db, `
		db.Table(users).Join(orders, ON users.id = orders.user_id).
		Where(amount > 5).All()
	`)
	if err != nil {
		t.Fatalf("JOIN+WHERE failed: %v", err)
	}

	// amount > 5: apple(10), cherry(20) -> 2 rows
	if len(res.Rows) != 2 {
		t.Errorf("expected 2 rows, got %d", len(res.Rows))
	}
}

// TestJoinWithSelect tests JOIN + column selection
func TestJoinWithSelect(t *testing.T) {
	db, _ := joinTestDB(t)

	res, err := EvaluateQueryNoQuotes(db, `
		db.Table(users).Select(name, product).Join(orders, ON users.id = orders.user_id).All()
	`)
	if err != nil {
		t.Fatalf("JOIN+SELECT failed: %v", err)
	}

	if len(res.Rows) != 3 {
		t.Errorf("expected 3 rows, got %d", len(res.Rows))
	}
	for _, row := range res.Rows {
		if _, hasId := row["id"]; hasId {
			t.Errorf("'id' should not be in Select(name, product) result: %+v", row)
		}
		if _, hasName := row["name"]; !hasName {
			t.Errorf("missing 'name': %+v", row)
		}
	}
}

// TestJoinMultiChain tests chained multi-table joins
func TestJoinMultiChain(t *testing.T) {
	m := newMockAdapter()
	_ = m.CreateTable(NewTableSchema("a", NewColumn("id", "INTEGER", false)))
	_ = m.CreateTable(NewTableSchema("b", NewColumn("id", "INTEGER", false), NewColumn("a_id", "INTEGER", false)))
	_ = m.CreateTable(NewTableSchema("c", NewColumn("id", "INTEGER", false), NewColumn("b_id", "INTEGER", false)))
	_ = m.InsertRows("a", []map[string]interface{}{{"id": int64(1)}, {"id": int64(2)}})
	_ = m.InsertRows("b", []map[string]interface{}{
		{"id": int64(10), "a_id": int64(1)},
		{"id": int64(11), "a_id": int64(2)},
	})
	_ = m.InsertRows("c", []map[string]interface{}{
		{"id": int64(100), "b_id": int64(10)},
		{"id": int64(101), "b_id": int64(11)},
	})

	db := NewDatabase(m)
	res, err := EvaluateQueryNoQuotes(db, `
		db.Table(a).Join(b, ON a.id = b.a_id).Join(c, ON b.id = c.b_id).All()
	`)
	if err != nil {
		t.Fatalf("Multi JOIN failed: %v", err)
	}
	if len(res.Rows) != 2 {
		t.Errorf("expected 2 rows from 3-way join, got %d", len(res.Rows))
	}
}

// TestJoinAST exercises JOIN AST parsing
func TestJoinAST(t *testing.T) {
	db, _ := joinTestDB(t)
	_, err := EvaluateQueryNoQuotes(db, `
		db.Table(users).Join(orders, ON users.id = orders.user_id).All()
	`)
	if err != nil {
		t.Fatalf("JOIN parse/exec failed: %v", err)
	}
}

// TestJoinEmptyRightTable tests behavior with empty right table
func TestJoinEmptyRightTable(t *testing.T) {
	m := newMockAdapter()
	_ = m.CreateTable(NewTableSchema("left_t", NewColumn("id", "INTEGER", false)))
	_ = m.CreateTable(NewTableSchema("right_t", NewColumn("id", "INTEGER", false)))
	_ = m.InsertRows("left_t", []map[string]interface{}{{"id": int64(1)}})
	// right_t intentionally empty

	db := NewDatabase(m)
	res, err := EvaluateQueryNoQuotes(db, `
		db.Table(left_t).LeftJoin(right_t, ON left_t.id = right_t.id).All()
	`)
	if err != nil {
		t.Fatalf("LEFT JOIN empty right failed: %v", err)
	}
	// 1 row: left row + nil right fields
	if len(res.Rows) != 1 {
		t.Errorf("expected 1 row, got %d", len(res.Rows))
	}
}
