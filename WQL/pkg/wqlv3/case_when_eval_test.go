// WQL v3.2: CASE WHEN / COALESCE / NULLIF / CAST 解析与求值测试
//go:build !integration

package wqlv3

import (
	"testing"
)

func TestCaseWhen_Searched_Evaluate(t *testing.T) {
	row := map[string]interface{}{"amount": int64(200)}
	expr, err := ParseValueExpression("CASE WHEN amount > 100 THEN high WHEN amount > 50 THEN mid ELSE low END")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	v, err := expr.Evaluate(row)
	if err != nil {
		t.Fatalf("eval: %v", err)
	}
	if v != "high" {
		t.Errorf("expected high, got %v", v)
	}
}

func TestCaseWhen_Simple_Evaluate(t *testing.T) {
	row := map[string]interface{}{"status": "active"}
	expr, err := ParseValueExpression("CASE status WHEN active THEN ok WHEN pending THEN wait ELSE cancel END")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	v, err := expr.Evaluate(row)
	if err != nil {
		t.Fatalf("eval: %v", err)
	}
	if v != "ok" {
		t.Errorf("expected ok, got %v", v)
	}
}

func TestCaseWhen_NoElseReturnsNil(t *testing.T) {
	row := map[string]interface{}{"amount": int64(5)}
	expr, err := ParseValueExpression("CASE WHEN amount > 100 THEN high END")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	v, err := expr.Evaluate(row)
	if err != nil {
		t.Fatalf("eval: %v", err)
	}
	if v != nil {
		t.Errorf("expected nil, got %v", v)
	}
}

func TestCoalesce_Evaluate(t *testing.T) {
	// COALESCE 函数参数遵循 WQL 无双引号设计：bare identifier 视为字符串字面量
	// COALESCE(first, second, default) 三个参数都是字符串
	row1 := map[string]interface{}{}
	expr, err := ParseValueExpression("COALESCE(first, second, default)")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	// 三个参数都是字面量 "first", "second", "default" - 全部非空，返回第一个 "first"
	if v, _ := expr.Evaluate(row1); v != "first" {
		t.Errorf("expected first, got %v", v)
	}
	// 数字字面量在 COALESCE 中有效
	expr2, _ := ParseValueExpression("COALESCE(NULL, NULL, 99)")
	if v, _ := expr2.Evaluate(row1); v != int64(99) {
		t.Errorf("expected 99, got %v", v)
	}
}

func TestNullIf_Evaluate(t *testing.T) {
	// NULLIF 遵循 WQL 无双引号设计：参数为 bare identifier 视为字符串字面量
	row := map[string]interface{}{}
	expr, err := ParseValueExpression("NULLIF(x, x)")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	// 两值都是 "x"，相等，返回 nil
	if v, _ := expr.Evaluate(row); v != nil {
		t.Errorf("expected nil for equal values, got %v", v)
	}
	// 两值不同，返回第一个
	expr2, _ := ParseValueExpression("NULLIF(x, y)")
	if v, _ := expr2.Evaluate(row); v != "x" {
		t.Errorf("expected x for different values, got %v", v)
	}
}

func TestCast_Evaluate(t *testing.T) {
	// CAST 遵循 WQL 无双引号设计：参数 bare identifier 视为字符串字面量
	// CAST(42 AS INTEGER) - 数字字面量可直接转换
	row := map[string]interface{}{}
	expr, err := ParseValueExpression("CAST(42 AS INTEGER)")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	v, err := expr.Evaluate(row)
	if err != nil {
		t.Fatalf("eval: %v", err)
	}
	if v != int64(42) {
		t.Errorf("expected 42, got %v (type %T)", v, v)
	}
	// CAST(string AS INTEGER) - 字符串字面量也可转换
	expr2, _ := ParseValueExpression("CAST(42 AS REAL)")
	v2, _ := expr2.Evaluate(row)
	if v2 != float64(42) {
		t.Errorf("expected 42.0, got %v (type %T)", v2, v2)
	}
}

func TestParseValueExpression_NoQuote(t *testing.T) {
	// WQL 无双引号设计：
	//   - 值上下文中 bare identifier 视为字符串字面量
	//   - 列上下文中 bare identifier 视为列引用
	// CASE WHEN 条件: status = active (列 = 列) → 都不存在时都为 nil，等于 0 比较结果为 true
	// THEN 1 (字面量); ELSE 0 (字面量)
	expr, err := ParseValueExpression("CASE WHEN status = active THEN 1 ELSE 0 END")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	// 验证：解析结构正确（status 与 active 都是 ColumnExpr）
	c, ok := expr.(*CaseWhenExpr)
	if !ok {
		t.Fatalf("expected CaseWhenExpr, got %T", expr)
	}
	be := c.WhenClauses[0].Condition.(*BinaryExpr)
	if _, ok := be.Left.(*ColumnExpr); !ok {
		t.Errorf("expected ColumnExpr left, got %T", be.Left)
	}
	if _, ok := be.Right.(*ColumnExpr); !ok {
		t.Errorf("expected ColumnExpr right, got %T", be.Right)
	}
	// 演示：COALESCE 函数中 bare identifier 是字面量
	expr2, _ := ParseValueExpression("COALESCE(active, default)")
	row := map[string]interface{}{"status": "active"}
	v2, _ := expr2.Evaluate(row)
	if v2 != "active" {
		t.Errorf("COALESCE bare identifier: expected active literal, got %v", v2)
	}
}
