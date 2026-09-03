package parser

import (
	"fmt"
	"strings"
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

// DMLOperation DML操作（INSERT/UPDATE/DELETE）的统一接口
type DMLOperation interface {
	Operation
	dmlNode()
}

// DDLOperation DDL操作（CREATE/DROP）的统一接口
type DDLOperation interface {
	Operation
	ddlNode()
}

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

// ===== DML Operations =====

// InsertOperation INSERT 操作
// 语法（无双引号设计）:
//     db.Table(users).Insert({id: 1, name: alice, age: 30}).Execute()
//     db.Table(users).Insert({id: 1, name: alice}, {id: 2, name: bob}).Execute()
//     db.Table(users).Insert({id: 1, name: alice}).OnConflict(UPDATE, id).Execute()
type InsertOperation struct {
	Table         Expression // 表名（来自 db.Table(name)）
	Rows          []Expression // 值列表，每个是 {col: val, ...} 形式的对象表达式
	OnConflict    string    // 可选: "UPDATE" / "IGNORE" / "DO NOTHING"
	OnConflictKey string    // 可选: 冲突检测键
}

// OnConflictOperation ON CONFLICT 子句：附加在 Insert 后
// 例: .OnConflict(UPDATE, id) — id 冲突时改为更新
//     .OnConflict(IGNORE)     — 冲突时跳过
type OnConflictOperation struct {
	Strategy string // "UPDATE" / "IGNORE" / "DO NOTHING"
	Key      string // 冲突键（列名）；空表示自动选择
}

func (o *OnConflictOperation) String() string {
	if o.Key != "" {
		return fmt.Sprintf("ON_CONFLICT(%s, %s)", o.Strategy, o.Key)
	}
	return fmt.Sprintf("ON_CONFLICT(%s)", o.Strategy)
}

func (o *OnConflictOperation) operationNode() {}

func (i *InsertOperation) String() string {
	rows := make([]string, len(i.Rows))
	for idx, r := range i.Rows {
		rows[idx] = r.String()
	}
	return fmt.Sprintf("Insert(%s, [%s])", i.Table, fmt.Sprintf("%v", rows))
}

func (i *InsertOperation) operationNode() {}
func (i *InsertOperation) dmlNode()      {}

// UpdateOperation UPDATE 操作
// 语法: db.Table(users).Set(name, "alice").Where(id = 1).Execute()
type UpdateOperation struct {
	Table     Expression // 表名
	Updates   []Expression // Set 调用的列=值对列表
	Condition Expression // WHERE 条件（可选）
}

func (u *UpdateOperation) String() string {
	updates := make([]string, len(u.Updates))
	for idx, u2 := range u.Updates {
		updates[idx] = u2.String()
	}
	if u.Condition != nil {
		return fmt.Sprintf("Update(%s, Set=[%s], Where=%s)", u.Table, fmt.Sprintf("%v", updates), u.Condition)
	}
	return fmt.Sprintf("Update(%s, Set=[%s])", u.Table, fmt.Sprintf("%v", updates))
}

func (u *UpdateOperation) operationNode() {}
func (u *UpdateOperation) dmlNode()      {}

// DeleteOperation DELETE 操作
// 语法: db.Table(users).Where(age < 18).Delete().Execute()
type DeleteOperation struct {
	Table     Expression // 表名
	Condition Expression // WHERE 条件（可选）
}

func (d *DeleteOperation) String() string {
	if d.Condition != nil {
		return fmt.Sprintf("Delete(%s, Where=%s)", d.Table, d.Condition)
	}
	return fmt.Sprintf("Delete(%s)", d.Table)
}

func (d *DeleteOperation) operationNode() {}
func (d *DeleteOperation) dmlNode()      {}

// SetOperation UPDATE 的 SET 子句
// 语法: .Set(col1, val1, col2, val2, ...)
// 通常紧跟在 db.Table(users) 之后，并由 .Execute() 终结
type SetOperation struct {
	Updates   []Expression // ObjectLiteralExpression 列表（每行一个对象的更新）
	Condition Expression   // 关联的 WHERE 条件（由 buildQueryBuilder 填充）
}

func (s *SetOperation) String() string {
	ups := make([]string, len(s.Updates))
	for i, u := range s.Updates {
		ups[i] = u.String()
	}
	return fmt.Sprintf("Set([%s])", fmt.Sprintf("%v", ups))
}

func (s *SetOperation) operationNode() {}

// ObjectLiteralExpression 对象字面量 {key: value, ...}
// 用于 Insert 和 Set 操作的行/更新定义
type ObjectLiteralExpression struct {
	Fields []ObjectField
}

// ObjectField 对象字面量的字段
type ObjectField struct {
	Key   Expression
	Value Expression
}

func (o *ObjectLiteralExpression) String() string {
	parts := make([]string, len(o.Fields))
	for i, f := range o.Fields {
		parts[i] = fmt.Sprintf("%s: %s", f.Key, f.Value)
	}
	return fmt.Sprintf("{%s}", strings.Join(parts, ", "))
}

func (o *ObjectLiteralExpression) expressionNode() {}

// ExecuteOperation 终结操作
// 语法: .Execute() 表示 DML 语句的终结
type ExecuteOperation struct {
	// 标记这是一个终结点
	Terminator bool
}

func (e *ExecuteOperation) String() string {
	return "Execute()"
}

func (e *ExecuteOperation) operationNode() {}

// ===== DDL Operations =====

// CreateTableOperation CREATE TABLE 操作
// 语法: db.Table(users).Create(id INTEGER, name TEXT, age INTEGER).Execute()
type CreateTableOperation struct {
	Table  Expression // 表名
	Columns []ColumnDef
}

func (c *CreateTableOperation) String() string {
	cols := make([]string, len(c.Columns))
	for idx, col := range c.Columns {
		cols[idx] = col.String()
	}
	return fmt.Sprintf("CreateTable(%s, [%s])", c.Table, fmt.Sprintf("%v", cols))
}

func (c *CreateTableOperation) operationNode() {}
func (c *CreateTableOperation) ddlNode()      {}

// DropTableOperation DROP TABLE 操作
// 语法: db.Table(users).Drop().Execute()
type DropTableOperation struct {
	Table Expression // 表名
}

func (d *DropTableOperation) String() string {
	return fmt.Sprintf("DropTable(%s)", d.Table)
}

func (d *DropTableOperation) operationNode() {}
func (d *DropTableOperation) ddlNode()      {}

// ColumnDef 列定义（用于 CREATE TABLE）
type ColumnDef struct {
	Name     string
	Type     string
	Nullable bool
	Primary  bool
}

func (c *ColumnDef) String() string {
	s := c.Name + " " + c.Type
	if c.Primary {
		s += " PRIMARY KEY"
	}
	if !c.Nullable {
		s += " NOT NULL"
	}
	return s
}

// ===== Aggregations =====
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
	Table *WQLQuery
	All   bool
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
	Table *WQLQuery
}

func (i *IntersectOperation) String() string {
	return fmt.Sprintf("Intersect(%s)", i.Table)
}

func (i *IntersectOperation) operationNode() {}

// ExceptOperation EXCEPT操作
type ExceptOperation struct {
	Table *WQLQuery
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

// UnaryExpression 一元表达式：NOT(...)
type UnaryExpression struct {
	Operator string // "NOT"
	Operand  Expression
}

func (u *UnaryExpression) String() string {
	return fmt.Sprintf("%s(%s)", u.Operator, u.Operand)
}

func (u *UnaryExpression) expressionNode() {}

// InExpression IN 表达式：col IN (v1, v2, v3)
type InExpression struct {
	Column Expression
	Values []Expression
	Not    bool // true = NOT IN
}

func (e *InExpression) String() string {
	values := make([]string, len(e.Values))
	for i, v := range e.Values {
		values[i] = v.String()
	}
	if e.Not {
		return fmt.Sprintf("(%s NOT IN (%s))", e.Column, fmt.Sprintf("%v", values))
	}
	return fmt.Sprintf("(%s IN (%s))", e.Column, fmt.Sprintf("%v", values))
}

func (e *InExpression) expressionNode() {}

// LikeExpression LIKE 表达式：col LIKE "pattern"
type LikeExpression struct {
	Column   Expression
	Pattern  Expression
	Not      bool // true = NOT LIKE
	CaseSensitive bool
}

func (e *LikeExpression) String() string {
	if e.Not {
		return fmt.Sprintf("(%s NOT LIKE %s)", e.Column, e.Pattern)
	}
	return fmt.Sprintf("(%s LIKE %s)", e.Column, e.Pattern)
}

func (e *LikeExpression) expressionNode() {}

// BetweenExpression BETWEEN 表达式：col BETWEEN low AND high
type BetweenExpression struct {
	Column Expression
	Low    Expression
	High   Expression
	Not    bool
}

func (e *BetweenExpression) String() string {
	if e.Not {
		return fmt.Sprintf("(%s NOT BETWEEN %s AND %s)", e.Column, e.Low, e.High)
	}
	return fmt.Sprintf("(%s BETWEEN %s AND %s)", e.Column, e.Low, e.High)
}

func (e *BetweenExpression) expressionNode() {}

// IsNullExpression IS NULL / IS NOT NULL 表达式
type IsNullExpression struct {
	Column Expression
	Not    bool
}

func (e *IsNullExpression) String() string {
	if e.Not {
		return fmt.Sprintf("(%s IS NOT NULL)", e.Column)
	}
	return fmt.Sprintf("(%s IS NULL)", e.Column)
}

func (e *IsNullExpression) expressionNode() {}

// CoalesceExpression COALESCE(a, b, c, ...) — 第一个非 NULL 值
type CoalesceExpression struct {
	Args []Expression
}

func (e *CoalesceExpression) String() string {
	parts := make([]string, len(e.Args))
	for i, a := range e.Args {
		parts[i] = a.String()
	}
	return fmt.Sprintf("COALESCE(%s)", strings.Join(parts, ", "))
}

func (e *CoalesceExpression) expressionNode() {}

// NullIfExpression NULLIF(a, b) — a == b 时返回 NULL，否则返回 a
type NullIfExpression struct {
	A Expression
	B Expression
}

func (e *NullIfExpression) String() string {
	return fmt.Sprintf("NULLIF(%s, %s)", e.A, e.B)
}

func (e *NullIfExpression) expressionNode() {}

// CastExpression CAST(expr AS TYPE) — 类型转换
type CastExpression struct {
	Expr Expression
	Type string
}

func (e *CastExpression) String() string {
	return fmt.Sprintf("CAST(%s AS %s)", e.Expr, e.Type)
}

func (e *CastExpression) expressionNode() {}

// CaseWhenExpression CASE WHEN cond THEN val [WHEN ...] [ELSE val] END
type CaseWhenExpression struct {
	// Optional: simple CASE (CASE expr WHEN val THEN result)
	Input Expression
	// Searched CASE: list of WHEN conditions
	WhenClauses []CaseWhenClause
	ElseValue   Expression
}

type CaseWhenClause struct {
	Condition Expression
	Result    Expression
}

func (e *CaseWhenExpression) String() string {
	var sb strings.Builder
	sb.WriteString("CASE")
	if e.Input != nil {
		sb.WriteString(" ")
		sb.WriteString(e.Input.String())
	}
	for _, w := range e.WhenClauses {
		sb.WriteString(" WHEN ")
		sb.WriteString(w.Condition.String())
		sb.WriteString(" THEN ")
		sb.WriteString(w.Result.String())
	}
	if e.ElseValue != nil {
		sb.WriteString(" ELSE ")
		sb.WriteString(e.ElseValue.String())
	}
	sb.WriteString(" END")
	return sb.String()
}

func (e *CaseWhenExpression) expressionNode() {}

// DistinctOperation DISTINCT 去重操作
type DistinctOperation struct {
	Columns []Expression
}

func (d *DistinctOperation) String() string {
	cols := make([]string, len(d.Columns))
	for i, c := range d.Columns {
		cols[i] = c.String()
	}
	return fmt.Sprintf("Distinct(%s)", fmt.Sprintf("%v", cols))
}

func (d *DistinctOperation) operationNode() {}

// TransactionOperation 事务操作：BEGIN / COMMIT / ROLLBACK
type TransactionOperation struct {
	Action string // "BEGIN", "COMMIT", "ROLLBACK"
}

func (t *TransactionOperation) String() string {
	return t.Action
}

func (t *TransactionOperation) operationNode() {}
func (t *TransactionOperation) dmlNode()      {}
