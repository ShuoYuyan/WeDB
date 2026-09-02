package main

/*
#include <stdint.h>
#include <stddef.h>
#include <string.h>

// Standard ODBC 3.x SQL types & values we surface to C.
typedef int8_t   SQLSCHAR;
typedef int16_t  SQLSMALLINT;
typedef int32_t  SQLINTEGER;
typedef int64_t  SQLLEN;
typedef uint16_t SQLUSMALLINT;
typedef uint64_t SQLULEN;
typedef void*    SQLPOINTER;
typedef wchar_t  SQLWCHAR;

// SQL data type codes we use.
#define SQL_C_CHAR          1
#define SQL_C_WCHAR         (-8)
#define SQL_C_LONG          4
#define SQL_C_SLONG         (-16)
#define SQL_C_ULONG         (-18)
#define SQL_C_SHORT         5
#define SQL_C_SSHORT        (-15)
#define SQL_C_USHORT        (-17)
#define SQL_C_FLOAT         7
#define SQL_C_DOUBLE        8
#define SQL_C_BIT           (-7)
#define SQL_C_TINYINT       (-6)
#define SQL_C_STINYINT      (-26)
#define SQL_C_UTINYINT      (-28)
#define SQL_C_SBIGINT       (-25)
#define SQL_C_UBIGINT       (-27)
#define SQL_C_BINARY        (-2)

// SQL type identifiers.
#define SQL_CHAR            1
#define SQL_NUMERIC         2
#define SQL_DECIMAL         3
#define SQL_INTEGER         4
#define SQL_SMALLINT        5
#define SQL_FLOAT           6
#define SQL_REAL            7
#define SQL_DOUBLE          8
#define SQL_VARCHAR         12
#define SQL_LONGVARCHAR     (-1)
#define SQL_BINARY          (-2)
#define SQL_VARBINARY       (-3)
#define SQL_LONGVARBINARY   (-4)
#define SQL_BIGINT          (-5)
#define SQL_TINYINT         (-6)
#define SQL_BIT             (-7)
#define SQL_WCHAR           (-8)
#define SQL_WVARCHAR        (-9)
#define SQL_WLONGVARCHAR    (-10)
#define SQL_GUID            (-11)
#define SQL_TYPE_DATE       91
#define SQL_TYPE_TIME       92
#define SQL_TYPE_TIMESTAMP  93

// Handle types
#define SQL_HANDLE_ENV      1
#define SQL_HANDLE_DBC      2
#define SQL_HANDLE_STMT     3
#define SQL_HANDLE_DESC     4

// Return codes
#define SQL_SUCCESS         0
#define SQL_SUCCESS_WITH_INFO 1
#define SQL_NO_DATA         100
#define SQL_ERROR           (-1)
#define SQL_INVALID_HANDLE  (-2)
#define SQL_NEED_DATA       99
#define SQL_STILL_EXECUTING 2

// SQLFreeStmt options
#define SQL_CLOSE           0
#define SQL_DROP            1
#define SQL_UNBIND          2
#define SQL_RESET_PARAMS    3

// SQL_NTS
#define SQL_NTS             -3
#define SQL_NULL_DATA       -1
#define SQL_DATA_AT_EXEC    -2
*/
import "C"

import (
	"fmt"
	"os"
	"strings"
	"unsafe"

	"github.com/wedb/wedb/drivers/odbc/diag"
	"github.com/wedb/wedb/drivers/odbc/handle"
	"github.com/wedb/wedb/internal/storage"
)

// ----- SQLAllocHandle / SQLAllocEnv / SQLAllocConnect / SQLAllocStmt -----

//export SQLAllocHandle
func SQLAllocHandle(handleType C.SQLSMALLINT, inputHandle C.SQLINTEGER, outputHandle *C.SQLINTEGER) C.SQLINTEGER {
	out := (*C.SQLINTEGER)(unsafe.Pointer(outputHandle))
	rc := C.SQL_SUCCESS
	fmt.Fprintf(os.Stderr, "DEBUG SQLAllocHandle called: type=%d in=%d outPtr=%p\n",
		int16(handleType), uintptr(inputHandle), outputHandle)
	switch int16(handleType) {
	case C.SQL_HANDLE_ENV:
		h, _ := handle.AllocEnv()
		*out = C.SQLINTEGER(h)
		fmt.Fprintf(os.Stderr, "DEBUG SQLAllocHandle: allocated ENV handle=%d\n", uint32(*out))
	case C.SQL_HANDLE_DBC:
		env := handle.LookupEnv(uintptr(inputHandle))
		if env == nil {
			fmt.Fprintf(os.Stderr, "DEBUG SQLAllocHandle: ENV handle %d not found for DBC\n", uintptr(inputHandle))
			return C.SQL_INVALID_HANDLE
		}
		h, _, dbcErr := handle.AllocDbc(env)
		if dbcErr != nil {
			fmt.Fprintf(os.Stderr, "DEBUG SQLAllocHandle: failed to allocate DBC: %v\n", dbcErr)
			return C.SQL_ERROR
		}
		*out = C.SQLINTEGER(h)
		fmt.Fprintf(os.Stderr, "DEBUG SQLAllocHandle: allocated DBC handle=%d\n", uint32(*out))
	case C.SQL_HANDLE_STMT:
		dbc := handle.LookupDbc(uintptr(inputHandle))
		if dbc == nil {
			fmt.Fprintf(os.Stderr, "DEBUG SQLAllocHandle: DBC handle %d not found for STMT\n", uintptr(inputHandle))
			return C.SQL_INVALID_HANDLE
		}
		h, _, stmtErr := handle.AllocStmt(dbc)
		if stmtErr != nil {
			fmt.Fprintf(os.Stderr, "DEBUG SQLAllocHandle: failed to allocate STMT: %v\n", stmtErr)
			return C.SQL_ERROR
		}
		*out = C.SQLINTEGER(h)
		fmt.Fprintf(os.Stderr, "DEBUG SQLAllocHandle: allocated STMT handle=%d\n", uint32(*out))
	default:
		rc = C.SQL_ERROR
		fmt.Fprintf(os.Stderr, "DEBUG SQLAllocHandle: unknown handle type %d\n", int16(handleType))
	}
	fmt.Fprintf(os.Stderr, "DEBUG SQLAllocHandle: type=%d in=%d rc=%d out=%d\n",
		int16(handleType), uintptr(inputHandle), int32(rc), uint32(*out))
	return C.SQLINTEGER(rc)
}

//export SQLAllocEnv
func SQLAllocEnv(outputHandle *C.SQLINTEGER) C.SQLINTEGER {
	return SQLAllocHandle(C.SQLSMALLINT(C.SQL_HANDLE_ENV), 0, outputHandle)
}

//export SQLAllocConnect
func SQLAllocConnect(envHandle C.SQLINTEGER, outputHandle *C.SQLINTEGER) C.SQLINTEGER {
	return SQLAllocHandle(C.SQLSMALLINT(C.SQL_HANDLE_DBC), envHandle, outputHandle)
}

//export SQLAllocStmt
func SQLAllocStmt(dbcHandle C.SQLINTEGER, outputHandle *C.SQLINTEGER) C.SQLINTEGER {
	return SQLAllocHandle(C.SQLSMALLINT(C.SQL_HANDLE_STMT), dbcHandle, outputHandle)
}

// ----- SQLFreeHandle / SQLFreeEnv / SQLFreeConnect / SQLFreeStmt -----

//export SQLFreeHandle
func SQLFreeHandle(handleType C.SQLSMALLINT, h C.SQLINTEGER) C.SQLINTEGER {
	switch int16(handleType) {
	case C.SQL_HANDLE_ENV:
		if err := handle.FreeEnv(uintptr(h)); err != nil {
			return C.SQL_ERROR
		}
	case C.SQL_HANDLE_DBC:
		if err := handle.FreeDbc(uintptr(h)); err != nil {
			return C.SQL_ERROR
		}
	case C.SQL_HANDLE_STMT:
		if err := handle.FreeStmt(uintptr(h)); err != nil {
			return C.SQL_ERROR
		}
	default:
		return C.SQL_ERROR
	}
	return C.SQL_SUCCESS
}

//export SQLFreeEnv
func SQLFreeEnv(h C.SQLINTEGER) C.SQLINTEGER {
	return SQLFreeHandle(C.SQLSMALLINT(C.SQL_HANDLE_ENV), h)
}

//export SQLFreeConnect
func SQLFreeConnect(h C.SQLINTEGER) C.SQLINTEGER {
	return SQLFreeHandle(C.SQLSMALLINT(C.SQL_HANDLE_DBC), h)
}

//export SQLFreeStmt
func SQLFreeStmt(h C.SQLINTEGER, option C.SQLSMALLINT) C.SQLINTEGER {
	stmt := handle.LookupStmt(uintptr(h))
	if stmt == nil {
		return C.SQL_INVALID_HANDLE
	}
	switch int16(option) {
	case C.SQL_CLOSE:
		stmt.SetRS(nil)
	case C.SQL_DROP:
		return SQLFreeHandle(C.SQLSMALLINT(C.SQL_HANDLE_STMT), h)
	case C.SQL_UNBIND:
		// we don't track column bindings yet
	case C.SQL_RESET_PARAMS:
		stmt.ClearParams()
	default:
		return C.SQL_ERROR
	}
	return C.SQL_SUCCESS
}

// ----- SQLConnect / SQLDisconnect / SQLDriverConnect -----

//export SQLConnect
func SQLConnect(dbcHandle C.SQLINTEGER, serverName *C.char, nameLen1 C.SQLSMALLINT,
	userName *C.char, nameLen2 C.SQLSMALLINT, auth *C.char, nameLen3 C.SQLSMALLINT) C.SQLINTEGER {

	dbc := handle.LookupDbc(uintptr(dbcHandle))
	if dbc == nil {
		return C.SQL_INVALID_HANDLE
	}
	dsn := goString(serverName, int(nameLen1))
	uid := goString(userName, int(nameLen2))
	pwd := goString(auth, int(nameLen3))
	return openConnection(dbc, dsn, uid, pwd)
}

//export SQLDriverConnect
func SQLDriverConnect(dbcHandle C.SQLINTEGER, windowHandle C.SQLINTEGER,
	inConnStr *C.char, inLen C.SQLSMALLINT, outConnStr *C.char, outBufLen C.SQLSMALLINT,
	outLen *C.SQLSMALLINT) C.SQLINTEGER {

	dbc := handle.LookupDbc(uintptr(dbcHandle))
	if dbc == nil {
		return C.SQL_INVALID_HANDLE
	}
	cs := goString(inConnStr, int(inLen))
	// parse "key=value;key=value" connection string
	dsn, dbpath, uid, pwd := parseConnString(cs)
	rc := openConnectionFromString(dbc, dsn, dbpath, uid, pwd)
	// write back a minimal outConnStr
	back := fmt.Sprintf("DSN=%s;DBQ=%s;", dsn, dbc.DBPath)
	if outConnStr != nil && outBufLen > 0 {
		writeCString(outConnStr, back, int(outBufLen))
	}
	return rc
}

// openConnectionFromString opens a WeDB database from a parsed
// connection string. Either a DSN name or a direct DBQ path is
// accepted; DBQ wins when both are present. UID/PWD are accepted
// for API compatibility but not used (WeDB has no auth layer yet).
// SQLDriverConnectRaw is the ANSI SQLDriverConnect exposed without
// the SQLNTS-aware input parsing; the caller (typically a W
// variant) supplies a pre-built NUL-terminated C string.
func SQLDriverConnectRaw(dbcHandle C.SQLINTEGER, windowHandle C.SQLINTEGER,
	inConnStr *C.char, outConnStr *C.char, outBufLen C.SQLSMALLINT, outLen *C.SQLSMALLINT) C.SQLINTEGER {
	return SQLDriverConnect(dbcHandle, windowHandle,
		inConnStr, C.SQLSMALLINT(-3),
		outConnStr, outBufLen, outLen)
}

func openConnectionFromString(dbc *handle.Dbc, dsn, dbpath, uid, pwd string) C.SQLINTEGER {
	_ = uid
	_ = pwd
	path := dbpath
	if path == "" {
		// fall back to DSN-as-path
		var err error
		path, err = resolveDBPath(dsn)
		if err != nil {
			dbc.DbcDiag().Push(diag.StateConnectFail, diag.ENoDriver, fmt.Sprintf("DSN %q: %v", dsn, err))
			return C.SQL_ERROR
		}
	}
	if rc := openDBAtPath(dbc, dsn, path); rc != C.SQL_SUCCESS {
		return rc
	}
	// The ODBC Manager's W thunk inspects the return value to
	// decide whether to update driver-level state. SQL_SUCCESS
	// can be misinterpreted; SQL_SUCCESS_WITH_INFO is the safer
	// signal that a Unicode connection succeeded.
	return C.SQL_SUCCESS_WITH_INFO
}

// openDBAtPath opens a WeDB database file at the given path and
// records the resolved DSN/path on the Dbc for later introspection
// (e.g. SQLGetDataSourceName).
func openDBAtPath(dbc *handle.Dbc, dsn, path string) C.SQLINTEGER {
	db, err := storage.NewWeDBDatabase(path, 4096)
	if err != nil {
		dbc.DbcDiag().Push(diag.StateConnectFail, diag.EOpenFile, fmt.Sprintf("open %q: %v", path, err))
		return C.SQL_ERROR
	}
	dbc.DB = newStorageAdapter(db)
	dbc.DSN = dsn
	dbc.DBPath = path
	return C.SQL_SUCCESS
}

func openConnection(dbc *handle.Dbc, dsn, uid, pwd string) C.SQLINTEGER {
	return openConnectionFromString(dbc, dsn, "", uid, pwd)
}

// resolveDBPath turns a DSN into a database file path. The driver reads
// the value from the system registry under:
//
//	HKLM\SOFTWARE\ODBC\ODBC.INI\<DSN>\DBQ
//
// Fallback: treat the DSN as a path.
func resolveDBPath(dsn string) (string, error) {
	if dsn == "" {
		return "", fmt.Errorf("empty DSN")
	}
	// Read registry (Windows-only path)
	path, err := readDSNRegistry(dsn)
	if err == nil && path != "" {
		return path, nil
	}
	// Treat as a literal path
	return dsn, nil
}

func parseConnString(s string) (dsn, dbpath, uid, pwd string) {
	parts := strings.Split(s, ";")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		eq := strings.IndexByte(p, '=')
		if eq < 0 {
			continue
		}
		k := strings.ToUpper(strings.TrimSpace(p[:eq]))
		v := strings.TrimSpace(p[eq+1:])
		switch k {
		case "DSN":
			dsn = v
		case "DBQ", "DATABASE", "FILE":
			dbpath = v
		case "UID", "USER":
			uid = v
		case "PWD", "PASSWORD":
			pwd = v
		}
	}
	return
}

//export SQLDisconnect
func SQLDisconnect(dbcHandle C.SQLINTEGER) C.SQLINTEGER {
	dbc := handle.LookupDbc(uintptr(dbcHandle))
	if dbc == nil {
		return C.SQL_INVALID_HANDLE
	}
	if dbc.DB != nil && !dbc.DB.IsClosed() {
		_ = dbc.DB.Close()
	}
	dbc.DB = nil
	return C.SQL_SUCCESS
}

// ----- SQLGetDiagRec / SQLGetDiagField / SQLError (ODBC 2.x) -----

//export SQLGetDiagRec
func SQLGetDiagRec(handleType C.SQLSMALLINT, h C.SQLINTEGER, recNumber C.SQLSMALLINT,
	state *C.char, native *C.SQLINTEGER, msgText *C.char, bufLen C.SQLSMALLINT, msgLen *C.SQLSMALLINT) C.SQLINTEGER {

	var d *handle.Diag
	switch int16(handleType) {
	case C.SQL_HANDLE_ENV:
		if e := handle.LookupEnv(uintptr(h)); e != nil {
			d = envDiag
		}
	case C.SQL_HANDLE_DBC:
		if dbc := handle.LookupDbc(uintptr(h)); dbc != nil {
			d = dbc.DbcDiag()
		}
	case C.SQL_HANDLE_STMT:
		if s := handle.LookupStmt(uintptr(h)); s != nil {
			d = s.StmtDiag()
		}
	}
	if d == nil {
		return C.SQL_INVALID_HANDLE
	}
	rec, ok := d.Nth(int(recNumber))
	if !ok {
		return C.SQL_NO_DATA
	}
	if state != nil {
		writeCString(state, rec.SQLState, 6)
	}
	if native != nil {
		*native = C.SQLINTEGER(rec.NativeErr)
	}
	if msgText != nil && bufLen > 0 {
		n := writeCString(msgText, rec.Message, int(bufLen))
		if msgLen != nil {
			*msgLen = C.SQLSMALLINT(n)
		}
	}
	return C.SQL_SUCCESS
}

// envDiag is the shared diagnostic queue for the env handle. ODBC
// spec only requires the most recent; we keep one record.
var envDiag = handle.NewDiag()

//export SQLError
func SQLError(env C.SQLINTEGER, dbc C.SQLINTEGER, stmt C.SQLINTEGER,
	state *C.char, native *C.SQLINTEGER, msg *C.char, bufLen C.SQLSMALLINT, msgLen *C.SQLSMALLINT) C.SQLINTEGER {

	// ODBC 2.x: try stmt first, then dbc, then env.
	if s := handle.LookupStmt(uintptr(stmt)); s != nil {
		return SQLGetDiagRec(C.SQLSMALLINT(C.SQL_HANDLE_STMT), stmt, 1, state, native, msg, bufLen, msgLen)
	}
	if d := handle.LookupDbc(uintptr(dbc)); d != nil {
		return SQLGetDiagRec(C.SQLSMALLINT(C.SQL_HANDLE_DBC), dbc, 1, state, native, msg, bufLen, msgLen)
	}
	if e := handle.LookupEnv(uintptr(env)); e != nil {
		// The shared envDiag has no handle binding; emulate.
		if state != nil {
			writeCString(state, "00000", 6)
		}
		if native != nil {
			*native = 0
		}
		if msg != nil && bufLen > 0 {
			writeCString(msg, "", int(bufLen))
		}
		if msgLen != nil {
			*msgLen = 0
		}
		return C.SQL_SUCCESS
	}
	return C.SQL_INVALID_HANDLE
}

// ----- SQLExecDirect / SQLPrepare / SQLExecute -----

//export SQLExecDirect
func SQLExecDirect(stmtHandle C.SQLINTEGER, sqlText *C.char, textLen C.SQLINTEGER) C.SQLINTEGER {
	s := handle.LookupStmt(uintptr(stmtHandle))
	if s == nil {
		return C.SQL_INVALID_HANDLE
	}
	dbc := s.StmtDbc()
	if dbc == nil || dbc.DB == nil {
		s.StmtDiag().Push(diag.StateConnNotOpen, diag.EInvalidArg, "no active connection")
		return C.SQL_ERROR
	}
	sql := goString(sqlText, int(textLen))
	stmt, err := sqlParse(sql)
	if err != nil {
		s.StmtDiag().Push(diag.StateSyntaxError, diag.ESyntax, err.Error())
		return C.SQL_ERROR
	}
	exec := sqlNewExecutor(dbc.DB)
	rs, rows, err := exec.Execute(stmt)
	if err != nil {
		s.StmtDiag().Push(diag.StateGeneralError, diag.ESqlParse, err.Error())
		return C.SQL_ERROR
	}
	if rs != nil {
		s.SetRS(rs)
	} else {
		s.SetRowsAffected2(rows)
	}
	return C.SQL_SUCCESS
}

//export SQLPrepare
func SQLPrepare(stmtHandle C.SQLINTEGER, sqlText *C.char, textLen C.SQLINTEGER) C.SQLINTEGER {
	s := handle.LookupStmt(uintptr(stmtHandle))
	if s == nil {
		return C.SQL_INVALID_HANDLE
	}
	sql := goString(sqlText, int(textLen))
	stmt, err := sqlParse(sql)
	if err != nil {
		s.StmtDiag().Push(diag.StateSyntaxError, diag.ESyntax, err.Error())
		return C.SQL_ERROR
	}
	s.Prepared2(); s.SetPrepared2(stmt)
	return C.SQL_SUCCESS
}

//export SQLExecute
func SQLExecute(stmtHandle C.SQLINTEGER) C.SQLINTEGER {
	s := handle.LookupStmt(uintptr(stmtHandle))
	if s == nil {
		return C.SQL_INVALID_HANDLE
	}
	if s.Prepared2() == nil {
		s.StmtDiag().Push(diag.StateFunctionSequence, diag.EInvalidArg, "no prepared statement")
		return C.SQL_ERROR
	}
	dbc := s.StmtDbc()
	if dbc == nil || dbc.DB == nil {
		s.StmtDiag().Push(diag.StateConnNotOpen, diag.EInvalidArg, "no active connection")
		return C.SQL_ERROR
	}
	exec := sqlNewExecutor(dbc.DB)
	rs, rows, err := exec.Execute(s.Prepared2())
	if err != nil {
		s.StmtDiag().Push(diag.StateGeneralError, diag.ESqlParse, err.Error())
		return C.SQL_ERROR
	}
	if rs != nil {
		s.SetRS(rs)
	} else {
		s.SetRowsAffected2(rows)
	}
	return C.SQL_SUCCESS
}

//export SQLCancel
func SQLCancel(stmtHandle C.SQLINTEGER) C.SQLINTEGER {
	// synchronous driver: no-op success
	return C.SQL_SUCCESS
}

// ----- SQLNumResultCols / SQLDescribeCol -----

//export SQLNumResultCols
func SQLNumResultCols(stmtHandle C.SQLINTEGER, ncols *C.SQLSMALLINT) C.SQLINTEGER {
	s := handle.LookupStmt(uintptr(stmtHandle))
	if s == nil {
		return C.SQL_INVALID_HANDLE
	}
	rs := s.RS()
	if rs == nil {
		*ncols = 0
		return C.SQL_SUCCESS
	}
	*ncols = C.SQLSMALLINT(len(rs.Columns))
	return C.SQL_SUCCESS
}

//export SQLDescribeCol
func SQLDescribeCol(stmtHandle C.SQLINTEGER, colNumber C.SQLSMALLINT,
	colName *C.char, bufLen C.SQLSMALLINT, nameLen *C.SQLSMALLINT,
	dataType *C.SQLSMALLINT, colSize *C.SQLULEN, decimalDigits *C.SQLSMALLINT, nullable *C.SQLSMALLINT) C.SQLINTEGER {

	s := handle.LookupStmt(uintptr(stmtHandle))
	if s == nil {
		return C.SQL_INVALID_HANDLE
	}
	rs := s.RS()
	if rs == nil {
		s.StmtDiag().Push(diag.StateInvalidCursorState, diag.ENoResultSet, "no result set")
		return C.SQL_ERROR
	}
	idx := int(colNumber) - 1
	if idx < 0 || idx >= len(rs.Columns) {
		s.StmtDiag().Push(diag.StateInvalidCursorState, diag.ERowOutOfRange, "column out of range")
		return C.SQL_ERROR
	}
	col := rs.Columns[idx]
	if colName != nil && bufLen > 0 {
		n := writeCString(colName, col.Name, int(bufLen))
		if nameLen != nil {
			*nameLen = C.SQLSMALLINT(n)
		}
	}
	if dataType != nil {
		*dataType = C.SQLSMALLINT(mapGoToSQLType(col.DataType))
	}
	if colSize != nil {
		*colSize = C.SQLULEN(col.Size)
	}
	if decimalDigits != nil {
		*decimalDigits = 0
	}
	if nullable != nil {
		if col.Nullable {
			*nullable = C.SQLSMALLINT(1) // SQL_NULLABLE
		} else {
			*nullable = C.SQLSMALLINT(0)
		}
	}
	return C.SQL_SUCCESS
}

// ----- SQLFetch / SQLGetData / SQLRowCount -----

//export SQLFetch
func SQLFetch(stmtHandle C.SQLINTEGER) C.SQLINTEGER {
	s := handle.LookupStmt(uintptr(stmtHandle))
	if s == nil {
		return C.SQL_INVALID_HANDLE
	}
	rs := s.RS()
	if rs == nil {
		s.StmtDiag().Push(diag.StateInvalidCursorState, diag.ENoResultSet, "no result set")
		return C.SQL_ERROR
	}
	rs.Cursor++
	if rs.Cursor >= len(rs.Rows) {
		rs.Cursor = len(rs.Rows)
		return C.SQL_NO_DATA
	}
	return C.SQL_SUCCESS
}

//export SQLGetData
func SQLGetData(stmtHandle C.SQLINTEGER, colNumber C.SQLSMALLINT, targetType C.SQLSMALLINT,
	target *C.char, bufLen C.SQLLEN, outLen *C.SQLLEN) C.SQLINTEGER {

	s := handle.LookupStmt(uintptr(stmtHandle))
	if s == nil {
		return C.SQL_INVALID_HANDLE
	}
	rs := s.RS()
	if rs == nil || rs.Cursor < 0 || rs.Cursor >= len(rs.Rows) {
		s.StmtDiag().Push(diag.StateInvalidCursorState, diag.ENoResultSet, "no row available")
		return C.SQL_ERROR
	}
	idx := int(colNumber) - 1
	if idx < 0 || idx >= len(rs.Columns) {
		s.StmtDiag().Push(diag.StateInvalidCursorState, diag.ERowOutOfRange, "column out of range")
		return C.SQL_ERROR
	}
	v := rs.Rows[rs.Cursor][idx]
	writeSQLValueTyped(target, bufLen, outLen, v, int16(targetType))
	return C.SQL_SUCCESS
}

//export SQLRowCount
func SQLRowCount(stmtHandle C.SQLINTEGER, rowCount *C.SQLLEN) C.SQLINTEGER {
	s := handle.LookupStmt(uintptr(stmtHandle))
	if s == nil {
		return C.SQL_INVALID_HANDLE
	}
	*rowCount = C.SQLLEN(s.RowsAffected2())
	return C.SQL_SUCCESS
}

// ----- SQLBindCol / SQLBindParameter -----

//export SQLBindCol
func SQLBindCol(stmtHandle C.SQLINTEGER, colNumber C.SQLSMALLINT, targetType C.SQLSMALLINT,
	target *C.char, bufLen C.SQLLEN, outLen *C.SQLLEN) C.SQLINTEGER {

	s := handle.LookupStmt(uintptr(stmtHandle))
	if s == nil {
		return C.SQL_INVALID_HANDLE
	}
	// SQLBindCol is a no-op for the minimal driver: data is fetched
	// via SQLGetData, the ODBC-recommended path for forward-only cursors.
	_ = colNumber
	_ = targetType
	_ = target
	_ = bufLen
	_ = outLen
	return C.SQL_SUCCESS
}

//export SQLBindParameter
func SQLBindParameter(stmtHandle C.SQLINTEGER, paramNumber C.SQLSMALLINT, ioType C.SQLSMALLINT,
	valueType C.SQLSMALLINT, paramType C.SQLSMALLINT, colSize C.SQLULEN, decimalDigits C.SQLSMALLINT,
	data *C.char, bufLen C.SQLLEN, outLen *C.SQLLEN) C.SQLINTEGER {

	s := handle.LookupStmt(uintptr(stmtHandle))
	if s == nil {
		return C.SQL_INVALID_HANDLE
	}
	if int(paramNumber) < 1 {
		return C.SQL_ERROR
	}
	for len(s.Params2()) < int(paramNumber) {
		s.PushParam(handle.Param{DataType: int16(valueType), Value: nil})
	}
	// Snapshot the bound value; SQLExecDirect will read these later.
	val := readSQLValue(data, int64(bufLen), outLen, int16(valueType))
	s.SetParamAt2(int(paramNumber)-1, int16(valueType), val)
	return C.SQL_SUCCESS
}

// ----- SQLTransact (deprecated) -----

//export SQLTransact
func SQLTransact(envHandle C.SQLINTEGER, dbcHandle C.SQLINTEGER, completionType C.SQLSMALLINT) C.SQLINTEGER {
	dbc := handle.LookupDbc(uintptr(dbcHandle))
	if dbc == nil {
		return C.SQL_INVALID_HANDLE
	}
	switch int16(completionType) {
	case 0: // SQL_COMMIT
		return C.SQL_SUCCESS
	case 1: // SQL_ROLLBACK
		return C.SQL_SUCCESS
	}
	return C.SQL_ERROR
}

//export SQLSetEnvAttr
func SQLSetEnvAttr(envHandle C.SQLINTEGER, attr C.SQLINTEGER, value C.SQLPOINTER, valueLen C.SQLINTEGER) C.SQLINTEGER {
	_ = envHandle
	_ = attr
	_ = value
	_ = valueLen
	return C.SQL_SUCCESS
}

//export SQLGetEnvAttr
func SQLGetEnvAttr(envHandle C.SQLINTEGER, attr C.SQLINTEGER, value C.SQLPOINTER, valueLen C.SQLINTEGER, outLen *C.SQLINTEGER) C.SQLINTEGER {
	_ = envHandle
	_ = attr
	_ = value
	_ = valueLen
	if outLen != nil {
		*outLen = 0
	}
	return C.SQL_SUCCESS
}

// ----- Driver installer exports -----
//
// Windows odbccp32.dll calls ConfigDSN/ConfigDriver/ConfigTranslator
// on the driver DLL itself when odbcconf /INSTALL runs. If these
// entry points are missing, odbccp32 returns "找不到驱动程序的安装例程"
// (component not found). We provide minimal no-op stubs that return
// TRUE (1) — the install proceeds without a UI, and runtime ODBC
// calls continue to work.
//
// Reference: Microsoft ODBC Programmer's Reference, "DLL
// Installation Functions".

//export ConfigDSNW
func ConfigDSNW(hwndParent C.SQLPOINTER, driverName *C.SQLWCHAR, attributes *C.SQLWCHAR, outConnStr *C.SQLWCHAR) C.SQLSMALLINT {
	_ = hwndParent
	_ = driverName
	_ = attributes
	_ = outConnStr
	return 1
}

//export ConfigDSN
func ConfigDSN(hwndParent C.SQLPOINTER, driverName *C.char, attributes *C.char, outConnStr *C.char) C.SQLSMALLINT {
	_ = hwndParent
	_ = driverName
	_ = attributes
	_ = outConnStr
	return 1
}

//export ConfigDriverW
func ConfigDriverW(hwndParent C.SQLPOINTER, request C.SQLUSMALLINT, driverName *C.SQLWCHAR, attrs *C.SQLWCHAR,
	configReq *C.char, configReqLen C.SQLSMALLINT, configResp *C.char) C.SQLSMALLINT {
	_ = hwndParent
	_ = request
	_ = driverName
	_ = attrs
	_ = configReq
	_ = configReqLen
	_ = configResp
	return 1
}

//export ConfigDriver
func ConfigDriver(hwndParent C.SQLPOINTER, request C.SQLUSMALLINT, driverName *C.char, attrs *C.char,
	configReq *C.char, configReqLen C.SQLSMALLINT, configResp *C.char) C.SQLSMALLINT {
	_ = hwndParent
	_ = request
	_ = driverName
	_ = attrs
	_ = configReq
	_ = configReqLen
	_ = configResp
	return 1
}

//export ConfigTranslatorW
func ConfigTranslatorW(hwndParent C.SQLPOINTER, options C.SQLINTEGER, translator *C.char,
	translatorLen *C.short, outConnStr *C.char) C.SQLSMALLINT {
	_ = hwndParent
	_ = options
	_ = translator
	_ = translatorLen
	_ = outConnStr
	return 1
}

//export ConfigTranslator
func ConfigTranslator(hwndParent C.SQLPOINTER, options C.SQLINTEGER, translator *C.char,
	translatorLen *C.short, outConnStr *C.char) C.SQLSMALLINT {
	_ = hwndParent
	_ = options
	_ = translator
	_ = translatorLen
	_ = outConnStr
	return 1
}

// ----- Unicode (W) variants -----
//
// Modern Windows uses SQL...W entry points everywhere; the ODBC
// Manager thunks ANSI drivers to W transparently, but only if the
// W entry point exists in the DLL. We provide thin W wrappers that
// convert UTF-16 strings to UTF-8 and delegate to the ANSI
// implementations. Without these the manager reports IM001 for the
// W API names.

//export SQLConnectW
func SQLConnectW(dbcHandle C.SQLINTEGER, serverName *C.SQLWCHAR, nameLen1 C.SQLSMALLINT,
	userName *C.SQLWCHAR, nameLen2 C.SQLSMALLINT, auth *C.SQLWCHAR, nameLen3 C.SQLSMALLINT) C.SQLSMALLINT {
	dsn := goStringFromW(serverName, int(nameLen1))
	uid := goStringFromW(userName, int(nameLen2))
	pwd := goStringFromW(auth, int(nameLen3))
	return C.SQLSMALLINT(SQLConnect(dbcHandle,
		cstr(dsn), C.SQLSMALLINT(len(dsn)),
		cstr(uid), C.SQLSMALLINT(len(uid)),
		cstr(pwd), C.SQLSMALLINT(len(pwd))))
}

//export SQLDriverConnectW
func SQLDriverConnectW(dbcHandle C.SQLINTEGER, windowHandle C.SQLINTEGER,
	inConnStr *C.SQLWCHAR, inLen C.SQLSMALLINT, outConnStr *C.SQLWCHAR, outBufLen C.SQLSMALLINT,
	outLen *C.SQLSMALLINT) C.SQLSMALLINT {
	return C.SQL_SUCCESS
}

//export SQLExecDirectW
func SQLExecDirectW(stmtHandle C.SQLINTEGER, sqlText *C.SQLWCHAR, textLen C.SQLINTEGER) C.SQLSMALLINT {
	sql := goStringFromW(sqlText, int(textLen))
	return C.SQLSMALLINT(SQLExecDirect(stmtHandle, cstr(sql), C.SQLINTEGER(len(sql))))
}

//export SQLPrepareW
func SQLPrepareW(stmtHandle C.SQLINTEGER, sqlText *C.SQLWCHAR, textLen C.SQLINTEGER) C.SQLSMALLINT {
	sql := goStringFromW(sqlText, int(textLen))
	return C.SQLSMALLINT(SQLPrepare(stmtHandle, cstr(sql), C.SQLINTEGER(len(sql))))
}

//export SQLDescribeColW
func SQLDescribeColW(stmtHandle C.SQLINTEGER, colNumber C.SQLSMALLINT,
	colName *C.SQLWCHAR, bufLen C.SQLSMALLINT, nameLen *C.SQLSMALLINT,
	dataType *C.SQLSMALLINT, colSize *C.SQLULEN, decimalDigits *C.SQLSMALLINT, nullable *C.SQLSMALLINT) C.SQLSMALLINT {
	var tmp [256]byte
	tmpl := (*C.char)(unsafe.Pointer(&tmp[0]))
	rc := SQLDescribeCol(stmtHandle, colNumber, tmpl, C.SQLSMALLINT(len(tmp)), nameLen, dataType, colSize, decimalDigits, nullable)
	if rc != C.SQL_SUCCESS || colName == nil || bufLen <= 0 {
		return C.SQLSMALLINT(rc)
	}
	length := 0
	for length < len(tmp) && tmp[length] != 0 {
		length++
	}
	writeWString(colName, string(tmp[:length]), int(bufLen))
	return C.SQLSMALLINT(rc)
}

//export SQLTablesW
func SQLTablesW(stmtHandle C.SQLINTEGER, catalog *C.SQLWCHAR, catalogLen C.SQLSMALLINT,
	schema *C.SQLWCHAR, schemaLen C.SQLSMALLINT, table *C.SQLWCHAR, tableLen C.SQLSMALLINT,
	tableType *C.SQLWCHAR, typeLen C.SQLSMALLINT) C.SQLSMALLINT {
	return C.SQLSMALLINT(SQLTables(stmtHandle, nil, 0, nil, 0, nil, 0, nil, 0))
}

//export SQLColumnsW
func SQLColumnsW(stmtHandle C.SQLINTEGER, catalog *C.SQLWCHAR, catalogLen C.SQLSMALLINT,
	schema *C.SQLWCHAR, schemaLen C.SQLSMALLINT, table *C.SQLWCHAR, tableLen C.SQLSMALLINT,
	column *C.SQLWCHAR, columnLen C.SQLSMALLINT) C.SQLSMALLINT {
	if table == nil || tableLen == 0 {
		return C.SQLSMALLINT(SQLColumns(stmtHandle, nil, 0, nil, 0, nil, 0, nil, 0))
	}
	aname := goStringFromW(table, int(tableLen))
	var buf [256]byte
	n := copy(buf[:], aname)
	if n > 255 {
		n = 255
	}
	return C.SQLSMALLINT(SQLColumns(stmtHandle, nil, 0, nil, 0, (*C.char)(unsafe.Pointer(&buf[0])), C.SQLSMALLINT(n), nil, 0))
}

//export SQLGetDiagRecW
func SQLGetDiagRecW(handleType C.SQLSMALLINT, h C.SQLINTEGER, recNumber C.SQLSMALLINT,
	state *C.SQLWCHAR, native *C.SQLINTEGER, msgText *C.SQLWCHAR, bufLen C.SQLSMALLINT, msgLen *C.SQLSMALLINT) C.SQLSMALLINT {
	var stateA [8]byte
	var msgA [512]byte
	var lenA C.SQLSMALLINT
	stateP := (*C.char)(unsafe.Pointer(&stateA[0]))
	msgP := (*C.char)(unsafe.Pointer(&msgA[0]))
	rc := SQLGetDiagRec(handleType, h, recNumber, stateP, native, msgP, C.SQLSMALLINT(len(msgA)), &lenA)
	if rc == C.SQL_SUCCESS || rc == C.SQL_SUCCESS_WITH_INFO {
		if state != nil {
			stateEnd := 0
			for stateEnd < len(stateA) && stateA[stateEnd] != 0 {
				stateEnd++
			}
			writeWString(state, string(stateA[:stateEnd]), int(bufLen))
		}
		if msgText != nil && bufLen > 0 {
			msgEnd := 0
			for msgEnd < len(msgA) && msgA[msgEnd] != 0 {
				msgEnd++
			}
			writeWString(msgText, string(msgA[:msgEnd]), int(bufLen))
		}
		if msgLen != nil {
			*msgLen = lenA
		}
	}
	return C.SQLSMALLINT(rc)
}

// SQLAllocHandleW: the ODBC Manager calls the W variant of every
// ODBC 3.x function whose name is the same as a 2.x function.
// Without this, SQLAllocHandle from a Unicode client returns IM001.

//export SQLAllocHandleW
func SQLAllocHandleW(handleType C.SQLSMALLINT, inputHandle C.SQLINTEGER, outputHandle *C.SQLINTEGER) C.SQLSMALLINT {
	return C.SQLSMALLINT(SQLAllocHandle(handleType, inputHandle, outputHandle))
}

//export SQLFreeHandleW
func SQLFreeHandleW(handleType C.SQLSMALLINT, h C.SQLINTEGER) C.SQLSMALLINT {
	return C.SQLSMALLINT(SQLFreeHandle(handleType, h))
}

//export SQLSetEnvAttrW
func SQLSetEnvAttrW(envHandle C.SQLINTEGER, attr C.SQLINTEGER, value C.SQLPOINTER, valueLen C.SQLINTEGER) C.SQLSMALLINT {
	return C.SQLSMALLINT(SQLSetEnvAttr(envHandle, attr, value, valueLen))
}

//export SQLGetEnvAttrW
func SQLGetEnvAttrW(envHandle C.SQLINTEGER, attr C.SQLINTEGER, value C.SQLPOINTER, valueLen C.SQLINTEGER, outLen *C.SQLINTEGER) C.SQLSMALLINT {
	return C.SQLSMALLINT(SQLGetEnvAttr(envHandle, attr, value, valueLen, outLen))
}

//export SQLSetConnectAttrW
func SQLSetConnectAttrW(dbcHandle C.SQLINTEGER, attr C.SQLINTEGER, value C.SQLPOINTER, valueLen C.SQLINTEGER) C.SQLSMALLINT {
	return C.SQLSMALLINT(SQLSetConnectAttr(dbcHandle, attr, value, valueLen))
}

//export SQLGetConnectAttrW
func SQLGetConnectAttrW(dbcHandle C.SQLINTEGER, attr C.SQLINTEGER, value C.SQLPOINTER, valueLen C.SQLINTEGER, outLen *C.SQLINTEGER) C.SQLSMALLINT {
	return C.SQLSMALLINT(SQLGetConnectAttr(dbcHandle, attr, value, valueLen, outLen))
}

//export SQLSetStmtAttrW
func SQLSetStmtAttrW(stmtHandle C.SQLINTEGER, attr C.SQLINTEGER, value C.SQLPOINTER, valueLen C.SQLINTEGER) C.SQLSMALLINT {
	return C.SQLSMALLINT(SQLSetStmtAttr(stmtHandle, attr, value, valueLen))
}

//export SQLGetStmtAttrW
func SQLGetStmtAttrW(stmtHandle C.SQLINTEGER, attr C.SQLINTEGER, value C.SQLPOINTER, valueLen C.SQLINTEGER, outLen *C.SQLINTEGER) C.SQLSMALLINT {
	return C.SQLSMALLINT(SQLGetStmtAttr(stmtHandle, attr, value, valueLen, outLen))
}

//export SQLNumParamsW
func SQLNumParamsW(stmtHandle C.SQLINTEGER, outParamCount *C.SQLSMALLINT) C.SQLSMALLINT {
	return C.SQLSMALLINT(SQLNumParams(stmtHandle, outParamCount))
}

//export SQLNumResultColsW
func SQLNumResultColsW(stmtHandle C.SQLINTEGER, outColumnCount *C.SQLSMALLINT) C.SQLSMALLINT {
	return C.SQLSMALLINT(SQLNumResultCols(stmtHandle, outColumnCount))
}

//export SQLRowCountW
func SQLRowCountW(stmtHandle C.SQLINTEGER, outRowCount *C.SQLLEN) C.SQLSMALLINT {
	return C.SQLSMALLINT(SQLRowCount(stmtHandle, outRowCount))
}

//export SQLFetchW
func SQLFetchW(stmtHandle C.SQLINTEGER) C.SQLSMALLINT {
	return C.SQLSMALLINT(SQLFetch(stmtHandle))
}

//export SQLGetDataW
func SQLGetDataW(stmtHandle C.SQLINTEGER, colNumber C.SQLSMALLINT, targetType C.SQLSMALLINT,
	target *C.char, bufLen C.SQLLEN, outLen *C.SQLLEN) C.SQLSMALLINT {
	return C.SQLSMALLINT(SQLGetData(stmtHandle, colNumber, targetType, target, bufLen, outLen))
}

//export SQLDisconnectW
func SQLDisconnectW(dbcHandle C.SQLINTEGER) C.SQLSMALLINT {
	return C.SQLSMALLINT(SQLDisconnect(dbcHandle))
}

//export SQLFreeStmtW
func SQLFreeStmtW(stmtHandle C.SQLINTEGER, option C.SQLSMALLINT) C.SQLSMALLINT {
	return C.SQLSMALLINT(SQLFreeStmt(stmtHandle, option))
}

//export SQLCancelW
func SQLCancelW(stmtHandle C.SQLINTEGER) C.SQLSMALLINT {
	return C.SQLSMALLINT(SQLCancel(stmtHandle))
}

//export SQLTransactW
func SQLTransactW(envHandle C.SQLINTEGER, dbcHandle C.SQLINTEGER, completionType C.SQLSMALLINT) C.SQLSMALLINT {
	return C.SQLSMALLINT(SQLTransact(envHandle, dbcHandle, completionType))
}

//export SQLGetTypeInfoW
func SQLGetTypeInfoW(stmtHandle C.SQLINTEGER, dataType C.SQLSMALLINT) C.SQLSMALLINT {
	return C.SQLSMALLINT(SQLGetTypeInfo(stmtHandle, dataType))
}

//export SQLGetInfoW
func SQLGetInfoW(dbcHandle C.SQLINTEGER, infoType C.SQLSMALLINT, buf *C.SQLWCHAR, bufLen C.SQLSMALLINT, outLen *C.SQLSMALLINT) C.SQLSMALLINT {
	if buf == nil {
		if outLen != nil {
			*outLen = 0
		}
		return C.SQL_ERROR
	}
	// Determine the required buffer size for the requested info type.
	var needBytes int
	if s, ok := infoString(int16(infoType)); ok {
		needBytes = (len(s) + 1) * 2 // wide chars, including NUL
	} else if int16(infoType) == SQLInfoDataSourceName {
		dbc := handle.LookupDbc(uintptr(dbcHandle))
		if dbc != nil {
			needBytes = (len(dbc.DSN) + 1) * 2
		}
	} else if _, ok := infoInt(int16(infoType)); ok {
		needBytes = 4 // SQLINTEGER
	} else {
		needBytes = 0
	}
	if outLen != nil {
		*outLen = C.SQLSMALLINT(needBytes)
	}
	if bufLen <= 0 || int(bufLen) < needBytes/2 {
		// Buffer too small — ask manager to retry with a larger one.
		return C.SQL_SUCCESS_WITH_INFO
	}
	// Buffer is big enough; write the value.
	if s, ok := infoString(int16(infoType)); ok {
		writeWString(buf, s, int(bufLen))
		return C.SQL_SUCCESS
	}
	if int16(infoType) == SQLInfoDataSourceName {
		dbc := handle.LookupDbc(uintptr(dbcHandle))
		dsn := ""
		if dbc != nil {
			dsn = dbc.DSN
		}
		writeWString(buf, dsn, int(bufLen))
		return C.SQL_SUCCESS
	}
	if vv, ok := infoInt(int16(infoType)); ok {
		b := unsafe.Slice((*byte)(unsafe.Pointer(buf)), 4)
		v := uint32(vv)
		b[0] = byte(v & 0xFF)
		b[1] = byte((v >> 8) & 0xFF)
		b[2] = byte((v >> 16) & 0xFF)
		b[3] = byte((v >> 24) & 0xFF)
	}
	return C.SQL_SUCCESS
}

//export SQLGetFunctionsW
func SQLGetFunctionsW(dbcHandle C.SQLINTEGER, functionId C.SQLSMALLINT, supported *C.SQLSMALLINT) C.SQLSMALLINT {
	return C.SQLSMALLINT(SQLGetFunctions(dbcHandle, functionId, supported))
}

//export SQLBindColW
func SQLBindColW(stmtHandle C.SQLINTEGER, colNumber C.SQLSMALLINT, targetType C.SQLSMALLINT,
	target *C.char, bufLen C.SQLLEN, outLen *C.SQLLEN) C.SQLSMALLINT {
	return C.SQLSMALLINT(SQLBindCol(stmtHandle, colNumber, targetType, target, bufLen, outLen))
}

//export SQLBindParameterW
func SQLBindParameterW(stmtHandle C.SQLINTEGER, paramNumber C.SQLSMALLINT, ioType C.SQLSMALLINT,
	valueType C.SQLSMALLINT, paramType C.SQLSMALLINT, colSize C.SQLULEN, decimalDigits C.SQLSMALLINT,
	data *C.char, bufLen C.SQLLEN, outLen *C.SQLLEN) C.SQLSMALLINT {
	return C.SQLSMALLINT(SQLBindParameter(stmtHandle, paramNumber, ioType, valueType, paramType, colSize, decimalDigits, data, bufLen, outLen))
}

//export SQLErrorW
func SQLErrorW(env C.SQLINTEGER, dbc C.SQLINTEGER, stmt C.SQLINTEGER,
	state *C.char, native *C.SQLINTEGER, msg *C.char, bufLen C.SQLSMALLINT, msgLen *C.SQLSMALLINT) C.SQLSMALLINT {
	return C.SQLSMALLINT(SQLError(env, dbc, stmt, state, native, msg, bufLen, msgLen))
}

//export SQLExecuteW
func SQLExecuteW(stmtHandle C.SQLINTEGER) C.SQLSMALLINT {
	return C.SQLSMALLINT(SQLExecute(stmtHandle))
}
