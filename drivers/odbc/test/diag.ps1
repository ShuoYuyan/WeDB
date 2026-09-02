# diag.ps1 — try to load the driver DLL via ODBC Manager and report
# what the manager complains about.
$dll = "C:\Users\HP\Documents\WeDB\drivers\odbc\test\wedb_odbc.dll"
$conn = "DRIVER={WeDB ODBC Driver};DBQ=C:\Users\HP\Documents\WeDB\drivers\odbc\test\diag.db;"

# load odbc32 P/Invoke
$src = @"
using System;
using System.Runtime.InteropServices;
public class ODBC {
    [DllImport("odbccp32.dll", CharSet=CharSet.Unicode, SetLastError=true)]
    public static extern int SQLAllocHandle(short type, IntPtr inH, out IntPtr outH);
    [DllImport("odbccp32.dll", CharSet=CharSet.Unicode, SetLastError=true)]
    public static extern int SQLSetEnvAttr(IntPtr env, int attr, IntPtr val, int len);
    [DllImport("odbccp32.dll", CharSet=CharSet.Unicode, SetLastError=true)]
    public static extern int SQLDriverConnect(IntPtr dbc, IntPtr wnd, string cs, short csLen, System.Text.StringBuilder out, short outMax, out short outLen, short drvCompl);
    [DllImport("odbccp32.dll", CharSet=CharSet.Unicode, SetLastError=true)]
    public static extern int SQLGetDiagRec(short type, IntPtr h, short rec, System.Text.StringBuilder state, out int native, System.Text.StringBuilder msg, short msgMax, out short msgLen);
    [DllImport("odbccp32.dll", CharSet=CharSet.Unicode, SetLastError=true)]
    public static extern int SQLFreeHandle(short type, IntPtr h);
}
"@
Add-Type -TypeDefinition $src

$env2 = [IntPtr]::Zero; $dbc = [IntPtr]::Zero
[void][ODBC]::SQLAllocHandle(1, [IntPtr]::Zero, [ref]$env2)
[void][ODBC]::SQLSetEnvAttr($env2, 200, [IntPtr]3, 0)  # SQL_ATTR_ODBC_VERSION=200, SQL_OV_ODBC3=3
[void][ODBC]::SQLAllocHandle(2, $env2, [ref]$dbc)
$out = New-Object System.Text.StringBuilder 1024
$outLen = 0
$rc = [ODBC]::SQLDriverConnect($dbc, [IntPtr]::Zero, $conn, -3, $out, 1024, [ref]$outLen, 0)
Write-Host "rc=$rc"
Write-Host "out='$($out.ToString())'"

$state = New-Object System.Text.StringBuilder 8
$msg   = New-Object System.Text.StringBuilder 512
$native = 0
$msgLen = 0
$dr = [ODBC]::SQLGetDiagRec(2, $dbc, 1, $state, [ref]$native, $msg, 512, [ref]$msgLen)
Write-Host "dbc diag: rc=$dr state='$($state.ToString())' native=$native msg='$($msg.ToString())'"

# also try env
$state2 = New-Object System.Text.StringBuilder 8
$msg2   = New-Object System.Text.StringBuilder 512
$native2 = 0
$msgLen2 = 0
$dr2 = [ODBC]::SQLGetDiagRec(1, $env2, 1, $state2, [ref]$native2, $msg2, 512, [ref]$msgLen2)
Write-Host "env diag: rc=$dr2 state='$($state2.ToString())' native=$native2 msg='$($msg2.ToString())'"

[void][ODBC]::SQLFreeHandle(2, $dbc)
[void][ODBC]::SQLFreeHandle(1, $env2)
