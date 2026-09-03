package storage

import "testing"

// TestEvaluateNewOps exercises the new IN/LIKE/BETWEEN/IS NULL/NOT operators
// against real row data.
func TestEvaluateNewOps(t *testing.T) {
	rows := []map[string]interface{}{
		{"id": int64(1), "name": "alice", "age": int64(30), "dept": "Engineering"},
		{"id": int64(2), "name": "bob", "age": int64(25), "dept": "Sales"},
		{"id": int64(3), "name": "carol", "age": int64(35), "dept": "Engineering"},
		{"id": int64(4), "name": "dave", "age": int64(17), "dept": "Sales"},
		{"id": int64(5), "name": "anna", "age": int64(45), "dept": "Marketing"},
	}

	cases := []struct {
		where string
		want  []int64 // expected ids
	}{
		{"id IN (1, 3, 5)", []int64{1, 3, 5}},
		{"id NOT IN (1, 2)", []int64{3, 4, 5}},
		{"age BETWEEN 25 AND 35", []int64{1, 2, 3}},
		{"age NOT BETWEEN 25 AND 35", []int64{4, 5}},
		{`name LIKE "a%"`, []int64{1, 5}},
		{`name NOT LIKE "a%"`, []int64{2, 3, 4}},
		{"dept IS NOT NULL", []int64{1, 2, 3, 4, 5}},
		{"dept IS NULL", []int64{}},
		{`NOT(dept = "Sales")`, []int64{1, 3, 5}},
	}
	for _, c := range cases {
		w, err := ParseWhereClause(c.where)
		if err != nil {
			t.Errorf("parse %q failed: %v", c.where, err)
			continue
		}
		var got []int64
		for _, r := range rows {
			match, err := evaluateCondition(&w.Conditions[0], r)
			if err != nil {
				t.Errorf("eval %q on %+v: %v", c.where, r, err)
				continue
			}
			if match {
				got = append(got, r["id"].(int64))
			}
		}
		if !equalInt64Slices(got, c.want) {
			t.Errorf("where=%q: got %v, want %v", c.where, got, c.want)
		}
	}
}

func equalInt64Slices(a, b []int64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
