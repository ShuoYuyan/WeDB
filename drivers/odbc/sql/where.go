package sql

import (
	"fmt"
	"strconv"
	"strings"
)

// whereExpr is a parsed WHERE clause evaluated against a single row.
type whereExpr struct {
	Left  *whereTerm
	Right *whereExpr // optional, for AND/OR chains
	Op    string     // "" (single), "AND", "OR"
}

type whereTerm struct {
	Column string
	Op     string
	Value  interface{}
}

func (w *whereExpr) Eval(row map[string]interface{}) bool {
	if w == nil {
		return true
	}
	cur := w
	for cur != nil {
		left := evalTerm(cur.Left, row)
		if cur.Right == nil {
			return left
		}
		right := cur.Right.Eval(row)
		switch cur.Op {
		case "AND":
			if !(left && right) {
				return false
			}
		case "OR":
			if left || right {
				return true
			}
		}
		cur = cur.Right
	}
	return true
}

func evalTerm(t *whereTerm, row map[string]interface{}) bool {
	if t == nil {
		return true
	}
	v, ok := row[t.Column]
	if !ok {
		return false
	}
	return compareOp(v, t.Op, t.Value)
}

func compareOp(a interface{}, op string, b interface{}) bool {
	av, aok := asFloat(a)
	bv, bok := asFloat(b)
	if aok && bok {
		switch op {
		case "=":
			return av == bv
		case "!=", "<>":
			return av != bv
		case "<":
			return av < bv
		case "<=":
			return av <= bv
		case ">":
			return av > bv
		case ">=":
			return av >= bv
		}
	}
	as, bs := fmt.Sprintf("%v", a), fmt.Sprintf("%v", b)
	switch op {
	case "=":
		return as == bs
	case "!=", "<>":
		return as != bs
	case "<":
		return as < bs
	case "<=":
		return as <= bs
	case ">":
		return as > bs
	case ">=":
		return as >= bs
	case "LIKE":
		return likeMatch(as, bs)
	}
	return false
}

func asFloat(v interface{}) (float64, bool) {
	switch x := v.(type) {
	case int:
		return float64(x), true
	case int32:
		return float64(x), true
	case int64:
		return float64(x), true
	case float32:
		return float64(x), true
	case float64:
		return x, true
	}
	return 0, false
}

func likeMatch(s, pattern string) bool {
	// Convert SQL LIKE to a simple anchored substring match.
	parts := strings.Split(pattern, "%")
	if len(parts) == 1 {
		return s == pattern
	}
	// prefix
	if !strings.HasPrefix(pattern, "%") {
		if !strings.HasPrefix(s, parts[0]) {
			return false
		}
		s = s[len(parts[0]):]
		parts = parts[1:]
	}
	// suffix
	if !strings.HasSuffix(pattern, "%") {
		last := parts[len(parts)-1]
		if !strings.HasSuffix(s, last) {
			return false
		}
		s = s[:len(s)-len(last)]
		parts = parts[:len(parts)-1]
	}
	// middle substrings
	idx := 0
	for _, p := range parts {
		if p == "" {
			continue
		}
		j := strings.Index(s[idx:], p)
		if j < 0 {
			return false
		}
		idx += j + len(p)
	}
	return true
}

// parseWhereExpr parses a SQL-92 WHERE clause into a chained whereExpr.
// The grammar is intentionally tiny:
//
//	expr := term ( ("AND" | "OR") term )*
//	term := ident op value
//	op   := "=" | "!=" | "<>" | "<" | "<=" | ">" | ">=" | "LIKE"
//	value:= number | "'...'" | ident
func parseWhereExpr(where string) (*whereExpr, error) {
	where = strings.TrimSpace(where)
	if where == "" {
		return nil, nil
	}
	// Tokenize.
	type tok struct {
		kind string // word, num, str, punct
		val  string
	}
	var toks []tok
	i := 0
	for i < len(where) {
		c := where[i]
		if c == ' ' || c == '\t' {
			i++
			continue
		}
		if c == '(' || c == ')' {
			toks = append(toks, tok{"punct", string(c)})
			i++
			continue
		}
		if c == '\'' || c == '"' {
			q := c
			i++
			j := i
			var b strings.Builder
			for j < len(where) {
				if where[j] == q {
					if j+1 < len(where) && where[j+1] == q {
						b.WriteByte(q)
						j += 2
						continue
					}
					break
				}
				b.WriteByte(where[j])
				j++
			}
			if j >= len(where) {
				return nil, fmt.Errorf("unterminated string in WHERE")
			}
			toks = append(toks, tok{"str", b.String()})
			i = j + 1
			continue
		}
		if (c >= '0' && c <= '9') || c == '-' || c == '+' {
			j := i
			if c == '-' || c == '+' {
				j++
			}
			for j < len(where) && (where[j] >= '0' && where[j] <= '9' || where[j] == '.') {
				j++
			}
			toks = append(toks, tok{"num", where[i:j]})
			i = j
			continue
		}
		// identifier / word / operator
		j := i
		for j < len(where) && where[j] != ' ' && where[j] != '(' && where[j] != ')' {
			j++
		}
		word := where[i:j]
		upper := strings.ToUpper(word)
		switch upper {
		case "AND", "OR", "LIKE", "IS", "NOT", "NULL", "IN":
			toks = append(toks, tok{"word", upper})
		case "<=", ">=", "!=", "<>", "==":
			toks = append(toks, tok{"word", upper})
		default:
			if _, err := strconv.ParseFloat(word, 64); err == nil {
				toks = append(toks, tok{"num", word})
			} else {
				toks = append(toks, tok{"ident", word})
			}
		}
		i = j
	}
	// Recursive descent parser. Grammar:
	//   expr  := term ( ("AND"|"OR") term )*
	//   term  := ident op value
	//   op    := "=" | "<" | "<=" | ">" | ">=" | "!=" | "<>" | "LIKE"
	//   value := num | str | ident
	type cursor struct{ i int }
	cur := &cursor{}
	peek := func() (tok, bool) {
		if cur.i >= len(toks) {
			return tok{}, false
		}
		return toks[cur.i], true
	}
	parseValue := func() (interface{}, error) {
		t, ok := peek()
		if !ok {
			return nil, fmt.Errorf("expected value")
		}
		switch t.kind {
		case "num":
			cur.i++
			if strings.Contains(t.val, ".") {
				return strconv.ParseFloat(t.val, 64)
			}
			return strconv.ParseInt(t.val, 10, 64)
		case "str":
			cur.i++
			return t.val, nil
		case "ident":
			cur.i++
			return t.val, nil
		}
		return nil, fmt.Errorf("expected value, got %v", t)
	}
	parseTerm := func() (*whereTerm, error) {
		t, ok := peek()
		if !ok || t.kind != "ident" {
			return nil, fmt.Errorf("expected column name")
		}
		col := t.val
		cur.i++
		// operator
		t, ok = peek()
		if !ok {
			return nil, fmt.Errorf("expected operator after %s", col)
		}
		op := strings.ToUpper(t.val)
		cur.i++
		v, err := parseValue()
		if err != nil {
			return nil, err
		}
		return &whereTerm{Column: col, Op: op, Value: v}, nil
	}
	root := &whereExpr{}
	curTerm := root
	for {
		t, err := parseTerm()
		if err != nil {
			return nil, err
		}
		curTerm.Left = t
		// AND/OR
		next, ok := peek()
		if !ok {
			return root, nil
		}
		if next.kind == "word" && (next.val == "AND" || next.val == "OR") {
			curTerm.Op = next.val
			cur.i++
			nextTerm := &whereExpr{}
			curTerm.Right = nextTerm
			curTerm = nextTerm
			continue
		}
		return root, nil
	}
}
