// WQL v3.2: CASE WHEN end-to-end smoke test
//go:build integration

package wqlv3_test

import (
	"os"
	"testing"

	"github.com/wedb/wedb/WQL/pkg/wqlv3"
	"github.com/wedb/wedb/internal/storage"
)

func TestCaseWhen_Smoke(t *testing.T) {
	os.Remove("case_test.db")
	os.Remove("case_test.db.metadata")
	wedb, err := storage.NewWeDBDatabase("case_test.db", 4096)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		wedb.Close()
		os.Remove("case_test.db")
		os.Remove("case_test.db.metadata")
	})
	db := wqlv3.NewDatabase(wqlv3.NewWeDBAdapter(wedb))
	wqlv3.SetColorEnabled(false)

	// 至少能创建表、插入、查询
	mustQuery(t, db, `db.Table(orders).Create(id INTEGER PRIMARY KEY, amount REAL).Execute()`)
	mustQuery(t, db, `db.Table(orders).Insert({id: 1, amount: 200}).Execute()`)
	mustQuery(t, db, `db.Table(orders).Insert({id: 2, amount: 60}).Execute()`)
	mustQuery(t, db, `db.Table(orders).Insert({id: 3, amount: 10}).Execute()`)

	rows := mustQuery(t, db, `db.Table(orders).All()`).Rows
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}
}
