//go:build integration

package wqlv3

import (
	"os"
	"strings"
	"testing"

	"github.com/wedb/wedb/internal/storage"
)

// TestIntegration_FullFeatureMatrix exercises the full WQL v3 feature set
// against a real WeDB storage engine. This test is gated by the
// "integration" build tag so it can be run on demand via:
//   go test -tags=integration -count=1 ./pkg/wqlv3/...
func TestIntegration_FullFeatureMatrix(t *testing.T) {
	dbFile := "integration_test.db"
	os.Remove(dbFile)
	os.Remove(dbFile + ".metadata")
	wedb, err := storage.NewWeDBDatabase(dbFile, 4096)
	if err != nil {
		t.Fatalf("NewWeDBDatabase: %v", err)
	}
	defer wedb.Close()
	defer os.Remove(dbFile)
	defer os.Remove(dbFile + ".metadata")

	adapter := NewWeDBAdapter(wedb)
	db := NewDatabase(adapter)
	SetColorEnabled(false)

	// 1. DDL
	ddlStmts := []string{
		`db.Table(users).Create(id INTEGER PRIMARY KEY, name TEXT, age INTEGER, dept TEXT).Execute()`,
		`db.Table(orders).Create(id INTEGER PRIMARY KEY, user_id INTEGER, amount REAL, product TEXT).Execute()`,
		`db.Table(departments).Create(id INTEGER PRIMARY KEY, name TEXT).Execute()`,
	}
	for _, q := range ddlStmts {
		if _, err := EvaluateQueryNoQuotes(db, q); err != nil {
			t.Fatalf("DDL failed: %q -> %v", q, err)
		}
	}

	// 2. INSERT — WQL 无双引号设计：字符串值不需要引号
	inserts := []string{
		`db.Table(departments).Insert({id: 1, name: Engineering}).Execute()`,
		`db.Table(departments).Insert({id: 2, name: Sales}).Execute()`,
		`db.Table(departments).Insert({id: 3, name: Marketing}).Execute()`,
		`db.Table(users).Insert({id: 1, name: alice, age: 30, dept: Engineering}).Execute()`,
		`db.Table(users).Insert({id: 2, name: bob, age: 25, dept: Sales}).Execute()`,
		`db.Table(users).Insert({id: 3, name: carol, age: 35, dept: Engineering}).Execute()`,
		`db.Table(users).Insert({id: 4, name: dave, age: 17, dept: Sales}).Execute()`,
		`db.Table(users).Insert({id: 5, name: eve, age: 45, dept: Marketing}).Execute()`,
		`db.Table(orders).Insert({id: 1, user_id: 1, amount: 100.50, product: laptop}).Execute()`,
		`db.Table(orders).Insert({id: 2, user_id: 1, amount: 20.00, product: mouse}).Execute()`,
		`db.Table(orders).Insert({id: 3, user_id: 2, amount: 50.00, product: keyboard}).Execute()`,
		`db.Table(orders).Insert({id: 4, user_id: 3, amount: 200.00, product: monitor}).Execute()`,
		`db.Table(orders).Insert({id: 5, user_id: 5, amount: 500.00, product: phone}).Execute()`,
	}
	for _, q := range inserts {
		if _, err := EvaluateQueryNoQuotes(db, q); err != nil {
			t.Fatalf("INSERT failed: %q -> %v", q, err)
		}
	}

	// 3. SELECT — 全部使用无双引号字符串字面量
	cases := []struct {
		name   string
		query  string
		expect int // expected row count
	}{
		// 基础
		{"全表扫描", `db.Table(users).All()`, 5},
		{"投影", `db.Table(users).Select(name, age).All()`, 5},
		{"WHERE 等于", `db.Table(users).Where(name = alice).All()`, 1},
		{"WHERE 不等", `db.Table(users).Where(age != 30).All()`, 4},
		{"WHERE 大于", `db.Table(users).Where(age > 30).All()`, 2},
		{"WHERE AND", `db.Table(users).Where(age > 18 AND dept = Sales).All()`, 1},
		{"WHERE OR", `db.Table(users).Where(age < 20 OR age > 40).All()`, 2},
		{"WHERE NOT", `db.Table(users).Where(NOT(dept = Sales)).All()`, 3},
		// 新算符
		{"IN", `db.Table(users).Where(id IN (1, 3, 5)).All()`, 3},
		{"NOT IN", `db.Table(users).Where(id NOT IN (1, 2)).All()`, 3},
		{"BETWEEN", `db.Table(users).Where(age BETWEEN 25 AND 35).All()`, 3},
		{"NOT BETWEEN", `db.Table(users).Where(age NOT BETWEEN 25 AND 35).All()`, 2},
		{"LIKE", `db.Table(users).Where(name LIKE a%).All()`, 1},
		{"NOT LIKE", `db.Table(users).Where(name NOT LIKE a%).All()`, 4},
		{"IS NOT NULL", `db.Table(users).Where(dept IS NOT NULL).All()`, 5},
		{"IS NULL", `db.Table(users).Where(dept IS NULL).All()`, 0},
		// 排序 / 分页
		{"ORDER BY ASC", `db.Table(users).OrderBy(age, ASC).Select(name, age).All()`, 5},
		{"ORDER BY DESC", `db.Table(users).OrderBy(age, DESC).Select(name, age).All()`, 5},
		{"TAKE", `db.Table(users).OrderBy(age, ASC).Take(2).All()`, 2},
		{"SKIP", `db.Table(users).OrderBy(age, ASC).Skip(3).All()`, 2},
		// 聚合
		{"Count", `db.Table(users).Count()`, 5},
		{"Sum amount", `db.Table(orders).Sum(amount)`, 1},
		{"Avg amount", `db.Table(orders).Avg(amount)`, 1},
		{"Min age", `db.Table(users).Min(age)`, 1},
		{"Max age", `db.Table(users).Max(age)`, 1},
		// 分组
		{"GROUP BY", `db.Table(users).GroupBy(dept).Select(dept, Count()).All()`, 3},
		{"GROUP BY + HAVING", `db.Table(users).GroupBy(dept).Having(Count() > 1).Select(dept, Count()).All()`, 2},
		// JOIN
		{"INNER JOIN", `db.Table(users).Join(orders, ON users.id = orders.user_id).Select(name, product, amount).All()`, 5},
		{"LEFT JOIN", `db.Table(users).LeftJoin(orders, ON users.id = orders.user_id).Select(name, product).All()`, 6},
		// DISTINCT
		{"DISTINCT all", `db.Table(users).Select(dept).Distinct().All()`, 3},
		{"DISTINCT col", `db.Table(users).Select(dept).Distinct(dept).All()`, 3},
		// 复合
		{"IN + AND", `db.Table(users).Where(id IN (1, 2, 3) AND age > 20).All()`, 3},
		{"复合 WHERE + ORDER + LIMIT", `db.Table(users).Where(age BETWEEN 20 AND 40).OrderBy(age, DESC).Take(2).All()`, 2},
		// DML
		{"UPDATE", `db.Table(users).Set(age, 31).Where(id = 1).Execute()`, 0}, // 返回 AffectedRows
		{"DELETE", `db.Table(users).Where(age < 18).Delete().Execute()`, 0},
		{"删除后再查", `db.Table(users).All()`, 4}, // dave 已被删
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			res, err := EvaluateQueryNoQuotes(db, c.query)
			if err != nil {
				t.Fatalf("query %q failed: %v", c.query, err)
			}
			if c.query == `db.Table(orders).Sum(amount)` || c.query == `db.Table(orders).Avg(amount)` ||
				c.query == `db.Table(users).Min(age)` || c.query == `db.Table(users).Max(age)` {
				// 聚合函数返回 Value 而非 Rows
				if res.Value == nil {
					t.Errorf("%s: expected value, got nil", c.name)
				}
				return
			}
			if c.query == `db.Table(users).Count()` {
				if res.Value == nil {
					t.Errorf("%s: expected value, got nil", c.name)
				}
				return
			}
			if len(res.Rows) != c.expect {
				t.Errorf("%s: got %d rows, want %d", c.name, len(res.Rows), c.expect)
			}
		})
	}
}

// TestIntegration_TransactionViaAPI: verify the WQL Transaction object can be
// acquired via the WeDBAdapter, and the underlying WeDB transaction is functional.
// (DML builders don't yet route through the active transaction, so this test
// uses the adapter-level API.)
func TestIntegration_TransactionViaAPI(t *testing.T) {
	dbFile := "tx_api_integration_test.db"
	os.Remove(dbFile)
	os.Remove(dbFile + ".metadata")
	wedb, err := storage.NewWeDBDatabase(dbFile, 4096)
	if err != nil {
		t.Fatalf("NewWeDBDatabase: %v", err)
	}
	defer wedb.Close()
	defer os.Remove(dbFile)
	defer os.Remove(dbFile + ".metadata")

	adapter := NewWeDBAdapter(wedb)
	db := NewDatabase(adapter)
	SetColorEnabled(false)

	if _, err := EvaluateQueryNoQuotes(db, `db.Table(accounts).Create(id INTEGER PRIMARY KEY, balance INTEGER).Execute()`); err != nil {
		t.Fatalf("CreateTable: %v", err)
	}
	if _, err := EvaluateQueryNoQuotes(db, `db.Table(accounts).Insert({id: 1, balance: 100}).Execute()`); err != nil {
		t.Fatalf("Insert: %v", err)
	}

	// WQL-level BEGIN/COMMIT：能执行语句，但 DML 自动路由到当前事务尚未实现。
	// 本测试仅验证语句不报错、currentTx 状态正确。
	if _, err := EvaluateQueryNoQuotes(db, `db.Table(accounts).Begin()`); err != nil {
		t.Fatalf("Begin: %v", err)
	}
	if db.CurrentTransaction() == nil || !db.CurrentTransaction().IsActive() {
		t.Error("expected active transaction after Begin")
	}
	if _, err := EvaluateQueryNoQuotes(db, `db.Table(accounts).Commit()`); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	if db.CurrentTransaction() != nil {
		t.Error("expected nil transaction after Commit")
	}

	// ROLLBACK
	if _, err := EvaluateQueryNoQuotes(db, `db.Table(accounts).Begin()`); err != nil {
		t.Fatalf("Begin2: %v", err)
	}
	if _, err := EvaluateQueryNoQuotes(db, `db.Table(accounts).Rollback()`); err != nil {
		t.Fatalf("Rollback: %v", err)
	}
	if db.CurrentTransaction() != nil {
		t.Error("expected nil transaction after Rollback")
	}
}

// TestIntegration_DDLOrder: DROP TABLE + CREATE TABLE roundtrip
func TestIntegration_DDLOrder(t *testing.T) {
	dbFile := "ddl_integration_test.db"
	os.Remove(dbFile)
	os.Remove(dbFile + ".metadata")
	wedb, _ := storage.NewWeDBDatabase(dbFile, 4096)
	defer wedb.Close()
	defer os.Remove(dbFile)
	defer os.Remove(dbFile + ".metadata")

	adapter := NewWeDBAdapter(wedb)
	db := NewDatabase(adapter)
	SetColorEnabled(false)

	if _, err := EvaluateQueryNoQuotes(db, `db.Table(temp).Create(id INTEGER PRIMARY KEY, name TEXT).Execute()`); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := EvaluateQueryNoQuotes(db, `db.Table(temp).Insert({id: 1, name: x}).Execute()`); err != nil {
		t.Fatalf("Insert: %v", err)
	}
	if _, err := EvaluateQueryNoQuotes(db, `db.Table(temp).Drop().Execute()`); err != nil {
		t.Fatalf("Drop: %v", err)
	}
	// 删除后查询应失败
	_, err := EvaluateQueryNoQuotes(db, `db.Table(temp).All()`)
	if err == nil {
		t.Error("expected error querying dropped table")
	} else if !strings.Contains(err.Error(), "does not exist") && !strings.Contains(err.Error(), "not found") {
		t.Logf("error: %v (acceptable)", err)
	}
}
