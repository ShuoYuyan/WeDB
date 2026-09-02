package parser

import (
	"fmt"
)

// Node AST节点接口
type Node interface {
	String() string
}

// Expression 表达式接口
type Expression interface {
	Node
	expressionNode()
}

// Statement 语句接口
type Statement interface {
	Node
	statementNode()
}

// ExpressionImpl 表达式实现
type ExpressionImpl struct{}

func (e ExpressionImpl) expressionNode() {}

// StatementImpl 语句实现
type StatementImpl struct{}

func (s StatementImpl) statementNode() {}

// WQLQuery WQL查询语句
type WQLQuery struct {
	Source      string
	Operations  []Operation
}

func (q *WQLQuery) String() string {
	return fmt.Sprintf("WQLQuery{Source: %s, Operations: %v}", q.Source, q.Operations)
}

func (q *WQLQuery) statementNode() {}

// Operation 操作接口
type Operation interface {
	Node
	operationNode()
}

// OperationImpl 操作实现
type OperationImpl struct{}

func (o OperationImpl) operationNode() {}

// TableOperation 表操作
type TableOperation struct {
	TableName Expression
}

func (t *TableOperation) String() string {
	return fmt.Sprintf("Table(%s)", t.TableName)
}

func (t *TableOperation) operationNode() {}

// SelectOperation SELECT操作
type SelectOperation struct {
	Columns []Expression
}

func (s *SelectOperation) String() string {
	cols := make([]string, len(s.Columns))
	for i, col := range s.Columns {
		cols[i] = col.String()
	}
	return fmt.Sprintf("Select(%s)", fmt.Sprintf("%v", cols))
}

func (s *SelectOperation) operationNode() {}

// WhereOperation WHERE操作
type WhereOperation struct {
	Condition Expression
}

func (w *WhereOperation) String() string {
	return fmt.Sprintf("Where(%s)", w.Condition)
}

func (w *WhereOperation) operationNode() {}

// JoinOperation JOIN操作
type JoinOperation struct {
	JoinType  string
	Table     Expression
	LeftKey   Expression
	RightKey  Expression
	Condition Expression
}

func (j *JoinOperation) String() string {
	return fmt.Sprintf("Join(%s, %s, %s)", j.JoinType, j.Table, j.LeftKey)
}

func (j *JoinOperation) operationNode() {}

// OrderByOperation ORDER BY操作
type OrderByOperation struct {
	Column    Expression
	Direction string
}

func (o *OrderByOperation) String() string {
	return fmt.Sprintf("OrderBy(%s, %s)", o.Column, o.Direction)
}

func (o *OrderByOperation) operationNode() {}

// GroupByOperation GROUP BY操作
type GroupByOperation struct {
	Columns []Expression
}

func (g *GroupByOperation) String() string {
	cols := make([]string, len(g.Columns))
	for i, col := range g.Columns {
		cols[i] = col.String()
	}
	return fmt.Sprintf("GroupBy(%v)", cols)
}

func (g *GroupByOperation) operationNode() {}

// HavingOperation HAVING操作
type HavingOperation struct {
	Condition Expression
}

func (h *HavingOperation) String() string {
	return fmt.Sprintf("Having(%s)", h.Condition)
}

func (h *HavingOperation) operationNode() {}

// LimitOperation LIMIT操作
type LimitOperation struct {
	Count  Expression
	Offset Expression
}

func (l *LimitOperation) String() string {
	if l.Offset != nil {
		return fmt.Sprintf("Limit(%s, %s)", l.Count, l.Offset)
	}
	return fmt.Sprintf("Limit(%s)", l.Count)
}

func (l *LimitOperation) operationNode() {}

// TakeOperation TAKE操作
type TakeOperation struct {
	Count Expression
}

func (t *TakeOperation) String() string {
	return fmt.Sprintf("Take(%s)", t.Count)
}

func (t *TakeOperation) operationNode() {}

// SkipOperation SKIP操作
type SkipOperation struct {
	Count Expression
}

func (s *SkipOperation) String() string {
	return fmt.Sprintf("Skip(%s)", s.Count)
}

func (s *SkipOperation) operationNode() {}

// FirstOperation FIRST操作
type FirstOperation struct{}

func (f *FirstOperation) String() string {
	return "First()"
}

func (f *FirstOperation) operationNode() {}

// AllOperation ALL操作
type AllOperation struct{}

func (a *AllOperation) String() string {
	return "All()"
}

func (a *AllOperation) operationNode() {}

// AggregateOperation 聚合操作
type AggregateOperation struct {
	Function string
	Column   Expression
	Alias    string
}

func (a *AggregateOperation) String() string {
	if a.Alias != "" {
		return fmt.Sprintf("%s(%s AS %s)", a.Function, a.Column, a.Alias)
	}
	return fmt.Sprintf("%s(%s)", a.Function, a.Column)
}

func (a *AggregateOperation) operationNode() {}

// Identifier 标识符表达式
type Identifier struct {
	Value string
}

func (i *Identifier) String() string {
	return i.Value
}

func (i *Identifier) expressionNode() {}

// BinaryExpression 二元表达式
type BinaryExpression struct {
	Left     Expression
	Operator string
	Right    Expression
}

func (b *BinaryExpression) String() string {
	return fmt.Sprintf("(%s %s %s)", b.Left, b.Operator, b.Right)
}

func (b *BinaryExpression) expressionNode() {}

// LiteralExpression 字面量表达式
type LiteralExpression struct {
	Value interface{}
}

func (l *LiteralExpression) String() string {
	return fmt.Sprintf("%v", l.Value)
}

func (l *LiteralExpression) expressionNode() {}

// FunctionCallExpression 函数调用表达式
type FunctionCallExpression struct {
	Name      string
	Arguments []Expression
}

func (f *FunctionCallExpression) String() string {
	args := make([]string, len(f.Arguments))
	for i, arg := range f.Arguments {
		args[i] = arg.String()
	}
	return fmt.Sprintf("%s(%s)", f.Name, fmt.Sprintf("%v", args))
}

func (f *FunctionCallExpression) expressionNode() {}

// CallExpression 方法调用表达式
type CallExpression struct {
	Callee    Expression
	Arguments []Expression
}

func (c *CallExpression) String() string {
	args := make([]string, len(c.Arguments))
	for i, arg := range c.Arguments {
		args[i] = arg.String()
	}
	return fmt.Sprintf("%s(%s)", c.Callee, fmt.Sprintf("%v", args))
}

func (c *CallExpression) expressionNode() {}

// ArrayExpression 数组表达式
type ArrayExpression struct {
	Elements []Expression
}

func (a *ArrayExpression) String() string {
	elements := make([]string, len(a.Elements))
	for i, elem := range a.Elements {
		elements[i] = elem.String()
	}
	return fmt.Sprintf("[%s]", fmt.Sprintf("%v", elements))
}

func (a *ArrayExpression) expressionNode() {}

// UnionOperation UNION操作
type UnionOperation struct {
	Table    Expression
	All      bool
}

func (u *UnionOperation) String() string {
	if u.All {
		return fmt.Sprintf("UnionAll(%s)", u.Table)
	}
	return fmt.Sprintf("Union(%s)", u.Table)
}

func (u *UnionOperation) operationNode() {}

// IntersectOperation INTERSECT操作
type IntersectOperation struct {
	Table Expression
}

func (i *IntersectOperation) String() string {
	return fmt.Sprintf("Intersect(%s)", i.Table)
}

func (i *IntersectOperation) operationNode() {}

// ExceptOperation EXCEPT操作
type ExceptOperation struct {
	Table Expression
}

func (e *ExceptOperation) String() string {
	return fmt.Sprintf("Except(%s)", e.Table)
}

func (e *ExceptOperation) operationNode() {}

// SubqueryExpression 子查询表达式
type SubqueryExpression struct {
	Query *WQLQuery
	Alias string
}

func (s *SubqueryExpression) String() string {
	if s.Alias != "" {
		return fmt.Sprintf("(%s) AS %s", s.Query.String(), s.Alias)
	}
	return fmt.Sprintf("(%s)", s.Query.String())
}

func (s *SubqueryExpression) expressionNode() {}
