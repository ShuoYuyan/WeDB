package storage

import (
	"fmt"
	"sync"
	"testing"

	"github.com/wedb/wedb/internal/api"
)

// TestParallelWritersDifferentTables 验证核心卖点：不同表的写完全并行且零丢失。
func TestParallelWritersDifferentTables(t *testing.T) {
	db, err := NewWeDBDatabase("test_par_writer.db", 4096)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { db.Close() }()

	const (
		numTables = 4
		rowsEach  = 150
	)

	for i := 0; i < numTables; i++ {
		tbl := fmt.Sprintf("t%d", i)
		if err := db.CreateTable(&api.TableSchema{
			TableName: tbl,
			Columns: []api.ColumnSchema{
				{Name: "id", Type: api.TypeInteger, PrimaryKey: true},
				{Name: "v", Type: api.TypeText},
			},
		}); err != nil {
			t.Fatalf("create %s: %v", tbl, err)
		}
	}

	var wg sync.WaitGroup
	for ti := 0; ti < numTables; ti++ {
		wg.Add(1)
		go func(ti int) {
			defer wg.Done()
			tbl := fmt.Sprintf("t%d", ti)
			for r := 1; r <= rowsEach; r++ {
				row := map[string]interface{}{
					"id": int64(r),
					"v":  fmt.Sprintf("t%d-row%04d", ti, r),
				}
				if err := db.InsertRow(tbl, row); err != nil {
					t.Errorf("insert %s/%d: %v", tbl, r, err)
					return
				}
			}
		}(ti)
	}
	wg.Wait()

	for i := 0; i < numTables; i++ {
		tbl := fmt.Sprintf("t%d", i)
		n, err := db.Count(tbl, "")
		if err != nil {
			t.Fatalf("count %s: %v", tbl, err)
		}
		if n != rowsEach {
			t.Fatalf("table %s: expected %d rows, got %d", tbl, rowsEach, n)
		}
	}
}

// TestConcurrentReadersWhileWriting 读不被跨表写阻塞（功能性验证：读者全程可见
// 行数单调不减，且最终读到全部数据）。
func TestConcurrentReadersWhileWriting(t *testing.T) {
	db, err := NewWeDBDatabase("test_par_mixed.db", 4096)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { db.Close() }()

	if err := db.CreateTable(&api.TableSchema{
		TableName: "src",
		Columns: []api.ColumnSchema{
			{Name: "id", Type: api.TypeInteger, PrimaryKey: true},
		},
	}); err != nil {
		t.Fatal(err)
	}
	if err := db.CreateTable(&api.TableSchema{
		TableName: "other",
		Columns: []api.ColumnSchema{
			{Name: "id", Type: api.TypeInteger, PrimaryKey: true},
		},
	}); err != nil {
		t.Fatal(err)
	}

	const total = 300
	var wg sync.WaitGroup
	stop := make(chan struct{})
	var lastSeen int64 = -1
	var mu sync.Mutex

	// 读者：持续扫描 other 表（写发生在 src 表）
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
			}
			rows, err := db.ScanTable("other")
			if err != nil {
				t.Errorf("reader scan: %v", err)
				return
			}
			mu.Lock()
			if int64(len(rows)) > lastSeen {
				lastSeen = int64(len(rows))
			}
			mu.Unlock()
		}
	}()

	for r := 1; r <= total; r++ {
		if err := db.InsertRow("src", map[string]interface{}{"id": int64(r)}); err != nil {
			t.Fatalf("write src/%d: %v", r, err)
		}
	}
	close(stop)
	wg.Wait()

	n, err := db.Count("src", "")
	if err != nil || n != total {
		t.Fatalf("src count=%d err=%v", n, err)
	}
}
