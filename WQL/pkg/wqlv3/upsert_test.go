// WQL v3.2: UPSERT / ON CONFLICT tests
//go:build integration

package wqlv3_test

import (
	"os"
	"testing"

	"github.com/wedb/wedb/WQL/pkg/wqlv3"
	"github.com/wedb/wedb/internal/storage"
)

func TestUpsert_OnConflict_Update(t *testing.T) {
	os.Remove("upsert.db")
	os.Remove("upsert.db.metadata")
	wedb, err := storage.NewWeDBDatabase("upsert.db", 4096)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		wedb.Close()
		os.Remove("upsert.db")
		os.Remove("upsert.db.metadata")
	})
	db := wqlv3.NewDatabase(wqlv3.NewWeDBAdapter(wedb))
	wqlv3.SetColorEnabled(false)

	mustQuery(t, db, `db.Table(products).Create(id INTEGER PRIMARY KEY, name TEXT, price REAL).Execute()`)
	mustQuery(t, db, `db.Table(products).Insert({id: 1, name: apple, price: 1.5}).Execute()`)
	mustQuery(t, db, `db.Table(products).Insert({id: 2, name: banana, price: 0.8}).Execute()`)

	// UPSERT: 重复 id=1 改为更新
	mustQuery(t, db, `db.Table(products).Insert({id: 1, name: apple, price: 2.0}).OnConflict(UPDATE, id).Execute()`)

	rows := mustQuery(t, db, `db.Table(products).All()`).Rows
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	// 找到 id=1
	for _, r := range rows {
		if r["id"] == int64(1) {
			if r["price"] != 2.0 {
				t.Errorf("expected price=2.0 after upsert, got %v", r["price"])
			}
		}
	}
}

func TestUpsert_OnConflict_Insert(t *testing.T) {
	os.Remove("upsert2.db")
	os.Remove("upsert2.db.metadata")
	wedb, err := storage.NewWeDBDatabase("upsert2.db", 4096)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		wedb.Close()
		os.Remove("upsert2.db")
		os.Remove("upsert2.db.metadata")
	})
	db := wqlv3.NewDatabase(wqlv3.NewWeDBAdapter(wedb))
	wqlv3.SetColorEnabled(false)

	mustQuery(t, db, `db.Table(items).Create(id INTEGER PRIMARY KEY, name TEXT).Execute()`)
	mustQuery(t, db, `db.Table(items).Insert({id: 1, name: first}).Execute()`)

	// 不带 OnConflict — 直接插入会失败（重复主键）
	_, err = mustQueryE(t, db, `db.Table(items).Insert({id: 1, name: dup}).Execute()`)
	if err == nil {
		t.Error("expected error on duplicate insert without OnConflict")
	}
}

// mustQueryE runs a query and returns (result, error) without failing the test on error
func mustQueryE(t *testing.T, db *wqlv3.Database, q string) (wqlv3.QueryResult, error) {
	t.Helper()
	return wqlv3.EvaluateQueryNoQuotes(db, q)
}
