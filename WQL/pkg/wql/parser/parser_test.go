package parser

import "testing"

func TestParseString(t *testing.T) {
	input := "db.Table(users).Where(age > 25).Take(3).All()"

	query, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	if query.Source != "users" {
		t.Errorf("query.Source = %s, want users", query.Source)
	}

	if len(query.Operations) != 3 {
		t.Fatalf("len(query.Operations) = %d, want 3", len(query.Operations))
	}
}

func TestParseSimpleSelect(t *testing.T) {
	input := "db.Table(users).Select(name, age).All()"

	query, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	if query.Source != "users" {
		t.Errorf("query.Source = %s, want users", query.Source)
	}
}

func TestParseWithJoin(t *testing.T) {
	input := "db.Table(users).Join(orders, users.id, orders.user_id).All()"

	query, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	if len(query.Operations) != 2 {
		t.Fatalf("len(query.Operations) = %d, want 2", len(query.Operations))
	}
}

func TestParseSubquery(t *testing.T) {
	// 测试简单的子查询
	input := "db.Table(users).Where(id IN (db.Table(orders).Select(id).Where(amount > 100))).All()"

	query, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	if query.Source != "users" {
		t.Errorf("query.Source = %s, want users", query.Source)
	}

	// 应该有Where操作
	if len(query.Operations) == 0 {
		t.Fatalf("len(query.Operations) = %d, want at least 1", len(query.Operations))
	}

	// 检查Where操作的Condition是否包含子查询
	whereOp, ok := query.Operations[0].(*WhereOperation)
	if !ok {
		t.Fatalf("first operation is not WhereOperation")
	}

	// 检查Condition是否是 InExpression（新的 IN 语法）
	inExpr, ok := whereOp.Condition.(*InExpression)
	if !ok {
		// 兼容旧式 BinaryExpression
		if be, ok2 := whereOp.Condition.(*BinaryExpression); ok2 {
			if _, ok3 := be.Right.(*SubqueryExpression); ok3 {
				return // OK 旧式
			}
		}
		t.Fatalf("Where condition is not InExpression or BinaryExpression with subquery, got %T", whereOp.Condition)
	}

	// 检查 InExpression 的 Values 是否包含 SubqueryExpression
	if len(inExpr.Values) == 0 {
		t.Fatal("InExpression.Values is empty")
	}
	subquery, ok := inExpr.Values[0].(*SubqueryExpression)
	if !ok {
		t.Fatalf("First InExpression value is not SubqueryExpression, got %T", inExpr.Values[0])
	}

	if subquery.Query == nil {
		t.Error("Subquery.Query is nil")
	}

	if subquery.Query.Source != "orders" {
		t.Errorf("Subquery.Query.Source = %s, want orders", subquery.Query.Source)
	}
}

func TestParseSubqueryWithAlias(t *testing.T) {
	// 测试带别名的子查询
	input := "db.Table(users).Join((db.Table(orders).Select(id, name)) AS o, users.id, o.id).All()"

	query, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	if query.Source != "users" {
		t.Errorf("query.Source = %s, want users", query.Source)
	}

	// 应该有Join操作
	if len(query.Operations) == 0 {
		t.Fatalf("len(query.Operations) = %d, want at least 1", len(query.Operations))
	}

	// 检查Join操作的Table是否是子查询
	joinOp, ok := query.Operations[0].(*JoinOperation)
	if !ok {
		t.Fatalf("first operation is not JoinOperation")
	}

	// 检查Table是否是SubqueryExpression
	subquery, ok := joinOp.Table.(*SubqueryExpression)
	if !ok {
		t.Fatalf("Join table is not SubqueryExpression, got %T", joinOp.Table)
	}

	if subquery.Alias != "o" {
		t.Errorf("Subquery.Alias = %s, want o", subquery.Alias)
	}

	if subquery.Query.Source != "orders" {
		t.Errorf("Subquery.Query.Source = %s, want orders", subquery.Query.Source)
	}
}
