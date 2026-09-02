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
	OpIn           ComparisonOperator = "IN"
	OpNotIn        ComparisonOperator = "NOT IN"
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

// splitByOperator splits a string by operator, but not inside quotes or parentheses
func splitByOperator(s, op string) []string {
	var parts []string
	var current strings.Builder
	inParen := 0

	for i := 0; i < len(s); i++ {
		if s[i] == '(' {
			inParen++
			current.WriteByte(s[i])
		} else if s[i] == ')' {
			inParen--
			current.WriteByte(s[i])
		} else if inParen == 0 && i+len(op) <= len(s) && s[i:i+len(op)] == op {
			parts = append(parts, strings.TrimSpace(current.String()))
			current.Reset()
			i += len(op) - 1
		} else {
			current.WriteByte(s[i])
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
	// 去除外层包裹的圆括号（WQL 生成的条件常带括号）
	for len(cond) >= 2 && cond[0] == '(' && cond[len(cond)-1] == ')' {
		// 确保括号是配对的（避免剥离字符串字面量内的括号）
		if isOuterParenPair(cond) {
			cond = strings.TrimSpace(cond[1 : len(cond)-1])
		} else {
			break
		}
	}

	// Try to parse each operator
	operators := []ComparisonOperator{
		OpNotEqual,
		OpLessEqual,
		OpGreaterEqual,
		OpNotIn,
		OpLessThan,
		OpGreaterThan,
		OpEqual,
		OpIn,
		OpLike,
	}

	for _, op := range operators {
		opStr := string(op)
		idx := strings.Index(strings.ToUpper(cond), opStr)
		if idx != -1 {
			// Split by operator
			left := strings.TrimSpace(cond[:idx])
			right := strings.TrimSpace(cond[idx+len(opStr):])

			// 验证列名
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
	}

	return nil, fmt.Errorf("invalid condition: %s", cond)
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
