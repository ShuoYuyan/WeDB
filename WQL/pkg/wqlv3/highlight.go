// WQL syntax highlighter and EXPLAIN support
package wqlv3

import (
	"fmt"
	"strings"

	"github.com/wedb/wedb/WQL/pkg/wql/lexer"
	"github.com/wedb/wedb/WQL/pkg/wql/parser"
)

// ANSI color codes (auto-disabled when NO_COLOR is set or non-terminal)
const (
	ansiReset   = "\033[0m"
	ansiBold    = "\033[1m"
	ansiRed     = "\033[31m"
	ansiGreen   = "\033[32m"
	ansiYellow  = "\033[33m"
	ansiBlue    = "\033[34m"
	ansiMagenta = "\033[35m"
	ansiCyan    = "\033[36m"
	ansiGray    = "\033[90m"
)

// useColor 控制是否输出 ANSI 颜色
var useColor = true

// SetColorEnabled 启用/禁用颜色
func SetColorEnabled(enabled bool) {
	useColor = enabled
}

// colorize 应用 ANSI 颜色码（如果启用）
func colorize(color, s string) string {
	if !useColor {
		return s
	}
	return color + s + ansiReset
}

// Highlight 简单 WQL 语法高亮（基于 lexer token 分类）
func Highlight(src string) string {
	lex := lexer.NewLexer(src)
	var colored strings.Builder
	for {
		tok := lex.NextToken()
		if tok.Type == lexer.TOKEN_EOF {
			break
		}
		if tok.Type == lexer.TOKEN_ILLEGAL {
			colored.WriteString(colorize(ansiRed, tok.Value))
			continue
		}
		piece := tok.Value
		if piece == "" {
			continue
		}
		switch tok.Type {
		case lexer.TOKEN_DB, lexer.TOKEN_TABLE, lexer.TOKEN_SELECT, lexer.TOKEN_WHERE,
			lexer.TOKEN_ORDER_BY, lexer.TOKEN_GROUP_BY, lexer.TOKEN_HAVING,
			lexer.TOKEN_JOIN, lexer.TOKEN_LEFT_JOIN, lexer.TOKEN_RIGHT_JOIN,
			lexer.TOKEN_LIMIT, lexer.TOKEN_TAKE, lexer.TOKEN_SKIP,
			lexer.TOKEN_FIRST, lexer.TOKEN_ALL, lexer.TOKEN_COUNT, lexer.TOKEN_SUM,
			lexer.TOKEN_AVG, lexer.TOKEN_MIN, lexer.TOKEN_MAX,
			lexer.TOKEN_ASC, lexer.TOKEN_DESC,
			lexer.TOKEN_INSERT, lexer.TOKEN_UPDATE, lexer.TOKEN_DELETE, lexer.TOKEN_SET,
			lexer.TOKEN_INTO, lexer.TOKEN_VALUES,
			lexer.TOKEN_EXECUTE, lexer.TOKEN_CREATE, lexer.TOKEN_DROP,
			lexer.TOKEN_UNION, lexer.TOKEN_UNION_ALL, lexer.TOKEN_INTERSECT, lexer.TOKEN_EXCEPT,
			lexer.TOKEN_ON, lexer.TOKEN_AS, lexer.TOKEN_PRIMARY, lexer.TOKEN_KEY,
			lexer.TOKEN_DEFAULT, lexer.TOKEN_SUBQUERY, lexer.TOKEN_JSON_EXTRACT,
			lexer.TOKEN_JSON_QUERY, lexer.TOKEN_JSON_VALUE:
			colored.WriteString(colorize(ansiBlue+ansiBold, piece))
		case lexer.TOKEN_AND, lexer.TOKEN_OR, lexer.TOKEN_NOT,
			lexer.TOKEN_LIKE, lexer.TOKEN_IN, lexer.TOKEN_BETWEEN, lexer.TOKEN_IS:
			colored.WriteString(colorize(ansiMagenta+ansiBold, piece))
		case lexer.TOKEN_EQ, lexer.TOKEN_NE, lexer.TOKEN_LT, lexer.TOKEN_LE,
			lexer.TOKEN_GT, lexer.TOKEN_GE:
			colored.WriteString(colorize(ansiRed, piece))
		case lexer.TOKEN_PLUS, lexer.TOKEN_MINUS, lexer.TOKEN_MULTIPLY,
			lexer.TOKEN_DIVIDE, lexer.TOKEN_MODULO, lexer.TOKEN_POWER:
			colored.WriteString(colorize(ansiRed, piece))
		case lexer.TOKEN_INTEGER, lexer.TOKEN_FLOAT:
			colored.WriteString(colorize(ansiCyan, piece))
		case lexer.TOKEN_STRING:
			colored.WriteString(colorize(ansiGreen, piece))
		case lexer.TOKEN_BOOLEAN, lexer.TOKEN_NULL:
			colored.WriteString(colorize(ansiYellow, piece))
		case lexer.TOKEN_DOT, lexer.TOKEN_COMMA, lexer.TOKEN_SEMICOLON,
			lexer.TOKEN_COLON, lexer.TOKEN_LPAREN, lexer.TOKEN_RPAREN,
			lexer.TOKEN_LBRACKET, lexer.TOKEN_RBRACKET, lexer.TOKEN_LBRACE, lexer.TOKEN_RBRACE,
			lexer.TOKEN_QUESTION:
			colored.WriteString(colorize(ansiGray, piece))
		case lexer.TOKEN_IDENTIFIER:
			colored.WriteString(piece)
		default:
			colored.WriteString(piece)
		}
		// Lexer 不保留空白；这里不强求格式保真（CLI 输入通常无空白）
		// 如果需要保留可后续在 Highlight 中插入空格分隔
	}
	// Lexer 会消耗空白，简化处理：使用空格分隔 token（不严格保留格式）
	return colored.String()
}

// HighlightSimple 基于正则/字符串的轻量级高亮（保留原始空白）
// 这是更友好的版本：直接在原文上替换关键 token
func HighlightSimple(src string) string {
	if !useColor {
		return src
	}
	// 关键字（按长度倒序，避免短词抢先匹配）
	keywords := []string{
		"LeftJoin", "RightJoin", "OrderBy", "GroupBy", "Having", "Insert", "Update", "Delete",
		"Create", "Drop", "Execute", "UnionAll", "Count", "Sum", "Avg", "Min", "Max",
		"Table", "Select", "Where", "Union", "Intersect", "Except",
		"Take", "Skip", "Limit", "First", "All", "Join", "ASC", "DESC", "AS", "ON", "Set",
		"Values", "Primary", "Key", "Default", "db",
	}
	// 逻辑关键字
	logicKeywords := []string{"AND", "OR", "NOT", "IN", "LIKE", "BETWEEN", "IS", "NULL", "TRUE", "FALSE"}

	out := src
	out = colorizeStrings(out)
	out = colorizeNumbers(out)
	for _, kw := range keywords {
		out = replaceWord(out, kw, colorize(ansiBlue+ansiBold, kw))
	}
	for _, kw := range logicKeywords {
		out = replaceWord(out, kw, colorize(ansiMagenta+ansiBold, kw))
	}
	for _, op := range []string{"!=", "<=", ">=", "==", "=", "<", ">"} {
		out = replaceWord(out, op, colorize(ansiRed, op))
	}
	return out
}

// replaceWord 替换完整 word（基于非标识符字符边界）
func replaceWord(s, word, repl string) string {
	var b strings.Builder
	i := 0
	for i < len(s) {
		if i+len(word) <= len(s) && s[i:i+len(word)] == word {
			before := i == 0 || !isIdentByte(s[i-1])
			after := i+len(word) == len(s) || !isIdentByte(s[i+len(word)])
			if before && after {
				b.WriteString(repl)
				i += len(word)
				continue
			}
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

// isIdentByte 是标识符字符？
func isIdentByte(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') || c == '_'
}

// colorizeStrings 给 "..." 与 '...' 字符串上色
func colorizeStrings(s string) string {
	var b strings.Builder
	i := 0
	for i < len(s) {
		c := s[i]
		if c == '"' || c == '\'' {
			quote := c
			// 找结束引号（跳过转义）
			j := i + 1
			for j < len(s) && s[j] != quote {
				if s[j] == '\\' && j+1 < len(s) {
					j += 2
					continue
				}
				j++
			}
			if j < len(s) {
				b.WriteString(colorize(ansiGreen, s[i:j+1]))
				i = j + 1
				continue
			}
		}
		b.WriteByte(c)
		i++
	}
	return b.String()
}

// colorizeNumbers 给数字字面量上色
func colorizeNumbers(s string) string {
	var b strings.Builder
	i := 0
	for i < len(s) {
		c := s[i]
		if c >= '0' && c <= '9' {
			before := i == 0 || !isIdentByte(s[i-1])
			if before {
				j := i
				for j < len(s) && (s[j] >= '0' && s[j] <= '9' || s[j] == '.') {
					j++
				}
				b.WriteString(colorize(ansiCyan, s[i:j]))
				i = j
				continue
			}
		}
		b.WriteByte(c)
		i++
	}
	return b.String()
}

// QueryPlan describes the steps a WQL query will take
type QueryPlan struct {
	Pushdown    bool     // WHERE/ORDER/LIMIT pushed to storage
	Table       string   // source table
	Joins       []string // joined tables in order
	HasGroupBy  bool
	HasHaving   bool
	Aggregates  []string
	HasOrderBy  bool
	HasLimit    bool
	HasOffset   bool
	SelectCols  []string
	WhereClause string
}

// Explain returns a QueryPlan describing how the query would be executed.
// It uses the parser to extract structure; it does NOT execute the query.
func Explain(query string) (*QueryPlan, error) {
	plan := &QueryPlan{}

	q, err := parser.ParseString(query)
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	plan.Table = q.Source
	for _, op := range q.Operations {
		switch o := op.(type) {
		case *parser.JoinOperation:
			plan.Joins = append(plan.Joins, fmt.Sprintf("%s(%s)", o.JoinType, exprString(o.Table)))
		case *parser.GroupByOperation:
			plan.HasGroupBy = true
		case *parser.HavingOperation:
			plan.HasHaving = true
		case *parser.SelectOperation:
			for _, c := range o.Columns {
				plan.SelectCols = append(plan.SelectCols, c.String())
			}
			for _, agg := range collectAggregates(o.Columns) {
				plan.Aggregates = append(plan.Aggregates, agg)
			}
		case *parser.WhereOperation:
			plan.WhereClause = exprString(o.Condition)
		case *parser.OrderByOperation:
			plan.HasOrderBy = true
		case *parser.LimitOperation:
			plan.HasLimit = true
		case *parser.TakeOperation:
			plan.HasLimit = true
		case *parser.SkipOperation:
			plan.HasOffset = true
		}
	}

	plan.Pushdown = !plan.HasGroupBy && !plan.HasHaving && len(plan.Joins) == 0 && len(plan.Aggregates) == 0
	return plan, nil
}

// exprString 安全获取 Expression 的字符串表示
func exprString(e parser.Expression) string {
	if e == nil {
		return ""
	}
	return e.String()
}

// collectAggregates 从 Select 列中收集聚合函数名
func collectAggregates(cols []parser.Expression) []string {
	aggNames := map[string]bool{
		"Count": true, "Sum": true, "Avg": true, "Min": true, "Max": true,
	}
	var found []string
	seen := map[string]bool{}
	for _, c := range cols {
		s := c.String()
		for name := range aggNames {
			if strings.Contains(s, name+"(") && !seen[name] {
				seen[name] = true
				found = append(found, name)
			}
		}
	}
	return found
}

// String formats the plan as a human-readable multi-line string
func (p *QueryPlan) String() string {
	var b strings.Builder
	b.WriteString("Query Plan:\n")
	b.WriteString("  Table:    " + p.Table + "\n")
	if len(p.Joins) > 0 {
		b.WriteString("  Joins:    ")
		for i, j := range p.Joins {
			if i > 0 {
				b.WriteString(" → ")
			}
			b.WriteString(j)
		}
		b.WriteString("\n")
	}
	if len(p.SelectCols) > 0 {
		b.WriteString("  Select:   ")
		for i, c := range p.SelectCols {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(c)
		}
		b.WriteString("\n")
	}
	if p.WhereClause != "" {
		b.WriteString("  Where:    " + p.WhereClause + "\n")
	}
	if p.HasGroupBy {
		b.WriteString("  GroupBy:  yes\n")
	}
	if p.HasHaving {
		b.WriteString("  Having:   yes\n")
	}
	if len(p.Aggregates) > 0 {
		b.WriteString("  AggFuncs: ")
		for i, a := range p.Aggregates {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(a)
		}
		b.WriteString("\n")
	}
	if p.HasOrderBy {
		b.WriteString("  OrderBy:  yes\n")
	}
	if p.HasOffset {
		b.WriteString("  Offset:   yes\n")
	}
	if p.HasLimit {
		b.WriteString("  Limit:    yes\n")
	}
	if p.Pushdown {
		b.WriteString(colorize(ansiGreen, "  Pushdown: WHERE/ORDER/LIMIT → storage engine ✓\n"))
	} else {
		b.WriteString(colorize(ansiYellow, "  Pushdown: no (in-memory evaluation)\n"))
	}
	return b.String()
}
