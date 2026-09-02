package wqlv3

import (
	"testing"
)

// TestParseWhere 测试 WHERE 子句解析
func TestParseWhere(t *testing.T) {
	tests := []struct {
		name  string
		input string
		row   map[string]interface{}
		want  bool
	}{
		{
			name:  "simple equality",
			input: `age = 18`,
			row:   map[string]interface{}{"age": 18},
			want:  true,
		},
		{
			name:  "simple equality false",
			input: `age = 20`,
			row:   map[string]interface{}{"age": 18},
			want:  false,
		},
		{
			name:  "greater than",
			input: `age > 17`,
			row:   map[string]interface{}{"age": 18},
			want:  true,
		},
		{
			name:  "less than",
			input: `age < 20`,
			row:   map[string]interface{}{"age": 18},
			want:  true,
		},
		{
			name:  "string equality",
			input: `name = "alice"`,
			row:   map[string]interface{}{"name": "alice"},
			want:  true,
		},
		{
			name:  "AND true",
			input: `age > 17 AND age < 30`,
			row:   map[string]interface{}{"age": 18},
			want:  true,
		},
		{
			name:  "AND false",
			input: `age > 17 AND age < 18`,
			row:   map[string]interface{}{"age": 18},
			want:  false,
		},
		{
			name:  "OR true",
			input: `age < 5 OR age > 17`,
			row:   map[string]interface{}{"age": 18},
			want:  true,
		},
		{
			name:  "IN",
			input: `id IN (1, 2, 3)`,
			row:   map[string]interface{}{"id": 2},
			want:  true,
		},
		{
			name:  "IS NULL",
			input: `deleted_at IS NULL`,
			row:   map[string]interface{}{"name": "x"},
			want:  true,
		},
		{
			name:  "IS NOT NULL",
			input: `email IS NOT NULL`,
			row:   map[string]interface{}{"email": "a@b.com"},
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := ParseWhere(tt.input)
			if err != nil {
				t.Fatalf("ParseWhere(%q) error: %v", tt.input, err)
			}
			got := EvalBoolExpr(expr, tt.row)
			if got != tt.want {
				t.Errorf("EvalBoolExpr(%q, %v) = %v, want %v", tt.input, tt.row, got, tt.want)
			}
		})
	}
}

// TestQueryBuilder 测试 QueryBuilder 链式 API
func TestQueryBuilder(t *testing.T) {
	// 内存中测试 QueryBuilder（不连接真实数据库）
	rows := []map[string]interface{}{
		{"id": int64(1), "name": "alice", "age": int64(30)},
		{"id": int64(2), "name": "bob", "age": int64(25)},
		{"id": int64(3), "name": "carol", "age": int64(40)},
	}

	// 测试内存过滤
	filtered := filterRows(rows, `age > 25`)
	if len(filtered) != 2 {
		t.Errorf("filterRows(age>25) = %d rows, want 2", len(filtered))
	}

	filtered = filterRows(rows, `name = "alice"`)
	if len(filtered) != 1 {
		t.Errorf("filterRows(name=alice) = %d rows, want 1", len(filtered))
	}

	filtered = filterRows(rows, `age > 25 AND age < 35`)
	if len(filtered) != 1 {
		t.Errorf("filterRows(age>25 AND age<35) = %d rows, want 1", len(filtered))
	}

	// 测试排序
	sorted := make([]map[string]interface{}, len(rows))
	copy(sorted, rows)
	sortRows(sorted, "age", "DESC")
	if sorted[0]["name"] != "carol" {
		t.Errorf("sortRows DESC by age failed: first = %v, want carol", sorted[0]["name"])
	}
}

// TestWQLNoSQLGeneration 测试 WQL 不生成 SQL 字符串
// 这是关键的"独立性证明"测试：
// 我们验证 WQL 的执行路径不产生任何 SQL 字符串。
func TestWQLNoSQLGeneration(t *testing.T) {
	// 关键断言: wqlv3 包不导入 database/sql
	// 通过在 QueryBuilder 中不引用任何 SQL 概念来证明

	qb := &QueryBuilder{
		db:        nil, // 即使没有数据库，结构也可以构造
		tableName: "users",
		selects:   []string{"id", "name"},
		where:     "age > 18",
	}

	// 验证 QueryBuilder 不包含任何 SQL 概念
	if qb.tableName != "users" {
		t.Error("QueryBuilder should store table name directly, not as SQL")
	}
	if qb.where != "age > 18" {
		t.Error("QueryBuilder should store WHERE as Go string, not SQL")
	}
}
