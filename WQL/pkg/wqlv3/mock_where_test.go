package wqlv3

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// filterByConditionWithNewOps 在 mockAdapter 中按 WHERE 条件过滤行。
// 与 storage WHERE 解析器保持一致：支持 =, !=, <, <=, >, >=, AND, OR, NOT, IN, NOT IN,
// BETWEEN, NOT BETWEEN, LIKE, NOT LIKE, IS NULL, IS NOT NULL。
// 不支持算符优先级/嵌套表达式的高级用例（足以覆盖 WQL 单元测试）。
func filterByConditionWithNewOps(rows []map[string]interface{}, cond string) []map[string]interface{} {
	// 顶层去除外层括号（仅一层）
	cond = strings.TrimSpace(cond)
	if len(cond) >= 2 && cond[0] == '(' && cond[len(cond)-1] == ')' {
		cond = strings.TrimSpace(cond[1 : len(cond)-1])
	}
	// 先拆分顶层 AND
	parts := splitByTopLevelOp(cond, " AND ")
	if len(parts) > 1 {
		out := rows
		for _, p := range parts {
			out = filterByConditionWithNewOps(out, p)
		}
		return out
	}
	// 再拆分顶层 OR
	parts = splitByTopLevelOp(cond, " OR ")
	if len(parts) > 1 {
		seen := make(map[string]bool)
		out := make([]map[string]interface{}, 0)
		for _, p := range parts {
			for _, r := range filterByConditionWithNewOps(rows, p) {
				key := fmt.Sprintf("%p", r)
				if seen[key] {
					continue
				}
				seen[key] = true
				out = append(out, r)
			}
		}
		return out
	}
	// 单个条件
	return filterBySingleCondition(rows, cond)
}

func splitByTopLevelOp(cond, op string) []string {
	upper := strings.ToUpper(cond)
	parts := []string{}
	var current strings.Builder
	inStr := byte(0)
	parenDepth := 0
	for i := 0; i < len(cond); i++ {
		c := cond[i]
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
			continue
		}
		if c == '(' {
			parenDepth++
			current.WriteByte(c)
			continue
		}
		if c == ')' {
			parenDepth--
			current.WriteByte(c)
			continue
		}
		// BETWEEN 内的 AND 不切分
		if op == " AND " && isInBetween(upper, i) {
			current.WriteByte(c)
			continue
		}
		// 只在顶层（parenDepth==0）切分
		if parenDepth == 0 && i+len(op) <= len(cond) && upper[i:i+len(op)] == op {
			parts = append(parts, strings.TrimSpace(current.String()))
			current.Reset()
			i += len(op) - 1
			continue
		}
		current.WriteByte(c)
	}
	if current.Len() > 0 {
		parts = append(parts, strings.TrimSpace(current.String()))
	}
	// 对每个部分去除外层括号
	for i, p := range parts {
		pp := strings.TrimSpace(p)
		for len(pp) >= 2 && pp[0] == '(' && pp[len(pp)-1] == ')' {
			pp = strings.TrimSpace(pp[1 : len(pp)-1])
		}
		parts[i] = pp
	}
	return parts
}

// isInBetween 报告位置 i 是否在 BETWEEN 子句的 AND 与其配对 AND 之间
// 简化：从头扫描到 i，统计未配对的 BETWEEN 关键字数
func isInBetween(upper string, i int) bool {
	betweenDepth := 0
	parenDepth := 0
	j := 0
	for j < i {
		// 跳过字符串字面量
		if upper[j] == '"' || upper[j] == '\'' {
			q := upper[j]
			j++
			for j < i && upper[j] != q {
				j++
			}
			j++
			continue
		}
		// 跳过括号（不影响 BETWEEN 深度）
		if upper[j] == '(' {
			parenDepth++
			j++
			continue
		}
		if upper[j] == ')' {
			parenDepth--
			j++
			continue
		}
		// 单词检测
		wordBoundary := j == 0 || !isIdentByteMock(upper[j-1])
		if wordBoundary && j+7 <= i && upper[j:j+7] == "BETWEEN" {
			if j+7 >= i || !isIdentByteMock(upper[j+7]) {
				betweenDepth++
				j += 7
				continue
			}
		}
		wordBoundary = j == 0 || !isIdentByteMock(upper[j-1])
		if wordBoundary && j+3 <= i && upper[j:j+3] == "AND" {
			if j+3 >= i || !isIdentByteMock(upper[j+3]) {
				if betweenDepth > 0 {
					betweenDepth--
				}
				j += 3
				continue
			}
		}
		j++
	}
	return betweenDepth > 0
}

func isWordBoundaryMock(s string, start, length int) bool {
	if start > 0 {
		prev := s[start-1]
		if isIdentByteMock(prev) {
			return false
		}
	}
	end := start + length
	if end < len(s) {
		next := s[end]
		if isIdentByteMock(next) {
			return false
		}
	}
	return true
}

func isIdentByteMock(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') || c == '_'
}

func filterBySingleCondition(rows []map[string]interface{}, cond string) []map[string]interface{} {
	cond = strings.TrimSpace(cond)
	// 去除外层括号
	for len(cond) >= 2 && cond[0] == '(' && cond[len(cond)-1] == ')' {
		cond = strings.TrimSpace(cond[1 : len(cond)-1])
	}
	// NOT(expr)
	upper := strings.ToUpper(cond)
	if strings.HasPrefix(upper, "NOT(") && strings.HasSuffix(cond, ")") {
		inner := cond[4 : len(cond)-1]
		matched := filterBySingleCondition(rows, inner)
		matchSet := make(map[string]bool)
		for _, m := range matched {
			matchSet[fmt.Sprintf("%p", m)] = true
		}
		out := make([]map[string]interface{}, 0)
		for _, r := range rows {
			if !matchSet[fmt.Sprintf("%p", r)] {
				out = append(out, r)
			}
		}
		return out
	}
	// IS NULL / IS NOT NULL
	if idx := findWordIndex(upper, " IS NOT NULL"); idx >= 0 {
		col := strings.TrimSpace(cond[:idx])
		out := make([]map[string]interface{}, 0)
		for _, r := range rows {
			if v, ok := r[col]; ok && v != nil {
				out = append(out, r)
			}
		}
		return out
	}
	if idx := findWordIndex(upper, " IS NULL"); idx >= 0 {
		col := strings.TrimSpace(cond[:idx])
		out := make([]map[string]interface{}, 0)
		for _, r := range rows {
			v, ok := r[col]
			if !ok || v == nil {
				out = append(out, r)
			}
		}
		return out
	}
	// BETWEEN / NOT BETWEEN
	if idx := findWordIndex(upper, " NOT BETWEEN "); idx >= 0 {
		col := strings.TrimSpace(cond[:idx])
		rest := strings.TrimSpace(cond[idx+13:])
		bounds := parseBetweenBounds(rest)
		if bounds != nil {
			out := make([]map[string]interface{}, 0)
			for _, r := range rows {
				v, ok := r[col]
				if !ok {
					continue
				}
				if !betweenInclusive(v, bounds[0], bounds[1]) {
					out = append(out, r)
				}
			}
			return out
		}
	}
	if idx := findWordIndex(upper, " BETWEEN "); idx >= 0 {
		col := strings.TrimSpace(cond[:idx])
		rest := strings.TrimSpace(cond[idx+9:])
		bounds := parseBetweenBounds(rest)
		if bounds != nil {
			out := make([]map[string]interface{}, 0)
			for _, r := range rows {
				v, ok := r[col]
				if !ok {
					continue
				}
				if betweenInclusive(v, bounds[0], bounds[1]) {
					out = append(out, r)
				}
			}
			return out
		}
	}
	// NOT LIKE
	if idx := findWordIndex(upper, " NOT LIKE "); idx >= 0 {
		col := strings.TrimSpace(cond[:idx])
		pat := strings.TrimSpace(cond[idx+10:])
		pat = unquote(pat)
		out := make([]map[string]interface{}, 0)
		for _, r := range rows {
			v := fmt.Sprintf("%v", r[col])
			if !likeMatch(v, pat) {
				out = append(out, r)
			}
		}
		return out
	}
	// LIKE
	if idx := findWordIndex(upper, " LIKE "); idx >= 0 {
		col := strings.TrimSpace(cond[:idx])
		pat := strings.TrimSpace(cond[idx+6:])
		pat = unquote(pat)
		out := make([]map[string]interface{}, 0)
		for _, r := range rows {
			v := fmt.Sprintf("%v", r[col])
			if likeMatch(v, pat) {
				out = append(out, r)
			}
		}
		return out
	}
	// NOT IN
	if idx := findWordIndex(upper, " NOT IN "); idx >= 0 {
		col := strings.TrimSpace(cond[:idx])
		rest := strings.TrimSpace(cond[idx+8:])
		vals := parseList(rest)
		matchSet := make(map[string]bool, len(vals))
		for _, v := range vals {
			matchSet[fmt.Sprintf("%v", v)] = true
		}
		out := make([]map[string]interface{}, 0)
		for _, r := range rows {
			key := fmt.Sprintf("%v", r[col])
			if !matchSet[key] {
				out = append(out, r)
			}
		}
		return out
	}
	// IN
	idxIn := findWordIndex(upper, " IN ")
	if idxIn >= 0 {
		col := strings.TrimSpace(cond[:idxIn])
		rest := strings.TrimSpace(cond[idxIn+4:])
		vals := parseList(rest)
		matchSet := make(map[string]bool, len(vals))
		for _, v := range vals {
			matchSet[fmt.Sprintf("%v", v)] = true
		}
		out := make([]map[string]interface{}, 0)
		for _, r := range rows {
			key := fmt.Sprintf("%v", r[col])
			if matchSet[key] {
				out = append(out, r)
			}
		}
		return out
	}
	// 简单二元比较
	ops := []struct {
		str string
		fn  func(a, b interface{}) bool
		neg bool
	}{
		{"!=", func(a, b interface{}) bool { return !cmpEq(a, b) }, false},
		{"<>", func(a, b interface{}) bool { return !cmpEq(a, b) }, false},
		{"<=", func(a, b interface{}) bool { return cmpLt(a, b) || cmpEq(a, b) }, false},
		{">=", func(a, b interface{}) bool { return !cmpLt(a, b) && !cmpEq(a, b) }, true}, // special
		{">", func(a, b interface{}) bool { return !cmpLt(a, b) && !cmpEq(a, b) }, false},
		{"<", func(a, b interface{}) bool { return cmpLt(a, b) }, false},
		{"=", func(a, b interface{}) bool { return cmpEq(a, b) }, false},
	}
	for _, op := range ops {
		if idx := strings.Index(cond, op.str); idx >= 0 {
			col := strings.TrimSpace(cond[:idx])
			valStr := strings.TrimSpace(cond[idx+len(op.str):])
			val := parseSingleValue(valStr)
			out := make([]map[string]interface{}, 0)
			for _, r := range rows {
				v, ok := r[col]
				if !ok {
					continue
				}
				if op.fn(v, val) {
					out = append(out, r)
				}
			}
			return out
		}
	}
	return rows
}

func findWordIndex(s, target string) int {
	// target 可能形如 " BETWEEN "（前后有空格），"NOT BETWEEN"（仅尾部空格），或纯关键字。
	// 单词边界检查：计算 target 内部的"关键词"区间（去掉前后空格），并对关键词做边界判定。
	stripLeading := 0
	stripTrailing := 0
	for stripLeading < len(target) && target[stripLeading] == ' ' {
		stripLeading++
	}
	for stripTrailing < len(target)-stripLeading && target[len(target)-1-stripTrailing] == ' ' {
		stripTrailing++
	}
	wordStart := 0
	wordLen := len(target)
	if stripLeading > 0 {
		wordStart = stripLeading
		wordLen = len(target) - stripLeading - stripTrailing
	}
	start := 0
	for {
		idx := strings.Index(s[start:], target)
		if idx < 0 {
			return -1
		}
		abs := start + idx
		if isWordBoundaryMock(s, abs+wordStart, wordLen) {
			return abs
		}
		start = abs + 1
	}
}

func parseList(s string) []interface{} {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "(") && strings.HasSuffix(s, ")") {
		s = s[1 : len(s)-1]
	}
	parts := splitTopLevel(s, ",")
	out := make([]interface{}, 0, len(parts))
	for _, p := range parts {
		out = append(out, parseSingleValue(strings.TrimSpace(p)))
	}
	return out
}

func splitTopLevel(s, sep string) []string {
	parts := []string{}
	var current strings.Builder
	inStr := byte(0)
	for i := 0; i < len(s); i++ {
		c := s[i]
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
			continue
		}
		if c == '(' {
			// 不切分
			depth := 1
			current.WriteByte(c)
			i++
			for i < len(s) && depth > 0 {
				cc := s[i]
				if cc == '(' {
					depth++
				} else if cc == ')' {
					depth--
				}
				current.WriteByte(cc)
				i++
			}
			i--
			continue
		}
		if i+len(sep) <= len(s) && s[i:i+len(sep)] == sep {
			parts = append(parts, strings.TrimSpace(current.String()))
			current.Reset()
			i += len(sep) - 1
			continue
		}
		current.WriteByte(c)
	}
	if current.Len() > 0 {
		parts = append(parts, strings.TrimSpace(current.String()))
	}
	return parts
}

func parseBetweenBounds(rest string) []interface{} {
	upper := strings.ToUpper(rest)
	idx := strings.Index(upper, " AND ")
	if idx < 0 {
		return nil
	}
	low := parseSingleValue(strings.TrimSpace(rest[:idx]))
	high := parseSingleValue(strings.TrimSpace(rest[idx+5:]))
	return []interface{}{low, high}
}

func betweenInclusive(v, low, high interface{}) bool {
	return !cmpLt(v, low) && (cmpLt(v, high) || cmpEq(v, high))
}

func likeMatch(value, pattern string) bool {
	// 把 LIKE 模式转为正则：% = .*, _ = .
	regexStr := "^"
	for i := 0; i < len(pattern); i++ {
		c := pattern[i]
		switch c {
		case '%':
			regexStr += ".*"
		case '_':
			regexStr += "."
		case '.', '+', '*', '?', '(', ')', '[', ']', '{', '}', '|', '\\', '^', '$':
			regexStr += "\\" + string(c)
		default:
			regexStr += string(c)
		}
	}
	regexStr += "$"
	re, err := regexp.Compile(regexStr)
	if err != nil {
		return false
	}
	return re.MatchString(value)
}

func unquote(s string) string {
	if len(s) >= 2 && (s[0] == '"' && s[len(s)-1] == '"' || s[0] == '\'' && s[len(s)-1] == '\'') {
		return s[1 : len(s)-1]
	}
	return s
}

func parseSingleValue(s string) interface{} {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && (s[0] == '"' && s[len(s)-1] == '"' || s[0] == '\'' && s[len(s)-1] == '\'') {
		return s[1 : len(s)-1]
	}
	if v, err := strconv.ParseInt(s, 10, 64); err == nil {
		return v
	}
	if v, err := strconv.ParseFloat(s, 64); err == nil {
		return v
	}
	if s == "true" {
		return true
	}
	if s == "false" {
		return false
	}
	return s
}

func cmpEq(a, b interface{}) bool {
	af, aok := toFloat(a)
	bf, bok := toFloat(b)
	if aok && bok {
		return af == bf
	}
	ai, aok := a.(int64)
	bi, bok := b.(int64)
	if aok && bok {
		return ai == bi
	}
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

func cmpLt(a, b interface{}) bool {
	af, aok := toFloat(a)
	bf, bok := toFloat(b)
	if aok && bok {
		return af < bf
	}
	return fmt.Sprintf("%v", a) < fmt.Sprintf("%v", b)
}

func toFloatMock(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case int:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	case float32:
		return float64(n), true
	case float64:
		return n, true
	}
	return 0, false
}
