#include <windows.h>
#include <stdio.h>

int main() {
    HMODULE dll = LoadLibraryA("wedb_odbc.dll");
    if (!dll) { printf("LoadLibrary failed: %lu\n", GetLastError()); return 1; }
    printf("DLL: %p\n", dll);

    // Try the exact export name
    FARPROC p = GetProcAddress(dll, "SQLAllocHandle");
    if (p) printf("SQLAllocHandle found at %p\n", p);
    else printf("SQLAllocHandle NOT found (err=%lu)\n", GetLastError());

    // Try a few others
    p = GetProcAddress(dll, "SQLSetEnvAttr");
    printf("SQLSetEnvAttr: %s\n", p ? "found" : "not found");

    p = GetProcAddress(dll, "ConfigDSN");
    printf("ConfigDSN: %s\n", p ? "found" : "not found");

    p = GetProcAddress(dll, "SQLAllocEnv");
    printf("SQLAllocEnv: %s\n", p ? "found" : "not found");

    FreeLibrary(dll);
    return 0;
}
