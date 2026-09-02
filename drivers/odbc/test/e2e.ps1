# e2e.ps1 - Build and run the end-to-end test for wedb_odbc.dll.
param([string]$DbPath = "odbc_e2e.db", [switch]$Direct, [switch]$Manager, [switch]$KeepDb)
if (-not $Direct -and -not $Manager) { $Direct = $true; $Manager = $true }
$ErrorActionPreference = "Stop"
$vcvars = "C:\Program Files (x86)\Microsoft Visual Studio\2019\Community\VC\Auxiliary\Build\vcvars64.bat"
if (-not (Test-Path $vcvars)) { throw "vcvars64.bat not found" }
$dllSrc = Join-Path $PSScriptRoot "..\build\wedb_odbc.dll"
if (-not (Test-Path $dllSrc)) { throw "DLL not found" }
Copy-Item -Force $dllSrc (Join-Path $PSScriptRoot "wedb_odbc.dll")
Get-Process | Where-Object { $_.ProcessName -eq "odbc_e2e" -or $_.ProcessName -eq "direct" } | Stop-Process -Force -ErrorAction SilentlyContinue
Start-Sleep -Milliseconds 500
if (-not $KeepDb) { Remove-Item $DbPath -ErrorAction SilentlyContinue; Remove-Item "$DbPath.metadata" -ErrorAction SilentlyContinue }
Push-Location $PSScriptRoot
$ok = $true

# Helper: run a Windows command via Start-Process. This avoids the
# PS 5.1 string-escape issues with `>` and `&&` inside cmd /c.
function Run-Process {
    param([string]$Exe, [string]$ExeArgs, [string]$Cwd)
    $log = Join-Path $env:TEMP ("wedb_" + [System.Guid]::NewGuid().ToString("N") + ".log")
    $p = Start-Process -FilePath $Exe -ArgumentList $ExeArgs -WorkingDirectory $Cwd `
        -RedirectStandardOutput $log `
        -NoNewWindow -PassThru -Wait
    $code = $p.ExitCode
    $out = Get-Content $log -Raw
    Remove-Item $log -ErrorAction SilentlyContinue
    return @{ Code = $code; Output = $out }
}

# Helper: run a cmd.exe with /c and a single string argument.
# Stderr is merged into stdout via `2>&1` in the command.
function Run-Cmd {
    param([string]$CmdString)
    $log = Join-Path $env:TEMP ("wedb_" + [System.Guid]::NewGuid().ToString("N") + ".log")
    $p = Start-Process -FilePath "cmd.exe" -ArgumentList "/c `"$CmdString 2>&1`"" `
        -RedirectStandardOutput $log `
        -NoNewWindow -PassThru -Wait
    $code = $p.ExitCode
    $out = Get-Content $log -Raw
    Remove-Item $log -ErrorAction SilentlyContinue
    return @{ Code = $code; Output = $out }
}

if ($Direct) {
    Write-Host "== Compiling direct.c =="
    $r = Run-Cmd ("call `"$vcvars`" >NUL 2>&1 && cl /nologo /EHsc /W3 direct.c /Fe:direct.exe")
    if ($r.Code -ne 0) { Write-Warning ("cl direct.c failed " + $r.Output) ; $ok = $false }
    else {
        Write-Host "== Running direct.exe =="
        $r = Run-Process ".\direct.exe" ('"' + $DbPath + '"') $PSScriptRoot
        if ($r.Code -ne 0) { Write-Host $r.Output; Write-Warning "direct failed $($r.Code)" ; $ok = $false }
        else { Write-Host $r.Output }
    }
}

if ($Manager) {
    $isAdmin = ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
    if (-not $isAdmin) { Write-Warning "Not admin; manager test requires admin. Skipping." }
    else {
        $instBase = "HKLM:\SOFTWARE\ODBC\ODBCINST.INI"
        $dsnBase = "HKLM:\SOFTWARE\ODBC\ODBC.INI"
        $driverName = "WeDB ODBC Driver"
        $dsnName = "WeDB Sample"
        $system32 = "$env:SystemRoot\System32"
        $targetDll = Join-Path $system32 "wedb_odbc.dll"
        $restoreDll = $false
        if ((Test-Path $targetDll) -and ((Get-Item $dllSrc).LastWriteTime -gt (Get-Item $targetDll).LastWriteTime)) {
            Remove-Item $targetDll -Force -ErrorAction SilentlyContinue
        }
        if (-not (Test-Path $targetDll)) { Copy-Item $dllSrc $targetDll; $restoreDll = $true }
        if (-not (Test-Path "$instBase\ODBC Drivers")) { New-Item -Path "$instBase\ODBC Drivers" -Force | Out-Null }
        Set-ItemProperty -Path "$instBase\ODBC Drivers" -Name $driverName -Value "Installed" -Force
        if (-not (Test-Path "$instBase\$driverName")) { New-Item -Path "$instBase\$driverName" -Force | Out-Null }
        $dllTarget = if (Test-Path $targetDll) { $targetDll } else { $dllSrc }
        Set-ItemProperty -Path "$instBase\$driverName" -Name "Driver" -Value $dllTarget -Force
        Set-ItemProperty -Path "$instBase\$driverName" -Name "DriverODBCVer" -Value "03.00" -Force
        Set-ItemProperty -Path "$instBase\$driverName" -Name "FileExtns" -Value "*" -Force
        Set-ItemProperty -Path "$instBase\$driverName" -Name "FileUsage" -Value "1" -Force
        if (-not (Test-Path "$dsnBase\ODBC Data Sources")) { New-Item -Path "$dsnBase\ODBC Data Sources" -Force | Out-Null }
        Set-ItemProperty -Path "$dsnBase\ODBC Data Sources" -Name $dsnName -Value $driverName -Force
        if (-not (Test-Path "$dsnBase\$dsnName")) { New-Item -Path "$dsnBase\$dsnName" -Force | Out-Null }
        Set-ItemProperty -Path "$dsnBase\$dsnName" -Name "Driver" -Value $driverName -Force
        Set-ItemProperty -Path "$dsnBase\$dsnName" -Name "DBQ" -Value (Join-Path $PSScriptRoot $DbPath) -Force
        if (-not (Test-Path "C:\temp")) { New-Item -Path "C:\temp" -ItemType Directory -Force | Out-Null }
        if (-not (Test-Path "$dsnBase\ODBC")) { New-Item -Path "$dsnBase\ODBC" -Force | Out-Null }
        Set-ItemProperty -Path "$dsnBase\ODBC" -Name "Trace" -Value "1" -Force
        Set-ItemProperty -Path "$dsnBase\ODBC" -Name "TraceFile" -Value "C:\temp\odbctrace.log" -Force

        Write-Host "== Compiling odbc_e2e.c =="
        $r = Run-Cmd ("call `"$vcvars`" >NUL 2>&1 && cl /nologo /EHsc /W3 /DUNICODE /D_UNICODE odbc_e2e.c odbc32.lib /Fe:odbc_e2e.exe")
        if ($r.Code -ne 0) { Write-Warning ("cl odbc_e2e.c failed " + $r.Output) ; $ok = $false } else {
            Write-Host "== Running odbc_e2e.exe =="
            $r = Run-Process ".\odbc_e2e.exe" ('"' + $DbPath + '"') $PSScriptRoot
            if ($r.Code -ne 0) { Write-Host $r.Output; Write-Warning "odbc_e2e failed $($r.Code)" ; $ok = $false }
            else { Write-Host $r.Output }
        }

        Remove-ItemProperty -Path "$instBase\ODBC Drivers" -Name $driverName -ErrorAction SilentlyContinue
        Remove-Item -Path "$instBase\$driverName" -Recurse -Force -ErrorAction SilentlyContinue
        Remove-ItemProperty -Path "$dsnBase\ODBC Data Sources" -Name $dsnName -ErrorAction SilentlyContinue
        Remove-Item -Path "$dsnBase\$dsnName" -Recurse -Force -ErrorAction SilentlyContinue
        Remove-ItemProperty -Path "$dsnBase\ODBC" -Name "Trace" -ErrorAction SilentlyContinue
        Remove-ItemProperty -Path "$dsnBase\ODBC" -Name "TraceFile" -ErrorAction SilentlyContinue
        if ($restoreDll) { Remove-Item $targetDll -ErrorAction SilentlyContinue }
    }
}

Pop-Location
if (-not $KeepDb) { Remove-Item $DbPath -ErrorAction SilentlyContinue; Remove-Item "$DbPath.metadata" -ErrorAction SilentlyContinue }
if ($ok) { Write-Host "== E2E PASS ==" } else { Write-Host "== E2E DONE (with failures) ==" }
