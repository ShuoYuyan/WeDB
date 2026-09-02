// Package handle implements the ODBC handle hierarchy (env / dbc / stmt).
//
// WeDB does not need separate env/dbc/stmt classes internally, but ODBC
// mandates the structure. Each handle is a uintptr published to C land;
// the Go side keeps a map from handle to *Env / *Dbc / *Stmt.
//
// The C side is expected to never fabricate handles 鈥?it only passes back
// values that this package handed out. We still validate magic numbers
// defensively.
package handle

import (
	"fmt"
	"sync"

	"github.com/wedb/wedb/drivers/odbc/sql"
)

// Magic bytes that prefix every handle. They let us detect misuse and
// reject garbage pointers from C callers (e.g. SQLFreeHandle on a value
// never allocated by SQLAllocHandle).
const (
	MagicEnv  uint32 = 0x5745_4442 // 'WEDB' env
	MagicDbc  uint32 = 0x5744_4243 // 'WDBC' dbc
	MagicStmt uint32 = 0x5753_544D // 'WSTM' stmt
)

// invalid handle value returned to C on error.
const invalidHandle uintptr = 0

// Env is the ODBC environment handle root.
type Env struct {
	Magic      uint32
	dbcList    *Dbc // head of singly-linked list
	odbcVer    int32
	cp         uint16 // SQLGetInfo SQL_ODBC_API_CONFORMANCE bucketing
}

// Dbc is an ODBC connection handle.
type Dbc struct {
	Magic      uint32
	Env        *Env
	Next       *Dbc
	DB         Database
	DSN        string
	DBPath     string
	ReadOnly   bool
	AutoCommit bool
	Diag       *Diag
}

// Stmt is an ODBC statement handle.
type Stmt struct {
	Magic       uint32
	dbc         *Dbc
	rs          *ResultSet
	params      []Param
	diag        *Diag
	inExec      bool
	prepared    interface{} // *sql.Statement (kept as interface to avoid import cycle)
	rowsAffected int64
}

// ResultSet is a re-export of sql.ResultSet via interface{}. We keep
// the type opaque in the handle package to avoid a circular import;
// accessors in drivers/odbc/accessors.go cast it back.
type ResultSet = sql.ResultSet

// Param holds bound parameter state for a prepared statement.
type Param struct {
	Value    interface{}
	DataType int16 // SQL type
	Nullable bool
}

// Database is the minimal interface a Dbc must hold. It is satisfied by
// *storage.WeDBDatabase, but we keep the dependency in the parent package
// to avoid a hard import.
type Database interface {
	Ping() error
	Close() error
	IsClosed() bool

	ListTables() []string
	TableExists(name string) bool
	GetTableSchema(name string) (interface{}, error)
	CreateTable(schema interface{}) error
	DropTable(name string) error

	ScanTable(name string) ([]map[string]interface{}, error)
	ScanTableWithColumns(name string, cols []string) ([]map[string]interface{}, error)
	InsertRow(name string, row map[string]interface{}) error
	UpdateRow(name string, row map[string]interface{}, where string) error
	DeleteRow(name string, where string) error

	Count(name, where string) (int64, error)
	Min(name, col, where string) (interface{}, error)
	Max(name, col, where string) (interface{}, error)
	Sum(name, col, where string) (float64, error)
	Avg(name, col, where string) (float64, error)
}

// Diag is a per-handle diagnostic record queue. The first record is the
// most recent error; older records follow. We expose up to 8 records per
// handle as ODBC requires only a small ring.
type Diag struct {
	mu      sync.Mutex
	records []DiagRecord
}

// DiagRecord is one SQLSTATE + native error + message triple.
type DiagRecord struct {
	SQLState  string
	NativeErr int32
	Message   string
}

// NewDiag allocates a fresh diagnostic queue.
func NewDiag() *Diag { return &Diag{} }

// Push adds a record. The first record is the most recent.
func (d *Diag) Push(state string, native int32, msg string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(d.records) >= 16 {
		// keep at most 16 records; drop the oldest
		d.records = append(d.records[:15], DiagRecord{state, native, msg})
		return
	}
	d.records = append(d.records, DiagRecord{state, native, msg})
}

// First returns the most recent record.
func (d *Diag) First() (DiagRecord, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(d.records) == 0 {
		return DiagRecord{}, false
	}
	return d.records[len(d.records)-1], true
}

// Nth returns the (1-indexed) record requested by SQLGetDiagRec.
func (d *Diag) Nth(n int) (DiagRecord, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if n < 1 || n > len(d.records) {
		return DiagRecord{}, false
	}
	// SQLGetDiagRec numbers from most recent (1) backward.
	idx := len(d.records) - n
	return d.records[idx], true
}

// Count returns the number of records currently queued.
func (d *Diag) Count() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return len(d.records)
}

// Clear empties the queue.
func (d *Diag) Clear() {
	d.mu.Lock()
	d.records = d.records[:0]
	d.mu.Unlock()
}

// Pool is the global handle allocator. It is shared across env/dbc/stmt
// spaces; the magic prefix prevents cross-type confusion.
type Pool struct {
	mu   sync.Mutex
	envs map[uintptr]*Env
	dbcs map[uintptr]*Dbc
	stmts map[uintptr]*Stmt
	next  uintptr
}

var globalPool = &Pool{
	envs:  map[uintptr]*Env{},
	dbcs:  map[uintptr]*Dbc{},
	stmts: map[uintptr]*Stmt{},
	next:  0x10000, // skip the low page so 0 stays a sentinel
}

// AllocEnv creates a new Env and returns its handle.
func AllocEnv() (uintptr, *Env) {
	globalPool.mu.Lock()
	defer globalPool.mu.Unlock()
	h := globalPool.next
	globalPool.next++
	e := &Env{Magic: MagicEnv, odbcVer: 0x0300, cp: 1}
	globalPool.envs[h] = e
	return h, e
}

// AllocDbc creates a new Dbc under the given env.
func AllocDbc(env *Env) (uintptr, *Dbc, error) {
	if env == nil || env.Magic != MagicEnv {
		return invalidHandle, nil, fmt.Errorf("invalid env handle")
	}
	globalPool.mu.Lock()
	defer globalPool.mu.Unlock()
	h := globalPool.next
	globalPool.next++
	d := &Dbc{Magic: MagicDbc, Env: env, AutoCommit: true, Diag: NewDiag()}
	d.Next = env.dbcList
	env.dbcList = d
	globalPool.dbcs[h] = d
	return h, d, nil
}

// AllocStmt creates a new Stmt under the given dbc.
func AllocStmt(dbc *Dbc) (uintptr, *Stmt, error) {
	if dbc == nil || dbc.Magic != MagicDbc {
		return invalidHandle, nil, fmt.Errorf("invalid dbc handle")
	}
	globalPool.mu.Lock()
	defer globalPool.mu.Unlock()
	h := globalPool.next
	globalPool.next++
	s := &Stmt{Magic: MagicStmt, dbc: dbc, diag: NewDiag()}
	globalPool.stmts[h] = s
	return h, s, nil
}

// LookupEnv / LookupDbc / LookupStmt retrieve a handle by value, checking
// the magic prefix. They return nil if the value was never allocated by
// this driver.
func LookupEnv(h uintptr) *Env {
	if h == invalidHandle {
		return nil
	}
	globalPool.mu.Lock()
	defer globalPool.mu.Unlock()
	e := globalPool.envs[h]
	if e == nil || e.Magic != MagicEnv {
		return nil
	}
	return e
}
func LookupDbc(h uintptr) *Dbc {
	if h == invalidHandle {
		return nil
	}
	globalPool.mu.Lock()
	defer globalPool.mu.Unlock()
	d := globalPool.dbcs[h]
	if d == nil || d.Magic != MagicDbc {
		return nil
	}
	return d
}
func LookupStmt(h uintptr) *Stmt {
	if h == invalidHandle {
		return nil
	}
	globalPool.mu.Lock()
	defer globalPool.mu.Unlock()
	s := globalPool.stmts[h]
	if s == nil || s.Magic != MagicStmt {
		return nil
	}
	return s
}

// FreeEnv removes an env and all its dbcs/stmts.
func FreeEnv(h uintptr) error {
	e := LookupEnv(h)
	if e == nil {
		return fmt.Errorf("invalid env handle")
	}
	globalPool.mu.Lock()
	defer globalPool.mu.Unlock()
	// free all child dbcs
	for d := e.dbcList; d != nil; {
		dn := d.Next
		for sh, s := range globalPool.stmts {
			if s.dbc == d {
				delete(globalPool.stmts, sh)
			}
		}
		for dh, dd := range globalPool.dbcs {
			if dd == d {
				delete(globalPool.dbcs, dh)
			}
		}
		d = dn
	}
	delete(globalPool.envs, h)
	return nil
}

// FreeDbc removes a dbc and all its stmts.
func FreeDbc(h uintptr) error {
	d := LookupDbc(h)
	if d == nil {
		return fmt.Errorf("invalid dbc handle")
	}
	if d.DB != nil && !d.DB.IsClosed() {
		_ = d.DB.Close()
	}
	globalPool.mu.Lock()
	defer globalPool.mu.Unlock()
	for sh, s := range globalPool.stmts {
		if s.dbc == d {
			delete(globalPool.stmts, sh)
		}
	}
	// unlink from env
	prev := d.Env.dbcList
	if prev == d {
		d.Env.dbcList = d.Next
	} else {
		for prev != nil && prev.Next != d {
			prev = prev.Next
		}
		if prev != nil {
			prev.Next = d.Next
		}
	}
	delete(globalPool.dbcs, h)
	return nil
}

// FreeStmt removes a stmt.
func FreeStmt(h uintptr) error {
	s := LookupStmt(h)
	if s == nil {
		return fmt.Errorf("invalid stmt handle")
	}
	globalPool.mu.Lock()
	defer globalPool.mu.Unlock()
	delete(globalPool.stmts, h)
	return nil
}
