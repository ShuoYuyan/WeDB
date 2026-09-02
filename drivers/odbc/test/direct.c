// direct.c - bypass ODBC Manager, call driver DLL directly via
// LoadLibrary + GetProcAddress. This is the primary e2e test because
// it doesn't require admin privileges (no HKLM registry write).
//
// Build: cl /EHsc /W3 direct.c /Fe:direct.exe
// Run:   direct.exe <path-to-db>

#include <windows.h>
#include <stdio.h>
#include <stdint.h>
#include <string.h>

typedef int   SQLRETURN;
typedef int   SQLINTEGER;
typedef short SQLSMALLINT;
typedef void* SQLPOINTER;
typedef long long SQLLEN;

typedef SQLRETURN (*pfnSQLAllocHandle)(SQLSMALLINT, SQLINTEGER, SQLINTEGER*);
typedef SQLRETURN (*pfnSQLSetEnvAttr)(SQLINTEGER, SQLINTEGER, SQLPOINTER, SQLINTEGER);
typedef SQLRETURN (*pfnSQLAllocConnect)(SQLINTEGER, SQLINTEGER*);
typedef SQLRETURN (*pfnSQLAllocStmt)(SQLINTEGER, SQLINTEGER*);
typedef SQLRETURN (*pfnSQLDriverConnect)(SQLINTEGER, void*, const char*, SQLSMALLINT,
                                          char*, SQLSMALLINT, SQLSMALLINT*, SQLSMALLINT);
typedef SQLRETURN (*pfnSQLExecDirect)(SQLINTEGER, const char*, SQLINTEGER);
typedef SQLRETURN (*pfnSQLPrepare)(SQLINTEGER, const char*, SQLINTEGER);
typedef SQLRETURN (*pfnSQLExecute)(SQLINTEGER);
typedef SQLRETURN (*pfnSQLFetch)(SQLINTEGER);
typedef SQLRETURN (*pfnSQLGetData)(SQLINTEGER, SQLSMALLINT, SQLSMALLINT,
                                    void*, SQLLEN, SQLLEN*);
typedef SQLRETURN (*pfnSQLRowCount)(SQLINTEGER, SQLLEN*);
typedef SQLRETURN (*pfnSQLNumResultCols)(SQLINTEGER, SQLSMALLINT*);
typedef SQLRETURN (*pfnSQLDescribeCol)(SQLINTEGER, SQLSMALLINT, char*, SQLSMALLINT, SQLSMALLINT*,
                                        SQLSMALLINT*, SQLLEN*, SQLSMALLINT*, SQLSMALLINT*);
typedef SQLRETURN (*pfnSQLFreeStmt)(SQLINTEGER, SQLSMALLINT);
typedef SQLRETURN (*pfnSQLTables)(SQLINTEGER, char*, SQLSMALLINT, char*, SQLSMALLINT,
                                   char*, SQLSMALLINT, char*, SQLSMALLINT);
typedef SQLRETURN (*pfnSQLColumns)(SQLINTEGER, char*, SQLSMALLINT, char*, SQLSMALLINT,
                                    char*, SQLSMALLINT, char*, SQLSMALLINT);
typedef SQLRETURN (*pfnSQLGetTypeInfo)(SQLINTEGER, SQLSMALLINT);
typedef SQLRETURN (*pfnSQLGetInfo)(SQLINTEGER, SQLSMALLINT, void*, SQLSMALLINT, SQLSMALLINT*);
typedef SQLRETURN (*pfnSQLGetFunctions)(SQLINTEGER, SQLSMALLINT, SQLSMALLINT*);
typedef SQLRETURN (*pfnSQLGetDiagRec)(SQLSMALLINT, SQLINTEGER, SQLSMALLINT,
                                      char*, SQLINTEGER*, char*, SQLSMALLINT, SQLSMALLINT*);
typedef SQLRETURN (*pfnSQLDisconnect)(SQLINTEGER);
typedef SQLRETURN (*pfnSQLFreeHandle)(SQLSMALLINT, SQLINTEGER);

// Use the values the driver expects (mirrors sql.h).
#define SQL_HANDLE_ENV    1
#define SQL_HANDLE_DBC    2
#define SQL_HANDLE_STMT   3
#define SQL_NTS           (-3)
#define SQL_SUCCESS       0
#define SQL_SUCCESS_WITH_INFO 1
#define SQL_NO_DATA       100
#define SQL_ERROR         (-1)
#define SQL_ATTR_ODBC_VERSION 200
#define SQL_OV_ODBC3      3
#define SQL_CLOSE         0
#define SQL_DROP          1
#define SQL_C_CHAR        1
#define SQL_C_LONG        4
#define SQL_ALL_TYPES     0
#define SQL_DBMS_NAME     17
#define SQL_DRIVER_NAME   6
#define SQL_ODBC_VER      113
#define SQL_TXN_ISOLATION 26
#define SQL_API_SQLEXECDIRECT 11

static FARPROC get(HMODULE dll, const char* name) {
    FARPROC p = GetProcAddress(dll, name);
    if (!p) printf("missing: %s (err=%lu)\n", name, GetLastError());
    return p;
}

#define GET(h, name, var) do { var = (void*)get(h, name); if (!var) return 1; } while(0)

static void check(SQLRETURN rc, const char* what) {
    if (rc == SQL_SUCCESS || rc == SQL_SUCCESS_WITH_INFO) {
        printf("  OK %s\n", what);
        return;
    }
    if (rc == SQL_NO_DATA) {
        printf("  OK (no data) %s\n", what);
        return;
    }
    printf("  FAIL %s rc=%d\n", what, rc);
}

int main(int argc, char** argv) {
    const char* path = argc > 1 ? argv[1] : "direct.db";
    HMODULE dll = LoadLibraryA("wedb_odbc.dll");
    if (!dll) { printf("LoadLibrary failed: %lu\n", GetLastError()); return 1; }
    printf("DLL: %p\n", dll);

    pfnSQLAllocHandle pAllocHandle; pfnSQLSetEnvAttr pSetEnvAttr;
    pfnSQLAllocConnect pAllocConnect; pfnSQLAllocStmt pAllocStmt;
    pfnSQLDriverConnect pDriverConnect;
    pfnSQLExecDirect pExecDirect; pfnSQLPrepare pPrepare; pfnSQLExecute pExecute;
    pfnSQLFetch pFetch; pfnSQLGetData pGetData; pfnSQLRowCount pRowCount;
    pfnSQLNumResultCols pNumResultCols; pfnSQLDescribeCol pDescribeCol;
    pfnSQLFreeStmt pFreeStmt;
    pfnSQLTables pTables; pfnSQLColumns pColumns; pfnSQLGetTypeInfo pGetTypeInfo;
    pfnSQLGetInfo pGetInfo; pfnSQLGetFunctions pGetFunctions;
    pfnSQLGetDiagRec pGetDiagRec;
    pfnSQLDisconnect pDisconnect; pfnSQLFreeHandle pFreeHandle;

    GET(dll, "SQLAllocHandle", pAllocHandle);
    GET(dll, "SQLSetEnvAttr", pSetEnvAttr);
    GET(dll, "SQLAllocConnect", pAllocConnect);
    GET(dll, "SQLAllocStmt", pAllocStmt);
    GET(dll, "SQLDriverConnect", pDriverConnect);
    GET(dll, "SQLExecDirect", pExecDirect);
    GET(dll, "SQLPrepare", pPrepare);
    GET(dll, "SQLExecute", pExecute);
    GET(dll, "SQLFetch", pFetch);
    GET(dll, "SQLGetData", pGetData);
    GET(dll, "SQLRowCount", pRowCount);
    GET(dll, "SQLNumResultCols", pNumResultCols);
    GET(dll, "SQLDescribeCol", pDescribeCol);
    GET(dll, "SQLFreeStmt", pFreeStmt);
    GET(dll, "SQLTables", pTables);
    GET(dll, "SQLColumns", pColumns);
    GET(dll, "SQLGetTypeInfo", pGetTypeInfo);
    GET(dll, "SQLGetInfo", pGetInfo);
    GET(dll, "SQLGetFunctions", pGetFunctions);
    GET(dll, "SQLGetDiagRec", pGetDiagRec);
    GET(dll, "SQLDisconnect", pDisconnect);
    GET(dll, "SQLFreeHandle", pFreeHandle);

    SQLINTEGER env = 0, dbc = 0, stmt = 0;
    check(pAllocHandle(SQL_HANDLE_ENV, 0, &env), "AllocEnv");
    check(pSetEnvAttr(env, SQL_ATTR_ODBC_VERSION, (SQLPOINTER)SQL_OV_ODBC3, 0), "SetEnvAttr");
    check(pAllocConnect(env, &dbc), "AllocConnect");

    char conn[512];
    snprintf(conn, sizeof(conn), "DRIVER={WeDB ODBC Driver};DBQ=%s;", path);
    char out[1024] = {0};
    SQLSMALLINT outLen = 0;
    check(pDriverConnect(dbc, NULL, conn, SQL_NTS, out, sizeof(out), &outLen, 0), "DriverConnect");
    printf("  out: %s\n", out);
    if (out[0] == 0) {
        char state[8] = {0}, msg[512] = {0};
        SQLINTEGER native = 0; SQLSMALLINT len = 0;
        pGetDiagRec(SQL_HANDLE_DBC, dbc, 1, state, &native, msg, sizeof(msg), &len);
        printf("  DIAG: state=%s native=%ld msg=%s\n", state, (long)native, msg);
    }

    check(pAllocStmt(dbc, &stmt), "AllocStmt");

    // DDL
    check(pExecDirect(stmt, "DROP TABLE IF EXISTS t", SQL_NTS), "DROP TABLE");
    check(pExecDirect(stmt, "CREATE TABLE t (id INTEGER PRIMARY KEY, name TEXT, age INTEGER)", SQL_NTS), "CREATE TABLE");

    // DML
    check(pExecDirect(stmt, "INSERT INTO t (id, name, age) VALUES (1, 'alice', 30)", SQL_NTS), "INSERT 1");
    check(pExecDirect(stmt, "INSERT INTO t (id, name, age) VALUES (2, 'bob', 25)", SQL_NTS), "INSERT 2");
    check(pExecDirect(stmt, "INSERT INTO t (id, name, age) VALUES (3, 'carol', 40)", SQL_NTS), "INSERT 3");

    SQLLEN rows = 0;
    pRowCount(stmt, &rows);
    printf("  RowCount after INSERT: %lld\n", (long long)rows);

    // Prepared
    check(pPrepare(stmt, "SELECT id, name, age FROM t WHERE id = ?", SQL_NTS), "PREPARE");
    // (parameter binding is stored but not yet substituted by the driver,
    // so the next SQLExecute will see the literal text we prepared. We
    // expect a parse error here; if so, SQLExecute is fine for unbound use.)
    {
        SQLRETURN rc = pExecute(stmt);
        if (rc == SQL_ERROR) {
            char state[8] = {0}, msg[256] = {0};
            SQLINTEGER native = 0; SQLSMALLINT len = 0;
            pGetDiagRec(SQL_HANDLE_STMT, stmt, 1, state, &native, msg, sizeof(msg), &len);
            printf("  (prepared execute returned error: %s -- expected for unbound ?)\n", msg);
        } else {
            printf("  prepared execute rc=%d\n", rc);
        }
    }
    pFreeStmt(stmt, SQL_CLOSE);

    // SELECT
    check(pExecDirect(stmt, "SELECT id, name, age FROM t ORDER BY id", SQL_NTS), "SELECT");
    SQLSMALLINT ncols = 0;
    pNumResultCols(stmt, &ncols);
    printf("  columns: %d\n", ncols);
    for (SQLSMALLINT i = 1; i <= ncols; i++) {
        char colName[64] = {0};
        SQLSMALLINT nameLen = 0, type = 0, nullable = 0, dec = 0;
        SQLLEN size = 0;
        pDescribeCol(stmt, i, colName, sizeof(colName), &nameLen, &type, &size, &dec, &nullable);
        printf("    [%d] %s type=%d\n", i, colName, type);
    }
    int fetched = 0;
    while (pFetch(stmt) == SQL_SUCCESS) {
        SQLLEN id = 0, age = 0, nameLen = 0;
        char name[64] = {0};
        pGetData(stmt, 1, SQL_C_LONG, &id, 0, NULL);
        pGetData(stmt, 2, SQL_C_CHAR, name, sizeof(name), &nameLen);
        pGetData(stmt, 3, SQL_C_LONG, &age, 0, NULL);
        unsigned char* ib = (unsigned char*)&id;
        unsigned char* ab = (unsigned char*)&age;
        printf("  row: id=%lld (raw %02X %02X %02X %02X) name=%s age=%lld (raw %02X %02X %02X %02X)\n",
               (long long)id, ib[0], ib[1], ib[2], ib[3], name, (long long)age, ab[0], ab[1], ab[2], ab[3]);
        fetched++;
    }
    printf("  total rows fetched: %d\n", fetched);

    pFreeStmt(stmt, SQL_CLOSE);

    // Aggregate
    check(pExecDirect(stmt, "SELECT COUNT(*), SUM(age), AVG(age), MIN(age), MAX(age) FROM t", SQL_NTS), "SELECT aggregate");
    if (pFetch(stmt) == SQL_SUCCESS) {
        char buf[8][64] = {{0}};
        for (int i = 0; i < 5; i++) {
            SQLLEN ind = 0;
            pGetData(stmt, (SQLSMALLINT)(i + 1), SQL_C_CHAR, buf[i], sizeof(buf[i]), &ind);
        }
        printf("  COUNT=%s SUM=%s AVG=%s MIN=%s MAX=%s\n", buf[0], buf[1], buf[2], buf[3], buf[4]);
    }

    pFreeStmt(stmt, SQL_CLOSE);

    // WHERE
    check(pExecDirect(stmt, "SELECT name FROM t WHERE age > 18 AND name = 'alice'", SQL_NTS), "SELECT WHERE");
    if (pFetch(stmt) == SQL_SUCCESS) {
        char name[64] = {0};
        SQLLEN ind = 0;
        pGetData(stmt, 1, SQL_C_CHAR, name, sizeof(name), &ind);
        printf("  WHERE match: %s\n", name);
    }

    pFreeStmt(stmt, SQL_CLOSE);

    // SQLTables
    check(pTables(stmt, NULL, 0, NULL, 0, NULL, SQL_NTS, NULL, 0), "SQLTables");
    int tblCount = 0;
    while (pFetch(stmt) == SQL_SUCCESS) {
        char cat[32]={0}, schem[32]={0}, name[64]={0}, type[32]={0};
        SQLLEN ind;
        pGetData(stmt, 1, SQL_C_CHAR, cat, sizeof(cat), &ind);
        pGetData(stmt, 2, SQL_C_CHAR, schem, sizeof(schem), &ind);
        pGetData(stmt, 3, SQL_C_CHAR, name, sizeof(name), &ind);
        pGetData(stmt, 4, SQL_C_CHAR, type, sizeof(type), &ind);
        printf("  Table: %s.%s.%s type=%s\n", cat, schem, name, type);
        tblCount++;
    }
    printf("  tables: %d\n", tblCount);

    pFreeStmt(stmt, SQL_CLOSE);

    // SQLColumns
    check(pColumns(stmt, NULL, 0, NULL, 0, (char*)"t", SQL_NTS, NULL, 0), "SQLColumns(t)");
    int colCount = 0;
    while (pFetch(stmt) == SQL_SUCCESS) {
        char col[64] = {0}, typeName[32] = {0};
        SQLINTEGER dataType = 0; SQLLEN ind;
        pGetData(stmt, 4, SQL_C_CHAR, col, sizeof(col), &ind);
        pGetData(stmt, 5, SQL_C_LONG, &dataType, 0, &ind);
        pGetData(stmt, 6, SQL_C_CHAR, typeName, sizeof(typeName), &ind);
        printf("  Column: %s type=%ld (%s)\n", col, (long)dataType, typeName);
        colCount++;
    }
    printf("  columns for t: %d\n", colCount);

    pFreeStmt(stmt, SQL_CLOSE);

    // SQLGetInfo
    {
        char buf[64] = {0};
        SQLSMALLINT len = 0;
        pGetInfo(dbc, SQL_DBMS_NAME, buf, sizeof(buf), &len);
        printf("  DBMS_NAME=%s\n", buf);
        memset(buf, 0, sizeof(buf));
        pGetInfo(dbc, SQL_DRIVER_NAME, buf, sizeof(buf), &len);
        printf("  DRIVER_NAME=%s\n", buf);
        memset(buf, 0, sizeof(buf));
        pGetInfo(dbc, SQL_ODBC_VER, buf, sizeof(buf), &len);
        printf("  ODBC_VER=%s\n", buf);
        SQLINTEGER iso = 0;
        pGetInfo(dbc, SQL_TXN_ISOLATION, &iso, sizeof(iso), &len);
        printf("  TXN_ISOLATION=%ld\n", (long)iso);
    }

    // SQLGetFunctions
    {
        SQLSMALLINT supp = 0;
        pGetFunctions(dbc, SQL_API_SQLEXECDIRECT, &supp);
        printf("  SQLExecDirect supported: %d\n", supp);
    }

    // SQLGetTypeInfo
    pFreeStmt(stmt, SQL_CLOSE);
    check(pGetTypeInfo(stmt, SQL_ALL_TYPES), "SQLGetTypeInfo");
    int n = 0;
    while (pFetch(stmt) == SQL_SUCCESS && n < 5) {
        char name[64] = {0};
        SQLINTEGER dataType = 0; SQLLEN ind;
        pGetData(stmt, 1, SQL_C_CHAR, name, sizeof(name), &ind);
        pGetData(stmt, 2, SQL_C_LONG, &dataType, 0, &ind);
        printf("  Type: %s (%ld)\n", name, (long)dataType);
        n++;
    }
    printf("  types shown: %d\n", n);

    pFreeStmt(stmt, SQL_DROP);
    pDisconnect(dbc);
    pFreeHandle(SQL_HANDLE_DBC, dbc);
    pFreeHandle(SQL_HANDLE_ENV, env);
    FreeLibrary(dll);
    DeleteFileA(path);
    printf("OK\n");
    return 0;
}
