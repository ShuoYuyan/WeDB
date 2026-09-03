package storage

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/wedb/wedb/internal/util"
)

// WhereClause represents a parsed WHERE clause
type WhereClause struct {
	Conditions []Condition
	Operator   LogicalOperator // AND or OR
	validColumns []string       // 允许的列名列表
}

// LogicalOperator represents AND or OR operator
type LogicalOperator string

const (
	LogicalAnd LogicalOperator = "AND"
	LogicalOr  LogicalOperator = "OR"
)

// Condition represents a single condition
type Condition struct {
	Column    string
	Operator  ComparisonOperator
	Value     interface{}
	Condition LogicalOperator // For nested conditions
}

// ComparisonOperator represents comparison operators
type ComparisonOperator string

const (
	OpEqual        ComparisonOperator = "="
	OpNotEqual     ComparisonOperator = "!="
	OpLessThan     ComparisonOperator = "<"
	OpLessEqual    ComparisonOperator = "<="
	OpGreaterThan  ComparisonOperator = ">"
	OpGreaterEqual ComparisonOperator = ">="
	OpLike         ComparisonOperator = "LIKE"
	OpNotLike      ComparisonOperator = "NOT LIKE"
	OpIn           ComparisonOperator = "IN"
	OpNotIn        ComparisonOperator = "NOT IN"
	OpBetween      ComparisonOperator = "BETWEEN"
	OpNotBetween   ComparisonOperator = "NOT BETWEEN"
	OpIsNull       ComparisonOperator = "IS NULL"
	OpIsNotNull    ComparisonOperator = "IS NOT NULL"
)

// ParseWhereClause parses a WHERE clause string
// validColumns is a list of valid column names to validate against
func ParseWhereClause(where string) (*WhereClause, error) {
	return ParseWhereClauseWithValidation(where, nil)
}

// ParseWhereClauseWithValidation parses a WHERE clause string with column validation
// validColumns is a list of valid column names to validate against
func ParseWhereClauseWithValidation(where string, validColumns []string) (*WhereClause, error) {
	// 验证WHERE条件
	if err := util.ValidateWhereClause(where); err != nil {
		return nil, err
	}

	if where == "" || where == "*" {
		return &WhereClause{
			Conditions:  []Condition{},
			Operator:    LogicalAnd,
			validColumns: validColumns,
		}, nil
	}

	// Split by AND/OR operators (simplified implementation)
	// This is a basic parser for simple WHERE clauses
	// For complex expressions, a full parser would be needed

	// Check for AND/OR operators
	andParts := splitByOperator(where, "AND")
	if len(andParts) > 1 {
		// AND condition
		whereClause := &WhereClause{
			Operator:     LogicalAnd,
			validColumns: validColumns,
		}
		for _, part := range andParts {
			cond, err := parseSingleCondition(strings.TrimSpace(part), validColumns)
			if err != nil {
				return nil, err
			}
			whereClause.Conditions = append(whereClause.Conditions, *cond)
		}
		return whereClause, nil
	}

	orParts := splitByOperator(where, "OR")
	if len(orParts) > 1 {
		// OR condition
		whereClause := &WhereClause{
			Operator:     LogicalOr,
			validColumns: validColumns,
		}
		for _, part := range orParts {
			cond, err := parseSingleCondition(strings.TrimSpace(part), validColumns)
			if err != nil {
				return nil, err
			}
			whereClause.Conditions = append(whereClause.Conditions, *cond)
		}
		return whereClause, nil
	}

	// Single condition
	cond, err := parseSingleCondition(strings.TrimSpace(where), validColumns)
	if err != nil {
		return nil, err
	}

	return &WhereClause{
		Conditions:  []Condition{*cond},
		Operator:    LogicalAnd,
		validColumns: validColumns,
	}, nil
}

// splitByOperator splits a string by operator, but not inside quotes, parentheses,
// or BETWEEN clauses (BETWEEN x AND y is one condition, not two).
func splitByOperator(s, op string) []string {
	var parts []string
	var current strings.Builder
	inParen := 0
	inStr := byte(0)
	// BETWEEN 状态机：当看到 "BETWEEN" 关键字时，set inBetween=true，
	// 直到再次出现 AND 关键字（作为 BETWEEN 的连接）才 reset。
	// 这里把 BETWEEN 视为一种"小括号"，将内层的 AND 吞下。
	betweenDepth := 0

	upper := strings.ToUpper(s)

	for i := 0; i < len(s); i++ {
		c := s[i]
		// 字符串字面量：吞下整个字面量
		if inStr != 0 {
			current.WriteByte(c)
			if c == inStr {
				inStr = 0
			}
			continue
		}
		if c == '"' || c == '\'' {
			inStr = c
			current.WriteByte(c)
			// 吞到匹配的引号
			i++
			for i < len(s) {
				ch := s[i]
				current.WriteByte(ch)
				if ch == '\\' && i+1 < len(s) {
					current.WriteByte(s[i+1])
					i += 2
					continue
				}
				if ch == inStr {
					inStr = 0
					break
				}
				i++
			}
			continue
		}
		// 普通括号
		if c == '(' {
			inParen++
			current.WriteByte(c)
			continue
		}
		if c == ')' {
			inParen--
			current.WriteByte(c)
			continue
		}
		// BETWEEN 关键字检测（顶层或括号外）
		if inParen == 0 && betweenDepth == 0 {
			// 检查是否是单词 "BETWEEN"
			beforeOK := i == 0 || !isIdentByteForOp(s[i-1])
			afterOK := i+7 >= len(s) || !isIdentByteForOp(s[i+7])
			if beforeOK && afterOK && i+7 <= len(s) && upper[i:i+7] == "BETWEEN" {
				betweenDepth = 1
				current.WriteString("BETWEEN")
				i += 6
				continue
			}
		}
		// 顶层 AND/OR 切分
		if inParen == 0 && betweenDepth == 0 && i+len(op) <= len(s) && upper[i:i+len(op)] == op {
			// 检查单词边界
			beforeOK := i == 0 || !isIdentByteForOp(s[i-1])
			afterOK := i+len(op) == len(s) || !isIdentByteForOp(s[i+len(op)])
			if beforeOK && afterOK {
				parts = append(parts, strings.TrimSpace(current.String()))
				current.Reset()
				i += len(op) - 1
				continue
			}
		}
		current.WriteByte(c)
		// BETWEEN 内部遇到 AND：消耗它（不切分）
		if betweenDepth > 0 && inParen == 0 && i+3 < len(s) && upper[i+1:i+4] == "AND" {
			// 当前字符 c 已写入；下一个位置开始的 3 字符是 AND
			// 验证 AND 单词边界
			after := i + 4
			if after == len(s) || !isIdentByteForOp(s[after]) {
				current.WriteString("AND")
				i += 3
				// BETWEEN 只消耗一个 AND 即可
				betweenDepth = 0
			}
		}
	}

	if current.Len() > 0 {
		parts = append(parts, strings.TrimSpace(current.String()))
	}

	return parts
}

// isOuterParenPair 报告 cond 最外层是否是一对匹配的圆括号（不考虑字符串内的括号）
func isOuterParenPair(cond string) bool {
	if len(cond) < 2 || cond[0] != '(' || cond[len(cond)-1] != ')' {
		return false
	}
	depth := 0
	inStr := byte(0)
	for i := 0; i < len(cond); i++ {
		c := cond[i]
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
			if depth == 0 {
				return i == len(cond)-1
			}
		}
	}
	return false
}

// parseSingleCondition parses a single condition like "age > 25"
func parseSingleCondition(cond string, validColumns []string) (*Condition, error) {
	cond = strings.TrimSpace(cond)
	// 处理 NOT(...) 一元否定：将 NOT(cond) 视为对 cond 取反
	// 通过两个子条件 + 顶层 OR 模拟? 简化做法：把 NOT 吸收到内部
	upper := strings.ToUpper(cond)
	if strings.HasPrefix(upper, "NOT(") && strings.HasSuffix(cond, ")") {
		// 提取 NOT 内的表达式
		inner := strings.TrimSpace(cond[4 : len(cond)-1])
		// 解析内层条件
		innerCond, err := parseSingleCondition(inner, validColumns)
		if err != nil {
			return nil, err
		}
		// NOT 语义：把内层算符取反
		switch innerCond.Operator {
		case OpEqual:
			innerCond.Operator = OpNotEqual
		case OpNotEqual:
			innerCond.Operator = OpEqual
		case OpLike:
			innerCond.Operator = OpNotLike
		case OpNotLike:
			innerCond.Operator = OpLike
		case OpIn:
			innerCond.Operator = OpNotIn
		case OpNotIn:
			innerCond.Operator = OpIn
		case OpBetween:
			innerCond.Operator = OpNotBetween
		case OpNotBetween:
			innerCond.Operator = OpBetween
		case OpIsNull:
			innerCond.Operator = OpIsNotNull
		case OpIsNotNull:
			innerCond.Operator = OpIsNull
		default:
			return nil, fmt.Errorf("NOT cannot be applied to operator %s", innerCond.Operator)
		}
		return innerCond, nil
	}
	// 去除外层包裹的圆括号（WQL 生成的条件常带括号）
	for len(cond) >= 2 && cond[0] == '(' && cond[len(cond)-1] == ')' {
		// 确保括号是配对的（避免剥离字符串字面量内的括号）
		if isOuterParenPair(cond) {
			cond = strings.TrimSpace(cond[1 : len(cond)-1])
		} else {
			break
		}
	}
	// 如果 paren-strip 后又变成 NOT(...)（说明原来是 ((NOT(...)))，重新处理
	upper = strings.ToUpper(cond)
	if strings.HasPrefix(upper, "NOT(") && strings.HasSuffix(cond, ")") {
		return parseSingleCondition(cond, validColumns) // 复用上面的 NOT 逻辑
	}

	// Try to parse each operator
	// 顺序：多字符算符先于短字符（避免 "=" 抢先匹配 "IN" 内的 "="）
	// 特殊算符：IS NULL / IS NOT NULL / BETWEEN 是单词边界敏感的
	operators := []ComparisonOperator{
		OpIsNotNull,
		OpIsNull,
		OpNotBetween,
		OpNotIn,
		OpNotLike,
		OpBetween,
		OpNotEqual,
		OpLessEqual,
		OpGreaterEqual,
		OpLessThan,
		OpGreaterThan,
		OpIn,
		OpLike,
		OpEqual,
	}

	for _, op := range operators {
		opStr := string(op)
		idx := findOperatorIndex(cond, opStr)
		if idx < 0 {
			continue
		}

		// IS NULL / IS NOT NULL 特殊处理
		if op == OpIsNull || op == OpIsNotNull {
			col := strings.TrimSpace(cond[:idx])
			if col == "" {
				continue
			}
			if validColumns != nil {
				if err := util.ValidateColumnNameInTable(col, validColumns); err != nil {
					return nil, err
				}
			} else {
				if err := util.ValidateColumnName(col); err != nil {
					return nil, err
				}
			}
			return &Condition{Column: col, Operator: op, Value: nil}, nil
		}

		// BETWEEN 特殊处理：col BETWEEN low AND high
		if op == OpBetween || op == OpNotBetween {
			col := strings.TrimSpace(cond[:idx])
			if col == "" {
				continue
			}
			rest := strings.TrimSpace(cond[idx+len(opStr):])
			// 期望格式：low AND high
			andIdx := strings.Index(strings.ToUpper(rest), " AND ")
			if andIdx < 0 {
				// 不是合法 BETWEEN，跳过
				continue
			}
			lowStr := strings.TrimSpace(rest[:andIdx])
			highStr := strings.TrimSpace(rest[andIdx+5:])
			low, err := parseValue(lowStr)
			if err != nil {
				return nil, fmt.Errorf("failed to parse BETWEEN low '%s': %w", lowStr, err)
			}
			high, err := parseValue(highStr)
			if err != nil {
				return nil, fmt.Errorf("failed to parse BETWEEN high '%s': %w", highStr, err)
			}
			if validColumns != nil {
				if err := util.ValidateColumnNameInTable(col, validColumns); err != nil {
					return nil, err
				}
			} else {
				if err := util.ValidateColumnName(col); err != nil {
					return nil, err
				}
			}
			return &Condition{Column: col, Operator: op, Value: []interface{}{low, high}}, nil
		}

		// 验证列名
		left := strings.TrimSpace(cond[:idx])
		right := strings.TrimSpace(cond[idx+len(opStr):])
		if validColumns != nil {
			if err := util.ValidateColumnNameInTable(left, validColumns); err != nil {
				return nil, err
			}
		} else {
			// 如果没有提供validColumns，至少验证列名格式
			if err := util.ValidateColumnName(left); err != nil {
				return nil, err
			}
		}

		// Parse value
		value, err := parseValue(right)
		if err != nil {
			return nil, fmt.Errorf("failed to parse value '%s': %w", right, err)
		}

		return &Condition{
			Column:   left,
			Operator: op,
			Value:    value,
		}, nil
	}

	return nil, fmt.Errorf("invalid condition: %s", cond)
}

// findOperatorIndex 找到 opStr 在 cond 中作为算符（而非子串）的位置。
// 算符必须以单词边界开头：要么 cond 开头，要么前面不是字母/数字/下划线。
// 算符必须以单词边界结束：要么 cond 结尾，要么后面不是字母/数字/下划线。
func findOperatorIndex(cond, opStr string) int {
	upper := strings.ToUpper(cond)
	idx := 0
	for {
		i := strings.Index(upper[idx:], opStr)
		if i < 0 {
			return -1
		}
		absIdx := idx + i
		// 检查前边界
		if absIdx > 0 {
			prev := cond[absIdx-1]
			if isIdentByteForOp(prev) {
				idx = absIdx + 1
				continue
			}
		}
		// 检查后边界
		endIdx := absIdx + len(opStr)
		if endIdx < len(cond) {
			next := cond[endIdx]
			if isIdentByteForOp(next) {
				idx = absIdx + 1
				continue
			}
		}
		return absIdx
	}
}

// isIdentByteForOp 报告 c 是否是标识符字符（用于算符边界判断）
func isIdentByteForOp(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') || c == '_'
}

// parseValue parses a value string into appropriate type
func parseValue(val string) (interface{}, error) {
	val = strings.TrimSpace(val)

	// Check if it's a quoted string
	if len(val) >= 2 && ((val[0] == '\'' && val[len(val)-1] == '\'') || (val[0] == '"' && val[len(val)-1] == '"')) {
		return val[1 : len(val)-1], nil
	}

	// Check if it's a number
	if intVal, err := strconv.ParseInt(val, 10, 64); err == nil {
		return intVal, nil
	}

	if floatVal, err := strconv.ParseFloat(val, 64); err == nil {
		return floatVal, nil
	}

	// Check if it's a boolean
	if strings.ToLower(val) == "true" {
		return true, nil
	}
	if strings.ToLower(val) == "false" {
		return false, nil
	}

	// Check if it's a list (for IN operator)
	if strings.HasPrefix(val, "(") && strings.HasSuffix(val, ")") {
		return parseList(val[1 : len(val)-1])
	}

	// Return as string
	return val, nil
}

// parseValue parses a list string (for IN operator)
func parseList(listStr string) ([]interface{}, error) {
	listStr = strings.TrimSpace(listStr)
	if listStr == "" {
		return []interface{}{}, nil
	}

	parts := strings.Split(listStr, ",")
	result := make([]interface{}, 0, len(parts))

	for _, part := range parts {
		val, err := parseValue(strings.TrimSpace(part))
		if err != nil {
			return nil, err
		}
		result = append(result, val)
	}

	return result, nil
}

// Evaluate evaluates a WHERE clause against a row
func (wc *WhereClause) Evaluate(row map[string]interface{}) (bool, error) {
	if len(wc.Conditions) == 0 {
		return true, nil
	}

	if wc.Operator == LogicalAnd {
		// All conditions must be true
		for _, cond := range wc.Conditions {
			matched, err := evaluateCondition(&cond, row)
			if err != nil {
				return false, err
			}
			if !matched {
				return false, nil
			}
		}
		return true, nil
	} else {
		// Any condition must be true
		for _, cond := range wc.Conditions {
			matched, err := evaluateCondition(&cond, row)
			if err != nil {
				return false, err
			}
			if matched {
				return true, nil
			}
		}
		return false, nil
	}
}

// evaluateCondition evaluates a single condition
func evaluateCondition(cond *Condition, row map[string]interface{}) (bool, error) {
	rowValue, exists := row[cond.Column]
	if !exists {
		// Column doesn't exist in row
		return false, nil
	}

	switch cond.Operator {
	case OpEqual:
		return compareEqual(rowValue, cond.Value), nil
	case OpNotEqual:
		return !compareEqual(rowValue, cond.Value), nil
	case OpLessThan:
		return compareLessThan(rowValue, cond.Value), nil
	case OpLessEqual:
		return compareLessThan(rowValue, cond.Value) || compareEqual(rowValue, cond.Value), nil
	case OpGreaterThan:
		return !compareLessThan(rowValue, cond.Value) && !compareEqual(rowValue, cond.Value), nil
	case OpGreaterEqual:
		return !compareLessThan(rowValue, cond.Value), nil
	case OpLike:
		result, err := matchLike(rowValue, cond.Value)
		if err != nil {
			return false, err
		}
		return result, nil
	case OpIn:
		return checkIn(rowValue, cond.Value), nil
	case OpNotIn:
		return !checkIn(rowValue, cond.Value), nil
	case OpNotLike:
		result, err := matchLike(rowValue, cond.Value)
		if err != nil {
			return false, err
		}
		return !result, nil
	case OpBetween, OpNotBetween:
		bounds, ok := cond.Value.([]interface{})
		if !ok || len(bounds) != 2 {
			return false, fmt.Errorf("BETWEEN expects [low, high] value pair")
		}
		low, high := bounds[0], bounds[1]
		// low <= val <= high
		geLow := !compareLessThan(rowValue, low)        // rowValue >= low
		leHigh := compareLessThan(rowValue, high) || compareEqual(rowValue, high)
		inRange := geLow && leHigh
		if cond.Operator == OpNotBetween {
			return !inRange, nil
		}
		return inRange, nil
	case OpIsNull:
		return rowValue == nil, nil
	case OpIsNotNull:
		return rowValue != nil, nil
	default:
		return false, fmt.Errorf("unsupported operator: %s", cond.Operator)
	}
}

// compareEqual compares two values for equality
func compareEqual(a, b interface{}) bool {
	aVal, aOk := a.(int64)
	bVal, bOk := b.(int64)
	if aOk && bOk {
		return aVal == bVal
	}

	aFloat, aOk := a.(float64)
	bFloat, bOk := b.(float64)
	if aOk && bOk {
		return aFloat == bFloat
	}

	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

// compareLessThan compares two values
func compareLessThan(a, b interface{}) bool {
	aInt, aOk := a.(int)
	bInt, bOk := b.(int)
	if aOk && bOk {
		return aInt < bInt
	}

	aInt64, aOk := a.(int64)
	bInt64, bOk := b.(int64)
	if aOk && bOk {
		return aInt64 < bInt64
	}

	aFloat, aOk := a.(float64)
	bFloat, bOk := b.(float64)
	if aOk && bOk {
		return aFloat < bFloat
	}

	return fmt.Sprintf("%v", a) < fmt.Sprintf("%v", b)
}

// matchLike implements LIKE pattern matching
func matchLike(value, pattern interface{}) (bool, error) {
	valueStr := fmt.Sprintf("%v", value)
	patternStr := fmt.Sprintf("%v", pattern)

	// 转义LIKE模式中的特殊字符，防止注入
	patternStr = util.EscapeLikePattern(patternStr)

	// Simple implementation: convert SQL LIKE pattern to regex
	// % matches any sequence of characters
	// _ matches any single character

	// 转义正则表达式特殊字符（除了LIKE通配符）
	// 先转义所有正则表达式特殊字符
	regexChars := []string{".", "+", "?", "|", "{", "}", "[", "]", "(", ")", "^", "$"}
	for _, char := range regexChars {
		patternStr = strings.ReplaceAll(patternStr, char, "\\"+char)
	}

	// 将LIKE通配符转换为正则表达式通配符
	// 注意：因为前面已经转义了\，所以需要特殊处理
	regex := strings.ReplaceAll(patternStr, "%", ".*")
	regex = strings.ReplaceAll(regex, "_", ".")
	regex = "^" + regex + "$"

	// 使用简单的字符串匹配，避免正则表达式相关的安全问题
	// 处理各种LIKE模式
	if strings.Contains(patternStr, "%") {
		// 模式包含%
		if patternStr == "%" {
			// 模式只是"%" - 匹配所有内容
			return true, nil
		}

		// 分割模式
		parts := strings.Split(patternStr, "%")

		// 处理各种情况
		if len(parts) == 2 && parts[0] == "" && parts[1] == "" {
			// "%"
			return true, nil
		} else if len(parts) == 2 && parts[0] == "" {
			// "%xxx" - 以xxx结尾
			return strings.HasSuffix(valueStr, parts[1]), nil
		} else if len(parts) == 2 && parts[1] == "" {
			// "xxx%" - 以xxx开头
			return strings.HasPrefix(valueStr, parts[0]), nil
		} else if len(parts) >= 2 {
			// "xxx%yyy" - 包含xxx和yyy，且顺序正确
			// 找到第一个非空部分
			for _, part := range parts {
				if part != "" {
					break
				}
			}

			// 检查是否匹配
			index := 0
			for _, part := range parts {
				if part == "" {
					continue
				}

				pos := strings.Index(valueStr[index:], part)
				if pos == -1 {
					return false, nil
				}
				index += pos + len(part)
			}

			return true, nil
		}
	}

	// 不包含通配符，直接比较
	return valueStr == patternStr, nil
}

// checkIn checks if value is in list
func checkIn(value, list interface{}) bool {
	listSlice, ok := list.([]interface{})
	if !ok {
		return false
	}

	for _, item := range listSlice {
		if compareEqual(value, item) {
			return true
		}
	}

	return false
}
