// WQL v3.2: CASE WHEN parser AST tests
//go:build !integration

package wqlv3

import (
	"testing"

	"github.com/wedb/wedb/WQL/pkg/wql/lexer"
	"github.com/wedb/wedb/WQL/pkg/wql/parser"
)

func findSelect(t *testing.T, q *parser.WQLQuery) *parser.SelectOperation {
	t.Helper()
	for _, op := range q.Operations {
		if s, ok := op.(*parser.SelectOperation); ok {
			return s
		}
	}
	t.Fatal("no SelectOperation found")
	return nil
}

func TestParser_CaseWhen_Searched(t *testing.T) {
	lex := lexer.NewLexer("db.Table(orders).Select(CASE WHEN amount > 100 THEN high WHEN amount > 50 THEN mid ELSE low END).All()")
	p := parser.NewParser(lex)
	q, err := p.Parse()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	sel := findSelect(t, q)
	ce, ok := sel.Columns[0].(*parser.CaseWhenExpression)
	if !ok {
		t.Fatalf("expected CaseWhenExpression, got %T", sel.Columns[0])
	}
	if len(ce.WhenClauses) != 2 {
		t.Fatalf("expected 2 WHEN clauses, got %d", len(ce.WhenClauses))
	}
	if ce.ElseValue == nil {
		t.Fatal("expected ELSE clause")
	}
}

func TestParser_CaseWhen_Simple(t *testing.T) {
	lex := lexer.NewLexer("db.Table(orders).Select(CASE status WHEN active THEN ok WHEN pending THEN wait ELSE cancel END).All()")
	p := parser.NewParser(lex)
	q, err := p.Parse()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	sel := findSelect(t, q)
	ce, ok := sel.Columns[0].(*parser.CaseWhenExpression)
	if !ok {
		t.Fatalf("expected CaseWhenExpression, got %T", sel.Columns[0])
	}
	if ce.Input == nil {
		t.Fatal("expected simple CASE input")
	}
	if len(ce.WhenClauses) != 2 {
		t.Fatalf("expected 2 WHEN clauses, got %d", len(ce.WhenClauses))
	}
}

func TestParser_CaseWhen_NoQuoteStrings(t *testing.T) {
	// 验证 WQL 无双引号设计：CASE WHEN 内的字符串也是 bare identifier
	lex := lexer.NewLexer("db.Table(orders).Select(CASE WHEN status = active THEN 1 ELSE 0 END).All()")
	p := parser.NewParser(lex)
	q, err := p.Parse()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	sel := findSelect(t, q)
	ce := sel.Columns[0].(*parser.CaseWhenExpression)
	if ce == nil {
		t.Fatal("expected CaseWhenExpression")
	}
	// 验证 String() 输出
	got := ce.String()
	want := "CASE WHEN (status = active) THEN 1 ELSE 0 END"
	if got != want {
		t.Errorf("CASE WHEN String() = %q, want %q", got, want)
	}
}
