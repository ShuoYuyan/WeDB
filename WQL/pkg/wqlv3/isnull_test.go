// WQL v3.2: IS NULL / IS NOT NULL in-memory WHERE tests
//go:build integration

package wqlv3_test

import (
	"os"
	"testing"

	"github.com/wedb/wedb/WQL/pkg/wqlv3"
	"github.com/wedb/wedb/internal/storage"
)

func TestIsNull_InMemoryWHERE(t *testing.T) {
	os.Remove("isnull_test.db")
	os.Remove("isnull_test.db.metadata")
	wedb, err := storage.NewWeDBDatabase("isnull_test.db", 4096)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		wedb.Close()
		os.Remove("isnull_test.db")
		os.Remove("isnull_test.db.metadata")
	})
	db := wqlv3.NewDatabase(wqlv3.NewWeDBAdapter(wedb))
	wqlv3.SetColorEnabled(false)

	mustQuery(t, db, `db.Table(people).Create(id INTEGER PRIMARY KEY, name TEXT, age INTEGER).Execute()`)
	mustQuery(t, db, `db.Table(people).Insert({id: 1, name: alice, age: 30}).Execute()`)
	mustQuery(t, db, `db.Table(people).Insert({id: 2, name: bob, age: 25}).Execute()`)
	mustQuery(t, db, `db.Table(people).Insert({id: 3, name: carol, age: 35}).Execute()`)

	// IS NULL — 全部都没有 NULL 值（在列上是 INTEGER/TEXT）
	// 所以 is null 应该返回 0 行
	rows := mustQuery(t, db, `db.Table(people).Where(age IS NULL).All()`).Rows
	if len(rows) != 0 {
		t.Fatalf("IS NULL: expected 0 rows, got %d", len(rows))
	}

	// IS NOT NULL — 应该返回全部 3 行
	rows = mustQuery(t, db, `db.Table(people).Where(age IS NOT NULL).All()`).Rows
	if len(rows) != 3 {
		t.Fatalf("IS NOT NULL: expected 3 rows, got %d", len(rows))
	}

	// 名字字段同理
	rows = mustQuery(t, db, `db.Table(people).Where(name IS NOT NULL).All()`).Rows
	if len(rows) != 3 {
		t.Fatalf("name IS NOT NULL: expected 3 rows, got %d", len(rows))
	}

	// 组合：age IS NOT NULL AND name = alice
	rows = mustQuery(t, db, `db.Table(people).Where(age IS NOT NULL AND name = alice).All()`).Rows
	if len(rows) != 1 {
		t.Fatalf("IS NOT NULL AND name = alice: expected 1 row, got %d", len(rows))
	}
}
