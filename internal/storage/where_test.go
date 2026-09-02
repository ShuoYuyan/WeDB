package storage

import (
	"testing"
)

func TestIsOuterParenPair(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"(name = \"alice\")", true},
		{"(age > 18)", true},
		{"((a > 1))", true},
		{"name = \"alice\"", false},
		{"age > 18", false},
		{"(name = \"(foo)\")", true},     // 字符串内的括号不应影响外层剥离
		{"(a) AND (b)", false},           // 不是配对
		{"(name = \"x\") AND (age > 1)", false}, // 复杂表达式
		{"", false},
		{"(", false},
		{"()", true},
	}
	for _, c := range cases {
		got := isOuterParenPair(c.in)
		if got != c.want {
			t.Errorf("isOuterParenPair(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestParseSingleCondition_StripsParens(t *testing.T) {
	// 条件带外层括号，应被剥离后正确解析
	c, err := parseSingleCondition(`(name = "alice")`, nil)
	if err != nil {
		t.Fatalf("expected parse to succeed with stripped parens, got: %v", err)
	}
	if c == nil {
		t.Fatal("expected non-nil condition")
	}
	if c.Column != "name" {
		t.Errorf("expected column=name, got %q", c.Column)
	}
}

func TestParseSingleCondition_NoParens(t *testing.T) {
	c, err := parseSingleCondition(`age > 18`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Column != "age" {
		t.Errorf("expected column=age, got %q", c.Column)
	}
}

func TestParseSingleCondition_NestedParens(t *testing.T) {
	c, err := parseSingleCondition(`((age > 18))`, nil)
	if err != nil {
		t.Fatalf("nested parens should be stripped iteratively: %v", err)
	}
	if c.Column != "age" {
		t.Errorf("expected column=age after stripping, got %q", c.Column)
	}
}
