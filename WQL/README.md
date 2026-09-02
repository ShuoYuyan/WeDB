# WQL - WeDB Native Query Language

## 概述

WQL 是 WeDB 的**原生查询语言**，完全独立实现的查询计划器。
WQL 不生成任何 SQL 字符串 —— 它直接调用 WeDB 的 Go API（`storage.WeDBDatabase`）来执行查询。

## 设计原则

1. **零 SQL 字符串**：WQL 引擎从不生成或执行 SQL 语句
2. **直接调用存储 API**：通过 `Adapter` 接口直接调用 WeDB 的 Go 方法
3. **完全类型化**：使用 Go 类型系统表达所有表达式和操作
4. **可扩展**：未来可实现 `PostgreSQLAdapter`、`MySQLAdapter` 等

## 架构

```
┌─────────────────┐
│  CLI / REPL      │  cmd/wql/
│  (字符串接口)     │
└────────┬────────┘
         │ 字符串解析
         ▼
┌─────────────────┐
│  wqlv3.QueryBuilder │  pkg/wqlv3/wqlv3.go
│  (链式 Go API)     │  db.Table("users").Where("age > 18").All()
└────────┬────────┘
         │ Expression AST
         ▼
┌─────────────────┐
│  Expression      │  pkg/wqlv3/expression.go
│  (表达式求值)     │  ParseWhere() + EvalBoolExpr()
└────────┬────────┘
         │ Adapter interface
         ▼
┌─────────────────┐
│  WeDBAdapter     │  pkg/wqlv3/wedb_adapter.go
│  (WeDB 适配器)    │  直接调用 WeDB Go API
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  WeDB Storage    │  internal/storage/
│  (B-Tree + MVCC)  │
└─────────────────┘
```

## WQL 语法

WQL 提供**两种**使用方式：

### 方式 1: Go 链式 API（推荐用于 Go 程序）

```go
import "github.com/wedb/wedb/WQL/pkg/wqlv3"

db := storage.NewWeDBDatabase("test.db", 4096)
adapter := wqlv3.NewWeDBAdapter(db)
wdb := wqlv3.NewDatabase(adapter)

// 全表查询
rows, _ := wdb.Table("users").All()

// 条件查询
rows, _ := wdb.Table("users").Where("age > 18").All()

// 选择列
rows, _ := wdb.Table("users").Select("id", "name").Where("age > 18").All()

// 排序 + 分页
rows, _ := wdb.Table("users").OrderBy("age", "DESC").Skip(10).Take(20).All()

// 聚合
count, _ := wdb.Table("users").Where("age > 18").Count()
total, _ := wdb.Table("orders").Sum("amount")
avg, _ := wdb.Table("orders").Avg("amount")

// 第一行
row, _ := wdb.Table("users").Where("email = 'alice@x.com'").First()
```

### 方式 2: 字符串接口（用于 CLI / 配置文件 / 远程协议）

```go
result, err := wqlv3.EvaluateQuery(wdb, `T("users").Where("age > 18").All()`)
result, err := wqlv3.EvaluateQuery(wdb, `T("orders").Count()`)
```

## CLI 用法

### 启动 REPL
```cmd
> wql-cli test.db

 ╦ ╦╔═╗╦  ╔═╗
 ║║║║╣ ║  ╠═╣
 ╚╩╝╚═╝╩═╝╩ ╩ WeDB Native Query Language

 v0.1.0  •  backed by WeDB pure-Go storage engine
 type 'help' for commands, 'quit' to exit

  database: test.db
  backend:  wqlv3 + WeDB native Go storage

wql> help
Available commands:
  tables              - List all tables
  schema <table>      - Show table schema
  help                - Show this help
  quit / exit          - Exit the REPL

WQL Syntax (fluent API, no SQL strings):
  T(<table>)                 - Reference a table
  .Select(col1, col2, ...)   - Select columns
  .Where(condition)          - Filter rows
  .OrderBy(col, "ASC|DESC")  - Sort results
  .Take(n)                   - Limit rows
  .Skip(n)                   - Offset rows
  .All()                     - Execute and return []map[string]any
  .First()                   - Return first row
  .Count()                   - Count rows
  .Sum(col) / .Avg(col)      - Aggregations

wql> tables
  Found 1 table(s):
    - users

wql> T("users").All()
  id  name  age
  --  ----  ---
  1   alice 30
  2   bob   25
  3   carol 40

  3 row(s) in 0.123ms

wql> T("users").Where("age > 18").Count()
  3
  result: 3

wql> T("users").Sum("age")
  result: 95

wql> T("users").Where("name = 'alice'").First()
  id  name  age
  --  ----  ---
  1   alice 30

wql> quit
Bye!
```

### 单次查询模式
```cmd
> wql-cli test.db 'T("users").Count()'
> wql-cli test.db 'T("users").All()'
> wql-cli test.db 'T("users").Where("age > 18").All()'
```

## WHERE 子句语法

支持的运算符和表达式：

| 类型 | 语法 | 示例 |
|---|---|---|
| 等于 | `=` | `age = 18` |
| 不等于 | `!=` 或 `<>` | `status != "active"` |
| 大于 | `>` | `age > 18` |
| 大于等于 | `>=` | `age >= 18` |
| 小于 | `<` | `price < 100` |
| 小于等于 | `<=` | `price <= 100` |
| 逻辑与 | `AND` | `age > 18 AND age < 30` |
| 逻辑或 | `OR` | `status = "active" OR status = "pending"` |
| 逻辑非 | `NOT` | `NOT deleted` |
| IN | `IN (a, b, c)` | `id IN (1, 2, 3)` |
| LIKE | `LIKE "pattern"` | `name LIKE "a%"` (其中 % = 任意, _ = 单字符) |
| IS NULL | `IS NULL` | `deleted_at IS NULL` |
| IS NOT NULL | `IS NOT NULL` | `email IS NOT NULL` |

## 不依赖 SQL 的证明

WQL 的执行路径**完全绕过 SQL**：

| 传统 SQL 路径 | WQL 路径 |
|---|---|
| `SQLExecDirect("SELECT ...")` | `wqlv3.QueryBuilder.All()` |
| → 解析 SQL 字符串 | → 直接构造 Go 对象 |
| → 生成查询计划 | → 已经是查询计划 |
| → 调用 B-tree | → 通过 Adapter 直接调用 B-tree |

WQL 包**不导入** `database/sql`。验证方法：
```bash
$ grep -r "database/sql" pkg/wqlv3/
# 无输出
```

## 测试

```bash
cd WQL
go test ./pkg/wqlv3/
```

测试覆盖：
- WHERE 子句解析（=, !=, <, >, AND, OR, NOT, IN, IS NULL, LIKE）
- QueryBuilder 内存过滤
- 排序（ASC/DESC）
- 不生成 SQL 字符串的证明测试

## 已知限制

当前 wqlv3 实现的功能：
- ✅ SELECT（带列过滤、WHERE、ORDER BY、SKIP、TAKE）
- ✅ 聚合（Count, Sum, Avg, Min, Max）
- ✅ WHERE 过滤（=, !=, <, >, AND, OR, NOT, IN, LIKE, IS NULL, IS NOT NULL）
- ❌ INSERT / UPDATE / DELETE（待实现，需扩展 Adapter 接口）
- ❌ JOIN（待实现）
- ❌ 嵌套子查询（待实现）
- ❌ GROUP BY / HAVING（待实现）
- ❌ 窗口函数（待实现）

## 项目历史

WQL 经历了三个主要版本：
- **v1**（在 `_attic_pkg_wql/`）：最初实现，基于 SQLite 后端
- **v2**（在 `_attic_pkg_wqlv2/`）：重构尝试
- **v3**（当前在 `pkg/wqlv3/`）：完全重写，基于 WeDB 原生 Go API，**不依赖任何 SQL**

v3 是当前正式版本。v1 和 v2 已归档保留供参考。

## 文件结构

```
WQL/
├── go.mod                              # 模块定义
├── README.md                           # 本文件
├── cmd/
│   └── wql/
│       └── main.go                     # CLI 入口
├── pkg/
│   └── wqlv3/                          # WQL v3 正式版
│       ├── wqlv3.go                    # QueryBuilder (Fluent API)
│       ├── expression.go                # Expression AST + ParseWhere
│       ├── expression_test.go           # 单元测试
│       ├── wedb_adapter.go              # WeDB Adapter 实现
│       └── cli_helpers.go              # CLI 辅助函数
├── _attic_pkg_wql/                      # 归档: WQL v1 (SQLite 后端)
├── _attic_pkg_wqlv2/                   # 归档: WQL v2
├── _attic_examples/                    # 归档: 旧示例
├── _attic_tools/                        # 归档: 旧工具
├── _attic_verification/                 # 归档: 旧验证
├── _attic_wql-editor/                   # 归档: PyQt5 IDE
├── _attic_cmd_*/                       # 归档: 旧测试程序
```

## 许可

WQL 与 WeDB 一样，遵循 AGPL-3.0 协议。详见根目录的 `LICENSE` 文件。
