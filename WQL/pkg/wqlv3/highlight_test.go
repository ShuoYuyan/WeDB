package wqlv3

import (
	"strings"
	"testing"
)

func TestHighlightSimple_Keywords(t *testing.T) {
	in := "db.Table(users).Where(age > 18).All()"
	out := HighlightSimple(in)
	if !strings.Contains(out, "Table") || !strings.Contains(out, "Where") {
		t.Errorf("expected keywords to be preserved (but may be wrapped in ANSI): got %q", out)
	}
}

func TestHighlightSimple_Strings(t *testing.T) {
	// WQL 无双引号设计：字符串字面量作为 bare identifier
	in := `db.Table(users).Where(name = alice).All()`
	// 关闭颜色以检查纯文本
	SetColorEnabled(false)
	defer SetColorEnabled(true)
	out := HighlightSimple(in)
	if !strings.Contains(out, "alice") {
		t.Errorf("expected no-quote string literal preserved: got %q", out)
	}
}

func TestHighlightSimple_Numbers(t *testing.T) {
	SetColorEnabled(false)
	defer SetColorEnabled(true)
	out := HighlightSimple("db.Table(users).Where(age > 18).All()")
	if !strings.Contains(out, "18") {
		t.Errorf("expected number preserved: got %q", out)
	}
}

func TestExplain_SimpleQuery(t *testing.T) {
	plan, err := Explain("db.Table(users).Where(age > 18).All()")
	if err != nil {
		t.Fatalf("Explain failed: %v", err)
	}
	if plan.Table != "users" {
		t.Errorf("expected table=users, got %q", plan.Table)
	}
	if !plan.Pushdown {
		t.Errorf("expected pushdown=true for simple WHERE, got false")
	}
	if plan.WhereClause == "" {
		t.Errorf("expected WHERE clause to be captured")
	}
}

func TestExplain_WithJoin(t *testing.T) {
	plan, err := Explain("db.Table(orders).Join(users, ON orders.user_id = users.id).All()")
	if err != nil {
		t.Fatalf("Explain failed: %v", err)
	}
	if plan.Table != "orders" {
		t.Errorf("expected table=orders, got %q", plan.Table)
	}
	if len(plan.Joins) == 0 {
		t.Errorf("expected join to be detected")
	}
	if plan.Pushdown {
		t.Errorf("expected pushdown=false (has JOIN)")
	}
}

func TestExplain_WithGroupBy(t *testing.T) {
	plan, err := Explain("db.Table(orders).GroupBy(user_id).All()")
	if err != nil {
		t.Fatalf("Explain failed: %v", err)
	}
	if !plan.HasGroupBy {
		t.Errorf("expected GroupBy detected")
	}
	if plan.Pushdown {
		t.Errorf("expected pushdown=false (has GROUP BY)")
	}
}

func TestExplain_WithAggregate(t *testing.T) {
	plan, err := Explain("db.Table(orders).Select(Sum(amount)).All()")
	if err != nil {
		t.Fatalf("Explain failed: %v", err)
	}
	if len(plan.Aggregates) == 0 {
		t.Errorf("expected Sum to be detected as aggregate")
	}
}

func TestExplain_OrderByAndLimit(t *testing.T) {
	plan, err := Explain("db.Table(users).OrderBy(age, DESC).Take(10).All()")
	if err != nil {
		t.Fatalf("Explain failed: %v", err)
	}
	if !plan.HasOrderBy {
		t.Errorf("expected OrderBy detected")
	}
	if !plan.HasLimit {
		t.Errorf("expected Take detected as limit")
	}
}

func TestExplain_SkipOffset(t *testing.T) {
	plan, err := Explain("db.Table(users).Skip(5).Take(10).All()")
	if err != nil {
		t.Fatalf("Explain failed: %v", err)
	}
	if !plan.HasOffset {
		t.Errorf("expected Skip detected as offset")
	}
	if !plan.HasLimit {
		t.Errorf("expected Take detected as limit")
	}
}

func TestExplain_InvalidQuery(t *testing.T) {
	_, err := Explain("not a valid query")
	if err == nil {
		t.Errorf("expected error for invalid query")
	}
}

func TestQueryPlan_String(t *testing.T) {
	SetColorEnabled(false)
	defer SetColorEnabled(true)
	plan := &QueryPlan{
		Table:       "users",
		SelectCols:  []string{"id", "name"},
		WhereClause: "age > 18",
		HasOrderBy:  true,
		HasLimit:    true,
		Pushdown:    true,
	}
	s := plan.String()
	if !strings.Contains(s, "users") {
		t.Errorf("expected table name in plan string: %q", s)
	}
	if !strings.Contains(s, "WHERE/ORDER/LIMIT") {
		t.Errorf("expected pushdown indicator: %q", s)
	}
}
