# WQL v3 - Cookbook of Common Patterns

This document shows real WQL v3 queries you can paste into the REPL
(`wql-cli demo.db`), the demo program (`go run ./cmd/wql-demo`), or
embed in Go code.

WQL v3 is a **fluent, no-quotes** Go API. Identifiers (table names, column
names) are written bare. String values still need quotes.

## Setup

```go
import (
    "github.com/wedb/wedb/WQL/pkg/wqlv3"
    "github.com/wedb/wedb/internal/storage"
)

wedb, _ := storage.NewWeDBDatabase("demo.db", 4096)
defer wedb.Close()
adapter := wqlv3.NewWeDBAdapter(wedb)
db := wqlv3.NewDatabase(adapter)
```

Or in the REPL:

```bash
$ wql-cli demo.db
wql> db.Table(users).Create(id INTEGER PRIMARY KEY, name TEXT, age INTEGER).Execute()
```

## DDL — Data Definition

### Create a table

```wql
db.Table(users).Create(id INTEGER PRIMARY KEY, name TEXT, age INTEGER, dept TEXT).Execute()
```

### Create with NOT NULL constraints

```wql
db.Table(orders).Create(
    id INTEGER PRIMARY KEY,
    user_id INTEGER NOT NULL,
    amount REAL NOT NULL,
    product TEXT
).Execute()
```

### Drop a table

```wql
db.Table(old_logs).Drop().Execute()
```

## DML — Data Manipulation

### Insert a single row

```wql
db.Table(users).Insert({id: 1, name: "alice", age: 30, dept: "Engineering"}).Execute()
```

### Insert multiple rows

```wql
db.Table(users).Insert({id: 2, name: "bob",   age: 25, dept: "Sales"}).Execute()
db.Table(users).Insert({id: 3, name: "carol", age: 35, dept: "Engineering"}).Execute()
```

### Update rows

```wql
-- give alice a 5-year raise
db.Table(users).Set(age, 35).Where(name = "alice").Execute()

-- batch update with object literal
db.Table(users).Sets({age: 40, dept: "Staff"}).Where(dept = "Engineering").Execute()
```

### Delete rows

```wql
-- remove users under 18
db.Table(users).Where(age < 18).Delete().Execute()

-- delete by id
db.Table(orders).Where(id = 100).Delete().Execute()
```

## SELECT — Queries

### Basic projection

```wql
-- all columns, first 5 rows
db.Table(users).Take(5).All()

-- specific columns
db.Table(users).Select(id, name, age).All()

-- ordering + limit
db.Table(users).OrderBy(age, DESC).Take(3).Select(name, age).All()
```

### WHERE — filtering

```wql
-- simple predicate
db.Table(users).Where(age >= 25).Select(name, age).All()

-- compound predicate (AND / OR)
db.Table(users).Where(age > 20 AND dept = "Sales").Select(name, age).All()
db.Table(users).Where(age < 18 OR age > 65).Select(name, age).All()
db.Table(users).Where(NOT(dept = "Sales")).Select(name, age).All()
```

Supported operators: `=`, `!=`, `<`, `<=`, `>`, `>=`, `AND`, `OR`, `NOT`.

### ORDER BY + LIMIT + OFFSET

```wql
-- top 3 oldest, skip the 2 oldest (paging)
db.Table(users).OrderBy(age, DESC).Skip(2).Take(3).Select(name, age).All()
```

### Aggregates

```wql
-- count
db.Table(users).Count()
db.Table(users).Where(age > 18).Count()

-- terminal aggregates (return single value)
db.Table(orders).Sum(amount)
db.Table(orders).Avg(amount)
db.Table(orders).Min(amount)
db.Table(orders).Max(amount)

-- in-line aggregates inside SELECT
db.Table(users).Select(Avg(age)).All()
```

### GROUP BY + HAVING

```wql
-- total amount per user, keep only those over 100
db.Table(orders).GroupBy(user_id).Select(user_id, Sum(amount)).Having(Sum(amount) > 100).All()

-- count users per department
db.Table(users).GroupBy(dept).Select(dept, Count() AS cnt).All()
```

### JOIN

```wql
-- INNER JOIN (only matching rows)
db.Table(orders).Join(users, ON orders.user_id = users.id).Select(users.name, orders.amount).OrderBy(orders.amount, DESC).All()

-- LEFT JOIN (preserve left side even without matches)
db.Table(users).LeftJoin(orders, ON users.id = orders.user_id).Select(users.name, orders.amount).All()

-- RIGHT JOIN
db.Table(users).RightJoin(orders, ON users.id = orders.user_id).Select(users.name, orders.amount).All()

-- chained joins
db.Table(orders).Join(users, ON orders.user_id = users.id).Join(products, ON orders.product = products.name).Select(users.name, products.price).All()
```

### Aliases

```wql
-- column alias
db.Table(orders).Select(Sum(amount) AS total).All()

-- function alias
db.Table(users).Count() AS total
```

## EXPLAIN — query plan

WQL v3 can parse a query and tell you *how* it will execute **without running it**:

```wql
explain db.Table(orders).Join(users, ON orders.user_id = users.id).Where(amount > 50).OrderBy(amount, DESC).Take(5).All()
```

Sample output:

```
Query Plan:
  Table:    orders
  Joins:    Join(users)
  Where:    (amount > 50)
  OrderBy:  yes
  Limit:    yes
  Pushdown: no (in-memory evaluation)
```

When pushdown is enabled (simple queries without JOIN/GROUP/aggregates),
the storage engine handles WHERE/ORDER/LIMIT directly:

```wql
explain db.Table(users).Where(age > 30).All()
```

```
Query Plan:
  Table:    users
  Where:    (age > 30)
  Pushdown: WHERE/ORDER/LIMIT → storage engine ✓
```

## Programmatic API (Go)

The string syntax is a thin wrapper over the Go API. Direct usage is
type-safe and IDE-friendly:

```go
// SELECT
rows, err := db.Table("users").
    Select("id", "name", "age").
    Where("age >= 18").
    OrderBy("age", "DESC").
    Take(10).
    All()

// aggregate
total, err := db.Table("orders").Sum("amount")

// INSERT
n, err := db.Insert("users").Value(map[string]interface{}{
    "id":   int64(1),
    "name": "alice",
    "age":  int64(30),
}).Execute()

// UPDATE
n, err := db.Update("users").Set("age", 35).Where("name = 'alice'").Execute()

// DELETE
n, err := db.Delete("users").Where("age < 18").Execute()
```

## Notes

- WQL is **not** SQL. There is no SQL string generation; the parser
  produces a Go AST and the executor calls the WeDB storage API directly.
- Identifier names (table/column) are always bare: `users`, `age`, `name`.
- String *values* still need quotes: `name = "alice"`, `dept = "Sales"`.
- Numbers are bare: `age > 18`, `amount > 50.0`.
- Booleans are bare: `active = true`, `deleted = false`.
- `null` is bare: `email = null`.
