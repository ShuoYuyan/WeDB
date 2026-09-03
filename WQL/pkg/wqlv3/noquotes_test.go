package wqlv3

import (
	"testing"

	"github.com/wedb/wedb/WQL/pkg/wql/lexer"
)

// TestNoQuotesParser validates no-quote WQL parsing.
// WQL's core design principle: identifiers and string values do NOT
// require double quotes.
func TestNoQuotesParser(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantErr  bool
		validate func(t *testing.T, result QueryResult)
	}{
		{
			name:    "basic query - no quotes",
			input:   `db.Table(users).All()`,
			wantErr: false,
			validate: func(t *testing.T, result QueryResult) {
				if len(result.Rows) != 3 {
					t.Errorf("expected 3 rows, got %d", len(result.Rows))
				}
			},
		},
		{
			name:    "WHERE - no quotes",
			input:   `db.Table(users).Where(age > 25).All()`,
			wantErr: false,
			validate: func(t *testing.T, result QueryResult) {
				if len(result.Rows) != 2 {
					t.Errorf("expected 2 rows (alice, carol), got %d", len(result.Rows))
				}
			},
		},
		{
			name:    "SELECT - no quotes",
			input:   `db.Table(users).Select(name, age).All()`,
			wantErr: false,
			validate: func(t *testing.T, result QueryResult) {
				if len(result.Rows) != 3 {
					t.Errorf("expected 3 rows, got %d", len(result.Rows))
				}
				if len(result.Rows) > 0 {
					row := result.Rows[0]
					if _, hasName := row["name"]; !hasName {
						t.Error("expected 'name' column in result")
					}
				}
			},
		},
		{
			name:    "composite - Select + Where + OrderBy + Take",
			input:   `db.Table(users).Select(name, age).Where(age > 18).OrderBy(age, DESC).Take(2).All()`,
			wantErr: false,
			validate: func(t *testing.T, result QueryResult) {
				if len(result.Rows) != 2 {
					t.Errorf("expected 2 rows, got %d", len(result.Rows))
				}
				if len(result.Rows) >= 1 {
					first := result.Rows[0]
					if age, ok := first["age"].(int64); !ok || age < 30 {
						t.Errorf("first row should be carol (age=40), got %v", first)
					}
				}
			},
		},
		{
			name:    "First - single row",
			input:   `db.Table(users).Where(name = alice).First()`,
			wantErr: false,
			validate: func(t *testing.T, result QueryResult) {
				if len(result.Rows) != 1 {
					t.Errorf("expected 1 row, got %d", len(result.Rows))
				}
			},
		},
		{
			name:    "no-quote string value (bare identifier as literal)",
			input:   `db.Table(users).Where(name = alice).First()`,
			wantErr: false,
			validate: func(t *testing.T, result QueryResult) {
				if len(result.Rows) != 1 {
					t.Errorf("expected 1 row for name=alice, got %d", len(result.Rows))
				}
			},
		},
		{
			name:    "Skip - offset",
			input:   `db.Table(users).Skip(1).Take(1).All()`,
			wantErr: false,
			validate: func(t *testing.T, result QueryResult) {
				if len(result.Rows) != 1 {
					t.Errorf("expected 1 row after skip+take, got %d", len(result.Rows))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, _ := setupTestDB(t)
			_, _ = db.Insert("users").Values(
				map[string]interface{}{"id": int64(1), "name": "alice", "age": int64(30)},
				map[string]interface{}{"id": int64(2), "name": "bob", "age": int64(25)},
				map[string]interface{}{"id": int64(3), "name": "carol", "age": int64(40)},
			).Execute()

			result, err := EvaluateQueryNoQuotes(db, tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("EvaluateQueryNoQuotes(%q) error: %v", tt.input, err)
			}
			if !tt.wantErr && tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

// TestNoQuotesLexerLexicalAnalysis verifies that the lexer produces
// IDENTIFIER tokens for table/column names without requiring quotes.
func TestNoQuotesLexerLexicalAnalysis(t *testing.T) {
	lex := lexer.NewLexer("db.Table(users).Select(name, age).All()")
	tokens := []lexer.Token{}
	for {
		tok := lex.NextToken()
		tokens = append(tokens, tok)
		if tok.Type == lexer.TOKEN_EOF {
			break
		}
	}
	t.Logf("Token count: %d", len(tokens))
	var hasDB, hasTable, hasSelect, hasAll, hasUsers bool
	for _, tok := range tokens {
		switch tok.Type {
		case lexer.TOKEN_DB:
			hasDB = true
		case lexer.TOKEN_TABLE:
			hasTable = true
		case lexer.TOKEN_SELECT:
			hasSelect = true
		case lexer.TOKEN_ALL:
			hasAll = true
		}
		if tok.Type == lexer.TOKEN_IDENTIFIER && tok.Value == "users" {
			hasUsers = true
		}
	}
	if !hasDB {
		t.Error("missing DB token")
	}
	if !hasTable {
		t.Error("missing TABLE token")
	}
	if !hasSelect {
		t.Error("missing SELECT token")
	}
	if !hasAll {
		t.Error("missing ALL token")
	}
	if !hasUsers {
		t.Error("missing users IDENTIFIER token (no quotes!)")
	}
}
