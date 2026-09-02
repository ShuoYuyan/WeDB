// Package wql is a stub replacement module for github.com/wedb/wedb/WQL.
//
// The full WQL parser and interpreter is not part of this build; this
// stub provides just enough surface so the cmd/wql_* example programs
// compile. Programs that depend on the stub APIs will panic at runtime.
package wql

import "errors"

// Database matches the historical WQL contract.
type Database interface {
	Close() error
}

// Expression is a parsed WQL expression.
type Expression interface {
	Evaluate(ctx *ExecutionContext) (interface{}, error)
}

// ExecutionContext carries per-row evaluation state.
type ExecutionContext struct {
	Row map[string]interface{}
}

// NewExecutionContext creates a new empty ExecutionContext.
func NewExecutionContext() *ExecutionContext {
	return &ExecutionContext{Row: map[string]interface{}{}}
}

// ParseExpressionString parses a WQL expression string.
func ParseExpressionString(s string) (Expression, error) {
	return nil, errors.New("wql stub: ParseExpressionString not implemented")
}

// aggregateFunc is a tiny helper for the stub aggregate funcs.
type aggregateFunc struct{ name string }

func (a *aggregateFunc) name_() string { return a.name }

// SumFunction implements the SUM() aggregate.
type SumFunction struct{ aggregateFunc }

// AvgFunction implements the AVG() aggregate.
type AvgFunction struct{ aggregateFunc }

// MinFunction implements the MIN() aggregate.
type MinFunction struct{ aggregateFunc }

// MaxFunction implements the MAX() aggregate.
type MaxFunction struct{ aggregateFunc }

// Execute runs the aggregate over the given values. The stub returns
// an error so the example programs fail loudly.
func (f *SumFunction) Execute(values ...interface{}) (interface{}, error) {
	return nil, errors.New("wql stub: SumFunction not implemented")
}
func (f *AvgFunction) Execute(values ...interface{}) (interface{}, error) {
	return nil, errors.New("wql stub: AvgFunction not implemented")
}
func (f *MinFunction) Execute(values ...interface{}) (interface{}, error) {
	return nil, errors.New("wql stub: MinFunction not implemented")
}
func (f *MaxFunction) Execute(values ...interface{}) (interface{}, error) {
	return nil, errors.New("wql stub: MaxFunction not implemented")
}
