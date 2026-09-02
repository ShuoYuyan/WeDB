#include <windows.h>
#include <sql.h>
#include <sqlext.h>
#include <stdio.h>

int main() {
    SQLHENV env = SQL_NULL_HENV;
    SQLHDBC dbc = SQL_NULL_HDBC;
    SQLHSTMT stmt = SQL_NULL_HSTMT;
    SQLRETURN rc;

    rc = SQLAllocHandle(SQL_HANDLE_ENV, SQL_NULL_HANDLE, &env);
    printf("AllocEnv rc=%d env=%p\n", rc, env);
    rc = SQLSetEnvAttr(env, SQL_ATTR_ODBC_VERSION, (SQLPOINTER)SQL_OV_ODBC3, 0);
    printf("SetEnvAttr rc=%d\n", rc);
    rc = SQLAllocHandle(SQL_HANDLE_DBC, env, &dbc);
    printf("AllocDbc rc=%d dbc=%p\n", rc, dbc);
    rc = SQLAllocHandle(SQL_HANDLE_STMT, dbc, &stmt);
    printf("AllocStmt rc=%d stmt=%p\n", rc, stmt);
    if (rc != SQL_SUCCESS) {
        SQLWCHAR state[8]={0}, msg[512]={0};
        SQLINTEGER native=0; SQLSMALLINT len=0;
        SQLGetDiagRecW(SQL_HANDLE_DBC, dbc, 1, state, &native, msg, sizeof(msg)/sizeof(SQLWCHAR), &len);
        char stateA[8]={0};
        for (int i=0;i<7&&state[i];i++) stateA[i]=(char)state[i];
        printf("DIAG state=%s native=%ld\n", stateA, (long)native);
    }
    SQLFreeHandle(SQL_HANDLE_STMT, stmt);
    SQLDisconnect(dbc);
    SQLFreeHandle(SQL_HANDLE_DBC, dbc);
    SQLFreeHandle(SQL_HANDLE_ENV, env);
    printf("OK\n");
    return 0;
}
