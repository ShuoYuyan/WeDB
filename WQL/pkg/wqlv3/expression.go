// 表达式系统：WQL WHERE 子句的内存评估引擎。
// 这是一个完整的、独立的表达式解析和求值系统。
package wqlv3

import (
	"fmt"
	"strconv"
	"strings"
)

// Expression 是一个可求值的布尔或值表达式
type Expression interface {
	Evaluate(row map[string]interface{}) (interface{}, error)
	String() string
}

// ===== 具体表达式类型 =====

// ColumnExpr 列引用
type ColumnExpr struct {
	Name string
}

func (e *ColumnExpr) Evaluate(row map[string]interface{}) (interface{}, error) {
	// 先尝试完整名（含 table.column 形式）
	if v, ok := row[e.Name]; ok {
		return v, nil
	}
	// 退而求其次：取最后一段（去表前缀），便于 JOIN 合并行查找
	if idx := strings.LastIndex(e.Name, "."); idx >= 0 {
		short := e.Name[idx+1:]
		if v, ok := row[short]; ok {
			return v, nil
		}
	}
	return nil, nil
}

func (e *ColumnExpr) String() string {
	return e.Name
}

// LiteralExpr 字面量值
type LiteralExpr struct {
	Value interface{}
}

func (e *LiteralExpr) Evaluate(row map[string]interface{}) (interface{}, error) {
	return e.Value, nil
}

func (e *LiteralExpr) String() string {
	return fmt.Sprintf("%v", e.Value)
}

// BinaryExpr 二元操作
type BinaryExpr struct {
	Left  Expression
	Right Expression
	Op    BinaryOp
}

func (e *BinaryExpr) Evaluate(row map[string]interface{}) (interface{}, error) {
	left, err := e.Left.Evaluate(row)
	if err != nil {
		return nil, err
	}
	right, err := e.Right.Evaluate(row)
	if err != nil {
		return nil, err
	}
	switch e.Op {
	case OpEq:
		return compareForFilter(left, right) == 0, nil
	case OpNe:
		return compareForFilter(left, right) != 0, nil
	case OpLt:
		return compareForFilter(left, right) < 0, nil
	case OpLe:
		return compareForFilter(left, right) <= 0, nil
	case OpGt:
		return compareForFilter(left, right) > 0, nil
	case OpGe:
		return compareForFilter(left, right) >= 0, nil
	case OpAnd:
		return toBoolForFilter(left) && toBoolForFilter(right), nil
	case OpOr:
		return toBoolForFilter(left) || toBoolForFilter(right), nil
	case OpLike:
		return matchLikeForFilter(toStringForFilter(left), toStringForFilter(right)), nil
	}
	return nil, fmt.Errorf("unsupported operator: %s", e.Op)
}

func (e *BinaryExpr) String() string {
	return fmt.Sprintf("(%s %s %s)", e.Left.String(), e.Op, e.Right.String())
}

// BinaryOp 二元操作符
type BinaryOp string

const (
	OpEq   BinaryOp = "="
	OpNe   BinaryOp = "!="
	OpLt   BinaryOp = "<"
	OpLe   BinaryOp = "<="
	OpGt   BinaryOp = ">"
	OpGe   BinaryOp = ">="
	OpAnd  BinaryOp = "AND"
	OpOr   BinaryOp = "OR"
	OpLike BinaryOp = "LIKE"
)

// UnaryExpr 一元操作（如 NOT）
type UnaryExpr struct {
	Op      string
	Operand Expression
}

func (e *UnaryExpr) Evaluate(row map[string]interface{}) (interface{}, error) {
	v, err := e.Operand.Evaluate(row)
	if err != nil {
		return nil, err
	}
	if e.Op == "NOT" {
		return !toBoolForFilter(v), nil
	}
	return v, nil
}

func (e *UnaryExpr) String() string {
	return fmt.Sprintf("%s %s", e.Op, e.Operand.String())
}

// InExpr IN 表达式
type InExpr struct {
	Column string
	Values []interface{}
}

func (e *InExpr) Evaluate(row map[string]interface{}) (interface{}, error) {
	v, exists := row[e.Column]
	if !exists {
		return false, nil
	}
	for _, val := range e.Values {
		if compareForFilter(v, val) == 0 {
			return true, nil
		}
	}
	return false, nil
}

func (e *InExpr) String() string {
	parts := make([]string, len(e.Values))
	for i, v := range e.Values {
		parts[i] = fmt.Sprintf("%v", v)
	}
	return fmt.Sprintf("%s IN (%s)", e.Column, strings.Join(parts, ", "))
}

// IsNullExpr IS NULL 表达式
type IsNullExpr struct {
	Column string
	Negate bool
}

func (e *IsNullExpr) Evaluate(row map[string]interface{}) (interface{}, error) {
	v, exists := row[e.Column]
	isNull := !exists || v == nil
	if e.Negate {
		return !isNull, nil
	}
	return isNull, nil
}

func (e *IsNullExpr) String() string {
	if e.Negate {
		return fmt.Sprintf("%s IS NOT NULL", e.Column)
	}
	return fmt.Sprintf("%s IS NULL", e.Column)
}

// ===== 工厂函数 =====

// Col 创建列表达式
func Col(name string) Expression {
	return &ColumnExpr{Name: name}
}

// Lit 创建字面量表达式
func Lit(value interface{}) Expression {
	return &LiteralExpr{Value: value}
}

// Eq 创建等于
func Eq(left, right interface{}) Expression {
	return &BinaryExpr{Left: toExprHelper(left), Right: toExprHelper(right), Op: OpEq}
}

// Ne 创建不等于
func Ne(left, right interface{}) Expression {
	return &BinaryExpr{Left: toExprHelper(left), Right: toExprHelper(right), Op: OpNe}
}

// Lt 创建小于
func Lt(left, right interface{}) Expression {
	return &BinaryExpr{Left: toExprHelper(left), Right: toExprHelper(right), Op: OpLt}
}

// Le 创建小于等于
func Le(left, right interface{}) Expression {
	return &BinaryExpr{Left: toExprHelper(left), Right: toExprHelper(right), Op: OpLe}
}

// Gt 创建大于
func Gt(left, right interface{}) Expression {
	return &BinaryExpr{Left: toExprHelper(left), Right: toExprHelper(right), Op: OpGt}
}

// Ge 创建大于等于
func Ge(left, right interface{}) Expression {
	return &BinaryExpr{Left: toExprHelper(left), Right: toExprHelper(right), Op: OpGe}
}

// AndExpr 创建 AND
func AndExpr(left, right Expression) Expression {
	return &BinaryExpr{Left: left, Right: right, Op: OpAnd}
}

// OrExpr 创建 OR
func OrExpr(left, right Expression) Expression {
	return &BinaryExpr{Left: left, Right: right, Op: OpOr}
}

// toExprHelper 将值转为 Expression
func toExprHelper(v interface{}) Expression {
	switch val := v.(type) {
	case Expression:
		return val
	case *ColumnExpr:
		return val
	case string:
		// 字符串字面量: "..." 或 '...'
		if len(val) >= 2 {
			if (val[0] == '"' && val[len(val)-1] == '"') ||
				(val[0] == '\'' && val[len(val)-1] == '\'') {
				return &LiteralExpr{Value: val[1 : len(val)-1]}
			}
		}
		// 简单判断：如果全是字母数字下划线开头且不含运算符，认为是列名
		if isIdentifier(val) {
			return &ColumnExpr{Name: val}
		}
		return &LiteralExpr{Value: val}
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return &LiteralExpr{Value: val}
	case float32, float64:
		return &LiteralExpr{Value: val}
	case bool:
		return &LiteralExpr{Value: val}
	case nil:
		return &LiteralExpr{Value: nil}
	default:
		return &LiteralExpr{Value: val}
	}
}

func isIdentifier(s string) bool {
	if s == "" {
		return false
	}
	for i, c := range s {
		if c == '.' {
			// 允许带点的复合标识符（如 users.id）
			if i == 0 || i == len(s)-1 {
				return false
			}
			continue
		}
		if i == 0 && (c >= '0' && c <= '9') {
			return false // 不能以数字开头
		}
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') || c == '_') {
			return false
		}
	}
	return true
}

// ===== 辅助函数 =====

func toBoolForFilter(v interface{}) bool {
	switch val := v.(type) {
	case bool:
		return val
	case int:
		return val != 0
	case int64:
		return val != 0
	case float64:
		return val != 0
	case string:
		return val != "" && strings.ToLower(val) != "false"
	}
	return v != nil
}

func toStringForFilter(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

// matchLikeForFilter 简单 LIKE 匹配：% 匹配任意，_ 匹配单字符
func matchLikeForFilter(s, pattern string) bool {
	// 简化: 不支持转义；% 和 _ 是通配符
	// 递归匹配
	return matchLikeHelper(s, 0, pattern, 0)
}

func matchLikeHelper(s string, si int, p string, pi int) bool {
	for pi < len(p) {
		if p[pi] == '%' {
			// 跳过连续的 %
			for pi < len(p) && p[pi] == '%' {
				pi++
			}
			if pi == len(p) {
				return true
			}
			// 尝试匹配 %
			for si <= len(s) {
				if matchLikeHelper(s, si, p, pi) {
					return true
				}
				si++
			}
			return false
		}
		if si >= len(s) {
			return false
		}
		if p[pi] == '_' {
			si++
			pi++
		} else if s[si] == p[pi] {
			si++
			pi++
		} else {
			return false
		}
	}
	return si == len(s)
}

// ===== 字符串解析器（ParseWhere）=====

// ParseConditionExpression 解析一个条件表达式（用于 CASE WHEN / WHERE 等布尔上下文）。
// 与 ParseValueExpression 区别: bare identifier 优先解析为列引用，而非字符串字面量。
// 支持: =, !=, <, <=, >, >=, AND, OR, NOT, IN, LIKE, IS NULL, IS NOT NULL
func ParseConditionExpression(s string) (Expression, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, fmt.Errorf("empty condition")
	}

	// 顶层 OR
	if idx := findTopKeyword(s, "OR"); idx > 0 {
		left, err := ParseConditionExpression(s[:idx])
		if err != nil {
			return nil, err
		}
		right, err := ParseConditionExpression(s[idx+2:])
		if err != nil {
			return nil, err
		}
		return &BinaryExpr{Left: left, Right: right, Op: OpOr}, nil
	}
	// 次层 AND
	if idx := findTopKeyword(s, "AND"); idx > 0 {
		left, err := ParseConditionExpression(s[:idx])
		if err != nil {
			return nil, err
		}
		right, err := ParseConditionExpression(s[idx+3:])
		if err != nil {
			return nil, err
		}
		return &BinaryExpr{Left: left, Right: right, Op: OpAnd}, nil
	}
	// 括号
	if strings.HasPrefix(s, "(") {
		if matched := matchParen(s); matched == len(s)-1 {
			return ParseConditionExpression(s[1 : len(s)-1])
		}
	}
	// NOT
	upper := strings.ToUpper(s)
	if strings.HasPrefix(upper, "NOT ") {
		inner, err := ParseConditionExpression(s[4:])
		if err != nil {
			return nil, err
		}
		return &UnaryExpr{Op: "NOT", Operand: inner}, nil
	}
	// IS NULL / IS NOT NULL
	if strings.HasSuffix(upper, " IS NULL") {
		return &IsNullExpr{Column: strings.TrimSpace(s[:len(s)-8]), Negate: false}, nil
	}
	if strings.HasSuffix(upper, " IS NOT NULL") {
		return &IsNullExpr{Column: strings.TrimSpace(s[:len(s)-12]), Negate: true}, nil
	}
	// IN (a, b, c)
	if idx := strings.Index(upper, " IN ("); idx > 0 {
		col := strings.TrimSpace(s[:idx])
		rest := s[idx+4:]
		if end := findClosingParen(rest); end >= 0 {
			valuesStr := rest[1:end]
			values, err := parseValueList(valuesStr)
			if err != nil {
				return nil, err
			}
			return &InExpr{Column: col, Values: values}, nil
		}
	}
	// LIKE pattern
	if idx := strings.Index(upper, " LIKE "); idx > 0 {
		col := strings.TrimSpace(s[:idx])
		pattern := strings.TrimSpace(s[idx+6:])
		pattern = strings.Trim(pattern, `"'`)
		return &BinaryExpr{
			Left:  &ColumnExpr{Name: col},
			Right: &LiteralExpr{Value: pattern},
			Op:    OpLike,
		}, nil
	}
	// 二元比较
	for _, op := range []BinaryOp{OpLe, OpGe, OpNe, OpEq, OpLt, OpGt} {
		if idx := findOperator(s, string(op)); idx > 0 {
			left := strings.TrimSpace(s[:idx])
			right := strings.TrimSpace(s[idx+len(op):])
			return &BinaryExpr{
				Left:  toExprHelper(left),
				Right: toExprHelper(right),
				Op:    op,
			}, nil
		}
	}
	// 单独列引用
	return &ColumnExpr{Name: s}, nil
}

// ParseValueExpression 解析一个值表达式（用于 SELECT 投影）。
// 支持: CASE WHEN, COALESCE, NULLIF, CAST, 二元比较, 字面量, 列引用
// 顶层返回的是 Expression，可使用 Evaluate 求值。
func ParseValueExpression(s string) (Expression, error) {	s = strings.TrimSpace(s)
	if s == "" {
		return nil, fmt.Errorf("empty expression")
	}

	// CASE WHEN ... END
	upper := strings.ToUpper(s)
	if strings.HasPrefix(upper, "CASE ") || upper == "CASE" {
		return parseCaseWhenString(s)
	}
	// COALESCE(...)
	if strings.HasPrefix(upper, "COALESCE(") {
		return parseFuncCall(s, "COALESCE", func(args []Expression) (Expression, error) {
			return &CoalesceExpr{Args: args}, nil
		})
	}
	// NULLIF(...)
	if strings.HasPrefix(upper, "NULLIF(") {
		return parseNullIfString(s)
	}
	// CAST(... AS TYPE)
	if strings.HasPrefix(upper, "CAST(") {
		return parseCastString(s)
	}

	// 二元比较
	for _, op := range []BinaryOp{OpLe, OpGe, OpNe, OpEq, OpLt, OpGt} {
		if idx := findOperator(s, string(op)); idx > 0 {
			left := strings.TrimSpace(s[:idx])
			right := strings.TrimSpace(s[idx+len(op):])
			return &BinaryExpr{
				Left:  toValueExpr(left),
				Right: toValueExpr(right),
				Op:    op,
			}, nil
		}
	}
	// 单值
	return toValueExpr(s), nil
}

// parseCaseWhenString 解析 CASE WHEN 字符串
func parseCaseWhenString(s string) (Expression, error) {
	rest := strings.TrimSpace(s)
	if len(rest) >= 4 && strings.EqualFold(rest[:4], "CASE") {
		rest = rest[4:]
	}
	rest = strings.TrimSpace(rest)

	expr := &CaseWhenExpr{}
	hasInput := false
	// 简单 CASE（带输入）vs 搜索 CASE（无输入）
	if !strings.HasPrefix(strings.ToUpper(rest), "WHEN") {
		// 取到下一个 WHEN 之前作为 Input（Simple CASE 形式）
		whenIdx := indexOfKeyword(rest, "WHEN")
		if whenIdx < 0 {
			return nil, fmt.Errorf("CASE missing WHEN")
		}
		inputStr := strings.TrimSpace(rest[:whenIdx])
		// Simple CASE 的 input 必须是列引用（无引号标识符）
		expr.Input = &ColumnExpr{Name: inputStr}
		hasInput = true
		rest = strings.TrimSpace(rest[whenIdx:])
	}

	for strings.HasPrefix(strings.ToUpper(rest), "WHEN") {
		rest = strings.TrimSpace(rest[4:]) // 跳过 WHEN
		// 取到 THEN 之前作为条件
		thenIdx := indexOfKeyword(rest, "THEN")
		if thenIdx < 0 {
			return nil, fmt.Errorf("CASE WHEN missing THEN")
		}
		var cond Expression
		var err error
		if hasInput {
			// Simple CASE: WHEN 后的值是字面量
			cond, err = toValueExpr(strings.TrimSpace(rest[:thenIdx])), error(nil)
		} else {
			// Searched CASE: 条件是布尔表达式
			cond, err = ParseConditionExpression(strings.TrimSpace(rest[:thenIdx]))
		}
		if err != nil {
			return nil, err
		}
		rest = strings.TrimSpace(rest[thenIdx+4:]) // 跳过 THEN
		// 条件后的 THEN result 块取到下一个 WHEN / ELSE / END
		endIdx := nextCaseBoundary(rest)
		if endIdx < 0 {
			return nil, fmt.Errorf("CASE THEN missing END")
		}
		result, err := ParseValueExpression(strings.TrimSpace(rest[:endIdx]))
		if err != nil {
			return nil, err
		}
		expr.WhenClauses = append(expr.WhenClauses, CaseWhenClause{
			Condition: cond,
			Result:    result,
		})
		rest = strings.TrimSpace(rest[endIdx:])
	}

	if strings.HasPrefix(strings.ToUpper(rest), "ELSE") {
		rest = strings.TrimSpace(rest[4:]) // 跳过 ELSE
		endIdx := nextCaseBoundary(rest)
		var elsePart string
		if endIdx < 0 {
			elsePart = strings.TrimSuffix(strings.TrimSpace(rest), "END")
			rest = "END"
		} else {
			elsePart = strings.TrimSpace(rest[:endIdx])
			rest = strings.TrimSpace(rest[endIdx:])
		}
		elseVal, err := ParseValueExpression(elsePart)
		if err != nil {
			return nil, err
		}
		expr.ElseValue = elseVal
	}

	if !strings.HasPrefix(strings.ToUpper(rest), "END") {
		return nil, fmt.Errorf("CASE missing END, got: %q", rest)
	}
	return expr, nil
}

// indexOfKeyword 在顶层（括号外）查找大写关键字位置
func indexOfKeyword(s, kw string) int {
	upper := strings.ToUpper(s)
	depth := 0
	i := 0
	for i < len(upper) {
		c := upper[i]
		if c == '(' {
			depth++
		} else if c == ')' {
			depth--
		} else if depth == 0 && i+len(kw) <= len(upper) && upper[i:i+len(kw)] == kw {
			// 边界：前是空格/开始，后是空格/结束
			prevOK := i == 0 || upper[i-1] == ' '
			nextOK := i+len(kw) == len(upper) || upper[i+len(kw)] == ' '
			if prevOK && nextOK {
				return i
			}
		}
		i++
	}
	return -1
}

// nextCaseBoundary 返回下一个顶层 WHEN / ELSE / END 关键字位置
func nextCaseBoundary(s string) int {
	positions := []int{}
	for _, kw := range []string{"WHEN", "ELSE", "END"} {
		if idx := indexOfKeyword(s, kw); idx >= 0 {
			positions = append(positions, idx)
		}
	}
	if len(positions) == 0 {
		return -1
	}
	min := positions[0]
	for _, p := range positions[1:] {
		if p < min {
			min = p
		}
	}
	return min
}

// parseFuncCall 解析简单函数调用 FUNCNAME(arg1, arg2, ...)
func parseFuncCall(s, name string, build func(args []Expression) (Expression, error)) (Expression, error) {
	rest := strings.TrimSpace(s)
	// Case-insensitive prefix trim (without mutating rest)
	if len(rest) >= len(name) && strings.EqualFold(rest[:len(name)], name) {
		rest = rest[len(name):]
	}
	rest = strings.TrimSpace(rest)
	if !strings.HasPrefix(rest, "(") {
		return nil, fmt.Errorf("%s missing (", name)
	}
	if matched := matchParen(rest); matched != len(rest)-1 {
		return nil, fmt.Errorf("%s paren not matched", name)
	}
	inner := rest[1 : len(rest)-1]
	args, err := parseArgList(inner)
	if err != nil {
		return nil, err
	}
	return build(args)
}

func parseNullIfString(s string) (Expression, error) {
	rest := strings.TrimSpace(s)
	if len(rest) >= 6 && strings.EqualFold(rest[:6], "NULLIF") {
		rest = rest[6:]
	}
	rest = strings.TrimSpace(rest)
	if !strings.HasPrefix(rest, "(") {
		return nil, fmt.Errorf("NULLIF missing (")
	}
	if matched := matchParen(rest); matched != len(rest)-1 {
		return nil, fmt.Errorf("NULLIF paren not matched")
	}
	inner := rest[1 : len(rest)-1]
	args, err := parseArgList(inner)
	if err != nil {
		return nil, err
	}
	if len(args) != 2 {
		return nil, fmt.Errorf("NULLIF expects 2 args, got %d", len(args))
	}
	return &NullIfExpr{First: args[0], Second: args[1]}, nil
}

func parseCastString(s string) (Expression, error) {
	rest := strings.TrimSpace(s)
	if len(rest) >= 4 && strings.EqualFold(rest[:4], "CAST") {
		rest = rest[4:]
	}
	rest = strings.TrimSpace(rest)
	if !strings.HasPrefix(rest, "(") {
		return nil, fmt.Errorf("CAST missing (")
	}
	if matched := matchParen(rest); matched != len(rest)-1 {
		return nil, fmt.Errorf("CAST paren not matched")
	}
	inner := rest[1 : len(rest)-1]
	// 期望格式: expr AS TYPE
	asIdx := indexOfKeyword(inner, "AS")
	if asIdx < 0 {
		return nil, fmt.Errorf("CAST missing AS")
	}
	exprStr := strings.TrimSpace(inner[:asIdx])
	typeStr := strings.TrimSpace(inner[asIdx+2:])
	expr, err := ParseValueExpression(exprStr)
	if err != nil {
		return nil, err
	}
	return &CastExpr{Expr: expr, Type: typeStr}, nil
}

func parseArgList(s string) ([]Expression, error) {
	if strings.TrimSpace(s) == "" {
		return nil, nil
	}
	parts := splitValueArgs(s)
	out := make([]Expression, 0, len(parts))
	for _, p := range parts {
		// 函数参数：遵循 WQL 无双引号设计 - bare identifier 在值上下文视为字符串字面量
		out = append(out, toValueExpr(strings.TrimSpace(p)))
	}
	return out, nil
}

func splitValueArgs(s string) []string {
	var out []string
	depth := 0
	last := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			depth--
		case ',':
			if depth == 0 {
				out = append(out, s[last:i])
				last = i + 1
			}
		}
	}
	out = append(out, s[last:])
	return out
}
// 支持: =, !=, <, <=, >, >=, AND, OR, NOT, IN, LIKE, IS NULL, IS NOT NULL, BETWEEN
func ParseWhere(s string) (Expression, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, fmt.Errorf("empty expression")
	}

	// 顶层 OR
	if idx := findTopKeyword(s, "OR"); idx > 0 {
		left, err := ParseWhere(s[:idx])
		if err != nil {
			return nil, err
		}
		right, err := ParseWhere(s[idx+2:])
		if err != nil {
			return nil, err
		}
		return &BinaryExpr{Left: left, Right: right, Op: OpOr}, nil
	}

	// 次层 AND
	if idx := findTopKeyword(s, "AND"); idx > 0 {
		left, err := ParseWhere(s[:idx])
		if err != nil {
			return nil, err
		}
		right, err := ParseWhere(s[idx+3:])
		if err != nil {
			return nil, err
		}
		return &BinaryExpr{Left: left, Right: right, Op: OpAnd}, nil
	}

	// 括号
	if strings.HasPrefix(s, "(") {
		if matched := matchParen(s); matched == len(s)-1 {
			return ParseWhere(s[1:len(s)-1])
		}
	}

	// NOT
	if upper := strings.ToUpper(s); strings.HasPrefix(upper, "NOT ") {
		inner, err := ParseWhere(s[4:])
		if err != nil {
			return nil, err
		}
		return &UnaryExpr{Op: "NOT", Operand: inner}, nil
	}

	// IS NULL / IS NOT NULL
	upper := strings.ToUpper(s)
	if strings.HasSuffix(upper, " IS NULL") {
		return &IsNullExpr{Column: strings.TrimSpace(s[:len(s)-8]), Negate: false}, nil
	}
	if strings.HasSuffix(upper, " IS NOT NULL") {
		return &IsNullExpr{Column: strings.TrimSpace(s[:len(s)-12]), Negate: true}, nil
	}

	// IN (a, b, c)
	if idx := strings.Index(upper, " IN ("); idx > 0 {
		col := strings.TrimSpace(s[:idx])
		// rest 现在以 ( 开头
		rest := s[idx+4:] // 跳过 " IN "，保留 ( 开头
		if end := findClosingParen(rest); end >= 0 {
			valuesStr := rest[1:end] // 去掉首尾的括号
			values, err := parseValueList(valuesStr)
			if err != nil {
				return nil, err
			}
			return &InExpr{Column: col, Values: values}, nil
		}
	}

	// LIKE "pattern"
	if idx := strings.Index(upper, " LIKE "); idx > 0 {
		col := strings.TrimSpace(s[:idx])
		pattern := strings.TrimSpace(s[idx+6:])
		pattern = strings.Trim(pattern, `"'`)
		return &BinaryExpr{
			Left:  &ColumnExpr{Name: col},
			Right: &LiteralExpr{Value: pattern},
			Op:    OpLike,
		}, nil
	}

	// 二元比较
	for _, op := range []BinaryOp{OpLe, OpGe, OpNe, OpEq, OpLt, OpGt} {
		if idx := findOperator(s, string(op)); idx > 0 {
			left := strings.TrimSpace(s[:idx])
			right := strings.TrimSpace(s[idx+len(op):])
			return &BinaryExpr{
				Left:  toExprHelper(left),
				Right: toValueExpr(right),
				Op:    op,
			}, nil
		}
	}

	return nil, fmt.Errorf("cannot parse expression: %s", s)
}

// EvalBoolExpr 评估布尔表达式
func EvalBoolExpr(expr Expression, row map[string]interface{}) bool {
	v, err := expr.Evaluate(row)
	if err != nil {
		return false
	}
	return toBoolForFilter(v)
}

// ===== 字符串解析辅助 =====

// findTopKeyword 找到顶层关键字位置
func findTopKeyword(s, kw string) int {
	depth := 0
	upper := strings.ToUpper(s)
	kupper := strings.ToUpper(kw)
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			depth--
		case '"', '\'':
			quote := s[i]
			i++
			for i < len(s) && s[i] != quote {
				i++
			}
			continue
		}
		if depth == 0 && i+len(kupper) <= len(upper) {
			if upper[i:i+len(kupper)] == kupper {
				before := i == 0 || !isWordByte(upper[i-1])
				after := i+len(kupper) == len(upper) || !isWordByte(upper[i+len(kupper)])
				if before && after {
					return i
				}
			}
		}
	}
	return -1
}

func findOperator(s string, op string) int {
	depth := 0
	for i := len(s) - len(op); i >= 0; i-- {
		switch s[i] {
		case ')':
			depth++
		case '(':
			depth--
		case '"', '\'':
			continue
		}
		if depth == 0 && i+len(op) <= len(s) && s[i:i+len(op)] == op {
			before := i == 0 || !isWordByte(s[i-1])
			after := i+len(op) == len(s) || !isWordByte(s[i+len(op)])
			if before && after {
				return i
			}
		}
	}
	return -1
}

func isWordByte(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') || c == '_'
}

func matchParen(s string) int {
	depth := 0
	for i, c := range s {
		switch c {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func findClosingParen(s string) int {
	depth := 0
	for i, c := range s {
		switch c {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func parseValueList(s string) ([]interface{}, error) {
	parts := splitValues(s)
	out := make([]interface{}, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		v, err := parseValue(p)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

func splitValues(s string) []string {
	var out []string
	depth := 0
	start := 0
	inStr := byte(0)
	for i, c := range []byte(s) {
		if inStr != 0 {
			if c == inStr {
				inStr = 0
			}
			continue
		}
		switch c {
		case '"', '\'':
			inStr = c
		case '(':
			depth++
		case ')':
			depth--
		case ',':
			if depth == 0 {
				out = append(out, s[start:i])
				start = i + 1
			}
		}
	}
	if start < len(s) {
		out = append(out, s[start:])
	}
	return out
}

func parseValue(s string) (interface{}, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, fmt.Errorf("empty value")
	}
	if (strings.HasPrefix(s, `"`) && strings.HasSuffix(s, `"`)) ||
		(strings.HasPrefix(s, `'`) && strings.HasSuffix(s, `'`)) {
		return s[1 : len(s)-1], nil
	}
	upper := strings.ToUpper(s)
	if upper == "NULL" {
		return nil, nil
	}
	if upper == "TRUE" {
		return true, nil
	}
	if upper == "FALSE" {
		return false, nil
	}
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return i, nil
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f, nil
	}
	return s, nil
}

// toValueExpr 把字符串解析为 value 位置的表达式
// WQL 无双引号设计：在 value 位置上的 bare identifier 视为字符串字面量
// （除非是带点的复合标识符如 users.id，那视为列引用——但这种情况在 SQL 中不会用 = 右侧）
func toValueExpr(s string) Expression {
	s = strings.TrimSpace(s)
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') ||
			(s[0] == '\'' && s[len(s)-1] == '\'') {
			return &LiteralExpr{Value: s[1 : len(s)-1]}
		}
	}
	// 数字
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return &LiteralExpr{Value: i}
	}
	// 浮点
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return &LiteralExpr{Value: f}
	}
	// 布尔
	upper := strings.ToUpper(s)
	if upper == "TRUE" {
		return &LiteralExpr{Value: true}
	}
	if upper == "FALSE" {
		return &LiteralExpr{Value: false}
	}
	if upper == "NULL" {
		return &LiteralExpr{Value: nil}
	}
	// 复合列引用（table.col）
	if strings.Contains(s, ".") {
		return &ColumnExpr{Name: s}
	}
	// WQL 无双引号设计：bare identifier 视为字符串字面量
	return &LiteralExpr{Value: s}
}

// ===== 标量函数与 CASE 表达式 =====

// CoalesceExpr COALESCE(expr1, expr2, ...) — 返回第一个非空参数
type CoalesceExpr struct {
	Args []Expression
}

func (e *CoalesceExpr) Evaluate(row map[string]interface{}) (interface{}, error) {
	for _, arg := range e.Args {
		v, err := arg.Evaluate(row)
		if err != nil {
			return nil, err
		}
		if v != nil {
			return v, nil
		}
	}
	return nil, nil
}

func (e *CoalesceExpr) String() string {
	parts := make([]string, len(e.Args))
	for i, a := range e.Args {
		parts[i] = a.String()
	}
	return fmt.Sprintf("COALESCE(%s)", strings.Join(parts, ", "))
}

// NullIfExpr NULLIF(expr1, expr2) — 若两值相等返回 NULL，否则返回 expr1
type NullIfExpr struct {
	First  Expression
	Second Expression
}

func (e *NullIfExpr) Evaluate(row map[string]interface{}) (interface{}, error) {
	v1, err := e.First.Evaluate(row)
	if err != nil {
		return nil, err
	}
	v2, err := e.Second.Evaluate(row)
	if err != nil {
		return nil, err
	}
	if compareForFilter(v1, v2) == 0 {
		return nil, nil
	}
	return v1, nil
}

func (e *NullIfExpr) String() string {
	return fmt.Sprintf("NULLIF(%s, %s)", e.First.String(), e.Second.String())
}

// CastExpr CAST(expr AS TYPE) — 类型转换
type CastExpr struct {
	Expr Expression
	Type string
}

func (e *CastExpr) Evaluate(row map[string]interface{}) (interface{}, error) {
	v, err := e.Expr.Evaluate(row)
	if err != nil {
		return nil, err
	}
	if v == nil {
		return nil, nil
	}
	return castValue(v, e.Type)
}

func (e *CastExpr) String() string {
	return fmt.Sprintf("CAST(%s AS %s)", e.Expr.String(), e.Type)
}

// castValue 简单类型转换
func castValue(v interface{}, t string) (interface{}, error) {
	switch strings.ToUpper(t) {
	case "INTEGER", "INT":
		switch x := v.(type) {
		case int64:
			return x, nil
		case int:
			return int64(x), nil
		case float64:
			return int64(x), nil
		}
		if s, ok := v.(string); ok {
			return strconv.ParseInt(strings.TrimSpace(s), 10, 64)
		}
		return nil, fmt.Errorf("cannot cast %T to INTEGER", v)
	case "TEXT", "STRING", "VARCHAR":
		return fmt.Sprintf("%v", v), nil
	case "REAL", "FLOAT", "DOUBLE":
		switch x := v.(type) {
		case float64:
			return x, nil
		case int64:
			return float64(x), nil
		case int:
			return float64(x), nil
		}
		if s, ok := v.(string); ok {
			return strconv.ParseFloat(strings.TrimSpace(s), 64)
		}
		return nil, fmt.Errorf("cannot cast %T to REAL", v)
	}
	return v, nil
}

// CaseWhenExpr CASE WHEN cond THEN val ... [ELSE val] END
type CaseWhenExpr struct {
	// Optional: simple CASE input (CASE expr WHEN v THEN r)
	Input Expression
	// Searched CASE: list of WHEN conditions
	WhenClauses []CaseWhenClause
	ElseValue   Expression
}

type CaseWhenClause struct {
	Condition Expression
	Result    Expression
}

func (e *CaseWhenExpr) Evaluate(row map[string]interface{}) (interface{}, error) {
	inputVal := interface{}(nil)
	hasInput := false
	if e.Input != nil {
		v, err := e.Input.Evaluate(row)
		if err != nil {
			return nil, err
		}
		inputVal = v
		hasInput = true
	}
	for _, w := range e.WhenClauses {
		if hasInput {
			// Simple CASE: inputVal = w.Condition（字面量）
			cv, err := w.Condition.Evaluate(row)
			if err != nil {
				return nil, err
			}
			if compareForFilter(inputVal, cv) == 0 {
				return w.Result.Evaluate(row)
			}
		} else {
			// Searched CASE: w.Condition 是布尔表达式
			match, err := w.Condition.Evaluate(row)
			if err != nil {
				return nil, err
			}
			if toBoolForFilter(match) {
				return w.Result.Evaluate(row)
			}
		}
	}
	if e.ElseValue != nil {
		return e.ElseValue.Evaluate(row)
	}
	return nil, nil
}

func (e *CaseWhenExpr) String() string {
	var sb strings.Builder
	sb.WriteString("CASE")
	if e.Input != nil {
		sb.WriteString(" ")
		sb.WriteString(e.Input.String())
	}
	for _, w := range e.WhenClauses {
		sb.WriteString(" WHEN ")
		sb.WriteString(w.Condition.String())
		sb.WriteString(" THEN ")
		sb.WriteString(w.Result.String())
	}
	if e.ElseValue != nil {
		sb.WriteString(" ELSE ")
		sb.WriteString(e.ElseValue.String())
	}
	sb.WriteString(" END")
	return sb.String()
}
