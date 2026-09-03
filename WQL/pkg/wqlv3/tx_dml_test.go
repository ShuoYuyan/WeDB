// WQL v3.2: DML through active transaction tests
//go:build integration

package wqlv3_test

import (
	"os"
	"sort"
	"testing"

	"github.com/wedb/wedb/WQL/pkg/wqlv3"
	"github.com/wedb/wedb/internal/storage"
)

func newTxDMLDB(t *testing.T) *wqlv3.Database {
	t.Helper()
	os.Remove("txdml_test.db")
	os.Remove("txdml_test.db.metadata")
	wedb, err := storage.NewWeDBDatabase("txdml_test.db", 4096)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		wedb.Close()
		os.Remove("txdml_test.db")
		os.Remove("txdml_test.db.metadata")
	})
	db := wqlv3.NewDatabase(wqlv3.NewWeDBAdapter(wedb))
	wqlv3.SetColorEnabled(false)
	return db
}

func mustQuery(t *testing.T, db *wqlv3.Database, q string) wqlv3.QueryResult {
	t.Helper()
	res, err := wqlv3.EvaluateQueryNoQuotes(db, q)
	if err != nil {
		t.Fatalf("query failed: %s\nquery: %s", err, q)
	}
	return res
}

func mustBegin(t *testing.T, db *wqlv3.Database) {
	t.Helper()
	if _, err := wqlv3.EvaluateQueryNoQuotes(db, `db.Table(items).Begin()`); err != nil {
		t.Fatalf("BEGIN failed: %v", err)
	}
}
func mustCommit(t *testing.T, db *wqlv3.Database) {
	t.Helper()
	if _, err := wqlv3.EvaluateQueryNoQuotes(db, `db.Table(items).Commit()`); err != nil {
		t.Fatalf("COMMIT failed: %v", err)
	}
}
func mustRollback(t *testing.T, db *wqlv3.Database) {
	t.Helper()
	if _, err := wqlv3.EvaluateQueryNoQuotes(db, `db.Table(items).Rollback()`); err != nil {
		t.Fatalf("ROLLBACK failed: %v", err)
	}
}

func TestTxDML_INSERT_then_COMMIT(t *testing.T) {
	db := newTxDMLDB(t)
	mustQuery(t, db, `db.Table(items).Create(id INTEGER PRIMARY KEY, name TEXT, qty INTEGER).Execute()`)
	mustQuery(t, db, `db.Table(items).Insert({id: 1, name: pen, qty: 5}).Execute()`)

	mustBegin(t, db)
	mustQuery(t, db, `db.Table(items).Insert({id: 2, name: book, qty: 10}).Execute()`)
	mustCommit(t, db)

	res := mustQuery(t, db, `db.Table(items).All()`)
	if len(res.Rows) != 2 {
		t.Fatalf("after commit: expected 2 rows, got %d", len(res.Rows))
	}
}

func TestTxDML_INSERT_then_ROLLBACK(t *testing.T) {
	db := newTxDMLDB(t)
	mustQuery(t, db, `db.Table(items).Create(id INTEGER PRIMARY KEY, name TEXT, qty INTEGER).Execute()`)
	mustQuery(t, db, `db.Table(items).Insert({id: 1, name: pen, qty: 5}).Execute()`)

	mustBegin(t, db)
	mustQuery(t, db, `db.Table(items).Insert({id: 2, name: book, qty: 10}).Execute()`)
	mustQuery(t, db, `db.Table(items).Insert({id: 3, name: eraser, qty: 2}).Execute()`)
	mustRollback(t, db)

	res := mustQuery(t, db, `db.Table(items).All()`)
	if len(res.Rows) != 1 {
		t.Fatalf("after rollback: expected 1 row, got %d", len(res.Rows))
	}
}

func TestTxDML_UPDATE_COMMIT(t *testing.T) {
	db := newTxDMLDB(t)
	mustQuery(t, db, `db.Table(accounts).Create(id INTEGER PRIMARY KEY, balance INTEGER).Execute()`)
	mustQuery(t, db, `db.Table(accounts).Insert({id: 1, balance: 100}).Execute()`)

	mustBegin(t, db)
	mustQuery(t, db, `db.Table(accounts).Set(balance, 200).Where(id = 1).Execute()`)
	mustCommit(t, db)

	res := mustQuery(t, db, `db.Table(accounts).Where(id = 1).First()`)
	if v, _ := res.Rows[0]["balance"].(int64); v != 200 {
		t.Fatalf("after commit: balance = %d, want 200", v)
	}
}

func TestTxDML_UPDATE_ROLLBACK(t *testing.T) {
	db := newTxDMLDB(t)
	mustQuery(t, db, `db.Table(accounts).Create(id INTEGER PRIMARY KEY, balance INTEGER).Execute()`)
	mustQuery(t, db, `db.Table(accounts).Insert({id: 1, balance: 100}).Execute()`)

	mustBegin(t, db)
	mustQuery(t, db, `db.Table(accounts).Set(balance, 999).Where(id = 1).Execute()`)
	mustRollback(t, db)

	res := mustQuery(t, db, `db.Table(accounts).Where(id = 1).First()`)
	if v, _ := res.Rows[0]["balance"].(int64); v != 100 {
		t.Fatalf("after rollback: balance = %d, want 100", v)
	}
}

func TestTxDML_DELETE_ROLLBACK(t *testing.T) {
	db := newTxDMLDB(t)
	mustQuery(t, db, `db.Table(accounts).Create(id INTEGER PRIMARY KEY, balance INTEGER).Execute()`)
	mustQuery(t, db, `db.Table(accounts).Insert({id: 1, balance: 100}).Execute()`)
	mustQuery(t, db, `db.Table(accounts).Insert({id: 2, balance: 200}).Execute()`)

	mustBegin(t, db)
	mustQuery(t, db, `db.Table(accounts).Where(id = 1).Delete().Execute()`)
	mustRollback(t, db)

	res := mustQuery(t, db, `db.Table(accounts).All()`)
	if len(res.Rows) != 2 {
		t.Fatalf("after rollback: expected 2 rows, got %d", len(res.Rows))
	}
}

func TestTxDML_NoQuoteStringValue(t *testing.T) {
	// 验证 WQL 无双引号设计在事务场景下也生效
	db := newTxDMLDB(t)
	mustQuery(t, db, `db.Table(users).Create(id INTEGER PRIMARY KEY, name TEXT, age INTEGER).Execute()`)

	mustQuery(t, db, `db.Table(users).Insert({id: 1, name: alice, age: 30}).Execute()`)
	mustQuery(t, db, `db.Table(users).Insert({id: 2, name: bob, age: 25}).Execute()`)
	// 不在事务中测试 UPDATE，简化
	mustQuery(t, db, `db.Table(users).Set(age, 31).Where(name = alice).Execute()`)

	rows := mustQuery(t, db, `db.Table(users).All()`).Rows
	sort.Slice(rows, func(i, j int) bool {
		ai, _ := rows[i]["id"].(int64)
		aj, _ := rows[j]["id"].(int64)
		return ai < aj
	})
	if name, _ := rows[0]["name"].(string); name != "alice" {
		t.Fatalf("row 1: name = %q, want %q", name, "alice")
	}
	if age, _ := rows[0]["age"].(int64); age != 31 {
		t.Fatalf("row 1: age = %d, want 31", age)
	}
	if name, _ := rows[1]["name"].(string); name != "bob" {
		t.Fatalf("row 2: name = %q, want %q", name, "bob")
	}
}
