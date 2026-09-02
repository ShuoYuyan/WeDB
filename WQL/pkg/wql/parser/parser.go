package parser

import (
	"fmt"
	"strings"

	"github.com/wedb/wedb/WQL/pkg/wql/lexer"
)

// Parser 语法分析器
type Parser struct {
	lexer        *lexer.Lexer
	currentToken lexer.Token
	peekToken    lexer.Token
	errors       []string
}

// NewParser 创建语法分析器
func NewParser(lex *lexer.Lexer) *Parser {
	p := &Parser{
		lexer:  lex,
		errors: []string{},
	}

	p.nextToken()
	p.nextToken()

	return p
}

// Parse 解析WQL查询
func (p *Parser) Parse() (*WQLQuery, error) {
	query := &WQLQuery{
		Operations: []Operation{},
	}

	if err := p.parseSource(query); err != nil {
		return nil, err
	}

	if err := p.parseOperations(query); err != nil {
		return nil, err
	}

	return query, nil
}

// parseSource 解析数据源
func (p *Parser) parseSource(query *WQLQuery) error {
	if p.currentToken.Type != lexer.TOKEN_DB {
		return fmt.Errorf("expected db, got %s", p.currentToken.Type)
	}

	p.nextToken()

	if p.currentToken.Type != lexer.TOKEN_DOT {
		return fmt.Errorf("expected '.', got %s", p.currentToken.Type)
	}

	p.nextToken()

	if p.currentToken.Type != lexer.TOKEN_TABLE {
		return fmt.Errorf("expected Table, got %s", p.currentToken.Type)
	}

	p.nextToken()

	if p.currentToken.Type != lexer.TOKEN_LPAREN {
		return fmt.Errorf("expected '(', got %s", p.currentToken.Type)
	}

	p.nextToken()

	tableName := p.currentToken.Value
	query.Source = tableName

	p.nextToken()

	if p.currentToken.Type != lexer.TOKEN_RPAREN {
		return fmt.Errorf("expected ')', got %s", p.currentToken.Type)
	}

	p.nextToken()

	return nil
}

// parseOperations 解析操作链
func (p *Parser) parseOperations(query *WQLQuery) error {
	for p.currentToken.Type != lexer.TOKEN_EOF {
		if p.currentToken.Type != lexer.TOKEN_DOT {
			break
		}

		p.nextToken()

		op, err := p.parseOperation()
		if err != nil {
			return err
		}

		query.Operations = append(query.Operations, op)
	}

	return nil
}

// parseOperation 解析单个操作
func (p *Parser) parseOperation() (Operation, error) {
	switch p.currentToken.Type {
	case lexer.TOKEN_SELECT:
		return p.parseSelectOperation()
	case lexer.TOKEN_WHERE:
		return p.parseWhereOperation()
	case lexer.TOKEN_JOIN, lexer.TOKEN_LEFT_JOIN, lexer.TOKEN_RIGHT_JOIN:
		return p.parseJoinOperation()
	case lexer.TOKEN_ORDER_BY:
		return p.parseOrderByOperation()
	case lexer.TOKEN_GROUP_BY:
		return p.parseGroupByOperation()
	case lexer.TOKEN_HAVING:
		return p.parseHavingOperation()
	case lexer.TOKEN_LIMIT, lexer.TOKEN_TAKE:
		return p.parseLimitOperation()
	case lexer.TOKEN_SKIP:
		return p.parseSkipOperation()
	case lexer.TOKEN_FIRST:
		return p.parseFirstOperation()
	case lexer.TOKEN_ALL:
		return p.parseAllOperation()
	case lexer.TOKEN_UNION, lexer.TOKEN_UNION_ALL:
		return p.parseUnionOperation()
	case lexer.TOKEN_INTERSECT:
		return p.parseIntersectOperation()
	case lexer.TOKEN_EXCEPT:
		return p.parseExceptOperation()
	case lexer.TOKEN_INSERT:
		return p.parseInsertOperation()
	case lexer.TOKEN_SET:
		return p.parseSetOperation()
	case lexer.TOKEN_DELETE:
		return p.parseDeleteOperation()
	case lexer.TOKEN_CREATE:
		return p.parseCreateTableOperation()
	case lexer.TOKEN_DROP:
		return p.parseDropTableOperation()
	case lexer.TOKEN_EXECUTE:
		return p.parseExecuteOperation()
	default:
		return nil, fmt.Errorf("unknown operation: %s", p.currentToken.Type)
	}
}

// parseSelectOperation 解析SELECT操作
func (p *Parser) parseSelectOperation() (Operation, error) {
	p.nextToken()

	if p.currentToken.Type != lexer.TOKEN_LPAREN {
		return nil, fmt.Errorf("expected '(', got %s", p.currentToken.Type)
	}

	p.nextToken()

	columns := []Expression{}

	for p.currentToken.Type != lexer.TOKEN_RPAREN {
		expr, err := p.parseExpression()
		if err != nil {
			return nil, err
		}

		columns = append(columns, expr)

		if p.currentToken.Type == lexer.TOKEN_COMMA {
			p.nextToken()
		}
	}

	p.nextToken()

	return &SelectOperation{Columns: columns}, nil
}

// parseWhereOperation 解析WHERE操作
func (p *Parser) parseWhereOperation() (*WhereOperation, error) {
	p.nextToken()

	if p.currentToken.Type != lexer.TOKEN_LPAREN {
		return nil, fmt.Errorf("expected '(', got %s", p.currentToken.Type)
	}

	p.nextToken()

	condition, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	if p.currentToken.Type != lexer.TOKEN_RPAREN {
		return nil, fmt.Errorf("expected ')', got %s", p.currentToken.Type)
	}

	p.nextToken()

	return &WhereOperation{Condition: condition}, nil
}

// parseJoinOperation 解析JOIN操作
func (p *Parser) parseJoinOperation() (*JoinOperation, error) {
	joinType := p.currentToken.Value

	p.nextToken()

	if p.currentToken.Type != lexer.TOKEN_LPAREN {
		return nil, fmt.Errorf("expected '(', got %s", p.currentToken.Type)
	}

	p.nextToken()

	table, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	if p.currentToken.Type != lexer.TOKEN_COMMA {
		return nil, fmt.Errorf("expected ',', got %s", p.currentToken.Type)
	}

	p.nextToken()

	leftKey, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	if p.currentToken.Type != lexer.TOKEN_COMMA {
		return nil, fmt.Errorf("expected ',', got %s", p.currentToken.Type)
	}

	p.nextToken()

	rightKey, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	if p.currentToken.Type != lexer.TOKEN_RPAREN {
		return nil, fmt.Errorf("expected ')', got %s", p.currentToken.Type)
	}

	p.nextToken()

	return &JoinOperation{
		JoinType: joinType,
		Table:    table,
		LeftKey:  leftKey,
		RightKey: rightKey,
	}, nil
}

// parseOrderByOperation 解析ORDER BY操作
func (p *Parser) parseOrderByOperation() (*OrderByOperation, error) {
	p.nextToken()

	if p.currentToken.Type != lexer.TOKEN_LPAREN {
		return nil, fmt.Errorf("expected '(', got %s", p.currentToken.Type)
	}

	p.nextToken()

	column, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	direction := "ASC"
	if p.currentToken.Type == lexer.TOKEN_COMMA {
		p.nextToken()

		if p.currentToken.Type == lexer.TOKEN_ASC || p.currentToken.Type == lexer.TOKEN_DESC {
			direction = p.currentToken.Value
			p.nextToken()
		}
	}

	if p.currentToken.Type != lexer.TOKEN_RPAREN {
		return nil, fmt.Errorf("expected ')', got %s", p.currentToken.Type)
	}

	p.nextToken()

	return &OrderByOperation{
		Column:    column,
		Direction: direction,
	}, nil
}

// parseGroupByOperation 解析GROUP BY操作
func (p *Parser) parseGroupByOperation() (*GroupByOperation, error) {
	p.nextToken()

	if p.currentToken.Type != lexer.TOKEN_LPAREN {
		return nil, fmt.Errorf("expected '(', got %s", p.currentToken.Type)
	}

	p.nextToken()

	columns := []Expression{}

	for p.currentToken.Type != lexer.TOKEN_RPAREN {
		expr, err := p.parseExpression()
		if err != nil {
			return nil, err
		}

		columns = append(columns, expr)

		if p.currentToken.Type == lexer.TOKEN_COMMA {
			p.nextToken()
		}
	}

	p.nextToken()

	return &GroupByOperation{Columns: columns}, nil
}

// parseHavingOperation 解析HAVING操作
func (p *Parser) parseHavingOperation() (*HavingOperation, error) {
	p.nextToken()

	if p.currentToken.Type != lexer.TOKEN_LPAREN {
		return nil, fmt.Errorf("expected '(', got %s", p.currentToken.Type)
	}

	p.nextToken()

	condition, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	if p.currentToken.Type != lexer.TOKEN_RPAREN {
		return nil, fmt.Errorf("expected ')', got %s", p.currentToken.Type)
	}

	p.nextToken()

	return &HavingOperation{Condition: condition}, nil
}

// parseLimitOperation 解析LIMIT操作
func (p *Parser) parseLimitOperation() (*LimitOperation, error) {
	p.nextToken()

	if p.currentToken.Type != lexer.TOKEN_LPAREN {
		return nil, fmt.Errorf("expected '(', got %s", p.currentToken.Type)
	}

	p.nextToken()

	count, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	var offset Expression = nil
	if p.currentToken.Type == lexer.TOKEN_COMMA {
		p.nextToken()
		offset, err = p.parseExpression()
		if err != nil {
			return nil, err
		}
	}

	if p.currentToken.Type != lexer.TOKEN_RPAREN {
		return nil, fmt.Errorf("expected ')', got %s", p.currentToken.Type)
	}

	p.nextToken()

	return &LimitOperation{
		Count:  count,
		Offset: offset,
	}, nil
}

// parseSkipOperation 解析SKIP操作
func (p *Parser) parseSkipOperation() (*SkipOperation, error) {
	p.nextToken()

	if p.currentToken.Type != lexer.TOKEN_LPAREN {
		return nil, fmt.Errorf("expected '(', got %s", p.currentToken.Type)
	}

	p.nextToken()

	count, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	if p.currentToken.Type != lexer.TOKEN_RPAREN {
		return nil, fmt.Errorf("expected ')', got %s", p.currentToken.Type)
	}

	p.nextToken()

	return &SkipOperation{Count: count}, nil
}

// parseFirstOperation 解析FIRST操作
func (p *Parser) parseFirstOperation() (*FirstOperation, error) {
	p.nextToken()

	if p.currentToken.Type != lexer.TOKEN_LPAREN {
		return nil, fmt.Errorf("expected '(', got %s", p.currentToken.Type)
	}

	p.nextToken()

	if p.currentToken.Type != lexer.TOKEN_RPAREN {
		return nil, fmt.Errorf("expected ')', got %s", p.currentToken.Type)
	}

	p.nextToken()

	return &FirstOperation{}, nil
}

// parseAllOperation 解析ALL操作
func (p *Parser) parseAllOperation() (*AllOperation, error) {
	p.nextToken()

	if p.currentToken.Type != lexer.TOKEN_LPAREN {
		return nil, fmt.Errorf("expected '(', got %s", p.currentToken.Type)
	}

	p.nextToken()

	if p.currentToken.Type != lexer.TOKEN_RPAREN {
		return nil, fmt.Errorf("expected ')', got %s", p.currentToken.Type)
	}

	p.nextToken()

	return &AllOperation{}, nil
}

// parseUnionOperation 解析UNION操作
func (p *Parser) parseUnionOperation() (*UnionOperation, error) {
	all := p.currentToken.Type == lexer.TOKEN_UNION_ALL
	p.nextToken()

	if p.currentToken.Type != lexer.TOKEN_LPAREN {
		return nil, fmt.Errorf("expected '(', got %s", p.currentToken.Type)
	}

	p.nextToken()

	table, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	if p.currentToken.Type != lexer.TOKEN_RPAREN {
		return nil, fmt.Errorf("expected ')', got %s", p.currentToken.Type)
	}

	p.nextToken()

	return &UnionOperation{
		Table: table,
		All:   all,
	}, nil
}

// parseIntersectOperation 解析INTERSECT操作
func (p *Parser) parseIntersectOperation() (*IntersectOperation, error) {
	p.nextToken()

	if p.currentToken.Type != lexer.TOKEN_LPAREN {
		return nil, fmt.Errorf("expected '(', got %s", p.currentToken.Type)
	}

	p.nextToken()

	table, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	if p.currentToken.Type != lexer.TOKEN_RPAREN {
		return nil, fmt.Errorf("expected ')', got %s", p.currentToken.Type)
	}

	p.nextToken()

	return &IntersectOperation{Table: table}, nil
}

// parseExceptOperation 解析EXCEPT操作
func (p *Parser) parseExceptOperation() (*ExceptOperation, error) {
	p.nextToken()

	if p.currentToken.Type != lexer.TOKEN_LPAREN {
		return nil, fmt.Errorf("expected '(', got %s", p.currentToken.Type)
	}

	p.nextToken()

	table, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	if p.currentToken.Type != lexer.TOKEN_RPAREN {
		return nil, fmt.Errorf("expected ')', got %s", p.currentToken.Type)
	}

	p.nextToken()

	return &ExceptOperation{Table: table}, nil
}

// parseExpression 解析表达式
func (p *Parser) parseExpression() (Expression, error) {
	return p.parseBinaryExpression(0)
}

// parseBinaryExpression 解析二元表达式
func (p *Parser) parseBinaryExpression(precedence int) (Expression, error) {
	left, err := p.parsePrimaryExpression()
	if err != nil {
		return nil, err
	}

	for {
		op := p.currentToken
		if !isBinaryOperator(op.Type) {
			break
		}

		opPrec := p.getOperatorPrecedence(op.Type)
		if opPrec < precedence {
			break
		}

		p.nextToken()

		right, err := p.parseBinaryExpression(opPrec + 1)
		if err != nil {
			return nil, err
		}

		left = &BinaryExpression{
			Left:     left,
			Operator: op.Value,
			Right:    right,
		}
	}

	return left, nil
}

// parsePrimaryExpression 解析基本表达式
func (p *Parser) parsePrimaryExpression() (Expression, error) {
	switch p.currentToken.Type {
	case lexer.TOKEN_IDENTIFIER, lexer.TOKEN_COUNT, lexer.TOKEN_SUM, lexer.TOKEN_AVG, lexer.TOKEN_MIN, lexer.TOKEN_MAX,
		lexer.TOKEN_JSON_EXTRACT, lexer.TOKEN_JSON_QUERY, lexer.TOKEN_JSON_VALUE:
		// 支持带点的标识符，如 users.id
		value := p.currentToken.Value
		p.nextToken()

		// 检查是否是函数调用：FunctionName(...)
		if p.currentToken.Type == lexer.TOKEN_LPAREN {
			return p.parseFunctionCall(value)
		}

		// 检查是否有点，如果有，则构造复合标识符
		if p.currentToken.Type == lexer.TOKEN_DOT {
			p.nextToken()
			if p.currentToken.Type == lexer.TOKEN_IDENTIFIER {
				// 复合标识符，如 users.id
				compoundValue := value + "." + p.currentToken.Value
				p.nextToken()
				return &Identifier{Value: compoundValue}, nil
			}
		}

		return &Identifier{Value: value}, nil
	case lexer.TOKEN_INTEGER, lexer.TOKEN_FLOAT, lexer.TOKEN_STRING, lexer.TOKEN_BOOLEAN:
		value := p.currentToken.Value
		p.nextToken()
		return &LiteralExpression{Value: value}, nil
	case lexer.TOKEN_LPAREN:
		p.nextToken()
		// 检查是否是子查询：(db.Table(...)...)
		if p.currentToken.Type == lexer.TOKEN_DB {
			return p.parseSubquery()
		}
		// 否则是普通的分组表达式
		expr, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		if p.currentToken.Type != lexer.TOKEN_RPAREN {
			return nil, fmt.Errorf("expected ')', got %s", p.currentToken.Type)
		}
		p.nextToken()
		return expr, nil
	default:
		return nil, fmt.Errorf("unexpected token in expression: %s", p.currentToken.Type)
	}
}

// parseSubquery 解析子查询
func (p *Parser) parseSubquery() (Expression, error) {
	// 实现步骤：
	// 1. 收集括号内的所有token
	// 2. 重新构建子查询字符串
	// 3. 创建新的parser解析子查询
	// 4. 返回SubqueryExpression

	var tokens []lexer.Token
	parenLevel := 1 // 从1开始，因为已经跳过了第一个左括号

	// 从当前token（TOKEN_SELECT）开始收集
	for {
		tokens = append(tokens, p.currentToken)

		// 跟踪括号嵌套
		if p.currentToken.Type == lexer.TOKEN_LPAREN {
			parenLevel++
		} else if p.currentToken.Type == lexer.TOKEN_RPAREN {
			parenLevel--
			// 当回到0时，表示找到了匹配的右括号
			if parenLevel == 0 {
				p.nextToken()
				break
			}
		}

		p.nextToken()
		if p.currentToken.Type == lexer.TOKEN_EOF {
			return nil, fmt.Errorf("unterminated subquery")
		}
	}

	// 重新构建子查询字符串
	var subqueryBuilder strings.Builder
	for _, tok := range tokens {
		subqueryBuilder.WriteString(tok.Value)
	}
	subqueryStr := subqueryBuilder.String()

	// 创建新的lexer和parser来解析子查询
	subLexer := lexer.NewLexer(subqueryStr)
	subParser := NewParser(subLexer)
	subQuery, err := subParser.Parse()
	if err != nil {
		return nil, fmt.Errorf("failed to parse subquery: %v", err)
	}

	// 检查是否有AS别名
	alias := ""
	if p.currentToken.Type == lexer.TOKEN_AS {
		p.nextToken()
		if p.currentToken.Type == lexer.TOKEN_IDENTIFIER {
			alias = p.currentToken.Value
			p.nextToken()
		}
	}

	return &SubqueryExpression{
		Query: subQuery,
		Alias: alias,
	}, nil
}

// getOperatorPrecedence 获取运算符优先级
func (p *Parser) getOperatorPrecedence(tokenType lexer.TokenType) int {
	switch tokenType {
	case lexer.TOKEN_OR:
		return 1
	case lexer.TOKEN_AND:
		return 2
	case lexer.TOKEN_EQ, lexer.TOKEN_NE, lexer.TOKEN_LT, lexer.TOKEN_LE, lexer.TOKEN_GT, lexer.TOKEN_GE, lexer.TOKEN_IN:
		return 3
	case lexer.TOKEN_PLUS, lexer.TOKEN_MINUS:
		return 4
	case lexer.TOKEN_MULTIPLY, lexer.TOKEN_DIVIDE, lexer.TOKEN_MODULO:
		return 5
	case lexer.TOKEN_POWER:
		return 6
	default:
		return 0
	}
}

// isBinaryOperator 检查是否是二元运算符
func isBinaryOperator(tokenType lexer.TokenType) bool {
	switch tokenType {
	case lexer.TOKEN_PLUS, lexer.TOKEN_MINUS, lexer.TOKEN_MULTIPLY, lexer.TOKEN_DIVIDE,
		lexer.TOKEN_MODULO, lexer.TOKEN_POWER, lexer.TOKEN_EQ, lexer.TOKEN_NE,
		lexer.TOKEN_LT, lexer.TOKEN_LE, lexer.TOKEN_GT, lexer.TOKEN_GE,
		lexer.TOKEN_AND, lexer.TOKEN_OR, lexer.TOKEN_IN:
		return true
	default:
		return false
	}
}

// parseFunctionCall 解析函数调用
func (p *Parser) parseFunctionCall(funcName string) (Expression, error) {
	p.nextToken() // 跳过左括号

	args := []Expression{}

	for p.currentToken.Type != lexer.TOKEN_RPAREN {
		arg, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		args = append(args, arg)

		if p.currentToken.Type == lexer.TOKEN_COMMA {
			p.nextToken()
		}
	}

	p.nextToken() // 跳过右括号

	return &FunctionCallExpression{
		Name:      funcName,
		Arguments: args,
	}, nil
}

// nextToken 读取下一个Token
func (p *Parser) nextToken() {
	p.currentToken = p.peekToken
	p.peekToken = p.lexer.NextToken()
}

// Errors 返回解析错误
func (p *Parser) Errors() []string {
	return p.errors
}

// ParseString 解析WQL字符串
func ParseString(input string) (*WQLQuery, error) {
	lex := lexer.NewLexer(input)
	p := NewParser(lex)
	return p.Parse()
}

// parseInsertOperation 解析 INSERT 操作
// 语法: .Insert(col1, val1, col2, val2, ...).Execute()
//        .Insert({col1: val1, col2: val2}).Execute()  (单行)
// 为了简单，采用 (col, val) 对列表形式
func (p *Parser) parseInsertOperation() (*InsertOperation, error) {
	// 当前 token 是 Insert
	p.nextToken() // 跳过 Insert

	if p.currentToken.Type != lexer.TOKEN_LPAREN {
		return nil, fmt.Errorf("expected '(' after Insert, got %s", p.currentToken.Type)
	}
	p.nextToken() // 跳过 (

	rows := []Expression{}

	// 解析多行：每行是 {col1: val1, col2: val2, ...} 形式
	// 简化: Insert(一对对 col, val, col, val...)
	// 我们解析成单行: 一对 (col,val) 对组成一个 map literal
	if p.currentToken.Type == lexer.TOKEN_LBRACE {
		// 对象字面量 {col: val, ...}
		obj, err := p.parseObjectLiteral()
		if err != nil {
			return nil, err
		}
		rows = append(rows, obj)

		if p.currentToken.Type == lexer.TOKEN_COMMA {
			p.nextToken()
			for p.currentToken.Type != lexer.TOKEN_RPAREN {
				obj2, err := p.parseObjectLiteral()
				if err != nil {
					return nil, err
				}
				rows = append(rows, obj2)
				if p.currentToken.Type == lexer.TOKEN_COMMA {
					p.nextToken()
				} else {
					break
				}
			}
		}
	} else {
		// 配对形式: col1, val1, col2, val2, ...
		row := &ObjectLiteralExpression{Fields: []ObjectField{}}
		for p.currentToken.Type != lexer.TOKEN_RPAREN {
			col, err := p.parseExpression()
			if err != nil {
				return nil, err
			}
			if p.currentToken.Type != lexer.TOKEN_COMMA {
				return nil, fmt.Errorf("expected ',' after column name in Insert, got %s", p.currentToken.Type)
			}
			p.nextToken() // 跳过 ,
			val, err := p.parseExpression()
			if err != nil {
				return nil, err
			}
			row.Fields = append(row.Fields, ObjectField{Key: col, Value: val})
			if p.currentToken.Type == lexer.TOKEN_COMMA {
				p.nextToken()
			} else {
				break
			}
		}
		rows = append(rows, row)
	}

	if p.currentToken.Type != lexer.TOKEN_RPAREN {
		return nil, fmt.Errorf("expected ')' to close Insert, got %s", p.currentToken.Type)
	}
	p.nextToken() // 跳过 )

	// Insert 必须依赖 db.Table 提供的表名, 返回的 Operation 暂留 Table 为 nil，
	// 在 buildQueryBuilder 中通过 query.Source 填充
	return &InsertOperation{Table: nil, Rows: rows}, nil
}

// parseObjectLiteral 解析 {key: val, key: val, ...}
func (p *Parser) parseObjectLiteral() (Expression, error) {
	if p.currentToken.Type != lexer.TOKEN_LBRACE {
		return nil, fmt.Errorf("expected '{', got %s", p.currentToken.Type)
	}
	p.nextToken() // 跳过 {

	obj := &ObjectLiteralExpression{Fields: []ObjectField{}}
	for p.currentToken.Type != lexer.TOKEN_RBRACE {
		// key 可以是 identifier 或 string
		var key Expression
		switch p.currentToken.Type {
		case lexer.TOKEN_IDENTIFIER, lexer.TOKEN_STRING:
			key = &Identifier{Value: p.currentToken.Value}
			p.nextToken()
		default:
			return nil, fmt.Errorf("expected key in object literal, got %s", p.currentToken.Type)
		}
		if p.currentToken.Type != lexer.TOKEN_COLON {
			return nil, fmt.Errorf("expected ':' after key, got %s", p.currentToken.Type)
		}
		p.nextToken() // 跳过 :
		val, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		obj.Fields = append(obj.Fields, ObjectField{Key: key, Value: val})
		if p.currentToken.Type == lexer.TOKEN_COMMA {
			p.nextToken()
		} else {
			break
		}
	}
	if p.currentToken.Type != lexer.TOKEN_RBRACE {
		return nil, fmt.Errorf("expected '}' to close object, got %s", p.currentToken.Type)
	}
	p.nextToken() // 跳过 }
	return obj, nil
}

// parseSetOperation 解析 SET 操作（用于 UPDATE）
// 语法: .Set(col1, val1, col2, val2, ...).Where(...).Execute()
func (p *Parser) parseSetOperation() (*SetOperation, error) {
	p.nextToken() // 跳过 Set

	if p.currentToken.Type != lexer.TOKEN_LPAREN {
		return nil, fmt.Errorf("expected '(' after Set, got %s", p.currentToken.Type)
	}
	p.nextToken() // 跳过 (

	updates := []Expression{}
	if p.currentToken.Type == lexer.TOKEN_LBRACE {
		obj, err := p.parseObjectLiteral()
		if err != nil {
			return nil, err
		}
		updates = append(updates, obj)
	} else {
		// 配对形式
		row := &ObjectLiteralExpression{Fields: []ObjectField{}}
		for p.currentToken.Type != lexer.TOKEN_RPAREN {
			col, err := p.parseExpression()
			if err != nil {
				return nil, err
			}
			if p.currentToken.Type != lexer.TOKEN_COMMA {
				return nil, fmt.Errorf("expected ',' after column in Set, got %s", p.currentToken.Type)
			}
			p.nextToken()
			val, err := p.parseExpression()
			if err != nil {
				return nil, err
			}
			row.Fields = append(row.Fields, ObjectField{Key: col, Value: val})
			if p.currentToken.Type == lexer.TOKEN_COMMA {
				p.nextToken()
			} else {
				break
			}
		}
		updates = append(updates, row)
	}

	if p.currentToken.Type != lexer.TOKEN_RPAREN {
		return nil, fmt.Errorf("expected ')' to close Set, got %s", p.currentToken.Type)
	}
	p.nextToken()

	return &SetOperation{Updates: updates}, nil
}

// parseDeleteOperation 解析 DELETE 操作
// 语法: .Delete()  （条件通过前面的 Where 提供）
func (p *Parser) parseDeleteOperation() (*DeleteOperation, error) {
	p.nextToken() // 跳过 Delete

	if p.currentToken.Type != lexer.TOKEN_LPAREN {
		return nil, fmt.Errorf("expected '(' after Delete, got %s", p.currentToken.Type)
	}
	p.nextToken()

	if p.currentToken.Type != lexer.TOKEN_RPAREN {
		return nil, fmt.Errorf("expected ')' to close Delete, got %s", p.currentToken.Type)
	}
	p.nextToken()

	return &DeleteOperation{Table: nil, Condition: nil}, nil
}

// parseCreateTableOperation 解析 CREATE TABLE 操作
// 语法: .Create(col1 TYPE, col2 TYPE, ..., colN TYPE).Execute()
func (p *Parser) parseCreateTableOperation() (*CreateTableOperation, error) {
	p.nextToken() // 跳过 Create

	if p.currentToken.Type != lexer.TOKEN_LPAREN {
		return nil, fmt.Errorf("expected '(' after Create, got %s", p.currentToken.Type)
	}
	p.nextToken()

	cols := []ColumnDef{}
	for p.currentToken.Type != lexer.TOKEN_RPAREN {
		if p.currentToken.Type != lexer.TOKEN_IDENTIFIER {
			return nil, fmt.Errorf("expected column name in Create, got %s", p.currentToken.Type)
		}
		colName := p.currentToken.Value
		p.nextToken()

		// 类型
		colType, err := p.parseColumnType()
		if err != nil {
			return nil, err
		}

		col := ColumnDef{Name: colName, Type: colType, Nullable: true, Primary: false}

		// 可选约束: PRIMARY KEY, NOT NULL, NULL
		for {
			switch p.currentToken.Type {
			case lexer.TOKEN_PRIMARY:
				p.nextToken()
				if p.currentToken.Type == lexer.TOKEN_KEY {
					p.nextToken()
				}
				col.Primary = true
				col.Nullable = false
			case lexer.TOKEN_NOT:
				p.nextToken()
				if p.currentToken.Type == lexer.TOKEN_NULL {
					p.nextToken()
				}
				col.Nullable = false
			case lexer.TOKEN_NULL:
				p.nextToken()
				col.Nullable = true
			default:
				goto doneConstraints
			}
		}
	doneConstraints:
		cols = append(cols, col)

		if p.currentToken.Type == lexer.TOKEN_COMMA {
			p.nextToken()
		} else {
			break
		}
	}

	if p.currentToken.Type != lexer.TOKEN_RPAREN {
		return nil, fmt.Errorf("expected ')' to close Create, got %s", p.currentToken.Type)
	}
	p.nextToken()

	return &CreateTableOperation{Table: nil, Columns: cols}, nil
}

// parseColumnType 解析列类型: INTEGER, TEXT, REAL, BLOB
func (p *Parser) parseColumnType() (string, error) {
	switch p.currentToken.Type {
	case lexer.TOKEN_INTEGER_TYPE:
		p.nextToken()
		return "INTEGER", nil
	case lexer.TOKEN_TEXT_TYPE:
		p.nextToken()
		return "TEXT", nil
	case lexer.TOKEN_REAL_TYPE:
		p.nextToken()
		return "REAL", nil
	case lexer.TOKEN_BLOB_TYPE:
		p.nextToken()
		return "BLOB", nil
	case lexer.TOKEN_IDENTIFIER:
		// 允许自定义类型 (大小写不敏感匹配)
		upper := strings.ToUpper(p.currentToken.Value)
		switch upper {
		case "INTEGER", "INT":
			p.nextToken()
			return "INTEGER", nil
		case "TEXT", "VARCHAR", "STRING":
			p.nextToken()
			return "TEXT", nil
		case "REAL", "FLOAT", "DOUBLE":
			p.nextToken()
			return "REAL", nil
		case "BLOB":
			p.nextToken()
			return "BLOB", nil
		}
		return "", fmt.Errorf("unknown column type: %s", p.currentToken.Value)
	}
	return "", fmt.Errorf("expected column type, got %s", p.currentToken.Type)
}

// parseDropTableOperation 解析 DROP TABLE
// 语法: .Drop()
func (p *Parser) parseDropTableOperation() (*DropTableOperation, error) {
	p.nextToken() // 跳过 Drop

	if p.currentToken.Type != lexer.TOKEN_LPAREN {
		return nil, fmt.Errorf("expected '(' after Drop, got %s", p.currentToken.Type)
	}
	p.nextToken()

	if p.currentToken.Type != lexer.TOKEN_RPAREN {
		return nil, fmt.Errorf("expected ')' to close Drop, got %s", p.currentToken.Type)
	}
	p.nextToken()

	return &DropTableOperation{Table: nil}, nil
}

// parseExecuteOperation 解析 Execute() 终结符
func (p *Parser) parseExecuteOperation() (*ExecuteOperation, error) {
	p.nextToken() // 跳过 Execute

	if p.currentToken.Type != lexer.TOKEN_LPAREN {
		return nil, fmt.Errorf("expected '(' after Execute, got %s", p.currentToken.Type)
	}
	p.nextToken()

	if p.currentToken.Type != lexer.TOKEN_RPAREN {
		return nil, fmt.Errorf("expected ')' to close Execute, got %s", p.currentToken.Type)
	}
	p.nextToken()

	return &ExecuteOperation{Terminator: true}, nil
}
