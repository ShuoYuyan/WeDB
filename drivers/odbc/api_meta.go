package main

/*
#include <stdint.h>
#include <stddef.h>
#include <string.h>

typedef int16_t  SQLSMALLINT;
typedef int32_t  SQLINTEGER;
typedef int64_t  SQLLEN;
typedef uint16_t SQLUSMALLINT;
typedef uint64_t SQLULEN;
typedef void*    SQLPOINTER;

#define SQL_HANDLE_ENV 1
#define SQL_HANDLE_DBC 2
#define SQL_HANDLE_STMT 3
#define SQL_HANDLE_DESC 4

#define SQL_SUCCESS 0
#define SQL_SUCCESS_WITH_INFO 1
#define SQL_NO_DATA 100
#define SQL_ERROR (-1)
#define SQL_INVALID_HANDLE (-2)
#define SQL_NEED_DATA 99
#define SQL_STILL_EXECUTING 2

// SQL types for catalog funcs (no overlap with info-type constants).
#define SQL_CHAR 1
#define SQL_VARCHAR 12
#define SQL_LONGVARCHAR (-1)
#define SQL_DECIMAL 3
#define SQL_NUMERIC 2
#define SQL_INTEGER 4
#define SQL_SMALLINT 5
#define SQL_FLOAT 6
#define SQL_REAL 7
#define SQL_DOUBLE 8
#define SQL_DATE 9
#define SQL_TIME 10
#define SQL_TIMESTAMP 11
#define SQL_BIT (-7)
#define SQL_TINYINT (-6)
#define SQL_BIGINT (-5)
#define SQL_BINARY (-2)
#define SQL_VARBINARY (-3)
#define SQL_LONGVARBINARY (-4)
#define SQL_TYPE_DATE 91
#define SQL_TYPE_TIME 92
#define SQL_TYPE_TIMESTAMP 93

// Transaction isolation bit values.
#define SQL_TXN_READ_UNCOMMITTED 1
#define SQL_TXN_READ_COMMITTED 2
#define SQL_TXN_REPEATABLE_READ 4
#define SQL_TXN_SERIALIZABLE 8

// NULL collation.
#define SQL_NC_HIGH 4
#define SQL_NULLABLE 1
#define SQL_NO_NULLS 0

// File usage.
#define SQL_FILE_CATALOG 1

// API function codes for SQLGetFunctions.
#define SQL_API_SQLALLOCCONNECT 1
#define SQL_API_SQLALLOCENV 2
#define SQL_API_SQLALLOCSTMT 3
#define SQL_API_SQLBINDCOL 4
#define SQL_API_SQLCANCEL 5
#define SQL_API_SQLCOLUMNS 40
#define SQL_API_SQLCONNECT 7
#define SQL_API_SQLDATASOURCES 57
#define SQL_API_SQLDESCRIBECOL 8
#define SQL_API_SQLDISCONNECT 9
#define SQL_API_SQLERROR 10
#define SQL_API_SQLEXECDIRECT 11
#define SQL_API_SQLEXECUTE 12
#define SQL_API_SQLFETCH 13
#define SQL_API_SQLFREECONNECT 14
#define SQL_API_SQLFREEENV 15
#define SQL_API_SQLFREESTMT 16
#define SQL_API_SQLGETCONNECTOPTION 62
#define SQL_API_SQLGETDATA 43
#define SQL_API_SQLGETFUNCTIONS 103
#define SQL_API_SQLGETINFO 104
#define SQL_API_SQLGETSTMTOPTION 67
#define SQL_API_SQLGETTYPEINFO 44
#define SQL_API_SQLNUMPARAMS 70
#define SQL_API_SQLNUMRESULTCOLS 18
#define SQL_API_SQLPARAMDATA 48
#define SQL_API_SQLPREPARE 19
#define SQL_API_SQLPUTDATA 53
#define SQL_API_SQLROWCOUNT 20
#define SQL_API_SQLSETCONNECTOPTION 72
#define SQL_API_SQLSETSTMTOPTION 73
#define SQL_API_SQLSPECIALCOLUMNS 56
#define SQL_API_SQLSTATISTICS 53
#define SQL_API_SQLTABLES 54
#define SQL_API_SQLTRANSACT 23
#define SQL_API_SQLBINDPARAMETER 72

// SQLGetFunctions special codes.
#define SQL_API_ALL_FUNCTIONS 0
#define SQL_API_ODBC3_ALL_FUNCTIONS 999
*/
import "C"

import (
	"unsafe"

	"github.com/wedb/wedb/drivers/odbc/handle"
	"github.com/wedb/wedb/drivers/odbc/sql"
)

// ----- SQLTables -----

//export SQLTables
func SQLTables(stmtHandle C.SQLINTEGER, catalog *C.char, catalogLen C.SQLSMALLINT,
	schema *C.char, schemaLen C.SQLSMALLINT, table *C.char, tableLen C.SQLSMALLINT,
	tableType *C.char, typeLen C.SQLSMALLINT) C.SQLINTEGER {

	_ = catalog
	_ = catalogLen
	_ = schema
	_ = schemaLen
	_ = tableType
	_ = typeLen

	s := handle.LookupStmt(uintptr(stmtHandle))
	if s == nil {
		return C.SQL_INVALID_HANDLE
	}
	dbc := s.StmtDbc()
	if dbc == nil || dbc.DB == nil {
		return C.SQL_ERROR
	}
	tableName := goString(table, int(tableLen))
	tables := dbc.DB.ListTables()
	rows := make([][]interface{}, 0, len(tables))
	for _, t := range tables {
		if tableName != "" && tableName != "%" && t != tableName {
			continue
		}
		// ODBC SQLTables column order: TABLE_CAT, TABLE_SCHEM,
		// TABLE_NAME, TABLE_TYPE, REMARKS. WeDB has no catalogs or
		// schemas, so we leave those as empty strings.
		rows = append(rows, []interface{}{"", "", t, "TABLE", ""})
	}
	cols := []sql.ColMeta{
		{Name: "TABLE_CAT", DataType: C.SQL_VARCHAR},
		{Name: "TABLE_SCHEM", DataType: C.SQL_VARCHAR},
		{Name: "TABLE_NAME", DataType: C.SQL_VARCHAR},
		{Name: "TABLE_TYPE", DataType: C.SQL_VARCHAR},
		{Name: "REMARKS", DataType: C.SQL_VARCHAR},
	}
	s.SetRS(&sql.ResultSet{Columns: cols, Rows: rows, Cursor: -1})
	return C.SQL_SUCCESS
}

// ----- SQLColumns -----

//export SQLColumns
func SQLColumns(stmtHandle C.SQLINTEGER, catalog *C.char, catalogLen C.SQLSMALLINT,
	schema *C.char, schemaLen C.SQLSMALLINT, table *C.char, tableLen C.SQLSMALLINT,
	column *C.char, columnLen C.SQLSMALLINT) C.SQLINTEGER {

	_ = catalog
	_ = catalogLen
	_ = schema
	_ = schemaLen
	_ = column
	_ = columnLen

	s := handle.LookupStmt(uintptr(stmtHandle))
	if s == nil {
		return C.SQL_INVALID_HANDLE
	}
	dbc := s.StmtDbc()
	if dbc == nil || dbc.DB == nil {
		return C.SQL_ERROR
	}
	tableName := goString(table, int(tableLen))
	all := [][]interface{}{}
	if tableName == "" || tableName == "%" {
		for _, tn := range dbc.DB.ListTables() {
			all = append(all, columnRowsFor(dbc.DB, tn)...)
		}
	} else {
		all = append(all, columnRowsFor(dbc.DB, tableName)...)
	}
	s.SetRS(&sql.ResultSet{Columns: columnMeta(), Rows: all, Cursor: -1})
	return C.SQL_SUCCESS
}

func columnMeta() []sql.ColMeta {
	return []sql.ColMeta{
		{Name: "TABLE_CAT", DataType: C.SQL_VARCHAR},
		{Name: "TABLE_SCHEM", DataType: C.SQL_VARCHAR},
		{Name: "TABLE_NAME", DataType: C.SQL_VARCHAR},
		{Name: "COLUMN_NAME", DataType: C.SQL_VARCHAR},
		{Name: "DATA_TYPE", DataType: C.SQL_INTEGER},
		{Name: "TYPE_NAME", DataType: C.SQL_VARCHAR},
		{Name: "COLUMN_SIZE", DataType: C.SQL_INTEGER},
		{Name: "BUFFER_LENGTH", DataType: C.SQL_INTEGER},
		{Name: "DECIMAL_DIGITS", DataType: C.SQL_SMALLINT},
		{Name: "NUM_PREC_RADIX", DataType: C.SQL_SMALLINT},
		{Name: "NULLABLE", DataType: C.SQL_SMALLINT},
		{Name: "REMARKS", DataType: C.SQL_VARCHAR},
		{Name: "COLUMN_DEF", DataType: C.SQL_VARCHAR},
		{Name: "SQL_DATA_TYPE", DataType: C.SQL_SMALLINT},
		{Name: "SQL_DATETIME_SUB", DataType: C.SQL_SMALLINT},
		{Name: "CHAR_OCTET_LENGTH", DataType: C.SQL_INTEGER},
		{Name: "ORDINAL_POSITION", DataType: C.SQL_INTEGER},
		{Name: "IS_NULLABLE", DataType: C.SQL_VARCHAR},
	}
}

func columnRowsFor(db handle.Database, table string) [][]interface{} {
	schema, err := db.GetTableSchema(table)
	if err != nil || schema == nil {
		return nil
	}
	out := [][]interface{}{}
	rows, ok := extractColumns(schema)
	if !ok {
		return nil
	}
	pos := 1
	for _, col := range rows {
		out = append(out, []interface{}{
			nil,                       // TABLE_CAT
			nil,                       // TABLE_SCHEM
			table,                     // TABLE_NAME
			col.Name,                  // COLUMN_NAME
			col.Type,                  // DATA_TYPE
			typeName(col.Type),        // TYPE_NAME
			col.Size,                  // COLUMN_SIZE
			col.Size,                  // BUFFER_LENGTH
			0,                         // DECIMAL_DIGITS
			10,                        // NUM_PREC_RADIX
			nullableInt(col.Nullable), // NULLABLE
			nil,                       // REMARKS
			nil,                       // COLUMN_DEF
			col.Type,                  // SQL_DATA_TYPE
			0,                         // SQL_DATETIME_SUB
			col.Size,                  // CHAR_OCTET_LENGTH
			pos,                       // ORDINAL_POSITION
			nullableStr(col.Nullable), // IS_NULLABLE
		})
		pos++
	}
	return out
}

// ----- SQLGetTypeInfo -----

//export SQLGetTypeInfo
func SQLGetTypeInfo(stmtHandle C.SQLINTEGER, dataType C.SQLSMALLINT) C.SQLINTEGER {
	s := handle.LookupStmt(uintptr(stmtHandle))
	if s == nil {
		return C.SQL_INVALID_HANDLE
	}
	cols := []sql.ColMeta{
		{Name: "TYPE_NAME", DataType: C.SQL_VARCHAR},
		{Name: "DATA_TYPE", DataType: C.SQL_SMALLINT},
		{Name: "COLUMN_SIZE", DataType: C.SQL_INTEGER},
		{Name: "LITERAL_PREFIX", DataType: C.SQL_VARCHAR},
		{Name: "LITERAL_SUFFIX", DataType: C.SQL_VARCHAR},
		{Name: "CREATE_PARAMS", DataType: C.SQL_VARCHAR},
		{Name: "NULLABLE", DataType: C.SQL_SMALLINT},
		{Name: "CASE_SENSITIVE", DataType: C.SQL_SMALLINT},
		{Name: "SEARCHABLE", DataType: C.SQL_SMALLINT},
		{Name: "UNSIGNED_ATTRIBUTE", DataType: C.SQL_SMALLINT},
		{Name: "FIXED_PREC_SCALE", DataType: C.SQL_SMALLINT},
		{Name: "AUTO_UNIQUE_VALUE", DataType: C.SQL_SMALLINT},
		{Name: "LOCAL_TYPE_NAME", DataType: C.SQL_VARCHAR},
		{Name: "MINIMUM_SCALE", DataType: C.SQL_SMALLINT},
		{Name: "MAXIMUM_SCALE", DataType: C.SQL_SMALLINT},
		{Name: "SQL_DATA_TYPE", DataType: C.SQL_SMALLINT},
		{Name: "SQL_DATETIME_SUB", DataType: C.SQL_SMALLINT},
		{Name: "NUM_PREC_RADIX", DataType: C.SQL_INTEGER},
		{Name: "INTERVAL_PRECISION", DataType: C.SQL_SMALLINT},
	}
	// Each row follows the ODBC spec column order for SQLGetTypeInfo:
	// TYPE_NAME, DATA_TYPE, COLUMN_SIZE, LITERAL_PREFIX, LITERAL_SUFFIX,
	// CREATE_PARAMS, NULLABLE, CASE_SENSITIVE, SEARCHABLE,
	// UNSIGNED_ATTRIBUTE, FIXED_PREC_SCALE, AUTO_UNIQUE_VALUE,
	// LOCAL_TYPE_NAME, MINIMUM_SCALE, MAXIMUM_SCALE, SQL_DATA_TYPE,
	// SQL_DATETIME_SUB, NUM_PREC_RADIX, INTERVAL_PRECISION.
	typeInfos := [][]interface{}{
		{"CHAR", C.SQL_CHAR, 255, "'", "'", nil, C.SQL_NULLABLE, C.SQLSMALLINT(1), C.SQLSMALLINT(3), C.SQLSMALLINT(0), C.SQLSMALLINT(0), nil, "CHAR", nil, nil, C.SQL_CHAR, nil, 0, nil},
		{"VARCHAR", C.SQL_VARCHAR, 65535, "'", "'", "length", C.SQL_NULLABLE, C.SQLSMALLINT(1), C.SQLSMALLINT(3), C.SQLSMALLINT(0), C.SQLSMALLINT(0), nil, "VARCHAR", nil, nil, C.SQL_VARCHAR, nil, 0, nil},
		{"TEXT", C.SQL_LONGVARCHAR, 1073741823, "'", "'", nil, C.SQL_NULLABLE, C.SQLSMALLINT(1), C.SQLSMALLINT(3), C.SQLSMALLINT(0), C.SQLSMALLINT(0), nil, "TEXT", nil, nil, C.SQL_LONGVARCHAR, nil, 0, nil},
		{"INTEGER", C.SQL_INTEGER, 19, nil, nil, nil, C.SQL_NULLABLE, C.SQLSMALLINT(0), C.SQLSMALLINT(2), C.SQLSMALLINT(0), C.SQLSMALLINT(0), nil, "INTEGER", 0, 0, C.SQL_INTEGER, nil, 10, nil},
		{"BIGINT", C.SQL_BIGINT, 19, nil, nil, nil, C.SQL_NULLABLE, C.SQLSMALLINT(0), C.SQLSMALLINT(2), C.SQLSMALLINT(0), C.SQLSMALLINT(0), nil, "BIGINT", 0, 0, C.SQL_BIGINT, nil, 10, nil},
		{"SMALLINT", C.SQL_SMALLINT, 5, nil, nil, nil, C.SQL_NULLABLE, C.SQLSMALLINT(0), C.SQLSMALLINT(2), C.SQLSMALLINT(0), C.SQLSMALLINT(0), nil, "SMALLINT", 0, 0, C.SQL_SMALLINT, nil, 10, nil},
		{"TINYINT", C.SQL_TINYINT, 3, nil, nil, nil, C.SQL_NULLABLE, C.SQLSMALLINT(0), C.SQLSMALLINT(2), C.SQLSMALLINT(0), C.SQLSMALLINT(0), nil, "TINYINT", 0, 0, C.SQL_TINYINT, nil, 10, nil},
		{"BIT", C.SQL_BIT, 1, nil, nil, nil, C.SQL_NULLABLE, C.SQLSMALLINT(0), C.SQLSMALLINT(2), C.SQLSMALLINT(0), C.SQLSMALLINT(0), nil, "BIT", nil, nil, C.SQL_BIT, nil, 0, nil},
		{"REAL", C.SQL_REAL, 7, nil, nil, nil, C.SQL_NULLABLE, C.SQLSMALLINT(0), C.SQLSMALLINT(2), C.SQLSMALLINT(0), C.SQLSMALLINT(0), nil, "REAL", 0, 0, C.SQL_REAL, nil, 10, nil},
		{"FLOAT", C.SQL_FLOAT, 15, nil, nil, nil, C.SQL_NULLABLE, C.SQLSMALLINT(0), C.SQLSMALLINT(2), C.SQLSMALLINT(0), C.SQLSMALLINT(0), nil, "FLOAT", 0, 0, C.SQL_FLOAT, nil, 10, nil},
		{"DOUBLE", C.SQL_DOUBLE, 15, nil, nil, nil, C.SQL_NULLABLE, C.SQLSMALLINT(0), C.SQLSMALLINT(2), C.SQLSMALLINT(0), C.SQLSMALLINT(0), nil, "DOUBLE", 0, 0, C.SQL_DOUBLE, nil, 10, nil},
		{"NUMERIC", C.SQL_NUMERIC, 38, nil, nil, "precision,scale", C.SQL_NULLABLE, C.SQLSMALLINT(0), C.SQLSMALLINT(2), C.SQLSMALLINT(0), C.SQLSMALLINT(0), nil, "NUMERIC", 0, 38, C.SQL_NUMERIC, nil, 10, nil},
		{"DECIMAL", C.SQL_DECIMAL, 38, nil, nil, "precision,scale", C.SQL_NULLABLE, C.SQLSMALLINT(0), C.SQLSMALLINT(2), C.SQLSMALLINT(0), C.SQLSMALLINT(0), nil, "DECIMAL", 0, 38, C.SQL_DECIMAL, nil, 10, nil},
		{"DATE", C.SQL_DATE, 10, "'", "'", nil, C.SQL_NULLABLE, C.SQLSMALLINT(0), C.SQLSMALLINT(2), C.SQLSMALLINT(0), C.SQLSMALLINT(0), nil, "DATE", nil, nil, C.SQL_DATE, nil, 0, nil},
		{"TIME", C.SQL_TIME, 8, "'", "'", nil, C.SQL_NULLABLE, C.SQLSMALLINT(0), C.SQLSMALLINT(2), C.SQLSMALLINT(0), C.SQLSMALLINT(0), nil, "TIME", nil, nil, C.SQL_TIME, nil, 0, nil},
		{"TIMESTAMP", C.SQL_TIMESTAMP, 26, "'", "'", nil, C.SQL_NULLABLE, C.SQLSMALLINT(0), C.SQLSMALLINT(2), C.SQLSMALLINT(0), C.SQLSMALLINT(0), nil, "TIMESTAMP", nil, nil, C.SQL_TIMESTAMP, nil, 0, nil},
		{"BINARY", C.SQL_BINARY, 2550, "0x", nil, nil, C.SQL_NULLABLE, C.SQLSMALLINT(0), C.SQLSMALLINT(2), C.SQLSMALLINT(0), C.SQLSMALLINT(0), nil, "BINARY", nil, nil, C.SQL_BINARY, nil, 0, nil},
		{"VARBINARY", C.SQL_VARBINARY, 65535, "0x", nil, "length", C.SQL_NULLABLE, C.SQLSMALLINT(0), C.SQLSMALLINT(2), C.SQLSMALLINT(0), C.SQLSMALLINT(0), nil, "VARBINARY", nil, nil, C.SQL_VARBINARY, nil, 0, nil},
		{"BLOB", C.SQL_LONGVARBINARY, 1073741823, "0x", nil, nil, C.SQL_NULLABLE, C.SQLSMALLINT(0), C.SQLSMALLINT(2), C.SQLSMALLINT(0), C.SQLSMALLINT(0), nil, "BLOB", nil, nil, C.SQL_LONGVARBINARY, nil, 0, nil},
	}
	_ = dataType // we report all types; filtering can be added later
	s.SetRS(&sql.ResultSet{Columns: cols, Rows: typeInfos, Cursor: -1})
	return C.SQL_SUCCESS
}

// ----- SQLGetInfo -----

//export SQLGetInfo
func SQLGetInfo(dbcHandle C.SQLINTEGER, infoType C.SQLSMALLINT,
	buf *C.char, bufLen C.SQLSMALLINT, outLen *C.SQLSMALLINT) C.SQLINTEGER {
	if handle.LookupDbc(uintptr(dbcHandle)) == nil {
		return C.SQL_INVALID_HANDLE
	}
	if buf == nil || bufLen <= 0 {
		if outLen != nil {
			*outLen = 0
		}
		return C.SQL_ERROR
	}
	// 32-bit integer fast path
	if bufLen >= 4 {
		if s, ok := infoString(int16(infoType)); ok {
			if outLen != nil {
				*outLen = C.SQLSMALLINT(len(s))
			}
			writeCString(buf, s, int(bufLen))
			return C.SQL_SUCCESS
		}
		if v, ok := infoInt(int16(infoType)); ok {
			*(*C.SQLINTEGER)(unsafe.Pointer(buf)) = C.SQLINTEGER(v)
			if outLen != nil {
				*outLen = 4
			}
			return C.SQL_SUCCESS
		}
		// Special handling: SQL_DATA_SOURCE_NAME reports the actual
		// DSN string. infoString can't do this because it needs the
		// handle, not just an info type.
		if int16(infoType) == SQLInfoDataSourceName {
			dbc := handle.LookupDbc(uintptr(dbcHandle))
			dsn := ""
			if dbc != nil {
				dsn = dbc.DSN
			}
			if outLen != nil {
				*outLen = C.SQLSMALLINT(len(dsn))
			}
			writeCString(buf, dsn, int(bufLen))
			return C.SQL_SUCCESS
		}
		// Default: zero
		*(*C.SQLINTEGER)(unsafe.Pointer(buf)) = 0
		if outLen != nil {
			*outLen = 4
		}
		return C.SQL_SUCCESS
	}
	// buf too small: report required length and return SQL_SUCCESS_WITH_INFO
	if outLen != nil {
		*outLen = 0
	}
	return C.SQL_SUCCESS_WITH_INFO
}

// ----- SQLGetFunctions -----

//export SQLGetFunctions
func SQLGetFunctions(dbcHandle C.SQLINTEGER, functionId C.SQLSMALLINT, supported *C.SQLSMALLINT) C.SQLINTEGER {
	if handle.LookupDbc(uintptr(dbcHandle)) == nil {
		return C.SQL_INVALID_HANDLE
	}
	// Declare every ODBC 3.x function we have an export for. The
	// manager uses this set to gate dispatch to the driver; if a
	// function code is missing, the manager returns IM001 without
	// ever calling us. Setting the bit to 1 tells the manager "yes
	// I support this function, dispatch to me". Functions that the
	// manager actually calls are implemented in api_core.go and
	// api_meta.go; the others return SQL_ERROR (not IM001) when
	// called, which the client can handle.
	supportedSet := map[int16]bool{
		// SQL 2.x API codes
		C.SQL_API_SQLALLOCCONNECT: true, C.SQL_API_SQLALLOCENV: true,
		C.SQL_API_SQLALLOCSTMT: true, C.SQL_API_SQLBINDCOL: true,
		C.SQL_API_SQLCANCEL: true, C.SQL_API_SQLCOLUMNS: true,
		C.SQL_API_SQLCONNECT: true, C.SQL_API_SQLDATASOURCES: true,
		C.SQL_API_SQLDESCRIBECOL: true, C.SQL_API_SQLDISCONNECT: true,
		C.SQL_API_SQLERROR: true, C.SQL_API_SQLEXECDIRECT: true,
		C.SQL_API_SQLEXECUTE: true, C.SQL_API_SQLFETCH: true,
		C.SQL_API_SQLFREECONNECT: true, C.SQL_API_SQLFREEENV: true,
		C.SQL_API_SQLFREESTMT: true, C.SQL_API_SQLGETCONNECTOPTION: false,
		C.SQL_API_SQLGETDATA: true, C.SQL_API_SQLGETFUNCTIONS: true,
		C.SQL_API_SQLGETINFO: true, C.SQL_API_SQLGETSTMTOPTION: false,
		C.SQL_API_SQLGETTYPEINFO: true, C.SQL_API_SQLNUMPARAMS: true,
		C.SQL_API_SQLNUMRESULTCOLS: true, C.SQL_API_SQLPARAMDATA: false,
		C.SQL_API_SQLPREPARE: true, C.SQL_API_SQLROWCOUNT: true,
		C.SQL_API_SQLSETCONNECTOPTION: false, C.SQL_API_SQLSETSTMTOPTION: false,
		C.SQL_API_SQLSPECIALCOLUMNS: false, C.SQL_API_SQLSTATISTICS: false,
		C.SQL_API_SQLTABLES: true, C.SQL_API_SQLTRANSACT: true,
		// SQL 3.x API codes (40..100) — only the ones that don't
		// collide with the legacy codes above.
		41 /* SQLDRIVERS */: true,
		// Modern codes 100..150
		100 /* SQLALLOCHANDLE */: true,
		101 /* SQLBINDPARAMETER */: true,
		102 /* SQLCLOSECURSOR */: true,
		// 103 = SQLGETFUNCTIONS and 104 = SQLGETINFO share codes
		// with the 2.x SQL_API_SQLGETFUNCTIONS / SQL_API_SQLGETINFO
		// constants above; they're already true.
		105 /* SQLNATIVESQL */: true,
		106 /* SQLSETDESCFIELD */: false,
		107 /* SQLSETENVATTR */: true,
		109 /* SQLGETSTMTATTR */: true,
		110 /* SQLGETCONNECTATTR */: true,
		114 /* SQLGETENVATTR */: true,
		115 /* SQLSETSTMTATTR */: true,
		116 /* SQLGETDIAGFIELD */: true,
		117 /* SQLGETDIAGREC */: true,
		118 /* SQLFETCHSCROLL */: true,
		119 /* SQLGETTYPEINFOEXTENDED */: true,
		120 /* SQLBULKOPERATIONS */: false,
		// 72 = SQL_API_SQLSETCONNECTOPTION. We don't implement the
		// 2.x SETCONNECTOPTION (use SetConnectAttr instead), but
		// the ODBC manager also maps code 72 to SQL_API_SQLBINDPARAMETER
		// in 3.x mode, and we do support that. Report true.
	}
	if int16(functionId) == C.SQL_API_ALL_FUNCTIONS {
		// SQL 2.x: 100-element array.
		arr := unsafe.Slice((*C.SQLUSMALLINT)(unsafe.Pointer(supported)), 100)
		for i := int16(0); i < 100; i++ {
			if supportedSet[i] {
				arr[i] = 1
			} else {
				arr[i] = 0
			}
		}
		return C.SQL_SUCCESS
	}
	if int16(functionId) == C.SQL_API_ODBC3_ALL_FUNCTIONS {
		// ODBC 3.x: 400-element array.
		arr := unsafe.Slice((*C.SQLUSMALLINT)(unsafe.Pointer(supported)), 400)
		for i := int16(0); i < 400; i++ {
			if supportedSet[i] {
				arr[i] = 1
			} else {
				arr[i] = 0
			}
		}
		return C.SQL_SUCCESS
	}
	*supported = 0
	if v, ok := supportedSet[int16(functionId)]; ok && v {
		*supported = 1
	}
	return C.SQL_SUCCESS
}

// ----- SQLDataSources / SQLDrivers -----

//export SQLDataSources
func SQLDataSources(envHandle C.SQLINTEGER, direction C.SQLSMALLINT,
	dsn *C.char, dsnLen C.SQLSMALLINT, dsnOutLen *C.SQLSMALLINT,
	desc *C.char, descLen C.SQLSMALLINT, descOutLen *C.SQLSMALLINT) C.SQLINTEGER {
	_ = envHandle
	_ = direction
	if dsn != nil && dsnLen > 0 {
		writeCString(dsn, "", int(dsnLen))
	}
	if dsnOutLen != nil {
		*dsnOutLen = 0
	}
	if desc != nil && descLen > 0 {
		writeCString(desc, "", int(descLen))
	}
	if descOutLen != nil {
		*descOutLen = 0
	}
	return C.SQL_NO_DATA
}

//export SQLDrivers
func SQLDrivers(envHandle C.SQLINTEGER, direction C.SQLSMALLINT,
	desc *C.char, descLen C.SQLSMALLINT, descOutLen *C.SQLSMALLINT,
	attrs *C.char, attrsLen C.SQLSMALLINT, attrsOutLen *C.SQLSMALLINT) C.SQLINTEGER {
	_ = envHandle
	_ = direction
	if desc != nil && descLen > 0 {
		writeCString(desc, "", int(descLen))
	}
	if descOutLen != nil {
		*descOutLen = 0
	}
	if attrs != nil && attrsLen > 0 {
		writeCString(attrs, "", int(attrsLen))
	}
	if attrsOutLen != nil {
		*attrsOutLen = 0
	}
	return C.SQL_NO_DATA
}

// ----- SQLSetConnectAttr / SQLGetConnectAttr -----

//export SQLSetConnectAttr
func SQLSetConnectAttr(dbcHandle C.SQLINTEGER, attr C.SQLINTEGER, value C.SQLPOINTER, valueLen C.SQLINTEGER) C.SQLINTEGER {
	_ = dbcHandle
	_ = attr
	_ = value
	_ = valueLen
	return C.SQL_SUCCESS
}

//export SQLGetConnectAttr
func SQLGetConnectAttr(dbcHandle C.SQLINTEGER, attr C.SQLINTEGER, value C.SQLPOINTER, valueLen C.SQLINTEGER, outLen *C.SQLINTEGER) C.SQLINTEGER {
	_ = dbcHandle
	_ = attr
	_ = value
	_ = valueLen
	if outLen != nil {
		*outLen = 0
	}
	return C.SQL_SUCCESS
}

// ----- SQLSetStmtAttr / SQLGetStmtAttr -----

//export SQLSetStmtAttr
func SQLSetStmtAttr(stmtHandle C.SQLINTEGER, attr C.SQLINTEGER, value C.SQLPOINTER, valueLen C.SQLINTEGER) C.SQLINTEGER {
	_ = stmtHandle
	_ = attr
	_ = value
	_ = valueLen
	return C.SQL_SUCCESS
}

//export SQLGetStmtAttr
func SQLGetStmtAttr(stmtHandle C.SQLINTEGER, attr C.SQLINTEGER, value C.SQLPOINTER, valueLen C.SQLINTEGER, outLen *C.SQLINTEGER) C.SQLINTEGER {
	_ = stmtHandle
	_ = attr
	_ = value
	_ = valueLen
	if outLen != nil {
		*outLen = 0
	}
	return C.SQL_SUCCESS
}

//export SQLNumParams
func SQLNumParams(stmtHandle C.SQLINTEGER, paramCount *C.SQLSMALLINT) C.SQLINTEGER {
	s := handle.LookupStmt(uintptr(stmtHandle))
	if s == nil {
		return C.SQL_ERROR
	}
	*paramCount = C.SQLSMALLINT(len(s.Params2()))
	return C.SQL_SUCCESS
}
