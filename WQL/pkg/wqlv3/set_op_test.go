// WQL v3.2: UNION / INTERSECT / EXCEPT integration tests
//go:build integration

package wqlv3_test

import (
	"os"
	"sort"
	"testing"

	"github.com/wedb/wedb/WQL/pkg/wqlv3"
	"github.com/wedb/wedb/internal/storage"
)

func newSetOpDB(t *testing.T) *wqlv3.Database {
	t.Helper()
	os.Remove("setop_test.db")
	os.Remove("setop_test.db.metadata")
	wedb, err := storage.NewWeDBDatabase("setop_test.db", 4096)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		wedb.Close()
		os.Remove("setop_test.db")
		os.Remove("setop_test.db.metadata")
	})
	adapter := wqlv3.NewWeDBAdapter(wedb)
	db := wqlv3.NewDatabase(adapter)
	wqlv3.SetColorEnabled(false)

	mustExec(t, db, `db.Table(employeesA).Create(id INTEGER PRIMARY KEY, name TEXT, dept TEXT).Execute()`)
	mustExec(t, db, `db.Table(employeesB).Create(id INTEGER PRIMARY KEY, name TEXT, dept TEXT).Execute()`)

	mustExec(t, db, `db.Table(employeesA).Insert({id: 1, name: alice, dept: eng}).Execute()`)
	mustExec(t, db, `db.Table(employeesA).Insert({id: 2, name: bob,   dept: sales}).Execute()`)
	mustExec(t, db, `db.Table(employeesA).Insert({id: 3, name: carol, dept: eng}).Execute()`)
	mustExec(t, db, `db.Table(employeesB).Insert({id: 2, name: bob,   dept: sales}).Execute()`)
	mustExec(t, db, `db.Table(employeesB).Insert({id: 3, name: carol, dept: eng}).Execute()`)
	mustExec(t, db, `db.Table(employeesB).Insert({id: 4, name: dave,  dept: hr}).Execute()`)
	return db
}

func mustExec(t *testing.T, db *wqlv3.Database, q string) {
	t.Helper()
	if _, err := wqlv3.EvaluateQueryNoQuotes(db, q); err != nil {
		t.Fatalf("exec failed: %s\nquery: %s", err, q)
	}
}

func runQuery(t *testing.T, db *wqlv3.Database, q string) []map[string]interface{} {
	t.Helper()
	res, err := wqlv3.EvaluateQueryNoQuotes(db, q)
	if err != nil {
		t.Fatalf("query failed: %s\nquery: %s", err, q)
	}
	return res.Rows
}

func sortedIDs(rows []map[string]interface{}) []int64 {
	out := make([]int64, 0, len(rows))
	for _, r := range rows {
		if v, ok := r["id"]; ok {
			switch n := v.(type) {
			case int64:
				out = append(out, n)
			case int:
				out = append(out, int64(n))
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func TestSetOp_UNION(t *testing.T) {
	db := newSetOpDB(t)
	rows := runQuery(t, db, `db.Table(employeesA).Select(id, name).Union(db.Table(employeesB).Select(id, name)).All()`)
	got := sortedIDs(rows)
	want := []int64{1, 2, 3, 4}
	if len(got) != len(want) {
		t.Fatalf("UNION: got %v, want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("UNION[%d]: got %d, want %d", i, got[i], want[i])
		}
	}
}

func TestSetOp_UNION_ALL(t *testing.T) {
	db := newSetOpDB(t)
	rows := runQuery(t, db, `db.Table(employeesA).Select(id, name).UnionAll(db.Table(employeesB).Select(id, name)).All()`)
	got := sortedIDs(rows)
	want := []int64{1, 2, 2, 3, 3, 4}
	if len(got) != len(want) {
		t.Fatalf("UNION ALL: got %v (%d), want %v (%d)", got, len(got), want, len(want))
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("UNION ALL[%d]: got %d, want %d", i, got[i], want[i])
		}
	}
}

func TestSetOp_INTERSECT(t *testing.T) {
	db := newSetOpDB(t)
	rows := runQuery(t, db, `db.Table(employeesA).Select(id, name).Intersect(db.Table(employeesB).Select(id, name)).All()`)
	got := sortedIDs(rows)
	want := []int64{2, 3}
	if len(got) != len(want) {
		t.Fatalf("INTERSECT: got %v, want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("INTERSECT[%d]: got %d, want %d", i, got[i], want[i])
		}
	}
}

func TestSetOp_EXCEPT(t *testing.T) {
	db := newSetOpDB(t)
	rows := runQuery(t, db, `db.Table(employeesA).Select(id, name).Except(db.Table(employeesB).Select(id, name)).All()`)
	got := sortedIDs(rows)
	want := []int64{1}
	if len(got) != len(want) {
		t.Fatalf("EXCEPT: got %v, want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("EXCEPT[%d]: got %d, want %d", i, got[i], want[i])
		}
	}
}

func TestSetOp_ChainedUnion(t *testing.T) {
	os.Remove("setop_chain.db")
	os.Remove("setop_chain.db.metadata")
	wedb, _ := storage.NewWeDBDatabase("setop_chain.db", 4096)
	t.Cleanup(func() {
		wedb.Close()
		os.Remove("setop_chain.db")
		os.Remove("setop_chain.db.metadata")
	})
	db := wqlv3.NewDatabase(wqlv3.NewWeDBAdapter(wedb))
	wqlv3.SetColorEnabled(false)
	mustExec(t, db, `db.Table(t1).Create(id INTEGER PRIMARY KEY, v TEXT).Execute()`)
	mustExec(t, db, `db.Table(t2).Create(id INTEGER PRIMARY KEY, v TEXT).Execute()`)
	mustExec(t, db, `db.Table(t3).Create(id INTEGER PRIMARY KEY, v TEXT).Execute()`)
	mustExec(t, db, `db.Table(t1).Insert({id: 1, v: a}).Execute()`)
	mustExec(t, db, `db.Table(t2).Insert({id: 2, v: b}).Execute()`)
	mustExec(t, db, `db.Table(t3).Insert({id: 3, v: c}).Execute()`)

	rows := runQuery(t, db, `db.Table(t1).Select(id).Union(db.Table(t2).Select(id)).Union(db.Table(t3).Select(id)).All()`)
	got := sortedIDs(rows)
	want := []int64{1, 2, 3}
	if len(got) != len(want) {
		t.Fatalf("chained UNION: got %v, want %v", got, want)
	}
}
