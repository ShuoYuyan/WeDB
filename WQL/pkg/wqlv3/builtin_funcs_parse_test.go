// WQL v3.2: COALESCE / NULLIF / CAST parser AST tests
//go:build !integration

package wqlv3

import (
	"testing"

	"github.com/wedb/wedb/WQL/pkg/wql/lexer"
	"github.com/wedb/wedb/WQL/pkg/wql/parser"
)

func TestParser_Coalesce(t *testing.T) {
	lex := lexer.NewLexer("db.Table(t).Select(COALESCE(name, unknown)).All()")
	p := parser.NewParser(lex)
	q, err := p.Parse()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	// Find the SelectOperation
	var sel *parser.SelectOperation
	for _, op := range q.Operations {
		if s, ok := op.(*parser.SelectOperation); ok {
			sel = s
		}
	}
	if sel == nil {
		t.Fatal("expected SelectOperation")
	}
	if len(sel.Columns) != 1 {
		t.Fatalf("expected 1 column, got %d", len(sel.Columns))
	}
	co, ok := sel.Columns[0].(*parser.CoalesceExpression)
	if !ok {
		t.Fatalf("expected CoalesceExpression, got %T", sel.Columns[0])
	}
	if len(co.Args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(co.Args))
	}
}

func TestParser_NullIf(t *testing.T) {
	lex := lexer.NewLexer("db.Table(t).Select(NULLIF(name, anonymous)).All()")
	p := parser.NewParser(lex)
	q, err := p.Parse()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	var sel *parser.SelectOperation
	for _, op := range q.Operations {
		if s, ok := op.(*parser.SelectOperation); ok {
			sel = s
		}
	}
	if sel == nil {
		t.Fatal("expected SelectOperation")
	}
	ne, ok := sel.Columns[0].(*parser.NullIfExpression)
	if !ok {
		t.Fatalf("expected NullIfExpression, got %T", sel.Columns[0])
	}
	if ne.A == nil || ne.B == nil {
		t.Fatal("NULLIF args should be non-nil")
	}
}

func TestParser_Cast(t *testing.T) {
	lex := lexer.NewLexer("db.Table(t).Select(CAST(age AS INTEGER)).All()")
	p := parser.NewParser(lex)
	q, err := p.Parse()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	var sel *parser.SelectOperation
	for _, op := range q.Operations {
		if s, ok := op.(*parser.SelectOperation); ok {
			sel = s
		}
	}
	if sel == nil {
		t.Fatal("expected SelectOperation")
	}
	ce, ok := sel.Columns[0].(*parser.CastExpression)
	if !ok {
		t.Fatalf("expected CastExpression, got %T", sel.Columns[0])
	}
	if ce.Type != "INTEGER" {
		t.Errorf("expected type=INTEGER, got %q", ce.Type)
	}
}
