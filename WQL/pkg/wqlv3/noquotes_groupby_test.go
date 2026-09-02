package wqlv3

import (
	"testing"
)

// groupTestDB creates a test DB with sales table for GROUP BY tests
func groupTestDB(t *testing.T) (*Database, *mockAdapter) {
	m := newMockAdapter()
	_ = m.CreateTable(NewTableSchema("sales",
		NewColumn("id", "INTEGER", false),
		NewColumn("region", "TEXT", false),
		NewColumn("product", "TEXT", false),
		NewColumn("amount", "INTEGER", false),
	))
	_ = m.InsertRows("sales", []map[string]interface{}{
		{"id": int64(1), "region": "east", "product": "apple", "amount": int64(10)},
		{"id": int64(2), "region": "east", "product": "banana", "amount": int64(20)},
		{"id": int64(3), "region": "east", "product": "apple", "amount": int64(30)},
		{"id": int64(4), "region": "west", "product": "apple", "amount": int64(5)},
		{"id": int64(5), "region": "west", "product": "banana", "amount": int64(15)},
		{"id": int64(6), "region": "west", "product": "banana", "amount": int64(25)},
	})
	db := NewDatabase(m)
	return db, m
}

// TestGroupByCount tests GROUP BY + Count aggregate
func TestGroupByCount(t *testing.T) {
	db, _ := groupTestDB(t)

	res, err := EvaluateQueryNoQuotes(db, `
		db.Table(sales).Select(region, Count()).GroupBy(region).All()
	`)
	if err != nil {
		t.Fatalf("GROUP BY Count failed: %v", err)
	}

	if len(res.Rows) != 2 {
		t.Errorf("expected 2 groups, got %d (%+v)", len(res.Rows), res.Rows)
	}
	// east: 3, west: 3
	countByRegion := map[string]int64{}
	for _, r := range res.Rows {
		key := r["region"].(string)
		cnt, ok := r["Count()"].(int64)
		if !ok {
			t.Errorf("Count() should be int64, got %T: %+v", r["Count()"], r)
		}
		countByRegion[key] = cnt
	}
	if countByRegion["east"] != 3 {
		t.Errorf("east count = %d, want 3", countByRegion["east"])
	}
	if countByRegion["west"] != 3 {
		t.Errorf("west count = %d, want 3", countByRegion["west"])
	}
}

// TestGroupBySum tests GROUP BY + Sum aggregate
func TestGroupBySum(t *testing.T) {
	db, _ := groupTestDB(t)

	res, err := EvaluateQueryNoQuotes(db, `
		db.Table(sales).Select(region, Sum(amount)).GroupBy(region).All()
	`)
	if err != nil {
		t.Fatalf("GROUP BY Sum failed: %v", err)
	}

	if len(res.Rows) != 2 {
		t.Errorf("expected 2 groups, got %d", len(res.Rows))
	}
	sumByRegion := map[string]float64{}
	for _, r := range res.Rows {
		key := r["region"].(string)
		s, ok := r["Sum(amount)"].(float64)
		if !ok {
			t.Errorf("Sum(amount) should be float64, got %T: %+v", r["Sum(amount)"], r)
		}
		sumByRegion[key] = s
	}
	// east: 10+20+30 = 60
	// west: 5+15+25 = 45
	if sumByRegion["east"] != 60 {
		t.Errorf("east sum = %v, want 60", sumByRegion["east"])
	}
	if sumByRegion["west"] != 45 {
		t.Errorf("west sum = %v, want 45", sumByRegion["west"])
	}
}

// TestGroupByMulti tests GROUP BY on multiple columns
func TestGroupByMulti(t *testing.T) {
	db, _ := groupTestDB(t)

	res, err := EvaluateQueryNoQuotes(db, `
		db.Table(sales).Select(region, product, Sum(amount)).GroupBy(region, product).All()
	`)
	if err != nil {
		t.Fatalf("GROUP BY multi failed: %v", err)
	}

	// (east,apple)=40, (east,banana)=20, (west,apple)=5, (west,banana)=40 => 4 groups
	if len(res.Rows) != 4 {
		t.Errorf("expected 4 groups, got %d (%+v)", len(res.Rows), res.Rows)
	}
}

// TestGroupByAvg tests AVG aggregate
func TestGroupByAvg(t *testing.T) {
	db, _ := groupTestDB(t)

	res, err := EvaluateQueryNoQuotes(db, `
		db.Table(sales).Select(region, Avg(amount)).GroupBy(region).All()
	`)
	if err != nil {
		t.Fatalf("GROUP BY Avg failed: %v", err)
	}

	// east avg: 60/3 = 20, west avg: 45/3 = 15
	avgByRegion := map[string]float64{}
	for _, r := range res.Rows {
		key := r["region"].(string)
		avgByRegion[key] = r["Avg(amount)"].(float64)
	}
	if avgByRegion["east"] != 20 {
		t.Errorf("east avg = %v, want 20", avgByRegion["east"])
	}
	if avgByRegion["west"] != 15 {
		t.Errorf("west avg = %v, want 15", avgByRegion["west"])
	}
}

// TestGroupByMinMax tests MIN and MAX aggregates
func TestGroupByMinMax(t *testing.T) {
	db, _ := groupTestDB(t)

	res, err := EvaluateQueryNoQuotes(db, `
		db.Table(sales).Select(region, Min(amount), Max(amount)).GroupBy(region).All()
	`)
	if err != nil {
		t.Fatalf("GROUP BY Min/Max failed: %v", err)
	}

	for _, r := range res.Rows {
		key := r["region"].(string)
		// east: min=10, max=30
		// west: min=5, max=25
		if key == "east" {
			if r["Min(amount)"] != int64(10) {
				t.Errorf("east min = %v, want 10", r["Min(amount)"])
			}
			if r["Max(amount)"] != int64(30) {
				t.Errorf("east max = %v, want 30", r["Max(amount)"])
			}
		}
		if key == "west" {
			if r["Min(amount)"] != int64(5) {
				t.Errorf("west min = %v, want 5", r["Min(amount)"])
			}
			if r["Max(amount)"] != int64(25) {
				t.Errorf("west max = %v, want 25", r["Max(amount)"])
			}
		}
	}
}

// TestGroupByWithWhere tests GROUP BY + WHERE
func TestGroupByWithWhere(t *testing.T) {
	db, _ := groupTestDB(t)

	// 只统计 amount > 10 的行
	res, err := EvaluateQueryNoQuotes(db, `
		db.Table(sales).Select(region, Sum(amount)).Where(amount > 10).GroupBy(region).All()
	`)
	if err != nil {
		t.Fatalf("GROUP BY + WHERE failed: %v", err)
	}

	sumByRegion := map[string]float64{}
	for _, r := range res.Rows {
		key := r["region"].(string)
		sumByRegion[key] = r["Sum(amount)"].(float64)
	}
	// east (amount > 10): 20+30 = 50
	// west (amount > 10): 15+25 = 40
	if sumByRegion["east"] != 50 {
		t.Errorf("east sum = %v, want 50", sumByRegion["east"])
	}
	if sumByRegion["west"] != 40 {
		t.Errorf("west sum = %v, want 40", sumByRegion["west"])
	}
}

// TestHaving tests HAVING filter on aggregated values
func TestHaving(t *testing.T) {
	db, _ := groupTestDB(t)

	// 找出总销售额 > 50 的地区
	res, err := EvaluateQueryNoQuotes(db, `
		db.Table(sales).Select(region, Sum(amount)).GroupBy(region).Having(Sum(amount) > 50).All()
	`)
	if err != nil {
		t.Fatalf("HAVING failed: %v", err)
	}

	// east: 60 (pass), west: 45 (fail) -> 1 row
	if len(res.Rows) != 1 {
		t.Errorf("expected 1 row, got %d (%+v)", len(res.Rows), res.Rows)
	}
	if len(res.Rows) > 0 && res.Rows[0]["region"] != "east" {
		t.Errorf("expected east, got %v", res.Rows[0]["region"])
	}
}

// TestGroupByWithoutAggregate tests GROUP BY-only (distinct)
func TestGroupByWithoutAggregate(t *testing.T) {
	db, _ := groupTestDB(t)

	res, err := EvaluateQueryNoQuotes(db, `
		db.Table(sales).Select(region).GroupBy(region).All()
	`)
	if err != nil {
		t.Fatalf("GROUP BY distinct failed: %v", err)
	}

	// 2 distinct regions
	if len(res.Rows) != 2 {
		t.Errorf("expected 2 distinct regions, got %d (%+v)", len(res.Rows), res.Rows)
	}
}

// TestAggregateWithoutGroupBy tests aggregate over all rows
func TestAggregateWithoutGroupBy(t *testing.T) {
	db, _ := groupTestDB(t)

	res, err := EvaluateQueryNoQuotes(db, `
		db.Table(sales).Select(Count(), Sum(amount)).All()
	`)
	if err != nil {
		t.Fatalf("Aggregate without GROUP BY failed: %v", err)
	}

	if len(res.Rows) != 1 {
		t.Errorf("expected 1 row, got %d", len(res.Rows))
	}
	if len(res.Rows) > 0 {
		if cnt := res.Rows[0]["Count()"]; cnt != int64(6) {
			t.Errorf("Count() = %v, want 6", cnt)
		}
		if sum := res.Rows[0]["Sum(amount)"]; sum != float64(105) {
			t.Errorf("Sum(amount) = %v, want 105", sum)
		}
	}
}
