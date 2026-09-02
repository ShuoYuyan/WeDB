package wqlv3

import (
	"testing"
)

// pushdownTestDB creates a test DB with 100 users
func pushdownTestDB(t *testing.T) (*Database, *mockAdapter) {
	m := newMockAdapter()
	_ = m.CreateTable(NewTableSchema("users",
		NewColumn("id", "INTEGER", false),
		NewColumn("name", "TEXT", false),
		NewColumn("age", "INTEGER", true),
	))
	rows := make([]map[string]interface{}, 0, 100)
	for i := 1; i <= 100; i++ {
		rows = append(rows, map[string]interface{}{
			"id":   int64(i),
			"name": "user",
			"age":  int64(20 + (i % 50)),
		})
	}
	_ = m.InsertRows("users", rows)
	db := NewDatabase(m)
	return db, m
}

// TestPushdownWhereCorrectness: WHERE pushed down to storage; result same as in-memory
func TestPushdownWhereCorrectness(t *testing.T) {
	db, _ := pushdownTestDB(t)

	res, err := EvaluateQueryNoQuotes(db, `db.Table(users).Where(age > 60).All()`)
	if err != nil {
		t.Fatalf("pushdown WHERE failed: %v", err)
	}

	// 期望: 100 行中 age > 60 的行 (age = 20 + i%50, so age values cycle 20-69)
	// 60 < age <= 69 -> i%50 in (40,49] -> 9 values per 50-cycle
	// 100 行 = 2 cycles -> 18 行
	if len(res.Rows) != 18 {
		t.Errorf("expected 18 rows with age > 60, got %d", len(res.Rows))
	}
	for _, r := range res.Rows {
		age := r["age"].(int64)
		if age <= 60 {
			t.Errorf("row has age %d, should be > 60", age)
		}
	}
}

// TestPushdownHitCounter: WHERE pushdown should increment pushdownHits
func TestPushdownHitCounter(t *testing.T) {
	db, m := pushdownTestDB(t)
	before := m.pushdownHits

	_, _ = EvaluateQueryNoQuotes(db, `db.Table(users).Where(age > 30).All()`)

	after := m.pushdownHits
	if after-before != 1 {
		t.Errorf("pushdownHits delta = %d, want 1 (WHERE should be pushed down)", after-before)
	}
}

// TestPushdownNoWhereNoHit: Without WHERE, no pushdown to ScanTableWithOptions
func TestPushdownNoWhereNoHit(t *testing.T) {
	db, m := pushdownTestDB(t)
	before := m.pushdownHits

	// 仅 All() 无 WHERE：理论上不需要下推（无可推内容）
	_, _ = EvaluateQueryNoQuotes(db, `db.Table(users).All()`)

	after := m.pushdownHits
	// 无 WHERE 时不强制下推；当前实现：若仅有 Take/Order 仍会下推（无 WHERE）
	// 这里我们允许 0 或 1 次
	if after-before > 1 {
		t.Errorf("pushdownHits delta = %d, expected <= 1", after-before)
	}
}

// TestPushdownJoinNotApplied: JOIN queries must NOT use pushdown (multi-table)
func TestPushdownJoinNotApplied(t *testing.T) {
	m := newMockAdapter()
	_ = m.CreateTable(NewTableSchema("users", NewColumn("id", "INTEGER", false), NewColumn("name", "TEXT", false)))
	_ = m.CreateTable(NewTableSchema("orders", NewColumn("id", "INTEGER", false), NewColumn("user_id", "INTEGER", false)))
	_ = m.InsertRows("users", []map[string]interface{}{{"id": int64(1), "name": "alice"}})
	_ = m.InsertRows("orders", []map[string]interface{}{{"id": int64(100), "user_id": int64(1)}})
	db := NewDatabase(m)
	before := m.pushdownHits

	_, _ = EvaluateQueryNoQuotes(db, `db.Table(users).Join(orders, ON users.id = orders.user_id).All()`)

	// JOIN 时不应下推主表 WHERE
	delta := m.pushdownHits - before
	if delta != 0 {
		t.Errorf("pushdownHits delta = %d with JOIN, expected 0 (no pushdown for joined queries)", delta)
	}
}

// TestPushdownGroupByNotApplied: GROUP BY queries must NOT use pushdown
func TestPushdownGroupByNotApplied(t *testing.T) {
	db, m := pushdownTestDB(t)
	before := m.pushdownHits

	_, _ = EvaluateQueryNoQuotes(db, `db.Table(users).Select(age, Count()).GroupBy(age).All()`)

	delta := m.pushdownHits - before
	if delta != 0 {
		t.Errorf("pushdownHits delta = %d with GROUP BY, expected 0", delta)
	}
}

// TestPushdownOrderByCorrectness: ORDER BY via pushdown returns sorted result
func TestPushdownOrderByCorrectness(t *testing.T) {
	db, _ := pushdownTestDB(t)

	res, err := EvaluateQueryNoQuotes(db, `db.Table(users).Where(age > 50).OrderBy(age, DESC).Take(5).All()`)
	if err != nil {
		t.Fatalf("pushdown ORDER BY failed: %v", err)
	}

	if len(res.Rows) != 5 {
		t.Errorf("expected 5 rows, got %d", len(res.Rows))
	}
	// 验证 DESC 排序
	for i := 1; i < len(res.Rows); i++ {
		if res.Rows[i]["age"].(int64) > res.Rows[i-1]["age"].(int64) {
			t.Errorf("rows not sorted DESC at index %d", i)
		}
	}
}

// TestPushdownTakeCorrectness: LIMIT via pushdown
func TestPushdownTakeCorrectness(t *testing.T) {
	db, _ := pushdownTestDB(t)

	res, err := EvaluateQueryNoQuotes(db, `db.Table(users).Where(age > 30).Take(3).All()`)
	if err != nil {
		t.Fatalf("pushdown LIMIT failed: %v", err)
	}

	if len(res.Rows) != 3 {
		t.Errorf("expected 3 rows after Take(3), got %d", len(res.Rows))
	}
}

// TestPushdownSkipCorrectness: OFFSET via pushdown
func TestPushdownSkipCorrectness(t *testing.T) {
	db, _ := pushdownTestDB(t)

	res, err := EvaluateQueryNoQuotes(db, `db.Table(users).Where(age > 30).Skip(2).Take(3).All()`)
	if err != nil {
		t.Fatalf("pushdown OFFSET failed: %v", err)
	}

	if len(res.Rows) != 3 {
		t.Errorf("expected 3 rows, got %d", len(res.Rows))
	}
}

// TestPushdownCombined: All pushdown features together
func TestPushdownCombined(t *testing.T) {
	db, m := pushdownTestDB(t)
	before := m.pushdownHits

	res, err := EvaluateQueryNoQuotes(db, `db.Table(users).Where(age > 30 AND age < 60).OrderBy(age, ASC).Skip(5).Take(10).All()`)
	if err != nil {
		t.Fatalf("combined pushdown failed: %v", err)
	}

	// 验证数量
	if len(res.Rows) != 10 {
		t.Errorf("expected 10 rows, got %d", len(res.Rows))
	}
	// 验证范围
	for _, r := range res.Rows {
		age := r["age"].(int64)
		if age <= 30 || age >= 60 {
			t.Errorf("age %d out of range (30,60)", age)
		}
	}
	// 验证 ASC 排序
	for i := 1; i < len(res.Rows); i++ {
		if res.Rows[i]["age"].(int64) < res.Rows[i-1]["age"].(int64) {
			t.Errorf("not ASC sorted at index %d", i)
		}
	}
	// 验证 pushdown 命中 1 次
	if m.pushdownHits-before != 1 {
		t.Errorf("expected 1 pushdown hit, got %d", m.pushdownHits-before)
	}
}
