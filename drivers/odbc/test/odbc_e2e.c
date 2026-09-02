// odbc_e2e.c -- end-to-end test for the WeDB ODBC driver through the
// Windows ODBC Manager. Expects the driver to be registered in HKLM
// (see test/e2e.ps1 which handles that).
//
// Build: cl /EHsc /W3 /DUNICODE /D_UNICODE odbc_e2e.c odbc32.lib /Fe:odbc_e2e.exe
// Run:   odbc_e2e.exe "C:\full\path\to\test.db"
//
// The connection string uses DSN=WeDB Sample; the script pre-creates
// that DSN with the right DBQ. We then run the same coverage as
// direct.c: CREATE, INSERT, SELECT, aggregate, WHERE, SQLTables,
// SQLColumns, SQLGetInfo, SQLGetFunctions, SQLGetTypeInfo.

#include <windows.h>
#include <sql.h>
#include <sqlext.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

static const char* DSN_NAME = "WeDB Sample";

static void readDiag(const char* what, SQLHENV env, SQLHDBC dbc, SQLHSTMT stmt) {
    struct { SQLSMALLINT type; SQLHANDLE h; const char* name; } tries[3] = {
        { SQL_HANDLE_STMT, stmt, "STMT" },
        { SQL_HANDLE_DBC,  dbc,  "DBC"  },
        { SQL_HANDLE_ENV,  env,  "ENV"  },
    };
    for (int i = 0; i < 3; i++) {
        if (!tries[i].h) continue;
        SQLWCHAR state[8] = {0}, msg[512] = {0};
        SQLINTEGER native = 0;
        SQLSMALLINT len = 0;
        SQLRETURN rc = SQLGetDiagRecW(tries[i].type, tries[i].h, 1, state, &native, msg, sizeof(msg)/sizeof(SQLWCHAR), &len);
        if (rc == SQL_SUCCESS || rc == SQL_SUCCESS_WITH_INFO) {
            char stateA[8] = {0}, msgA[512] = {0};
            for (int j = 0; j < 7 && state[j]; j++) stateA[j] = (char)state[j];
            for (int j = 0; j < 511 && j < len && msg[j]; j++) msgA[j] = (char)(msg[j] & 0x7F);
            fprintf(stderr, "  diag on %s: state=%s native=%ld msg='%s'\n",
                    tries[i].name, stateA, (long)native, msgA);
            // Also dump raw bytes
            fprintf(stderr, "  raw state bytes:");
            unsigned char* sb = (unsigned char*)state;
            for (int j = 0; j < 16; j++) fprintf(stderr, " %02X", sb[j]);
            fprintf(stderr, "\n  raw msg bytes:");
            unsigned char* mb = (unsigned char*)msg;
            for (int j = 0; j < 32; j++) fprintf(stderr, " %02X", mb[j]);
            fprintf(stderr, "\n");
            return;
        }
    }
    fprintf(stderr, "  (no diag records)\n");
}

static void check(SQLRETURN rc, const char* what, SQLHENV env, SQLHDBC dbc, SQLHSTMT stmt) {
    if (rc == SQL_SUCCESS || rc == SQL_SUCCESS_WITH_INFO) return;
    if (rc == SQL_NO_DATA) { printf("[OK no data] %s\n", what); return; }
    fprintf(stderr, "FAIL: %s rc=%d\n", what, rc);
    readDiag(what, env, dbc, stmt);
    exit(1);
}

int main(int argc, char** argv) {
    if (argc < 2) { fprintf(stderr, "usage: %s <path>\n", argv[0]); return 2; }
    const char* path = argv[1];

    SQLHENV env = SQL_NULL_HENV;
    SQLHDBC dbc = SQL_NULL_HDBC;
    SQLHSTMT stmt = SQL_NULL_HSTMT;

    check(SQLAllocHandle(SQL_HANDLE_ENV, SQL_NULL_HANDLE, &env), "AllocEnv", env, dbc, stmt);
    check(SQLSetEnvAttr(env, SQL_ATTR_ODBC_VERSION, (SQLPOINTER)SQL_OV_ODBC3, 0), "SetEnvAttr", env, dbc, stmt);
    check(SQLAllocHandle(SQL_HANDLE_DBC, env, &dbc), "AllocDbc", env, dbc, stmt);

    char conn[256];
    snprintf(conn, sizeof(conn), "DSN=WeDB Sample;");

    WCHAR wconn[1024];
    int connLen = (int)strlen(conn);
    MultiByteToWideChar(CP_ACP, 0, conn, connLen, wconn, 1024);
    wconn[connLen] = 0; // explicit NUL
    printf("connLen=%d\n", connLen);

    SQLWCHAR out[1024] = {0};
    SQLSMALLINT woutLen = 0;
    check(SQLDriverConnectW(dbc, NULL, wconn, SQL_NTS, out, sizeof(out)/sizeof(SQLWCHAR), &woutLen, SQL_DRIVER_NOPROMPT),
          "SQLDriverConnectW", env, dbc, stmt);
    printf("Connected.\n");

    check(SQLAllocHandle(SQL_HANDLE_STMT, dbc, &stmt), "AllocStmt", env, dbc, stmt);
    printf("Stmt allocated.\n");

    check(SQLExecDirect(stmt, (SQLCHAR*)"DROP TABLE IF EXISTS t", SQL_NTS), "DROP", env, dbc, stmt);
    check(SQLExecDirect(stmt, (SQLCHAR*)"CREATE TABLE t (id INTEGER PRIMARY KEY, name TEXT, age INTEGER)", SQL_NTS),
          "CREATE", env, dbc, stmt);
    check(SQLExecDirect(stmt, (SQLCHAR*)"INSERT INTO t (id, name, age) VALUES (1, 'alice', 30)", SQL_NTS), "I1", env, dbc, stmt);
    check(SQLExecDirect(stmt, (SQLCHAR*)"INSERT INTO t (id, name, age) VALUES (2, 'bob', 25)", SQL_NTS), "I2", env, dbc, stmt);
    check(SQLExecDirect(stmt, (SQLCHAR*)"INSERT INTO t (id, name, age) VALUES (3, 'carol', 40)", SQL_NTS), "I3", env, dbc, stmt);

    SQLLEN rc = 0;
    SQLRowCount(stmt, &rc);
    printf("Last RowCount: %ld\n", (long)rc);

    check(SQLFreeStmt(stmt, SQL_CLOSE), "FreeStmt(close)", env, dbc, stmt);
    check(SQLExecDirect(stmt, (SQLCHAR*)"SELECT id, name, age FROM t ORDER BY id", SQL_NTS), "SELECT", env, dbc, stmt);

    SQLSMALLINT ncols = 0;
    SQLNumResultCols(stmt, &ncols);
    printf("Columns: %d\n", ncols);
    for (SQLSMALLINT i = 1; i <= ncols; i++) {
        SQLCHAR name[64]; SQLSMALLINT nameLen = 0, type = 0;
        SQLULEN size = 0; SQLSMALLINT dec = 0, nullable = 0;
        SQLDescribeCol(stmt, i, name, sizeof(name), &nameLen, &type, &size, &dec, &nullable);
        printf("  col %d: %s type=%d\n", i, name, type);
    }
    while (SQLFetch(stmt) == SQL_SUCCESS) {
        SQLLEN id = 0, age = 0; char name[64] = {0}; SQLLEN nameLen = 0;
        SQLGetData(stmt, 1, SQL_C_LONG, &id, 0, NULL);
        SQLGetData(stmt, 2, SQL_C_CHAR, name, sizeof(name), &nameLen);
        SQLGetData(stmt, 3, SQL_C_LONG, &age, 0, NULL);
        printf("Row: id=%ld name=%s age=%ld\n", (long)id, name, (long)age);
    }

    SQLFreeStmt(stmt, SQL_CLOSE);
    check(SQLExecDirect(stmt, (SQLCHAR*)"SELECT COUNT(*), SUM(age), AVG(age), MIN(age), MAX(age) FROM t", SQL_NTS), "AGG", env, dbc, stmt);
    if (SQLFetch(stmt) == SQL_SUCCESS) {
        char buf[5][64] = {{0}}; SQLLEN ind = 0;
        for (int i = 0; i < 5; i++)
            SQLGetData(stmt, (SQLSMALLINT)(i+1), SQL_C_CHAR, buf[i], sizeof(buf[i]), &ind);
        printf("COUNT=%s SUM=%s AVG=%s MIN=%s MAX=%s\n", buf[0], buf[1], buf[2], buf[3], buf[4]);
    }

    SQLFreeStmt(stmt, SQL_CLOSE);
    check(SQLExecDirect(stmt, (SQLCHAR*)"SELECT name FROM t WHERE age > 18", SQL_NTS), "WHERE", env, dbc, stmt);
    while (SQLFetch(stmt) == SQL_SUCCESS) {
        char name[64] = {0}; SQLLEN ind = 0;
        SQLGetData(stmt, 1, SQL_C_CHAR, name, sizeof(name), &ind);
        printf("Filter: %s\n", name);
    }

    SQLFreeStmt(stmt, SQL_CLOSE);
    check(SQLTables(stmt, NULL, 0, NULL, 0, NULL, SQL_NTS, NULL, 0), "SQLTables", env, dbc, stmt);
    while (SQLFetch(stmt) == SQL_SUCCESS) {
        char name[64] = {0}, type[32] = {0}; SQLLEN ind;
        SQLGetData(stmt, 3, SQL_C_CHAR, name, sizeof(name), &ind);
        SQLGetData(stmt, 4, SQL_C_CHAR, type, sizeof(type), &ind);
        printf("Table: %s type=%s\n", name, type);
    }

    SQLFreeStmt(stmt, SQL_CLOSE);
    check(SQLColumns(stmt, NULL, 0, NULL, 0, (SQLCHAR*)"t", SQL_NTS, NULL, 0), "SQLColumns", env, dbc, stmt);
    while (SQLFetch(stmt) == SQL_SUCCESS) {
        char col[64] = {0}, typeName[32] = {0}; SQLINTEGER dataType = 0; SQLLEN ind;
        SQLGetData(stmt, 4, SQL_C_CHAR, col, sizeof(col), &ind);
        SQLGetData(stmt, 5, SQL_C_LONG, &dataType, 0, &ind);
        SQLGetData(stmt, 6, SQL_C_CHAR, typeName, sizeof(typeName), &ind);
        printf("Column: %s type=%ld (%s)\n", col, (long)dataType, typeName);
    }

    SQLCHAR infoBuf[64] = {0}; SQLSMALLINT infoLen = 0;
    SQLGetInfo(dbc, SQL_DBMS_NAME, infoBuf, sizeof(infoBuf), &infoLen);
    printf("DBMS_NAME=%s\n", infoBuf);
    memset(infoBuf, 0, sizeof(infoBuf));
    SQLGetInfo(dbc, SQL_DRIVER_NAME, infoBuf, sizeof(infoBuf), &infoLen);
    printf("DRIVER_NAME=%s\n", infoBuf);
    SQLUSMALLINT supp = 0;
    SQLGetFunctions(dbc, SQL_API_SQLEXECDIRECT, &supp);
    printf("SQLExecDirect supported: %d\n", supp);

    SQLFreeStmt(stmt, SQL_CLOSE);
    check(SQLGetTypeInfo(stmt, SQL_ALL_TYPES), "SQLGetTypeInfo", env, dbc, stmt);
    int n = 0;
    while (SQLFetch(stmt) == SQL_SUCCESS && n < 5) {
        char name[64] = {0}; SQLINTEGER dataType = 0; SQLLEN ind;
        SQLGetData(stmt, 1, SQL_C_CHAR, name, sizeof(name), &ind);
        SQLGetData(stmt, 2, SQL_C_LONG, &dataType, 0, &ind);
        printf("Type: %s (%ld)\n", name, (long)dataType);
        n++;
    }

    SQLFreeHandle(SQL_HANDLE_STMT, stmt);
    SQLDisconnect(dbc);
    SQLFreeHandle(SQL_HANDLE_DBC, dbc);
    SQLFreeHandle(SQL_HANDLE_ENV, env);
    DeleteFileA(path);
    printf("OK\n");
    return 0;
}
