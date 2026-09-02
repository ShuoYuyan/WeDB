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

### 方式 2: WQL 无双引号字符串（推荐 — 真正的 WQL 语法）

WQL 的设计原则是**无双引号**：标识符（表名、列名）不需要引号。

```go
result, err := wqlv3.EvaluateQueryNoQuotes(wdb, `db.Table(users).Where(age > 18).All()`)
result, err := wqlv3.EvaluateQueryNoQuotes(wdb, `db.Table(orders).Select(city, Count()).GroupBy(city).All()`)
result, err := wqlv3.EvaluateQueryNoQuotes(wdb, `db.Table(users).Where(name = "alice").First()`)
```

**语法要点**:
- 表名、列名**不加引号**：`db.Table(users)`, `Select(name, age)`
- 字符串值需要引号：`name = "alice"`
- 数字不加引号：`age > 18`
- 操作链：`db.Table(t).Select(...).Where(...).OrderBy(..., DESC).Take(N).All()`

### 方式 3: 旧字符串接口（向后兼容，带双引号）

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

wql> db.Table(users).All()
  id  name  age
  --  ----  ---
  1   alice 30
  2   bob   25
  3   carol 40

  3 row(s) in 0.123ms

wql> db.Table(users).Select(name, age).Where(age > 18).OrderBy(age, DESC).Take(2).All()
  name  age
  ----  ---
  carol 40
  alice 30

wql> db.Table(users).Where(name = "alice").First()
  id  name  age
  --  ----  ---
  1   alice 30

wql> quit
Bye!
```

> **设计原则**: WQL 使用**无双引号**语法 — 标识符（表名、列名）不需要引号，只有字符串值才需要。
> 例: `db.Table(users).Select(name, age).Where(name = "alice").All()`
> 而**不是**: `db.Table("users").Select("name", "age").Where("name = \"alice\"").All()`

### 单次查询模式
```cmd
> wql-cli test.db 'T("users").Count()'
> wql-cli test.db 'T("users").All()'
> wql-cli test.db 'T("users").Where("age > 18").All()'
```

## DML / DDL 无双引号语法

WQL v3 支持完整的 DML/DDL 链式语法，全部无需双引号：

### CREATE TABLE / DROP TABLE

```go
// 创建表
_, err := wqlv3.EvaluateQueryNoQuotes(wdb, `
    db.Table(products).Create(id INTEGER PRIMARY KEY, name TEXT, price REAL).Execute()
`)

// 删除表
_, err := wqlv3.EvaluateQueryNoQuotes(wdb, `db.Table(temp).Drop().Execute()`)
```

支持类型：`INTEGER`, `TEXT`, `REAL`, `BLOB`（`INT`/`VARCHAR`/`FLOAT`/`DOUBLE` 也可识别）
支持约束：`PRIMARY KEY`, `NOT NULL`, `NULL`

### INSERT

```go
// 对象字面量形式
_, err := wqlv3.EvaluateQueryNoQuotes(wdb, `
    db.Table(users).Insert({id: 1, name: "alice", age: 30}).Execute()
`)

// 列值对形式
_, err := wqlv3.EvaluateQueryNoQuotes(wdb, `
    db.Table(users).Insert(id, 2, name, "bob", age, 25).Execute()
`)

// 多行
_, err := wqlv3.EvaluateQueryNoQuotes(wdb, `
    db.Table(users).Insert(
        {id: 1, name: "alice", age: 30},
        {id: 2, name: "bob", age: 25}
    ).Execute()
`)
```

### UPDATE (Set + Where)

```go
_, err := wqlv3.EvaluateQueryNoQuotes(wdb, `
    db.Table(users).Set(age, 31).Where(id = 1).Execute()
`)
```

### DELETE (Where + Delete)

```go
_, err := wqlv3.EvaluateQueryNoQuotes(wdb, `
    db.Table(users).Where(age < 18).Delete().Execute()
`)
```

### 完整生命周期示例

```go
// 1. 建表
wqlv3.EvaluateQueryNoQuotes(wdb, `db.Table(orders).Create(
    id INTEGER PRIMARY KEY, product TEXT, qty INTEGER
).Execute()`)

// 2. 插数据
wqlv3.EvaluateQueryNoQuotes(wdb, `db.Table(orders).Insert(
    {id: 1, product: "apple", qty: 10},
    {id: 2, product: "banana", qty: 20}
).Execute()`)

// 3. 查
res, _ := wqlv3.EvaluateQueryNoQuotes(wdb, `db.Table(orders).All()`)

// 4. 改
wqlv3.EvaluateQueryNoQuotes(wdb, `db.Table(orders).Set(qty, 100).Where(product = "apple").Execute()`)

// 5. 删
wqlv3.EvaluateQueryNoQuotes(wdb, `db.Table(orders).Where(qty < 50).Delete().Execute()`)

// 6. 删表
wqlv3.EvaluateQueryNoQuotes(wdb, `db.Table(orders).Drop().Execute()`)
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
- ✅ INSERT / UPDATE / DELETE（DML 完整支持）
- ✅ CREATE TABLE / DROP TABLE（DDL 基础支持）
  - ✅ 聚合（Count, Sum, Avg, Min, Max — 通过 Go API）
  - ✅ WHERE 过滤（=, !=, <, >, AND, OR, NOT, IN, LIKE, IS NULL, IS NOT NULL）
  - ✅ WQL 无双引号解析器（lexer + parser + AST）
  - ✅ DML: Insert / Update (Set+Where) / Delete (Where+Delete)
  - ✅ DDL: CreateTable / DropTable
  - ✅ 对象字面量 `{col: val, ...}` 用于 Insert/Set
  - ❌ JOIN（parser 暂不支持）
  - ❌ GROUP BY / HAVING（parser 暂不支持）
- ❌ 嵌套子查询（parser 暂不支持）
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
