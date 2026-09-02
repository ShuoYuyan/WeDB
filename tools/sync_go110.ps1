# sync_go110.ps1 — 生成 WeDB 引擎的 Go 1.10 兼容镜像
#
# 源:   C:\CompanysPage\WeDB            (Go module, 现代工具链 C:\go)
# 目标: C:\CompanysPage\src\wedb        (GOPATH 平面包, 供 C:\go_1.10 编译)
#
# 转换规则:
#   1. import 路径  internal/* -> wedb/*
#   2. atomic.Int64/Uint64/Bool -> 本包垫片类型 AtomicInt64/... (方法名不变)
#   3. any -> interface{}
#   4. strings.Builder -> bytes.Buffer
#   5. os.ReadFile/WriteFile -> ioutil.*
#   6. %w -> %v ; 数字下划线字面量去除 ; errors.Is/As 手工处理点见输出提示
#
# 用法: powershell -ExecutionPolicy Bypass -File tools\sync_go110.ps1

$ErrorActionPreference = "Stop"
$SRC = "C:\CompanysPage\WeDB"
$DST = "C:\CompanysPage\src\wedb"

$PKGS = @(
    @{ from = "internal\api";     to = "api" },
    @{ from = "internal\types";   to = "types" },
    @{ from = "internal\util";    to = "util" },
    @{ from = "internal\storage"; to = "storage" },
    @{ from = "pkg\adapter";      to = "adapter" }
)

function Transform([string]$code) {
    # import 路径
    $code = $code -replace '"github\.com/wedb/wedb/internal/(api|types|util|storage)"', '"wedb/$1"'
    $code = $code -replace '"github\.com/wedb/wedb/pkg/adapter"', '"wedb/adapter"'

    # 原子类型 → 垫片(方法签名一致，调用点零修改)
    $code = $code -replace 'atomic\.Int64',  'AtomicInt64'
    $code = $code -replace 'atomic\.Uint64', 'AtomicUint64'
    $code = $code -replace 'atomic\.Int32',  'AtomicInt32'
    $code = $code -replace 'atomic\.Uint32', 'AtomicUint32'
    $code = $code -replace 'atomic\.Bool',   'AtomicBool'

    # any
    $code = [regex]::Replace($code, '\bany\b', 'interface{}')

    # 字符串构建器（保留原 strings 导入，另追加 bytes）
    if ($code.Contains('strings.Builder')) {
        $code = $code.Replace('strings.Builder', 'bytes.Buffer')
        if (-not $code.Contains('"bytes"')) {
            $code = $code.Replace([char]9 + '"strings"', [char]9 + '"bytes"' + [char]10 + [char]9 + '"strings"')
        }
    }

    # ioutil
    if ($code -match 'os\.(Read|Write)File') {
        $code = $code.Replace('os.ReadFile', 'ioutil.ReadFile').Replace('os.WriteFile', 'ioutil.WriteFile')
        if (-not $code.Contains('io/ioutil')) {
            $code = $code.Replace([char]9 + '"os"', [char]9 + '"os"' + [char]10 + [char]9 + '"io/ioutil"')
        }
    }

    # strings.ReplaceAll (Go1.12+) -> strings.Replace(...,-1)
    if ($code.Contains('strings.ReplaceAll')) {
        $code = [regex]::Replace($code, 'strings\.ReplaceAll\(([^;]+?), ([^;]+?), ([^;)]+?)\)', 'strings.Replace($1, $2, $3, -1)')
    }

    # 冗余的 errors.Is(os.ErrNotExist) 检查（前面已有 os.IsNotExist）
    $code = $code.Replace(' && !errors.Is(err, os.ErrNotExist)', '')

    if (-not $code.Contains("errors.")) {
        $code = [regex]::Replace($code, '\r?\n\t"errors"', '')
    }

    # 清理转换后未使用的导入
    if (-not $code.Contains('atomic.')) {
        $code = [regex]::Replace($code, '\r?\n\t"sync/atomic"', '')
    }
    if ($code -notmatch 'os\.') {
        $code = [regex]::Replace($code, '\r?\n\t"os"', '')
    }
    # 错误包装与数字字面量
    $code = $code -replace '%w', '%v'
    $code = [regex]::Replace($code, '(?<=\d)_(?=\d)', '')

    return $code
}

if (Test-Path $DST) { Remove-Item -LiteralPath $DST -Recurse -Force }
New-Item -ItemType Directory -Path (Join-Path $DST "storage") -Force | Out-Null
New-Item -ItemType Directory -Path (Join-Path $DST "api")    -Force | Out-Null
New-Item -ItemType Directory -Path (Join-Path $DST "types")  -Force | Out-Null
New-Item -ItemType Directory -Path (Join-Path $DST "util")   -Force | Out-Null

foreach ($p in $PKGS) {
    $fromDir = Join-Path $SRC $p.from
    $toDir   = Join-Path $DST $p.to
    New-Item -ItemType Directory -Path $toDir -Force | Out-Null
    Get-ChildItem -LiteralPath $fromDir -Filter *.go | ForEach-Object {
        $code = [System.IO.File]::ReadAllText($_.FullName, [System.Text.Encoding]::UTF8)
        $code = Transform $code
        [System.IO.File]::WriteAllText((Join-Path $toDir $_.Name), $code, (New-Object System.Text.UTF8Encoding($false)))
    }
    Write-Host ("synced {0} -> wedb/{1}" -f $p.from, $p.to)
}

# Go1.10 原子垫片（仅 storage 包用到原子类型）
Copy-Item (Join-Path $PSScriptRoot "go110_atomic_shim.go") (Join-Path $DST "storage") -Force
Write-Host "atomic shim installed into wedb/storage"

# CLI 工具 (cmd/wedb -> dst/cmd/wedb)
$cliSrc = Join-Path $SRC "cmd\wedb"
if (Test-Path $cliSrc) {
    $cliDst = Join-Path $DST "cmd\wedb"
    New-Item -ItemType Directory -Path $cliDst -Force | Out-Null
    Get-ChildItem -LiteralPath $cliSrc -Filter *.go | ForEach-Object {
        $code = [System.IO.File]::ReadAllText($_.FullName, [System.Text.Encoding]::UTF8)
        $code = Transform $code
        [System.IO.File]::WriteAllText((Join-Path $cliDst $_.Name), $code, (New-Object System.Text.UTF8Encoding($false)))
    }
    Write-Host "synced cmd\wedb -> wedb/cmd/wedb"
}

# errors.Is/As 需要人工确认的点
Select-String -Path (Join-Path $DST "*\*.go") -Pattern "errors\.Is|errors\.As" |
    ForEach-Object { Write-Host ("MANUAL CHECK: {0}:{1}" -f $_.Filename, $_.LineNumber) }
