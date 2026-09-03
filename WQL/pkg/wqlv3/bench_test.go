//go:build benchmark

package wqlv3

import (
	"fmt"
	"os"
	"testing"

	"github.com/wedb/wedb/internal/storage"
)

// BenchmarkPushdownVsInMemory measures the difference between query
// pushdown to the storage engine vs full in-memory filtering.
// Run with: go test -tags=benchmark -bench=. -benchtime=10x -run=^$ ./pkg/wqlv3/...
func BenchmarkPushdownVsInMemory(b *testing.B) {
	dbFile := "bench_wql.db"
	os.Remove(dbFile)
	os.Remove(dbFile + ".metadata")
	defer os.Remove(dbFile)
	defer os.Remove(dbFile + ".metadata")

	wedb, err := storage.NewWeDBDatabase(dbFile, 4096)
	if err != nil {
		b.Fatal(err)
	}
	adapter := NewWeDBAdapter(wedb)
	db := NewDatabase(adapter)
	SetColorEnabled(false)

	setup := []string{
		`db.Table(users).Create(id INTEGER PRIMARY KEY, name TEXT, age INTEGER, dept TEXT).Execute()`,
		`db.Table(orders).Create(id INTEGER PRIMARY KEY, user_id INTEGER, amount REAL, product TEXT).Execute()`,
	}
	for _, q := range setup {
		if _, err := EvaluateQueryNoQuotes(db, q); err != nil {
			b.Fatal(err)
		}
	}
	for i := 1; i <= 10000; i++ {
		dept := []string{"Engineering", "Sales", "Marketing", "HR"}[i%4]
		_, err := EvaluateQueryNoQuotes(db, fmt.Sprintf(
			`db.Table(users).Insert({id: %d, name: user_%d, age: %d, dept: %s}).Execute()`,
			i, i, 20+(i%50), dept))
		if err != nil {
			b.Fatal(err)
		}
	}
	for i := 1; i <= 5000; i++ {
		_, err := EvaluateQueryNoQuotes(db, fmt.Sprintf(
			`db.Table(orders).Insert({id: %d, user_id: %d, amount: %f, product: p%d}).Execute()`,
			i, 1+(i%10000), float64(i)*1.5, i))
		if err != nil {
			b.Fatal(err)
		}
	}
	wedb.Close()

	benchmarks := []struct {
		name  string
		query string
	}{
		{"Pushdown_IN", `db.Table(users).Where(id IN (1, 100, 500, 1000, 5000, 9999)).All()`},
		{"Pushdown_BETWEEN", `db.Table(users).Where(age BETWEEN 25 AND 35).All()`},
		{"Pushdown_LIKE", `db.Table(users).Where(name LIKE user_5%).All()`},
		{"Pushdown_FullScan", `db.Table(users).All()`},
		{"Pushdown_TopN", `db.Table(users).OrderBy(age, DESC).Take(10).All()`},
		{"InMemory_GROUP_BY", `db.Table(users).GroupBy(dept).Select(dept, Count()).All()`},
		{"InMemory_JOIN", `db.Table(users).Join(orders, ON users.id = orders.user_id).All()`},
		{"Pushdown_Count", `db.Table(users).Count()`},
		{"Pushdown_Sum", `db.Table(orders).Sum(amount)`},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			wdb, err := storage.NewWeDBDatabase(dbFile, 4096)
			if err != nil {
				b.Fatal(err)
			}
			defer wdb.Close()
			adapter := NewWeDBAdapter(wdb)
			d := NewDatabase(adapter)
			SetColorEnabled(false)

			// 预热：跑一次让缓存预热
			_, _ = EvaluateQueryNoQuotes(d, bm.query)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := EvaluateQueryNoQuotes(d, bm.query)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
