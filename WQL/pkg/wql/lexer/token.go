package lexer

import (
	"fmt"
	"strings"
)

// TokenType Token类型
type TokenType int

// Token WQL词法单元
type Token struct {
	Type   TokenType
	Value  string
	Line   int
	Column int
	Pos    int
}

// String 返回Token的字符串表示
func (t Token) String() string {
	return fmt.Sprintf("%s(%s)", t.Type, t.Value)
}

const (
	// 特殊Token
	TOKEN_ILLEGAL TokenType = iota
	TOKEN_EOF

	// 标识符和字面量
	TOKEN_IDENTIFIER
	TOKEN_INTEGER
	TOKEN_FLOAT
	TOKEN_STRING
	TOKEN_BOOLEAN
	TOKEN_NULL

	// 运算符
	TOKEN_PLUS
	TOKEN_MINUS
	TOKEN_MULTIPLY
	TOKEN_DIVIDE
	TOKEN_MODULO
	TOKEN_POWER

	// 比较运算符
	TOKEN_EQ
	TOKEN_NE
	TOKEN_LT
	TOKEN_LE
	TOKEN_GT
	TOKEN_GE
	TOKEN_LIKE
	TOKEN_IN
	TOKEN_BETWEEN
	TOKEN_IS

	// 逻辑运算符
	TOKEN_AND
	TOKEN_OR
	TOKEN_NOT

	// 标点符号
	TOKEN_LPAREN
	TOKEN_RPAREN
	TOKEN_LBRACKET
	TOKEN_RBRACKET
	TOKEN_COMMA
	TOKEN_DOT
	TOKEN_SEMICOLON
	TOKEN_COLON
	TOKEN_QUESTION

	// 关键字
	TOKEN_DB
	TOKEN_TABLE
	TOKEN_SELECT
	TOKEN_WHERE
	TOKEN_ORDER_BY
	TOKEN_GROUP_BY
	TOKEN_JOIN
	TOKEN_LEFT_JOIN
	TOKEN_RIGHT_JOIN
	TOKEN_LIMIT
	TOKEN_TAKE
	TOKEN_SKIP
	TOKEN_FIRST
	TOKEN_ALL
	TOKEN_COUNT
	TOKEN_SUM
	TOKEN_AVG
	TOKEN_MIN
	TOKEN_MAX
	TOKEN_ASC
	TOKEN_DESC
	TOKEN_JSON_EXTRACT
	TOKEN_JSON_QUERY
	TOKEN_JSON_VALUE
	TOKEN_HAVING
	TOKEN_UNION
	TOKEN_UNION_ALL
	TOKEN_INTERSECT
	TOKEN_EXCEPT
	TOKEN_AS
	TOKEN_SUBQUERY
)

var tokenNames = map[TokenType]string{
	TOKEN_ILLEGAL:   "ILLEGAL",
	TOKEN_EOF:       "EOF",
	TOKEN_IDENTIFIER:"IDENTIFIER",
	TOKEN_INTEGER:   "INTEGER",
	TOKEN_FLOAT:     "FLOAT",
	TOKEN_STRING:    "STRING",
	TOKEN_BOOLEAN:   "BOOLEAN",
	TOKEN_NULL:      "NULL",
	TOKEN_PLUS:      "+",
	TOKEN_MINUS:     "-",
	TOKEN_MULTIPLY:  "*",
	TOKEN_DIVIDE:    "/",
	TOKEN_MODULO:    "%",
	TOKEN_POWER:     "^",
	TOKEN_EQ:        "=",
	TOKEN_NE:        "!=",
	TOKEN_LT:        "<",
	TOKEN_LE:        "<=",
	TOKEN_GT:        ">",
	TOKEN_GE:        ">=",
	TOKEN_LIKE:      "LIKE",
	TOKEN_IN:        "IN",
	TOKEN_BETWEEN:   "BETWEEN",
	TOKEN_IS:        "IS",
	TOKEN_AND:       "AND",
	TOKEN_OR:        "OR",
	TOKEN_NOT:       "NOT",
	TOKEN_LPAREN:    "(",
	TOKEN_RPAREN:    ")",
	TOKEN_LBRACKET:  "[",
	TOKEN_RBRACKET:  "]",
	TOKEN_COMMA:     ",",
	TOKEN_DOT:       ".",
	TOKEN_SEMICOLON: ";",
	TOKEN_COLON:     ":",
	TOKEN_QUESTION:  "?",
	TOKEN_DB:        "DB",
	TOKEN_TABLE:     "TABLE",
	TOKEN_SELECT:    "SELECT",
	TOKEN_WHERE:     "WHERE",
	TOKEN_ORDER_BY:  "ORDER_BY",
	TOKEN_GROUP_BY:  "GROUP_BY",
	TOKEN_JOIN:      "JOIN",
	TOKEN_LEFT_JOIN: "LEFT_JOIN",
	TOKEN_RIGHT_JOIN:"RIGHT_JOIN",
	TOKEN_LIMIT:     "LIMIT",
	TOKEN_TAKE:      "TAKE",
	TOKEN_SKIP:      "SKIP",
	TOKEN_FIRST:     "FIRST",
	TOKEN_ALL:       "ALL",
	TOKEN_COUNT:     "COUNT",
	TOKEN_SUM:       "SUM",
	TOKEN_AVG:       "AVG",
	TOKEN_MIN:       "MIN",
	TOKEN_MAX:       "MAX",
	TOKEN_ASC:       "ASC",
	TOKEN_DESC:      "DESC",
	TOKEN_JSON_EXTRACT: "JSON_EXTRACT",
	TOKEN_JSON_QUERY: "JSON_QUERY",
	TOKEN_JSON_VALUE: "JSON_VALUE",
	TOKEN_HAVING:    "HAVING",
	TOKEN_UNION:     "UNION",
	TOKEN_UNION_ALL: "UNION_ALL",
	TOKEN_INTERSECT: "INTERSECT",
	TOKEN_EXCEPT:    "EXCEPT",
	TOKEN_AS:       "AS",
	TOKEN_SUBQUERY: "SUBQUERY",
}

func (tt TokenType) String() string {
	if name, ok := tokenNames[tt]; ok {
		return name
	}
	return fmt.Sprintf("UNKNOWN(%d)", tt)
}

var keywords = map[string]TokenType{
	"db":         TOKEN_DB,
	"Table":      TOKEN_TABLE,
	"Select":     TOKEN_SELECT,
	"Where":      TOKEN_WHERE,
	"OrderBy":    TOKEN_ORDER_BY,
	"GroupBy":    TOKEN_GROUP_BY,
	"Join":       TOKEN_JOIN,
	"LeftJoin":   TOKEN_LEFT_JOIN,
	"RightJoin":  TOKEN_RIGHT_JOIN,
	"Limit":      TOKEN_LIMIT,
	"Take":       TOKEN_TAKE,
	"Skip":       TOKEN_SKIP,
	"First":      TOKEN_FIRST,
	"All":        TOKEN_ALL,
	"Count":      TOKEN_COUNT,
	"Sum":        TOKEN_SUM,
	"Avg":        TOKEN_AVG,
	"Min":        TOKEN_MIN,
	"Max":        TOKEN_MAX,
	"ASC":        TOKEN_ASC,
	"DESC":       TOKEN_DESC,
	"AND":        TOKEN_AND,
	"OR":         TOKEN_OR,
	"NOT":        TOKEN_NOT,
	"LIKE":       TOKEN_LIKE,
	"IN":         TOKEN_IN,
	"BETWEEN":    TOKEN_BETWEEN,
	"IS":         TOKEN_IS,
	"NULL":       TOKEN_NULL,
	"true":       TOKEN_BOOLEAN,
	"false":      TOKEN_BOOLEAN,
	"JsonExtract": TOKEN_JSON_EXTRACT,
	"JsonQuery":   TOKEN_JSON_QUERY,
	"JsonValue":   TOKEN_JSON_VALUE,
	"Having":      TOKEN_HAVING,
	"Union":       TOKEN_UNION,
	"UnionAll":    TOKEN_UNION_ALL,
	"Intersect":   TOKEN_INTERSECT,
	"Except":      TOKEN_EXCEPT,
	"AS":          TOKEN_AS,
	"Subquery":    TOKEN_SUBQUERY,
}

func LookupIdentifier(ident string) TokenType {
	// 大小写不敏感查找关键词
	for keyword, tokType := range keywords {
		if strings.EqualFold(keyword, ident) {
			return tokType
		}
	}
	return TOKEN_IDENTIFIER
}
