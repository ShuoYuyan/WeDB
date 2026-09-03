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
	case lexer.TOKEN_COUNT, lexer.TOKEN_SUM, lexer.TOKEN_AVG,
		lexer.TOKEN_MIN, lexer.TOKEN_MAX:
		return p.parseAggregateOperation()
	case lexer.TOKEN_DISTINCT:
		return p.parseDistinctOperation()
	case lexer.TOKEN_BEGIN, lexer.TOKEN_COMMIT, lexer.TOKEN_ROLLBACK:
		return p.parseTransactionOperation()
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
// 语法:
//   .Join(otherTable)                                       -- 笛卡尔积
//   .Join(otherTable, leftCol, rightCol)                    -- 单列等值连接 (兼容旧语法)
//   .Join(otherTable, ON left.col = right.col)              -- ON 条件
//   .Join(otherTable, ON left.col = right.col AND l2 = r2)  -- 复合 ON 条件
func (p *Parser) parseJoinOperation() (*JoinOperation, error) {
	joinType := p.currentToken.Value

	p.nextToken()

	if p.currentToken.Type != lexer.TOKEN_LPAREN {
		return nil, fmt.Errorf("expected '(' after %s, got %s", joinType, p.currentToken.Type)
	}
	p.nextToken()

	// 第一个参数：表名（必须为标识符）
	table, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	join := &JoinOperation{JoinType: joinType, Table: table}

	// 检查是否有更多参数
	if p.currentToken.Type != lexer.TOKEN_COMMA {
		if p.currentToken.Type != lexer.TOKEN_RPAREN {
			return nil, fmt.Errorf("expected ',' or ')' in Join, got %s", p.currentToken.Type)
		}
		p.nextToken()
		return join, nil
	}
	p.nextToken()

	// 探测 ON 关键字
	if p.currentToken.Type == lexer.TOKEN_ON {
		p.nextToken()
		// 解析 ON 后的整个表达式
		cond, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		join.Condition = cond
		if p.currentToken.Type != lexer.TOKEN_RPAREN {
			return nil, fmt.Errorf("expected ')' after Join ON, got %s", p.currentToken.Type)
		}
		p.nextToken()
		return join, nil
	}

	// 兼容旧语法：Join(t, leftKey, rightKey)
	leftKey, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	join.LeftKey = leftKey

	if p.currentToken.Type != lexer.TOKEN_COMMA {
		return nil, fmt.Errorf("expected ',' in Join (leftKey, rightKey), got %s", p.currentToken.Type)
	}
	p.nextToken()

	rightKey, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	join.RightKey = rightKey

	if p.currentToken.Type != lexer.TOKEN_RPAREN {
		return nil, fmt.Errorf("expected ')' to close Join, got %s", p.currentToken.Type)
	}
	p.nextToken()

	return join, nil
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

// parseAggregateOperation 解析聚合终端操作：Count() / Sum(col) / Avg(col) / Min(col) / Max(col)
// 支持别名：Count() AS cnt, Sum(amount) AS total
func (p *Parser) parseAggregateOperation() (*AggregateOperation, error) {
	funcName := p.currentToken.Value
	p.nextToken()

	if p.currentToken.Type != lexer.TOKEN_LPAREN {
		return nil, fmt.Errorf("expected '(' after %s, got %s", funcName, p.currentToken.Type)
	}
	p.nextToken()

	agg := &AggregateOperation{Function: funcName}
	// Count() 不需要参数；Sum/Avg/Min/Max 需要 1 个参数
	if p.currentToken.Type != lexer.TOKEN_RPAREN {
		col, err := p.parseExpression()
		if err != nil {
			return nil, fmt.Errorf("failed to parse %s column: %w", funcName, err)
		}
		agg.Column = col
	}

	if p.currentToken.Type != lexer.TOKEN_RPAREN {
		return nil, fmt.Errorf("expected ')' in %s, got %s", funcName, p.currentToken.Type)
	}
	p.nextToken()

	// 可选: AS alias
	if p.currentToken.Type == lexer.TOKEN_AS {
		p.nextToken()
		if p.currentToken.Type != lexer.TOKEN_IDENTIFIER {
			return nil, fmt.Errorf("expected identifier after AS, got %s", p.currentToken.Type)
		}
		agg.Alias = p.currentToken.Value
		p.nextToken()
	}

	return agg, nil
}

// parseUnionOperation 解析UNION操作
// parseDistinctOperation 解析 DISTINCT 操作
// 语法: .Distinct() 或 .Distinct(col1, col2, ...)
func (p *Parser) parseDistinctOperation() (*DistinctOperation, error) {
	p.nextToken() // 跳过 DISTINCT

	if p.currentToken.Type != lexer.TOKEN_LPAREN {
		return nil, fmt.Errorf("expected '(' after Distinct, got %s", p.currentToken.Type)
	}
	p.nextToken()

	cols := []Expression{}
	for p.currentToken.Type != lexer.TOKEN_RPAREN {
		expr, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		cols = append(cols, expr)
		if p.currentToken.Type == lexer.TOKEN_COMMA {
			p.nextToken()
		}
	}
	p.nextToken() // 跳过 RPAREN

	return &DistinctOperation{Columns: cols}, nil
}

// parseTransactionOperation 解析事务操作：BEGIN / COMMIT / ROLLBACK
func (p *Parser) parseTransactionOperation() (*TransactionOperation, error) {
	action := strings.ToUpper(p.currentToken.Value)
	p.nextToken()
	return &TransactionOperation{Action: action}, nil
}

func (p *Parser) parseUnionOperation() (*UnionOperation, error) {
	all := p.currentToken.Type == lexer.TOKEN_UNION_ALL
	p.nextToken()

	if p.currentToken.Type != lexer.TOKEN_LPAREN {
		return nil, fmt.Errorf("expected '(', got %s", p.currentToken.Type)
	}
	p.nextToken()

	// 收集从当前位置到匹配右括号的 token，并保持括号嵌套
	// 期望子查询以 db.Table(...) 开头
	var tokens []lexer.Token
	parenLevel := 1 // 跨过左括号
	for {
		if p.currentToken.Type == lexer.TOKEN_LPAREN {
			parenLevel++
		} else if p.currentToken.Type == lexer.TOKEN_RPAREN {
			parenLevel--
			if parenLevel == 0 {
				break
			}
		} else if p.currentToken.Type == lexer.TOKEN_EOF {
			return nil, fmt.Errorf("unterminated Union subquery")
		}
		tokens = append(tokens, p.currentToken)
		p.nextToken()
	}
	p.nextToken() // 跳过 RPAREN

	if len(tokens) == 0 {
		return nil, fmt.Errorf("empty Union subquery")
	}

	subQuery, err := tokensToSubquery(tokens)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Union subquery: %w", err)
	}

	return &UnionOperation{
		Table: subQuery,
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

	var tokens []lexer.Token
	parenLevel := 1
	for {
		if p.currentToken.Type == lexer.TOKEN_LPAREN {
			parenLevel++
		} else if p.currentToken.Type == lexer.TOKEN_RPAREN {
			parenLevel--
			if parenLevel == 0 {
				break
			}
		} else if p.currentToken.Type == lexer.TOKEN_EOF {
			return nil, fmt.Errorf("unterminated Intersect subquery")
		}
		tokens = append(tokens, p.currentToken)
		p.nextToken()
	}
	p.nextToken()

	if len(tokens) == 0 {
		return nil, fmt.Errorf("empty Intersect subquery")
	}

	subQuery, err := tokensToSubquery(tokens)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Intersect subquery: %w", err)
	}

	return &IntersectOperation{Table: subQuery}, nil
}

// parseExceptOperation 解析EXCEPT操作
func (p *Parser) parseExceptOperation() (*ExceptOperation, error) {
	p.nextToken()

	if p.currentToken.Type != lexer.TOKEN_LPAREN {
		return nil, fmt.Errorf("expected '(', got %s", p.currentToken.Type)
	}
	p.nextToken()

	var tokens []lexer.Token
	parenLevel := 1
	for {
		if p.currentToken.Type == lexer.TOKEN_LPAREN {
			parenLevel++
		} else if p.currentToken.Type == lexer.TOKEN_RPAREN {
			parenLevel--
			if parenLevel == 0 {
				break
			}
		} else if p.currentToken.Type == lexer.TOKEN_EOF {
			return nil, fmt.Errorf("unterminated Except subquery")
		}
		tokens = append(tokens, p.currentToken)
		p.nextToken()
	}
	p.nextToken()

	if len(tokens) == 0 {
		return nil, fmt.Errorf("empty Except subquery")
	}

	subQuery, err := tokensToSubquery(tokens)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Except subquery: %w", err)
	}

	return &ExceptOperation{Table: subQuery}, nil
}

// tokensToSubquery 从 token 序列重建字符串并解析为 *WQLQuery
func tokensToSubquery(tokens []lexer.Token) (*WQLQuery, error) {
	var sb strings.Builder
	for i, t := range tokens {
		if i > 0 {
			sb.WriteString(" ")
		}
		sb.WriteString(t.Value)
	}
	subLexer := lexer.NewLexer(sb.String())
	subParser := NewParser(subLexer)
	return subParser.Parse()
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

	// 后缀处理：IN / NOT IN / BETWEEN / LIKE / IS NULL / IS NOT NULL
	for {
		// NOT IN / NOT LIKE / NOT BETWEEN — 用 peekToken 预判避免回退
		if p.currentToken.Type == lexer.TOKEN_NOT && isPostfixNotOp(p.peekToken.Type) {
			p.nextToken() // 跳过 NOT
			switch p.currentToken.Type {
			case lexer.TOKEN_IN:
				p.nextToken()
				if p.currentToken.Type != lexer.TOKEN_LPAREN {
					return nil, fmt.Errorf("expected '(' after NOT IN, got %s", p.currentToken.Type)
				}
				p.nextToken()
				values := []Expression{}
				for p.currentToken.Type != lexer.TOKEN_RPAREN {
					v, err := p.parseExpression()
					if err != nil {
						return nil, err
					}
					values = append(values, v)
					if p.currentToken.Type == lexer.TOKEN_COMMA {
						p.nextToken()
					}
				}
				if p.currentToken.Type != lexer.TOKEN_RPAREN {
					return nil, fmt.Errorf("expected ')' after NOT IN list, got %s", p.currentToken.Type)
				}
				p.nextToken()
				left = &InExpression{Column: left, Values: values, Not: true}
				continue
			case lexer.TOKEN_LIKE:
				// 不要 p.nextToken()：保留 currentToken=LIKE, peekToken=下一 token
				// 把 peekToken 当作 pattern 的开始；如果 lexer 当前位置是 % 或 _，
				// 就把它们 append 到 pattern。
				patTok := p.peekToken
				// 检查当前 lexer 位置，看能否扩展 patTok 的 value
				p.lexer.ExtendIdentifierValue(&patTok)
				p.nextToken() // 现在 currentToken = patTok
				p.nextToken() // 跳过 pattern
				left = &LikeExpression{Column: left, Pattern: &Identifier{Value: patTok.Value}, Not: true}
				continue
			case lexer.TOKEN_BETWEEN:
				p.nextToken()
				low, err := p.parsePrimaryExpression()
				if err != nil {
					return nil, err
				}
				if p.currentToken.Type != lexer.TOKEN_AND {
					return nil, fmt.Errorf("expected AND after NOT BETWEEN low, got %s", p.currentToken.Type)
				}
				p.nextToken()
				high, err := p.parsePrimaryExpression()
				if err != nil {
					return nil, err
				}
				left = &BetweenExpression{Column: left, Low: low, High: high, Not: true}
				continue
			}
		}

		switch p.currentToken.Type {
		case lexer.TOKEN_IN:
			p.nextToken()
			if p.currentToken.Type != lexer.TOKEN_LPAREN {
				return nil, fmt.Errorf("expected '(' after IN, got %s", p.currentToken.Type)
			}
			p.nextToken()
			values := []Expression{}
			for p.currentToken.Type != lexer.TOKEN_RPAREN {
				// 支持子查询作为 IN 的值
				if p.currentToken.Type == lexer.TOKEN_DB {
					sq, err := p.parseSubqueryStartingHere()
					if err != nil {
						return nil, err
					}
					values = append(values, sq)
				} else {
					v, err := p.parseExpression()
					if err != nil {
						return nil, err
					}
					values = append(values, v)
				}
				if p.currentToken.Type == lexer.TOKEN_COMMA {
					p.nextToken()
				}
			}
			if p.currentToken.Type != lexer.TOKEN_RPAREN {
				return nil, fmt.Errorf("expected ')' after IN list, got %s", p.currentToken.Type)
			}
			p.nextToken()
			left = &InExpression{Column: left, Values: values, Not: false}
			continue
		case lexer.TOKEN_BETWEEN:
			p.nextToken()
			low, err := p.parsePrimaryExpression()
			if err != nil {
				return nil, err
			}
			if p.currentToken.Type != lexer.TOKEN_AND {
				return nil, fmt.Errorf("expected AND after BETWEEN low, got %s", p.currentToken.Type)
			}
			p.nextToken()
			high, err := p.parsePrimaryExpression()
			if err != nil {
				return nil, err
			}
			left = &BetweenExpression{Column: left, Low: low, High: high, Not: false}
			continue
		case lexer.TOKEN_LIKE:
			patTok := p.peekToken
			p.lexer.ExtendIdentifierValue(&patTok)
			p.nextToken()
			p.nextToken()
			left = &LikeExpression{Column: left, Pattern: &Identifier{Value: patTok.Value}, Not: false}
			continue
		case lexer.TOKEN_IS:
			p.nextToken()
			notNull := false
			if p.currentToken.Type == lexer.TOKEN_NOT {
				notNull = true
				p.nextToken()
			}
			if p.currentToken.Type != lexer.TOKEN_NULL {
				return nil, fmt.Errorf("expected NULL after IS%s, got %s",
					map[bool]string{true: " NOT", false: ""}[notNull], p.currentToken.Type)
			}
			p.nextToken()
			left = &IsNullExpression{Column: left, Not: notNull}
			continue
		}
		break
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
	case lexer.TOKEN_NOT:
		// 一元 NOT(...)
		p.nextToken()
		if p.currentToken.Type != lexer.TOKEN_LPAREN {
			return nil, fmt.Errorf("expected '(' after NOT, got %s", p.currentToken.Type)
		}
		p.nextToken()
		expr, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		if p.currentToken.Type != lexer.TOKEN_RPAREN {
			return nil, fmt.Errorf("expected ')' to close NOT, got %s", p.currentToken.Type)
		}
		p.nextToken()
		return &UnaryExpression{Operator: "NOT", Operand: expr}, nil
	case lexer.TOKEN_LPAREN:
		// 可能是 IN (v1, v2, ...) / 子查询 / 括号表达式
		p.nextToken()
		// 子查询：(db.Table(...)...)
		if p.currentToken.Type == lexer.TOKEN_DB {
			return p.parseSubquery()
		}
		// 尝试解析为 IN 值列表：如果第一项是字面量/标识符，且后面有 ')' 紧跟 IS / 标识符，
		// 那么这是分组表达式
		expr, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		if p.currentToken.Type != lexer.TOKEN_RPAREN {
			return nil, fmt.Errorf("expected ')', got %s", p.currentToken.Type)
		}
		p.nextToken()
		return expr, nil
	case lexer.TOKEN_IDENTIFIER, lexer.TOKEN_COUNT, lexer.TOKEN_SUM, lexer.TOKEN_AVG, lexer.TOKEN_MIN, lexer.TOKEN_MAX,
		lexer.TOKEN_JSON_EXTRACT, lexer.TOKEN_JSON_QUERY, lexer.TOKEN_JSON_VALUE:
		// 支持带点的标识符，如 users.id
		value := p.currentToken.Value
		p.nextToken()

		// 检查是否是函数调用：FunctionName(...)
		if p.currentToken.Type == lexer.TOKEN_LPAREN {
			// 可能是 IN / 普通函数 / subquery
			return p.parseFunctionOrInCall(value)
		}

		// 检查是否有点，如果有，则构造复合标识符
		if p.currentToken.Type == lexer.TOKEN_DOT {
			p.nextToken()
			if p.currentToken.Type == lexer.TOKEN_IDENTIFIER {
				// 复合标识符，如 users.id
				compoundValue := value + "." + p.currentToken.Value
				p.nextToken()
				// 可能还有方法调用
				if p.currentToken.Type == lexer.TOKEN_LPAREN {
					return p.parseFunctionOrInCall(compoundValue)
				}
				return &Identifier{Value: compoundValue}, nil
			}
		}

		return &Identifier{Value: value}, nil
	case lexer.TOKEN_INTEGER, lexer.TOKEN_FLOAT, lexer.TOKEN_STRING, lexer.TOKEN_BOOLEAN:
		value := p.currentToken.Value
		p.nextToken()
		return &LiteralExpression{Value: value}, nil
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

// parseSubqueryStartingHere 期望 currentToken 是子查询的开头（如 db）。
// 与 parseSubquery 不同：本函数期望 currentToken 已经是子查询的第一个 token，
// 内部从头扫描整个子查询直到遇到匹配的右括号。
func (p *Parser) parseSubqueryStartingHere() (Expression, error) {
	var tokens []lexer.Token
	parenLevel := 0

	for {
		// db.Table(t).Where(...).All() 这种 — 整体作为一个表达式，里面含 ( )
		// 我们粗略收集到第一个未匹配的右括号或右方法
		if p.currentToken.Type == lexer.TOKEN_LPAREN {
			parenLevel++
		} else if p.currentToken.Type == lexer.TOKEN_RPAREN {
			if parenLevel == 0 {
				break
			}
			parenLevel--
		}
		tokens = append(tokens, p.currentToken)
		p.nextToken()
		if p.currentToken.Type == lexer.TOKEN_EOF {
			return nil, fmt.Errorf("unterminated subquery expression")
		}
	}

	if len(tokens) == 0 {
		return nil, fmt.Errorf("empty subquery")
	}

	// 重新构建子查询字符串
	var subqueryBuilder strings.Builder
	for i, tok := range tokens {
		if i > 0 {
			// 在 token 之间插入空格以便 lexer 正确切分
			subqueryBuilder.WriteString(" ")
		}
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

	return &SubqueryExpression{Query: subQuery}, nil
}

// getOperatorPrecedence 获取运算符优先级
func (p *Parser) getOperatorPrecedence(tokenType lexer.TokenType) int {
	switch tokenType {
	case lexer.TOKEN_OR:
		return 1
	case lexer.TOKEN_AND:
		return 2
	case lexer.TOKEN_EQ, lexer.TOKEN_NE, lexer.TOKEN_LT, lexer.TOKEN_LE, lexer.TOKEN_GT, lexer.TOKEN_GE:
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
		lexer.TOKEN_AND, lexer.TOKEN_OR:
		return true
	default:
		return false
	}
}

// isPostfixNotOp 报告 type 是否是 NOT 之后合法的后缀关键字（IN/LIKE/BETWEEN）
func isPostfixNotOp(tokenType lexer.TokenType) bool {
	switch tokenType {
	case lexer.TOKEN_IN, lexer.TOKEN_LIKE, lexer.TOKEN_BETWEEN:
		return true
	}
	return false
}

// parseFunctionCall 解析函数调用
// parseFunctionOrInCall 解析类似 Name(...) 的调用：
//   - 普通函数调用：Count(), Sum(amount), Avg(price)
//   - IN 表达式：IN (1, 2, 3) — 但 IN 是关键字而非列名，因此由 parsePrimaryExpression 直接处理
//   - 关键字调用：Count/Sum/Avg/Min/Max 走函数调用
func (p *Parser) parseFunctionOrInCall(name string) (Expression, error) {
	// 进入时 currentToken 已经在左括号
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
		Name:      name,
		Arguments: args,
	}, nil
}

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
// parseLikePattern 解析 LIKE 模式（无双引号：直接读取 a% / _x% 等）
// 返回模式字符串。模式中可以包含字母、数字、_、%。
func (p *Parser) parseLikePattern() (string, error) {
	if p.currentToken.Type == lexer.TOKEN_STRING {
		v := p.currentToken.Value
		p.nextToken()
		return v, nil
	}
	// 当前 token 应为标识符（包含 % 和 _）
	if p.currentToken.Type != lexer.TOKEN_IDENTIFIER {
		return "", fmt.Errorf("expected LIKE pattern, got %s", p.currentToken.Type)
	}
	return p.currentToken.Value, nil
}

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
