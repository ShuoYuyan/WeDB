package handle

import "github.com/wedb/wedb/drivers/odbc/sql"

// --- Dbc accessors ---

// DbcDiag returns the per-connection diagnostic queue.
func (d *Dbc) DbcDiag() *Diag { return d.Diag }

// --- Stmt accessors (field-backed to keep this file minimal). ---

// StmtDbc returns the parent connection.
func (s *Stmt) StmtDbc() *Dbc { return s.dbc }

// StmtDiag returns the per-statement diagnostic queue.
func (s *Stmt) StmtDiag() *Diag { return s.diag }

// RS returns the current result set, or nil.
func (s *Stmt) RS() *sql.ResultSet { return s.rs }

// SetRS replaces the current result set.
func (s *Stmt) SetRS(rs *sql.ResultSet) { s.rs = rs }

// RowsAffected2 returns the last DML rowcount.
func (s *Stmt) RowsAffected2() int64 { return s.rowsAffected }

// SetRowsAffected2 sets the rowcount for non-SELECT statements.
func (s *Stmt) SetRowsAffected2(n int64) { s.rowsAffected = n }

// Prepared2 returns the prepared statement, or nil.
func (s *Stmt) Prepared2() *sql.Statement {
	if p, ok := s.prepared.(*sql.Statement); ok {
		return p
	}
	return nil
}

// SetPrepared2 stores a parsed statement for later SQLExecute.
func (s *Stmt) SetPrepared2(p *sql.Statement) { s.prepared = p }

// Params2 returns the bound parameter list.
func (s *Stmt) Params2() []Param { return s.params }

// SetParam2 appends a parameter.
func (s *Stmt) SetParam2(t int16, v interface{}) {
	s.params = append(s.params, Param{DataType: t, Value: v})
}

// SetParamAt2 overwrites a parameter at idx.
func (s *Stmt) SetParamAt2(idx int, t int16, v interface{}) {
	if idx < 0 || idx >= len(s.params) {
		return
	}
	s.params[idx] = Param{DataType: t, Value: v}
}

// ClearParams empties the bound parameter list.
func (s *Stmt) ClearParams() { s.params = nil }

// PushParam is a convenience for callers that want to append a Param struct directly.
func (s *Stmt) PushParam(p Param) { s.params = append(s.params, p) }
