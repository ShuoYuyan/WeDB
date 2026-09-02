# WeDB ODBC 驱动

## 概述

WeDB ODBC 驱动是一个用 Go 编写、通过 CGO 导出为 Windows DLL 的 ODBC 3.x 兼容驱动。它允许任何支持 ODBC 的客户端应用程序（C/C++、Delphi、Java、Python、.NET 等）通过标准 ODBC API 访问 WeDB 数据库。

## 架构

```
┌─────────────────────┐
│   ODBC Client App   │  (C / C++ / Delphi / Python / Java / .NET)
│  (loads DLL via     │
│   LoadLibrary or    │
│   ODBC Manager)     │
└─────────┬───────────┘
          │  stdcall / cdecl
          ▼
┌─────────────────────┐
│   wedb_odbc.dll     │  Windows DLL (64-bit)
│  (Go + CGO exports) │
├─────────────────────┤
│   CGO Bridge Layer  │  api_core.go, api_meta.go, api_meta_info.go
│  (cgo exported      │
│   SQL* functions)   │
├─────────────────────┤
│   SQL Parser        │  sql/parse.go (SQL-92 subset)
│   SQL Executor      │  sql/exec.go (DML/DDL/SELECT)
│   WHERE Evaluator   │  sql/where.go
├─────────────────────┤
│   Handle Pool       │  handle/handle.go (env/dbc/stmt)
│   Diagnostics       │  diag/diag.go
├─────────────────────┤
│   Storage Engine    │  storage/ (WeDB native Go DB)
└─────────────────────┘
```

## 构建

### 前提条件
- Go 1.23 或更高版本
- GCC 工具链（MinGW-w64 / MSYS2 / Nuitka-bundled GCC）
- CGO_ENABLED=1

### 构建命令
```powershell
cd drivers/odbc
.\build.ps1
```

输出文件：
- `build\wedb_odbc.dll` — 主 DLL（64-bit Windows）
- `build\wedb_odbc.lib` — 导入库（静态链接用）
- `build\wedb_odbc.h` — C 头文件

### 32-bit 构建（用于 Delphi 7 等 32-bit 客户端）
需要 32-bit GCC 工具链：
```powershell
$env:GOARCH = "386"
$env:CGO_ENABLED = "1"
$env:CC = "i686-w64-mingw32-gcc"
go build -buildmode=c-shared -o build\wedb_odbc_32.dll .
```

## 导出的 ODBC 函数

驱动导出完整的 ODBC 3.x ANSI 和 Unicode（W）函数集：

| 函数 | ANSI | Unicode | 用途 |
|---|---|---|---|
| `SQLAllocHandle` | ✓ | ✓ | 分配 env/dbc/stmt 句柄 |
| `SQLFreeHandle` | ✓ | ✓ | 释放句柄 |
| `SQLSetEnvAttr` | ✓ | ✓ | 设置环境属性 |
| `SQLDriverConnect` | ✓ | ✓ | 连接数据库（支持连接字符串） |
| `SQLDisconnect` | ✓ | ✓ | 断开连接 |
| `SQLAllocStmt` | ✓ | — | 分配语句句柄（ODBC 2.x 兼容） |
| `SQLFreeStmt` | ✓ | ✓ | 释放/关闭语句 |
| `SQLExecDirect` | ✓ | ✓ | 直接执行 SQL |
| `SQLPrepare` | ✓ | ✓ | 预编译 SQL |
| `SQLExecute` | ✓ | ✓ | 执行预编译 SQL |
| `SQLFetch` | ✓ | ✓ | 取下一行 |
| `SQLGetData` | ✓ | ✓ | 获取列数据 |
| `SQLRowCount` | ✓ | ✓ | 受影响行数 |
| `SQLNumResultCols` | ✓ | ✓ | 结果集列数 |
| `SQLDescribeCol` | ✓ | ✓ | 描述列元数据 |
| `SQLTables` | ✓ | ✓ | 列举表 |
| `SQLColumns` | ✓ | ✓ | 列举列 |
| `SQLGetTypeInfo` | ✓ | ✓ | 数据类型信息 |
| `SQLGetInfo` | ✓ | ✓ | 连接级信息 |
| `SQLGetFunctions` | ✓ | ✓ | 函数支持查询 |
| `SQLGetDiagRec` | ✓ | ✓ | 诊断记录 |
| `SQLNumParams` | ✓ | ✓ | 参数数量 |
| `SQLBindParameter` | ✓ | ✓ | 绑定参数 |
| `SQLCancel` | ✓ | ✓ | 取消执行 |
| `ConfigDSN` / `ConfigDSNW` | ✓ | ✓ | DSN 配置（安装时用） |
| `ConfigDriver` / `ConfigDriverW` | ✓ | ✓ | 驱动配置 |
| `ConfigTranslator` | ✓ | ✓ | 转换器配置 |

## 连接字符串

```
DRIVER={WeDB ODBC Driver};DBQ=<path-to-db-file>;
```

- `DRIVER` 必须是 `WeDB ODBC Driver`（在注册表中注册的名字）
- `DBQ` 是 WeDB 数据库文件的完整路径（`.db`）

### 注册驱动和 DSN
```cmd
reg add "HKLM\SOFTWARE\ODBC\ODBCINST.INI\WeDB ODBC Driver" /v Driver /d "C:\path\to\wedb_odbc.dll" /f
reg add "HKLM\SOFTWARE\ODBC\ODBC.INI\WeDB Sample" /v Driver /d "WeDB ODBC Driver" /f
reg add "HKLM\SOFTWARE\ODBC\ODBC.INI\WeDB Sample" /v DBQ /d "C:\data\test.db" /f
```

## 支持的 SQL 子集

### DDL
- `CREATE TABLE`（INTEGER PRIMARY KEY, TEXT, INTEGER, REAL）
- `DROP TABLE`
- `DROP TABLE IF EXISTS`

### DML
- `INSERT INTO ... VALUES (...)`
- `SELECT`（`SELECT col, col FROM table WHERE ... ORDER BY col LIMIT n OFFSET k`）
- 聚合函数：`COUNT(*)`, `SUM`, `AVG`, `MIN`, `MAX`

### 类型映射
| WeDB 内部类型 | ODBC SQL 类型 | C 类型 |
|---|---|---|
| int64 | `SQL_INTEGER` (4) | `SQL_C_LONG` |
| float64 | `SQL_DOUBLE` (8) | `SQL_C_DOUBLE` |
| string | `SQL_VARCHAR` (12) | `SQL_C_CHAR` |
| []byte | `SQL_LONGVARBINARY` (-4) | `SQL_C_BINARY` |
| bool | `SQL_BIT` (-7) | `SQL_C_BIT` |

## 句柄管理

驱动内部使用小型句柄池（从 0x10000 开始的递增整数）。所有句柄通过 `uintptr` 值在 C 和 Go 之间传递。

```
环境句柄 (env)  →  连接句柄 (dbc)  →  语句句柄 (stmt)
  0x10000          0x10001             0x10002
```

## 测试

### 端到端测试

项目包含两个测试程序：

#### `direct.c` — 绕过 ODBC Manager 的直接调用测试
```cmd
cd drivers\odbc\test
cl /EHsc /W3 direct.c /Fe:direct.exe
direct.exe C:\data\test.db
```
直接通过 `LoadLibraryA` + `GetProcAddress` 调用 DLL，不经过 Windows ODBC Manager。

#### `WeDBODBC.dpr` — Delphi 7 64-bit 客户端测试
用 Delphi XE2 或更高版本打开 `WeDBODBC.dpr`，编译为 64-bit Windows 目标，运行：
```cmd
WeDBODBC.exe C:\data\test.db
```

### 测试覆盖
- ✓ AllocEnv / SetEnvAttr / AllocConnect
- ✓ DriverConnect（连接字符串解析）
- ✓ AllocStmt / SQLExecDirect
- ✓ CREATE TABLE / DROP TABLE
- ✓ INSERT 多行
- ✓ PREPARE + EXECUTE
- ✓ SELECT（列描述 + 数据获取）
- ✓ 聚合查询（COUNT, SUM, AVG, MIN, MAX）
- ✓ WHERE 子句过滤
- ✓ SQLTables / SQLColumns
- ✓ SQLGetTypeInfo
- ✓ SQLGetInfo
- ✓ SQLGetFunctions

## 已知问题与限制

### 1. Windows ODBC Manager 路径崩溃
通过 `odbcconf /INSTALL` 注册并经 Windows ODBC Manager 加载 DLL 时，在 `SQLDriverConnectW` 调用后会发生 `STATUS_STACK_BUFFER_OVERRUN` (0xC0000409)。这是 Go c-shared runtime 与 Windows ODBC Manager C runtime 之间的已知冲突。

**解决方案**：使用 `direct.c` 或 `WeDBODBC.dpr` 中的 `LoadLibraryA` 方式直接加载 DLL，绕过 ODBC Manager。

### 2. 64-bit 类型对齐
在 x64 平台上，`SQLULEN` 和 `SQLLEN` 是 64-bit。客户端代码必须使用 `uint64_t` / `int64_t` 而非 `unsigned long` / `long` 来接收这些类型的输出参数。

### 3. 参数绑定
`SQLBindParameter` 已实现基本支持，但带 `?` 占位符的预编译语句的完整参数替换流程仍在完善中。

## 文件结构

```
drivers/odbc/
├── api_core.go            # SQL 核心函数实现（30+ 入口）
├── api_meta.go            # SQLTables / SQLColumns / SQLGetTypeInfo
├── api_meta_info.go       # SQLGetInfo 常量与默认值
├── column.go              # 列元数据辅助
├── bridge.go              # storage 适配
├── bridge_sql.go          # SQL 层与 storage 桥接
├── handle/
│   ├── handle.go          # env/dbc/stmt 句柄池
│   └── accessors.go       # 句柄字段访问器
├── sql/
│   ├── parse.go           # SQL-92 子集解析器
│   ├── exec.go            # SQL 执行器
│   └── where.go           # WHERE 表达式求值器
├── diag/
│   └── diag.go            # SQLSTATE 常量
├── util_string.go         # Go ↔ C 字符串转换
├── registry_windows.go    # Windows 注册表写入
├── registry_other.go      # 其他平台占位
├── main.go                # 入口
├── build.ps1              # 构建脚本
├── wedb_odbc.h            # CGO 生成的头文件
├── wedb_odbc.dll          # 编译产物
└── test/
    ├── direct.c           # 绕过 Manager 的端到端测试
    ├── odbc_e2e.c         # 经 Manager 的端到端测试
    ├── e2e.ps1            # 自动化测试脚本
    ├── WeDBODBC.dpr       # Delphi 7 64-bit 客户端测试
    └── check_exports.c    # 导出符号检查
```

## 调试技巧

### 启用 ODBC Trace
```cmd
reg add "HKLM\SOFTWARE\ODBC\ODBC.INI\ODBC" /v Trace /t REG_SZ /d 1 /f
reg add "HKLM\SOFTWARE\ODBC\ODBC.INI\ODBC" /v TraceFile /t REG_SZ /d "C:\temp\odbctrace.log" /f
```

### 查看 DLL 导出
```cmd
objdump -p wedb_odbc.dll | findstr SQL
```

### 环境变量
- `WEDB_DEBUG=1` — 启用 Go 侧调试日志

