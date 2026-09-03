// WQL v3.2: CASE WHEN end-to-end integration test
//go:build integration

package wqlv3_test

import (
	"os"
	"testing"

	"github.com/wedb/wedb/WQL/pkg/wqlv3"
	"github.com/wedb/wedb/internal/storage"
)

func TestCaseWhen_Integration(t *testing.T) {
	os.Remove("case_int.db")
	os.Remove("case_int.db.metadata")
	wedb, err := storage.NewWeDBDatabase("case_int.db", 4096)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		wedb.Close()
		os.Remove("case_int.db")
		os.Remove("case_int.db.metadata")
	})
	db := wqlv3.NewDatabase(wqlv3.NewWeDBAdapter(wedb))
	wqlv3.SetColorEnabled(false)

	mustQuery(t, db, `db.Table(orders).Create(id INTEGER PRIMARY KEY, amount REAL, status TEXT).Execute()`)
	mustQuery(t, db, `db.Table(orders).Insert({id: 1, amount: 200, status: paid}).Execute()`)
	mustQuery(t, db, `db.Table(orders).Insert({id: 2, amount: 60, status: pending}).Execute()`)
	mustQuery(t, db, `db.Table(orders).Insert({id: 3, amount: 10, status: cancelled}).Execute()`)

	// CASE WHEN 投影
	res := mustQuery(t, db, `db.Table(orders).Select(CASE WHEN amount > 100 THEN high WHEN amount > 50 THEN mid ELSE low END).All()`)
	if len(res.Rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(res.Rows))
	}
	// 验证每行都有一个 CASE WHEN 结果列
	for i, row := range res.Rows {
		if len(row) != 1 {
			t.Errorf("row[%d] expected 1 col, got %d", i, len(row))
		}
	}
}
