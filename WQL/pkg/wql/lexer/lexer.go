package lexer

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Lexer WQL词法分析器
type Lexer struct {
	input        string
	position     int
	readPosition int
	ch           rune
	line         int
	column       int
}

// NewLexer 创建词法分析器
func NewLexer(input string) *Lexer {
	l := &Lexer{
		input: input,
		line:  1,
		column: 1,
	}
	l.readChar()
	return l
}

// readChar 读取下一个字符
func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = 0
	} else {
		l.ch, _ = utf8.DecodeRuneInString(l.input[l.readPosition:])
	}
	l.position = l.readPosition
	l.readPosition++
	l.column++
}

// peekChar 查看下一个字符但不移动位置
func (l *Lexer) peekChar() rune {
	if l.readPosition >= len(l.input) {
		return 0
	}
	ch, _ := utf8.DecodeRuneInString(l.input[l.readPosition:])
	return ch
}

// NextToken 读取下一个Token
func (l *Lexer) NextToken() Token {
	var tok Token

	l.skipWhitespace()

	switch l.ch {
	case '(':
		tok = Token{Type: TOKEN_LPAREN, Value: "(", Line: l.line, Column: l.column}
		l.readChar()
	case ')':
		tok = Token{Type: TOKEN_RPAREN, Value: ")", Line: l.line, Column: l.column}
		l.readChar()
	case '[':
		tok = Token{Type: TOKEN_LBRACKET, Value: "[", Line: l.line, Column: l.column}
		l.readChar()
	case ']':
		tok = Token{Type: TOKEN_RBRACKET, Value: "]", Line: l.line, Column: l.column}
		l.readChar()
	case '{':
		tok = Token{Type: TOKEN_LBRACE, Value: "{", Line: l.line, Column: l.column}
		l.readChar()
	case '}':
		tok = Token{Type: TOKEN_RBRACE, Value: "}", Line: l.line, Column: l.column}
		l.readChar()
	case ',':
		tok = Token{Type: TOKEN_COMMA, Value: ",", Line: l.line, Column: l.column}
		l.readChar()
	case '.':
		if l.peekChar() == '.' {
			tok = Token{Type: TOKEN_ILLEGAL, Value: "..", Line: l.line, Column: l.column}
			l.readChar()
			l.readChar()
		} else {
			tok = Token{Type: TOKEN_DOT, Value: ".", Line: l.line, Column: l.column}
			l.readChar()
		}
	case ';':
		tok = Token{Type: TOKEN_SEMICOLON, Value: ";", Line: l.line, Column: l.column}
		l.readChar()
	case ':':
		tok = Token{Type: TOKEN_COLON, Value: ":", Line: l.line, Column: l.column}
		l.readChar()
	case '?':
		if l.peekChar() == '?' {
			tok = Token{Type: TOKEN_ILLEGAL, Value: "??", Line: l.line, Column: l.column}
			l.readChar()
			l.readChar()
		} else {
			tok = Token{Type: TOKEN_QUESTION, Value: "?", Line: l.line, Column: l.column}
			l.readChar()
		}
	case '+':
		tok = Token{Type: TOKEN_PLUS, Value: "+", Line: l.line, Column: l.column}
		l.readChar()
	case '-':
		tok = Token{Type: TOKEN_MINUS, Value: "-", Line: l.line, Column: l.column}
		l.readChar()
	case '*':
		tok = Token{Type: TOKEN_MULTIPLY, Value: "*", Line: l.line, Column: l.column}
		l.readChar()
	case '/':
		tok = Token{Type: TOKEN_DIVIDE, Value: "/", Line: l.line, Column: l.column}
		l.readChar()
	case '%':
		tok = Token{Type: TOKEN_MODULO, Value: "%", Line: l.line, Column: l.column}
		l.readChar()
	case '^':
		tok = Token{Type: TOKEN_POWER, Value: "^", Line: l.line, Column: l.column}
		l.readChar()
	case '=':
		tok = Token{Type: TOKEN_EQ, Value: "=", Line: l.line, Column: l.column}
		l.readChar()
	case '!':
		if l.peekChar() == '=' {
			tok = Token{Type: TOKEN_NE, Value: "!=", Line: l.line, Column: l.column}
			l.readChar()
			l.readChar()
		} else {
			tok = Token{Type: TOKEN_NOT, Value: "!", Line: l.line, Column: l.column}
			l.readChar()
		}
	case '<':
		if l.peekChar() == '=' {
			tok = Token{Type: TOKEN_LE, Value: "<=", Line: l.line, Column: l.column}
			l.readChar()
			l.readChar()
		} else {
			tok = Token{Type: TOKEN_LT, Value: "<", Line: l.line, Column: l.column}
			l.readChar()
		}
	case '>':
		if l.peekChar() == '=' {
			tok = Token{Type: TOKEN_GE, Value: ">=", Line: l.line, Column: l.column}
			l.readChar()
			l.readChar()
		} else {
			tok = Token{Type: TOKEN_GT, Value: ">", Line: l.line, Column: l.column}
			l.readChar()
		}
	case 0:
		tok = Token{Type: TOKEN_EOF, Value: "", Line: l.line, Column: l.column}
	case '"', '\'':
		tok = l.readString()
	default:
		if isLetter(l.ch) {
			ident := l.readIdentifier()
			tokType := LookupIdentifier(ident)
			tok = Token{Type: tokType, Value: ident, Line: l.line, Column: l.column - len(ident)}
			return tok
		} else if isDigit(l.ch) {
			tok = l.readNumber()
			return tok
		} else {
			tok = Token{Type: TOKEN_ILLEGAL, Value: string(l.ch), Line: l.line, Column: l.column}
			l.readChar()
		}
	}

	return tok
}

// readIdentifier 读取标识符
func (l *Lexer) readIdentifier() string {
	position := l.position
	for isLetter(l.ch) || isDigit(l.ch) || l.ch == '_' {
		l.readChar()
	}
	return l.input[position:l.position]
}

// readLikePattern 读取 LIKE 模式（无双引号设计：直接读取直到空白/括号）
// 支持字符：字母、数字、_、%
func (l *Lexer) readLikePattern() string {
	position := l.position
	for isLetter(l.ch) || isDigit(l.ch) || l.ch == '_' || l.ch == '%' {
		l.readChar()
	}
	return l.input[position:l.position]
}

// readPatternToken 读取一个可能是 LIKE 模式的 token；
// 如果当前字符是字母/数字/_/%，则读取完整的 LIKE 模式。
// 否则按常规 NextToken。
// 这个函数专门给 parser 在 LIKE 之后使用。
func (l *Lexer) readPatternToken() Token {
	if isLetter(l.ch) || l.ch == '_' {
		ident := l.readLikePattern()
		// LIKE 模式总是视为 IDENTIFIER
		return Token{Type: TOKEN_IDENTIFIER, Value: ident, Line: l.line, Column: l.column - len(ident)}
	}
	return l.NextToken()
}

// ReadPatternToken 是 readPatternToken 的公开版本，供 parser 在 LIKE 之后使用。
func (l *Lexer) ReadPatternToken() Token {
	return l.readPatternToken()
}

// ExtendIdentifierValue 扩展当前 lexer position：如果当前位置是 % 或 _，
// 把它们加到 token 的值中（用于 LIKE 模式中 currentToken 是 a 而 peekToken 是 % 的情况）。
// 返回 true 表示扩展了字符。
func (l *Lexer) ExtendIdentifierValue(tok *Token) bool {
	extended := false
	for l.ch == '%' || l.ch == '_' {
		tok.Value += string(l.ch)
		l.readChar()
		extended = true
	}
	return extended
}

// readNumber 读取数字
func (l *Lexer) readNumber() Token {
	position := l.position
	isFloat := false

	for isDigit(l.ch) {
		l.readChar()
	}

	if l.ch == '.' && isDigit(l.peekChar()) {
		isFloat = true
		l.readChar()
		for isDigit(l.ch) {
			l.readChar()
		}
	}

	numStr := l.input[position:l.position]
	if isFloat {
		_, err := strconv.ParseFloat(numStr, 64)
		if err != nil {
			return Token{Type: TOKEN_ILLEGAL, Value: numStr, Line: l.line, Column: l.column}
		}
		return Token{Type: TOKEN_FLOAT, Value: numStr, Line: l.line, Column: l.column - len(numStr)}
	}

	_, err := strconv.ParseInt(numStr, 10, 64)
	if err != nil {
		return Token{Type: TOKEN_ILLEGAL, Value: numStr, Line: l.line, Column: l.column}
	}
	return Token{Type: TOKEN_INTEGER, Value: numStr, Line: l.line, Column: l.column - len(numStr)}
}

// readString 读取字符串
func (l *Lexer) readString() Token {
	quote := l.ch
	l.readChar()

	var value strings.Builder

	for l.ch != quote && l.ch != 0 {
		if l.ch == '\\' {
			l.readChar()
			switch l.ch {
			case 'n':
				value.WriteRune('\n')
			case 't':
				value.WriteRune('\t')
			case 'r':
				value.WriteRune('\r')
			case '\\':
				value.WriteRune('\\')
			case quote:
				value.WriteRune(quote)
			default:
				value.WriteRune(l.ch)
			}
		} else {
			value.WriteRune(l.ch)
		}
		l.readChar()
	}

	if l.ch != quote {
		return Token{Type: TOKEN_ILLEGAL, Value: "unterminated string", Line: l.line, Column: l.column}
	}

	l.readChar()
	return Token{Type: TOKEN_STRING, Value: value.String(), Line: l.line, Column: l.column}
}

// skipWhitespace 跳过空白字符
func (l *Lexer) skipWhitespace() {
	for unicode.IsSpace(l.ch) {
		if l.ch == '\n' {
			l.line++
			l.column = 1
		}
		l.readChar()
	}
}

// isLetter 检查是否是字母
func isLetter(ch rune) bool {
	return unicode.IsLetter(ch) || ch == '_'
}

// isDigit 检查是否是数字
func isDigit(ch rune) bool {
	return unicode.IsDigit(ch)
}

// Tokenize 将输入字符串转换为Token列表
func Tokenize(input string) ([]Token, error) {
	lexer := NewLexer(input)
	var tokens []Token

	for {
		tok := lexer.NextToken()
		tokens = append(tokens, tok)

		if tok.Type == TOKEN_EOF {
			break
		}

		if tok.Type == TOKEN_ILLEGAL {
			return nil, fmt.Errorf("illegal token at line %d, column %d: %s", tok.Line, tok.Column, tok.Value)
		}
	}

	return tokens, nil
}
