// connect_only.c — minimal connection smoke test. Prints whatever
// SQLGetDiagRec reports on env, dbc, then both.
// Build: cl /W3 /EHsc connect_only.c odbc32.lib advapi32.lib

#include <windows.h>
#include <sql.h>
#include <sqlext.h>
#include <stdio.h>

static void reg(const char* driver) {
    HKEY k; DWORD disp;
    char path[MAX_PATH];
    GetModuleFileNameA(NULL, path, MAX_PATH);
    char* p = strrchr(path, '\\');
    if (p) *(p+1) = 0;
    strcat(path, "wedb_odbc.dll");
    // Drivers list (system-wide; the manager reads HKLM)
    RegCreateKeyExA(HKEY_LOCAL_MACHINE, "SOFTWARE\\ODBC\\ODBC.INI\\ODBC Drivers", 0, NULL, 0, KEY_ALL_ACCESS, NULL, &k, &disp);
    RegSetValueExA(k, driver, 0, REG_SZ, (BYTE*)"Installed", 10);
    RegCloseKey(k);
    // Driver path (system-wide)
    char key[256]; snprintf(key, sizeof(key), "SOFTWARE\\ODBC\\ODBC.INI\\%s", driver);
    RegCreateKeyExA(HKEY_LOCAL_MACHINE, key, 0, NULL, 0, KEY_ALL_ACCESS, NULL, &k, &disp);
    RegSetValueExA(k, "Driver", 0, REG_SZ, (BYTE*)path, (DWORD)strlen(path)+1);
    RegSetValueExA(k, "DriverODBCVer", 0, REG_SZ, (BYTE*)"03.00", 5);
    RegCloseKey(k);
    printf("Registered driver at: %s\n", path);
}

static void unreg(const char* driver) {
    char key[256]; snprintf(key, sizeof(key), "SOFTWARE\\ODBC\\ODBC.INI\\%s", driver);
    RegDeleteKeyA(HKEY_LOCAL_MACHINE, key);
    HKEY k;
    if (RegOpenKeyExA(HKEY_LOCAL_MACHINE, "SOFTWARE\\ODBC\\ODBC.INI\\ODBC Drivers", 0, KEY_ALL_ACCESS, &k) == ERROR_SUCCESS) {
        RegDeleteValueA(k, driver);
        RegCloseKey(k);
    }
}

static void dumpDiag(SQLSMALLINT type, SQLHANDLE h, const char* tag) {
    for (SQLSMALLINT i = 1; i <= 5; i++) {
        SQLWCHAR state[8]={0};
        SQLWCHAR msg[512]={0};
        SQLINTEGER native=0;
        SQLSMALLINT len=0;
        SQLRETURN rc = SQLGetDiagRecW(type, h, i, state, &native, msg, sizeof(msg)/sizeof(SQLWCHAR), &len);
        if (rc == SQL_NO_DATA) break;
        // Dump raw state bytes
        printf("  %s #%d: rc=%d len=%d native=%ld\n", tag, i, rc, len, (long)native);
        printf("    state raw bytes:");
        unsigned char* sb = (unsigned char*)state;
        for (int j = 0; j < 12 && j < (len > 0 ? len*2+2 : 12); j++) {
            printf(" %02X", sb[j]);
        }
        printf("\n    msg raw bytes:");
        unsigned char* mb = (unsigned char*)msg;
        for (int j = 0; j < 24 && j < (len > 0 ? len*2+2 : 24); j++) {
            printf(" %02X", mb[j]);
        }
        printf("\n");
    }
}

int main(int argc, char** argv) {
    reg("WeDB ODBC Driver");
    const char* path = argc > 1 ? argv[1] : "smoke.db";
    char conn[512];
    snprintf(conn, sizeof(conn), "DRIVER={WeDB ODBC Driver};DBQ=%s;", path);

    SQLHENV env; SQLHDBC dbc;
    SQLRETURN rc;

    rc = SQLAllocHandle(SQL_HANDLE_ENV, SQL_NULL_HANDLE, &env);
    printf("AllocEnv rc=%d\n", rc);
    rc = SQLSetEnvAttr(env, SQL_ATTR_ODBC_VERSION, (SQLPOINTER)SQL_OV_ODBC3, 0);
    printf("SetEnvAttr rc=%d\n", rc);
    rc = SQLAllocHandle(SQL_HANDLE_DBC, env, &dbc);
    printf("AllocDbc rc=%d\n", rc);

    SQLWCHAR out[1024]={0};
    SQLSMALLINT outLen=0;
    rc = SQLDriverConnectW(dbc, NULL, (SQLWCHAR*)L"DRIVER={WeDB ODBC Driver};DBQ=smoke.db;", SQL_NTS, out, sizeof(out)/sizeof(SQLWCHAR), &outLen, SQL_DRIVER_NOPROMPT);
    printf("DriverConnect rc=%d\n", rc);
    {
        char outA[256]={0};
        for (int j=0;j<255 && out[j];j++) outA[j]=(char)out[j];
        printf("out='%s'\n", outA);
    }

    if (rc != SQL_SUCCESS && rc != SQL_SUCCESS_WITH_INFO) {
        dumpDiag(SQL_HANDLE_DBC, dbc, "dbc");
        dumpDiag(SQL_HANDLE_ENV, env, "env");
    } else {
        printf("Connected OK\n");
        SQLDisconnect(dbc);
    }

    SQLFreeHandle(SQL_HANDLE_DBC, dbc);
    SQLFreeHandle(SQL_HANDLE_ENV, env);
    unreg("WeDB ODBC Driver");
    DeleteFileA(path);
    return 0;
}
