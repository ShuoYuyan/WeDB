package main

// api_meta_info.go contains SQLInfoType constants and a value lookup
// table. We use a map rather than a switch so that constants sharing
// numeric values don't trip Go's duplicate-case check.

const (
	SQLInfoDriverName           int16 = 6
	SQLInfoDriverVer            int16 = 7
	SQLInfoDriverODBCVer        int16 = 77
	SQLInfoODBCVer              int16 = 113
	SQLInfoDBMSName             int16 = 17
	SQLInfoDBMSVer              int16 = 18
	SQLInfoDataSourceName       int16 = 2
	SQLInfoDataSourceReadOnly   int16 = 25
	SQLInfoIdentifierQuoteChar  int16 = 29
	SQLInfoCatalogName          int16 = 10003
	SQLInfoCatalogNameSep       int16 = 41
	SQLInfoTableTerm            int16 = 45
	SQLInfoSchemaTerm           int16 = 39
	SQLInfoCatalogTerm          int16 = 42
	SQLInfoNullCollation        int16 = 85
	SQLInfoTxnIsolation         int16 = 26
	SQLInfoTxnCapable           int16 = 46
	SQLInfoDefaultTxnIsolation  int16 = 26
	SQLInfoConvertFunctions     int16 = 28
	SQLInfoNumericFunctions     int16 = 23
	SQLInfoStringFunctions      int16 = 24
	SQLInfoSystemFunctions      int16 = 25
	SQLInfoTimedateFunctions    int16 = 22
	SQLInfoSearchPatternEscape  int16 = 14
	SQLInfoOuterJoins           int16 = 38
	SQLInfoMaxConcurrent        int16 = 1
	SQLInfoMaxDriverConn        int16 = 0
	SQLInfoActiveEnvironments   int16 = 116
	SQLInfoAsyncMode            int16 = 10021
	SQLInfoAsyncNotification    int16 = 10025
	SQLInfoBatchRowCount        int16 = 120
	SQLInfoBatchSupport         int16 = 121
	SQLInfoGetdataExtensions    int16 = 65
	SQLInfoLikeEscapeClause     int16 = 113
	SQLInfoMaxAsyncConcStmts    int16 = 10022
	SQLInfoMultipleActiveTxn    int16 = 37
	SQLInfoMultResultSets       int16 = 36
	SQLInfoNonNullableColumns   int16 = 75
	SQLInfoNullable             int16 = 0
	SQLInfoNullableUnknown      int16 = 2
	SQLInfoProcedures           int16 = 21
	SQLInfoRowUpdates           int16 = 20
	SQLInfoScrollOptions        int16 = 44
	SQLInfoScrollConcurrency    int16 = 43
	SQLInfoSensitivity          int16 = 15
	SQLInfoSpecialCharacters    int16 = 94
	SQLInfoTimedateAddIntervals int16 = 109
	SQLInfoTimedateDiffIntervals int16 = 110
	SQLInfoUnion                int16 = 96
	SQLInfoUnionAll             int16 = 97
	SQLInfoFetchDirection       int16 = 8
	SQLInfoFileUsage            int16 = 84
	SQLInfoKeysetCursorAttr1    int16 = 86
	SQLInfoKeysetCursorAttr2    int16 = 87
	SQLInfoLockTypes            int16 = 78
	SQLInfoMaxLength            int16 = 10004
	SQLInfoMaxRowSizeInclLong   int16 = 107
	SQLInfoOJCapabilities       int16 = 65
	SQLInfoOrderByColsInSelect  int16 = 99
	SQLInfoParamArrayRowCounts  int16 = 127
	SQLInfoParamArraySelects    int16 = 128
	SQLInfoPosOperations        int16 = 79
	SQLInfoPositionedStmts      int16 = 80
	SQLInfoPositionedUpdate     int16 = 82
	SQLInfoPositionedDelete     int16 = 81
	SQLInfoPrepareAuto          int16 = 81
	SQLInfoPrepareManual        int16 = 82
	SQLInfoStaticCursorAttr1    int16 = 83
	SQLInfoStaticCursorAttr2    int16 = 85
	SQLInfoDynamicCursorAttr1   int16 = 84
	SQLInfoDynamicCursorAttr2   int16 = 85
	SQLInfoTxnIsolationOption   int16 = 72
	SQLInfoMaxCatalogNameLen    int16 = 34
	SQLInfoMaxSchemaNameLen     int16 = 32
	SQLInfoMaxTableNameLen      int16 = 35
	SQLInfoMaxColumnNameLen     int16 = 30
	SQLInfoMaxCursorNameLen     int16 = 31
	SQLInfoMaxBinaryLitLen      int16 = 112
	SQLInfoMaxCharLitLen        int16 = 108
	SQLInfoMaxIndexSize         int16 = 102
	SQLInfoMaxRowSize           int16 = 104
	SQLInfoMaxStatementLen      int16 = 105
	SQLInfoMaxUserNameLen       int16 = 107
	// Additional critical attributes for ODBC Manager compatibility
	SQLInfoCursorCommitBehavior   int16 = 23
	SQLInfoCursorRollbackBehavior int16 = 24
	SQLInfoAccessibleTables       int16 = 19
	SQLInfoAccessibleProcedures   int16 = 20
	SQLInfoMaxTablesInSelect      int16 = 106
	SQLInfoMaxColumnsInSelect     int16 = 103
	SQLInfoMaxColumnsInTable      int16 = 101
	SQLInfoMaxColumnsInOrderBy    int16 = 108
	SQLInfoMaxColumnsInGroupBy    int16 = 97
	SQLInfoMaxUserFunctions      int16 = 98
	SQLInfoMaxProcedureNameLen    int16 = 32
	SQLInfoMaxCursorNameLenAlt    int16 = 31
	SQLInfoNeedLongDataLen        int16 = 111
	SQLInfoBookmarks              int16 = 12
	SQLInfoResultSets             int16 = 8
	SQLInfoSqlServer              int16 = 10
	SQLInfoIntegrity              int16 = 73
	SQLInfoProcTerms              int16 = 40
	SQLInfoProcedureTerm          int16 = 40
)

// infoString returns a string-form answer for an info type. Empty
// string means "not a string". Switch uses single-value cases to
// avoid Go's duplicate-case check (multiple ODBC info types share
// numeric values).
func infoString(info int16) (string, bool) {
	if info == SQLInfoDriverName || info == SQLInfoDBMSName {
		return "WeDB", true
	}
	if info == SQLInfoDriverVer {
		return "01.00.0000", true
	}
	if info == SQLInfoDriverODBCVer || info == SQLInfoODBCVer {
		return "03.00", true
	}
	if info == SQLInfoDBMSVer {
		return "1.0", true
	}
	if info == SQLInfoIdentifierQuoteChar {
		return "\"", true
	}
	if info == SQLInfoCatalogName || info == SQLInfoCatalogTerm {
		return "", true
	}
	if info == SQLInfoTableTerm || info == SQLInfoSchemaTerm {
		return "table", true
	}
	if info == SQLInfoCatalogNameSep {
		return ".", true
	}
	if info == SQLInfoLikeEscapeClause {
		return ";", true
	}
	if info == SQLInfoMaxRowSizeInclLong || info == SQLInfoRowUpdates ||
		info == SQLInfoOuterJoins || info == SQLInfoSpecialCharacters {
		return "N", true
	}
	if info == SQLInfoSearchPatternEscape {
		return "\\", true
	}
	if info == SQLInfoProcedureTerm || info == SQLInfoProcTerms {
		return "procedure", true
	}
	return "", false
}

// infoInt returns a 32-bit integer answer for an info type.
func infoInt(info int16) (uint32, bool) {
	if info == SQLInfoNullCollation {
		return 4, true // SQL_NC_HIGH
	}
	if info == SQLInfoTxnIsolation || info == SQLInfoDefaultTxnIsolation {
		return 2, true // SQL_TXN_READ_COMMITTED
	}
	if info == SQLInfoTxnCapable || info == SQLInfoTxnIsolationOption {
		// bitmask of supported isolation levels
		return 1 | 2 | 4 | 8, true
	}
	if info == SQLInfoMultipleActiveTxn {
		return 1, true
	}
	// Critical cursor behavior attributes for ODBC Manager compatibility
	if info == SQLInfoCursorCommitBehavior || info == SQLInfoCursorRollbackBehavior {
		return 0, true // SQL_CB_DELETE (cursors are closed on commit/rollback)
	}
	if info == SQLInfoAccessibleTables || info == SQLInfoAccessibleProcedures {
		return 1, true // Y (accessible)
	}
	if info == SQLInfoMaxTablesInSelect {
		return 100, true // reasonable max
	}
	if info == SQLInfoMaxColumnsInSelect {
		return 100, true // reasonable max
	}
	if info == SQLInfoMaxColumnsInTable {
		return 100, true // reasonable max
	}
	if info == SQLInfoMaxColumnsInOrderBy {
		return 100, true // reasonable max
	}
	if info == SQLInfoMaxColumnsInGroupBy {
		return 100, true // reasonable max
	}
	if info == SQLInfoMaxUserFunctions {
		return 0, true // no user functions
	}
	if info == SQLInfoMaxProcedureNameLen {
		return 128, true // reasonable max
	}
	if info == SQLInfoMaxCursorNameLen || info == SQLInfoMaxCursorNameLenAlt {
		return 128, true // reasonable max
	}
	if info == SQLInfoNeedLongDataLen {
		return 0, true // Y (don't need long data len)
	}
	if info == SQLInfoBookmarks {
		return 0, true // No bookmarks
	}
	if info == SQLInfoResultSets {
		return 1, true // Supports result sets
	}
	if info == SQLInfoSqlServer {
		return 0, true // No SQL server
	}
	if info == SQLInfoIntegrity {
		return 0, true // No integrity
	}
	return 0, false
}
