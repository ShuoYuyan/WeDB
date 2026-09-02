# build.ps1 — Build the WeDB ODBC driver DLL for Windows.
#
# Requires:
#   * Go 1.23+
#   * A GCC toolchain reachable via the CC environment variable.
#     The Nuitka-bundled MinGW 14.2 works; MSYS2 GCC 14+ is also fine.
#     Visual Studio's cl.exe is NOT supported because cgo requires gcc.
#
# Output:
#   build\wedb_odbc.dll
#   build\wedb_odbc.lib   (import library; created by Go)

[CmdletBinding()]
param(
    [string]$OutDir = "build",
    [string]$Cc = $env:CC
)

$ErrorActionPreference = "Stop"

if (-not $Cc) {
    $candidates = @(
        "C:\msys64\mingw64\bin\gcc.exe",
        "C:\Program Files\Nuitka\Nuitka\Cache\downloads\gcc\x86_64\14.2.0posix-19.1.1-12.0.0-msvcrt-r2\mingw64\bin\gcc.exe"
    )
    foreach ($c in $candidates) {
        if (Test-Path $c) { $Cc = $c; break }
    }
}
if (-not $Cc) {
    throw "No GCC found. Set -Cc or the CC environment variable."
}
Write-Host "Using CC: $Cc"

$env:CC = $Cc
$env:CGO_ENABLED = "1"
$env:GOOS = "windows"
$env:GOARCH = "amd64"

if (-not (Test-Path $OutDir)) {
    New-Item -ItemType Directory -Path $OutDir | Out-Null
}

Push-Location $PSScriptRoot
try {
    go build -buildmode=c-shared -ldflags="-s -w" -o (Join-Path $OutDir "wedb_odbc.dll") .
} finally {
    Pop-Location
}

Write-Host "Built: $OutDir\wedb_odbc.dll"
Get-Item (Join-Path $OutDir "wedb_odbc.dll") | Select-Object Name, Length
