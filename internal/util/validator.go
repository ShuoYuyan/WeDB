package util

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

// SQL关键字列表
var sqlKeywords = map[string]bool{
	"SELECT": true, "INSERT": true, "UPDATE": true, "DELETE": true,
	"DROP": true, "CREATE": true, "ALTER": true, "TRUNCATE": true,
	"WHERE": true, "FROM": true, "JOIN": true, "UNION": true,
	"AND": true, "OR": true, "NOT": true, "NULL": true,
	"TRUE": true, "FALSE": true, "DISTINCT": true, "ALL": true,
	"EXISTS": true, "BETWEEN": true, "IN": true, "LIKE": true,
	"IS": true, "LIMIT": true, "OFFSET": true, "ORDER": true,
	"BY": true, "GROUP": true, "HAVING": true, "CASE": true,
	"WHEN": true, "THEN": true, "ELSE": true, "END": true,
	"AS": true, "ASC": true, "DESC": true, "INNER": true,
	"OUTER": true, "LEFT": true, "RIGHT": true, "FULL": true,
	"CROSS": true, "NATURAL": true, "USING": true,
	"ON": true, "INTO": true, "VALUES": true, "SET": true,
	"PRIMARY": true, "KEY": true, "FOREIGN": true, "REFERENCES": true,
	"CONSTRAINT": true, "UNIQUE": true, "INDEX": true, "TABLE": true,
	"DATABASE": true, "SCHEMA": true, "VIEW": true, "TRIGGER": true,
	"PROCEDURE": true, "FUNCTION": true, "PRAGMA": true,
}

// 正则表达式：匹配有效的标识符（表名、列名）
// 只允许字母、数字、下划线，必须以字母或下划线开头
var validIdentifierRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// 最大长度限制
const (
	MaxTableNameLength    = 64
	MaxColumnNameLength   = 64
	MaxIndexNameLength    = 64
	MaxWhereClauseLength  = 10 * 1024 // 10KB
	MaxConditionCount     = 100
	MaxRecursionDepth     = 10
)

// ValidateIdentifier 验证标识符（表名、列名、索引名）
func ValidateIdentifier(name string, identifierType string) error {
	// 检查空值
	if name == "" {
		return fmt.Errorf("%s cannot be empty", identifierType)
	}

	// 检查长度
	maxLength := MaxColumnNameLength
	switch identifierType {
	case "table name":
		maxLength = MaxTableNameLength
	case "index name":
		maxLength = MaxIndexNameLength
	}
	if len(name) > maxLength {
		return fmt.Errorf("%s too long: maximum %d characters", identifierType, maxLength)
	}

	// 检查格式（只允许字母、数字、下划线）
	if !validIdentifierRegex.MatchString(name) {
		return fmt.Errorf("invalid %s format: only letters, numbers, and underscores are allowed", identifierType)
	}

	// 检查SQL关键字（不区分大小写）
	if sqlKeywords[strings.ToUpper(name)] {
		return fmt.Errorf("%s cannot be a SQL keyword", identifierType)
	}

	return nil
}

// ValidateTableName 验证表名
func ValidateTableName(name string) error {
	return ValidateIdentifier(name, "table name")
}

// ValidateColumnName 验证列名
func ValidateColumnName(name string) error {
	return ValidateIdentifier(name, "column name")
}

// ValidateIndexName 验证索引名
func ValidateIndexName(name string) error {
	return ValidateIdentifier(name, "index name")
}

// ValidateWhereClause 验证WHERE条件
func ValidateWhereClause(where string) error {
	// 检查空值（空WHERE条件是允许的）
	if where == "" || where == "*" {
		return nil
	}

	// 检查长度
	if len(where) > MaxWhereClauseLength {
		return fmt.Errorf("WHERE clause too long: maximum %d characters", MaxWhereClauseLength)
	}

	// 检查危险字符（防止注入）
	dangerousChars := []string{";", "--", "/*", "*/", "\x00"}
	for _, char := range dangerousChars {
		if strings.Contains(where, char) {
			return fmt.Errorf("WHERE clause contains dangerous characters")
		}
	}

	return nil
}

// ValidateColumnNameInTable 验证列名是否存在于表中
func ValidateColumnNameInTable(columnName string, validColumns []string) error {
	// 检查列名格式
	if err := ValidateColumnName(columnName); err != nil {
		return err
	}

	// 检查列名是否在有效列名列表中
	for _, col := range validColumns {
		if col == columnName {
			return nil
		}
	}

	return fmt.Errorf("column '%s' does not exist in table", columnName)
}

// EscapeLikePattern 转义LIKE模式中的特殊字符
func EscapeLikePattern(pattern string) string {
	// 转义LIKE通配符
	// 如果用户想要匹配字面的%或_，应该使用\转义
	// 这里我们假设输入是用户提供的原始模式

	// 先处理转义字符
	var result strings.Builder
	escape := false

	for i, ch := range pattern {
		if escape {
			// 前一个字符是转义字符，直接添加当前字符
			result.WriteRune(ch)
			escape = false
			continue
		}

		if ch == '\\' {
			// 转义字符，标记下一个字符需要转义
			escape = true
			result.WriteRune(ch)
			continue
		}

		// 检查是否是LIKE通配符
		if ch == '%' || ch == '_' {
			// 检查是否已经被转义
			if i > 0 && pattern[i-1] == '\\' {
				result.WriteRune(ch)
			} else {
				// 通配符，直接添加
				result.WriteRune(ch)
			}
		} else if unicode.IsControl(ch) {
			// 控制字符，跳过
			continue
		} else {
			result.WriteRune(ch)
		}
	}

	return result.String()
}

// SanitizeString 清理字符串，移除危险字符
func SanitizeString(input string) string {
	// 移除空字节
	result := strings.ReplaceAll(input, "\x00", "")

	// 移除其他危险字符
	dangerousChars := []string{"\r", "\n", "\t"}
	for _, char := range dangerousChars {
		result = strings.ReplaceAll(result, char, "")
	}

	return result
}

// ValidateInteger 验证整数
func ValidateInteger(value interface{}) (int64, error) {
	switch v := value.(type) {
	case int:
		return int64(v), nil
	case int8:
		return int64(v), nil
	case int16:
		return int64(v), nil
	case int32:
		return int64(v), nil
	case int64:
		return v, nil
	case uint:
		return int64(v), nil
	case uint8:
		return int64(v), nil
	case uint16:
		return int64(v), nil
	case uint32:
		return int64(v), nil
	case uint64:
		if v > 0x7FFFFFFFFFFFFFFF {
			return 0, fmt.Errorf("integer overflow")
		}
		return int64(v), nil
	case float32:
		return int64(v), nil
	case float64:
		return int64(v), nil
	case string:
		// 尝试解析字符串
		var result int64
		_, err := fmt.Sscanf(v, "%d", &result)
		if err != nil {
			return 0, fmt.Errorf("invalid integer: %s", v)
		}
		return result, nil
	default:
		return 0, fmt.Errorf("invalid integer type")
	}
}

// ValidateStringLength 验证字符串长度
func ValidateStringLength(value string, maxLength int) error {
	if len(value) > maxLength {
		return fmt.Errorf("string too long: maximum %d characters", maxLength)
	}
	return nil
}

// ContainsDangerousCharacters 检查字符串是否包含危险字符
func ContainsDangerousCharacters(input string) bool {
	dangerousChars := []string{";", "--", "/*", "*/", "\x00", "\r", "\n"}
	for _, char := range dangerousChars {
		if strings.Contains(input, char) {
			return true
		}
	}
	return false
}
