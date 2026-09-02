// Package diag centralizes SQLSTATE codes used by the WeDB ODBC driver.
//
// We follow the ODBC 3.x SQLSTATE classes. Codes are returned both by
// SQLGetDiagRec and by status word functions (SQLExecDirect, SQLExecute,
// SQLFetch, etc.) which set output SQLINTEGER pointers to one of:
//
//	SQL_SUCCESS           (0)
//	SQL_SUCCESS_WITH_INFO (1)
//	SQL_NO_DATA           (100)
//	SQL_ERROR             (-1)
//	SQL_INVALID_HANDLE    (-2)
//	SQL_NEED_DATA         (99)
//	SQL_STILL_EXECUTING   (2)
package diag

// Status word return values (SQLINTEGER codes).
const (
	Success           int32 = 0
	SuccessWithInfo   int32 = 1
	NoData            int32 = 100
	Error             int32 = -1
	InvalidHandle     int32 = -2
	NeedData          int32 = 99
	StillExecuting    int32 = 2
)

// SQLSTATE strings — ODBC 3.x.
const (
	// 07xxx: dynamic SQL error
	StateDynamicCursor   = "07005" // prepared statement not a cursor spec
	StateRestrictedType  = "07006" // attribute cannot be applied
	StateInvalidUseNull  = "07009" // invalid use of null pointer

	// 08xxx: connection exception
	StateConnectFail     = "08001" // unable to connect
	StateConnInUse       = "08002" // connection in use
	StateConnNotOpen     = "08003" // connection not open
	StateServerReject    = "08004" // server rejected
	StateCommLinkFail    = "08S01" // communication link failure

	// 21xxx: cardinality violation
	StateCardinality     = "21S01" // insert value list does not match column list

	// 22xxx: data exception
	StateStringRightTrunc = "22001" // string data right-truncated
	StateNumericOutOfRange= "22003" // numeric value out of range
	StateInvalidDatetimeFmt="22007" // invalid datetime format
	StateInvalidCharVal   = "22018" // invalid character value
	StateIntegrityConstraint="23000" // integrity constraint violation
	StateInvalidCursorState="24000" // invalid cursor state
	StateInvalidTransactionState="25000" // invalid transaction state

	// 3Bxxx: savepoint / DDL
	StateSyntaxError      = "42000" // syntax error or access violation
	StateBaseTableNotFound = "42S02" // base table or view not found
	StateColumnNotFound   = "42S22" // column not found
	StateTableExists      = "42S01" // table already exists
	StateIndexExists      = "42S11" // index already exists
	StateIndexNotFound    = "42S12" // index not found

	// HYxxx: CLI-specific
	StateGeneralError     = "HY000" // general error
	StateMemoryAlloc      = "HY001" // memory allocation failure
	StateInvalidHandle    = "HY009" // invalid use of null pointer / handle
	StateFunctionSequence = "HY010" // function sequence error
	StateInvalidParam     = "HY024" // invalid argument value
	StateOptChg           = "HY011" // attribute cannot be set now
	StateFetchOutOfSeq    = "HY019" // fetch out of sequence
	StateRowOutOfRange    = "HY107" // row value out of range
	StateDriverNotCapable = "HYC00" // optional feature not implemented
	StateNoDsn            = "IM002" // data source not found / no default driver
	StateConnTimeout      = "HYT00" // timeout expired
	StateNotImplemented   = "HYC00" // alias of above
)

// Native error codes (driver-specific, surfaced as fNativeError).
const (
	// Generic
	ENoMemory            int32 = 1001
	EInvalidArg          int32 = 1002
	ENotSupported        int32 = 1003
	ESyntax              int32 = 1004
	EUnknownTable        int32 = 1005
	EUnknownColumn       int32 = 1006
	EDuplicateKey        int32 = 1007
	ENoResultSet         int32 = 1008
	ENoRowsAffected      int32 = 1009
	ERowCountUnavailable  int32 = 1010
	EParamIndex          int32 = 1011
	ERowOutOfRange       int32 = 1018
	EParamType           int32 = 1012
	EParamValue          int32 = 1013
	EBindType            int32 = 1014
	EOpenFile            int32 = 1015
	ENoDriver            int32 = 1016
	ESqlParse            int32 = 1017
)
