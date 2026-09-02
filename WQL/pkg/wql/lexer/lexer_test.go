package lexer

import "testing"

func TestTokenize(t *testing.T) {
	input := `db.Table(users).Select(name, age).Where(age > 25).Take(3).All()`

	tokens, err := Tokenize(input)
	if err != nil {
		t.Fatalf("Tokenize() error = %v", err)
	}

	if len(tokens) == 0 {
		t.Fatal("Tokenize() got 0 tokens")
	}

	if tokens[0].Type != TOKEN_DB {
		t.Errorf("tokens[0].Type = %s, want %s", tokens[0].Type, TOKEN_DB)
	}

	if tokens[0].Value != "db" {
		t.Errorf("tokens[0].Value = %s, want %s", tokens[0].Value, "db")
	}
}

func TestTokenTypes(t *testing.T) {
	tests := []struct {
		input    string
		expected TokenType
	}{
		{"db", TOKEN_DB},
		{"Table", TOKEN_TABLE},
		{"Select", TOKEN_SELECT},
		{"Where", TOKEN_WHERE},
		{"Join", TOKEN_JOIN},
		{"Count", TOKEN_COUNT},
		{"AND", TOKEN_AND},
		{"OR", TOKEN_OR},
		{"ASC", TOKEN_ASC},
		{"DESC", TOKEN_DESC},
	}

	for _, tt := range tests {
		result := LookupIdentifier(tt.input)
		if result != tt.expected {
			t.Errorf("LookupIdentifier(%q) = %s, want %s", tt.input, result, tt.expected)
		}
	}
}

func TestNumberTokenization(t *testing.T) {
	tests := []struct {
		input       string
		expected    TokenType
		expectedVal string
	}{
		{"123", TOKEN_INTEGER, "123"},
		{"456", TOKEN_INTEGER, "456"},
		{"12.34", TOKEN_FLOAT, "12.34"},
		{"56.78", TOKEN_FLOAT, "56.78"},
	}

	for _, tt := range tests {
		lexer := NewLexer(tt.input)
		tok := lexer.NextToken()

		if tok.Type != tt.expected {
			t.Errorf("NextToken(%q).Type = %s, want %s", tt.input, tok.Type, tt.expected)
		}
		if tok.Value != tt.expectedVal {
			t.Errorf("NextToken(%q).Value = %s, want %s", tt.input, tok.Value, tt.expectedVal)
		}
	}
}

func TestStringTokenization(t *testing.T) {
	input := `"hello world"`
	lexer := NewLexer(input)
	tok := lexer.NextToken()

	if tok.Type != TOKEN_STRING {
		t.Errorf("String token type = %s, want %s", tok.Type, TOKEN_STRING)
	}
	if tok.Value != "hello world" {
		t.Errorf("String token value = %s, want 'hello world'", tok.Value)
	}
}
