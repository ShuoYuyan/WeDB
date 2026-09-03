package wqlv3

import (
	"strings"
	"testing"
)

// TestDistinctAll: DISTINCT() on all columns
func TestDistinctAll(t *testing.T) {
	m := newMockAdapter()
	db := NewDatabase(m)
	_ = m.CreateTable(NewTableSchema("users",
		NewColumn("id", "INTEGER", false),
		NewColumn("dept", "TEXT", false),
	))
	_ = m.InsertRows("users", []map[string]interface{}{
		{"id": int64(1), "dept": "Engineering"},
		{"id": int64(2), "dept": "Sales"},
		{"id": int64(3), "dept": "Engineering"},
		{"id": int64(4), "dept": "Sales"},
		{"id": int64(5), "dept": "Marketing"},
	})

	res, err := EvaluateQueryNoQuotes(db, `db.Table(users).Select(dept).Distinct().All()`)
	if err != nil {
		t.Fatalf("DISTINCT() failed: %v", err)
	}
	if len(res.Rows) != 3 {
		t.Errorf("expected 3 distinct depts (Engineering, Sales, Marketing), got %d", len(res.Rows))
	}
}

// TestDistinctOnColumn: DISTINCT(col) on specific column
func TestDistinctOnColumn(t *testing.T) {
	m := newMockAdapter()
	db := NewDatabase(m)
	_ = m.CreateTable(NewTableSchema("orders",
		NewColumn("id", "INTEGER", false),
		NewColumn("user_id", "INTEGER", false),
		NewColumn("product", "TEXT", false),
	))
	_ = m.InsertRows("orders", []map[string]interface{}{
		{"id": int64(1), "user_id": int64(1), "product": "apple"},
		{"id": int64(2), "user_id": int64(1), "product": "banana"},
		{"id": int64(3), "user_id": int64(2), "product": "apple"},
		{"id": int64(4), "user_id": int64(3), "product": "cherry"},
	})

	res, err := EvaluateQueryNoQuotes(db, `db.Table(orders).Select(product).Distinct(product).All()`)
	if err != nil {
		t.Fatalf("DISTINCT(col) failed: %v", err)
	}
	if len(res.Rows) != 3 {
		t.Errorf("expected 3 distinct products, got %d", len(res.Rows))
	}
	products := map[string]bool{}
	for _, r := range res.Rows {
		products[r["product"].(string)] = true
	}
	for _, p := range []string{"apple", "banana", "cherry"} {
		if !products[p] {
			t.Errorf("expected product %q in distinct result", p)
		}
	}
}

// TestTransactionBeginCommit: BEGIN/COMMIT/ROLLBACK operations
func TestTransactionBeginCommit(t *testing.T) {
	m := newMockAdapter()
	db := NewDatabase(m)
	_ = m.CreateTable(NewTableSchema("users",
		NewColumn("id", "INTEGER", false),
		NewColumn("name", "TEXT", false),
	))

	// BEGIN
	_, err := EvaluateQueryNoQuotes(db, `db.Table(users).Begin()`)
	if err != nil {
		t.Fatalf("Begin failed: %v", err)
	}
	if db.CurrentTransaction() == nil {
		t.Fatal("expected active transaction after Begin")
	}
	if !db.CurrentTransaction().IsActive() {
		t.Error("transaction should be active")
	}

	// INSERT
	_, err = EvaluateQueryNoQuotes(db, `db.Table(users).Insert({id: 1, name: alice}).Execute()`)
	if err != nil {
		t.Fatalf("Insert in tx failed: %v", err)
	}

	// COMMIT
	_, err = EvaluateQueryNoQuotes(db, `db.Table(users).Commit()`)
	if err != nil {
		t.Fatalf("Commit failed: %v", err)
	}
	if db.CurrentTransaction() != nil {
		t.Error("expected nil transaction after Commit")
	}
}

func TestTransactionRollback(t *testing.T) {
	m := newMockAdapter()
	db := NewDatabase(m)
	_ = m.CreateTable(NewTableSchema("users",
		NewColumn("id", "INTEGER", false),
	))

	// BEGIN + insert
	_, err := EvaluateQueryNoQuotes(db, `db.Table(users).Begin()`)
	if err != nil {
		t.Fatalf("Begin failed: %v", err)
	}
	_, err = EvaluateQueryNoQuotes(db, `db.Table(users).Insert({id: 1}).Execute()`)
	if err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	// ROLLBACK
	_, err = EvaluateQueryNoQuotes(db, `db.Table(users).Rollback()`)
	if err != nil {
		t.Fatalf("Rollback failed: %v", err)
	}
	if db.CurrentTransaction() != nil {
		t.Error("expected nil transaction after Rollback")
	}
}

func TestCommitWithoutBegin(t *testing.T) {
	m := newMockAdapter()
	db := NewDatabase(m)

	_, err := EvaluateQueryNoQuotes(db, `db.Table(users).Commit()`)
	if err == nil {
		t.Error("expected error for Commit without active transaction")
	}
	if !strings.Contains(err.Error(), "without active transaction") {
		t.Errorf("expected 'without active transaction' error, got: %v", err)
	}
}

// TestNewOperators_AllKnownOperators: comprehensive coverage of all new operators
func TestNewOperators_AllKnownOperators(t *testing.T) {
	m := newMockAdapter()
	db := NewDatabase(m)
	_ = m.CreateTable(NewTableSchema("users",
		NewColumn("id", "INTEGER", false),
		NewColumn("name", "TEXT", false),
		NewColumn("age", "INTEGER", true),
		NewColumn("dept", "TEXT", true),
	))
	_ = m.InsertRows("users", []map[string]interface{}{
		{"id": int64(1), "name": "alice", "age": int64(30), "dept": "Engineering"},
		{"id": int64(2), "name": "bob", "age": int64(25), "dept": "Sales"},
		{"id": int64(3), "name": "carol", "age": int64(35), "dept": "Engineering"},
		{"id": int64(4), "name": "dave", "age": int64(17), "dept": "Sales"},
		{"id": int64(5), "name": "eve", "age": int64(45), "dept": "Marketing"},
		{"id": int64(6), "name": "anna", "age": int64(28), "dept": "Sales"},
	})

	cases := []struct {
		name   string
		query  string
		expect []int64 // expected ids in order
	}{
		{"IN basic", `db.Table(users).Where(id IN (1, 3, 5)).Select(id).All()`, []int64{1, 3, 5}},
		{"NOT IN", `db.Table(users).Where(id NOT IN (1, 2)).Select(id).All()`, []int64{3, 4, 5, 6}},
		{"BETWEEN", `db.Table(users).Where(age BETWEEN 25 AND 35).Select(id).All()`, []int64{1, 2, 3, 6}},
		{"NOT BETWEEN", `db.Table(users).Where(age NOT BETWEEN 25 AND 35).Select(id).All()`, []int64{4, 5}},
		{"LIKE prefix", `db.Table(users).Where(name LIKE a%).Select(id).All()`, []int64{1, 6}},
		{"NOT LIKE", `db.Table(users).Where(name NOT LIKE a%).Select(id).All()`, []int64{2, 3, 4, 5}},
		{"NOT paren", `db.Table(users).Where(NOT(dept = Sales)).Select(id).All()`, []int64{1, 3, 5}},
		{"IS NOT NULL", `db.Table(users).Where(dept IS NOT NULL).Select(id).All()`, []int64{1, 2, 3, 4, 5, 6}},
		{"compound AND", `db.Table(users).Where(dept = Sales AND age > 20).Select(id).All()`, []int64{2, 6}},
		{"compound OR", `db.Table(users).Where(age < 20 OR age > 40).Select(id).All()`, []int64{4, 5}},
		{"compound IN+AND", `db.Table(users).Where(id IN (1, 2, 3) AND age > 20).Select(id).All()`, []int64{1, 2, 3}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			res, err := EvaluateQueryNoQuotes(db, c.query)
			if err != nil {
				t.Fatalf("query %q failed: %v", c.query, err)
			}
			got := make([]int64, 0, len(res.Rows))
			for _, r := range res.Rows {
				if v, ok := r["id"]; ok {
					got = append(got, v.(int64))
				}
			}
			if !int64SlicesEqual(got, c.expect) {
				t.Errorf("%s: got %v, want %v", c.name, got, c.expect)
			}
		})
	}
}

func int64SlicesEqual(a, b []int64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
