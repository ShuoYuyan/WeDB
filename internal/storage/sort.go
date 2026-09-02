package storage

import (
	"fmt"
	"sort"

	"github.com/wedb/wedb/internal/api"
)

// SortRows 对行数据进行排序
func SortRows(rows []map[string]interface{}, sortBy []api.SortBy) error {
	if len(rows) == 0 || len(sortBy) == 0 {
		return nil
	}

	// 验证所有列都存在
	for _, sb := range sortBy {
		if len(rows) > 0 {
			if _, exists := rows[0][sb.Column]; !exists {
				return fmt.Errorf("column not found: %s", sb.Column)
			}
		}
	}

	// 创建排序器
	sort.Slice(rows, func(i, j int) bool {
		for _, sb := range sortBy {
			valI := rows[i][sb.Column]
			valJ := rows[j][sb.Column]

			cmp := compareValues(valI, valJ)
			if cmp != 0 {
				// 根据排序方向返回
				if sb.Order == api.SortDesc {
					return cmp > 0
				}
				return cmp < 0
			}
		}
		return false
	})

	return nil
}

// compareValues 比较两个值
// 返回 -1 if a < b, 0 if a == b, 1 if a > b
func compareValues(a, b interface{}) int {
	// 处理 nil 值
	if a == nil && b == nil {
		return 0
	}
	if a == nil {
		return -1
	}
	if b == nil {
		return 1
	}

	// 尝试比较整数
	switch aVal := a.(type) {
	case int:
		if bVal, ok := b.(int); ok {
			if aVal < bVal {
				return -1
			} else if aVal > bVal {
				return 1
			}
			return 0
		}
	case int64:
		if bVal, ok := b.(int64); ok {
			if aVal < bVal {
				return -1
			} else if aVal > bVal {
				return 1
			}
			return 0
		}
		// 尝试将 int64 转换为 float64 进行比较
		if bVal, ok := b.(float64); ok {
			aFloat := float64(aVal)
			if aFloat < bVal {
				return -1
			} else if aFloat > bVal {
				return 1
			}
			return 0
		}
	case float64:
		if bVal, ok := b.(float64); ok {
			if aVal < bVal {
				return -1
			} else if aVal > bVal {
				return 1
			}
			return 0
		}
		// 尝试将 float64 与 int64 比较
		if bVal, ok := b.(int64); ok {
			bFloat := float64(bVal)
			if aVal < bFloat {
				return -1
			} else if aVal > bFloat {
				return 1
			}
			return 0
		}
	}

	// 尝试比较字符串
	aStr := fmt.Sprintf("%v", a)
	bStr := fmt.Sprintf("%v", b)

	if aStr < bStr {
		return -1
	} else if aStr > bStr {
		return 1
	}
	return 0
}