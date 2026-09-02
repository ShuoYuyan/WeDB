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

	// 检查Condition是否是BinaryExpression
	binaryExpr, ok := whereOp.Condition.(*BinaryExpression)
	if !ok {
		t.Fatalf("Where condition is not BinaryExpression")
	}

	// 检查Right是否是SubqueryExpression
	subquery, ok := binaryExpr.Right.(*SubqueryExpression)
	if !ok {
		t.Fatalf("Right side of binary expression is not SubqueryExpression, got %T", binaryExpr.Right)
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
