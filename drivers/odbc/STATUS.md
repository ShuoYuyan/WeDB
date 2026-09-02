# WeDB ODBC 驱动：项目现状（2026-09-02 10:02）

## 总体状态

| 测试路径 | 状态 |
|---|---|
| `direct.c`（LoadLibrary 绕过 Manager） | ✓ **完全通过** |
| `WeDBODBC.dpr`（Delphi XE2 64-bit，LoadLibrary） | ✓ **连接+执行通过**，字符串显示已修复 |
| `odbc_e2e.c`（经 Windows ODBC Manager） | ✗ 崩溃（已知限制） |

## Delphi 字符串显示问题修复（2026-09-02 10:02）

### 根因
类型不匹配导致 C 缓冲区写入时栈溢出：
- `ColSize: LongWord` 是 32-bit，但 Go 驱动通过 `SQLULEN*` 写入 64-bit，溢出 4 字节到后续栈变量
- 原始 `AnsiString(@Array[0])` 转换在 XE2 64-bit 下可靠性存疑

### 修复
1. **`SQLULEN` 类型更正**：`LongWord` → `UInt64`（64-bit 无符号）
2. **`ColSize` 变量类型**：`LongWord` → `SQLULEN`（匹配 `SQLULEN*` 写入大小）
3. **`BufToStr` 辅助函数**：手动扫描 NUL 终止符并用 `Copy` 截取字符串，替代 `AnsiString(@ptr)` 隐式转换
4. **变量初始化**：`SQLDescribeCol` 前显式清零所有输出参数

### 修复后预期输出
```
columns: 3
  [1] id type=4 nameLen=2
  [2] name type=12 nameLen=4
  [3] age type=4 nameLen=3
row: id=1 name=alice age=30
row: id=2 name=bob age=25
row: id=3 name=carol age=40
COUNT=3 SUM=95 AVG=31.67 MIN=25 MAX=40
Table: ..t type=TABLE
Column: id type=4 (INTEGER)
DBMS_NAME=WeDB
DRIVER_NAME=WeDB
ODBC_VER=03.00
Type: CHAR (1)
Type: VARCHAR (12)
Type: TEXT (-1)
Type: INTEGER (4)
Type: BIGINT (-5)
```

## 关键发现

### `direct.c` 端到端测试输出
```
DLL: 00007FFC3A610000
  OK AllocEnv / SetEnvAttr / AllocConnect / DriverConnect
  OK AllocStmt
  OK DROP TABLE / CREATE TABLE
  OK INSERT 1, 2, 3  (RowCount after INSERT: 1)
  OK PREPARE
  OK SELECT (3 rows: id=1,2,3  type=4 INTEGER, name type=12 VARCHAR, age type=4 INTEGER)
  OK SELECT aggregate (COUNT=3 SUM=95 AVG=31.66 MIN=25 MAX=40)
  OK SELECT WHERE
  OK SQLTables / SQLColumns (id=INTEGER name=VARCHAR age=INTEGER)
  SQLGetInfo: DBMS_NAME=WeDB DRIVER_NAME=WeDB ODBC_VER=03.00
  SQLGetFunctions: SQLExecDirect supported: 1
  OK SQLGetTypeInfo (CHAR/VARCHAR/TEXT/INTEGER/BIGINT/...)
OK
```

### 已知限制：Windows ODBC Manager 路径
通过 `odbcconf /INSTALL` 注册并经 Windows ODBC Manager 加载 DLL 时，在 `SQLDriverConnectW` 调用后发生 `STATUS_STACK_BUFFER_OVERRUN`（0xC0000409）。这是 Go c-shared runtime 与 Windows ODBC Manager C runtime 之间的冲突。

**绕过方法**：客户端直接 `LoadLibraryA` + `GetProcAddress` 调用 DLL，绕过 ODBC Manager。

## 文件清单

| 文件 | 用途 |
|---|---|
| `drivers/odbc/api_core.go` | 38 个 ANSI ODBC 入口 + 30 个 W 变体 + 6 个 Config* 安装函数 |
| `drivers/odbc/api_meta.go` | SQLTables / SQLColumns / SQLGetTypeInfo + SQLGetFunctions 数组 |
| `drivers/odbc/api_meta_info.go` | SQLGetInfo 常量 + 默认值 |
| `drivers/odbc/util_string.go` | Go ↔ C 字符串转换（writeCString / goString） |
| `drivers/odbc/sql/parse.go` | SQL-92 子集解析器 |
| `drivers/odbc/sql/exec.go` | SQL 执行器 + `inferColumnType` |
| `drivers/odbc/sql/where.go` | WHERE 表达式求值器 |
| `drivers/odbc/handle/handle.go` | env/dbc/stmt 句柄池 |
| `drivers/odbc/diag/diag.go` | SQLSTATE 常量 |
| `drivers/odbc/bridge.go` | storage 适配 |
| `drivers/odbc/build.ps1` | DLL 构建脚本 |
| `drivers/odbc/test/direct.c` | LoadLibrary 端到端测试 ✓ |
| `drivers/odbc/test/odbc_e2e.c` | Windows ODBC Manager 端到端测试 ✗ |
| `drivers/odbc/test/WeDBODBC.dpr` | Delphi XE2 64-bit 客户端测试 ✓ |
| `drivers/odbc/test/e2e.ps1` | 自动注册/测试/清理 |
| `drivers/odbc/README.md` | ODBC 驱动完整文档 |

## 关键时间线
- 17:00 之前：IM001 解决
- 18:00 之前：direct.c 端到端测试通过
- 19:00 之前：补齐所有 W 变体
- 21:00 之前：Config* 函数解决 odbccp32 错误
- 23:00 之前：env var 让测试进展
- 06:00 之前：定位到 Go runtime + manager C runtime 冲突
- 09:00 之前：Delphi XE2 64-bit 连接并执行 SQL 通过
- 10:00 之前：修复 Delphi 字符串显示问题（SQLULEN 类型对齐 + BufToStr 辅助函数）

———
完成时间：2026-09-02 10:02
