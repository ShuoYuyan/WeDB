# WQL v3 - 完全独立于SQL的查询语言

## 设计目标

✅ **完全链式查询** - 支持完整的方法链式调用  
✅ **DSL语法** - 类型安全的领域特定语言  
✅ **WQL字符串语法** - 简洁的字符串查询语法  
✅ **独立于SQL** - 完全不依赖database/sql  
✅ **WeDB唯一查询语言** - 与WeDB深度集成  
✅ **原生链式注释** - Hint、Prefer、Budget、Assume  
✅ **性能指标** - 查询结果的一部分  
✅ **跨数据库** - 设计为可扩展到其他数据库  

## 核心特性

### 1. 两种查询语法

#### DSL语法（类型安全）

```go
// 定义字段
age := Field("age")
name := Field("name")

// 使用DSL语法
result := db.Table(users)
    .Select(name, age)
    .Where(age.Gt(25).And(age.Lt(40)))
    .All()
```

#### WQL字符串语法（简洁）

```go
// 使用WQL字符串语法
result := db.Table(users)
    .Select(name, age)
    .Where("age>25 and age<40")
    .All()
```

### 2. 链式查询API

```go
// DSL语法
age := Field("age")
name := Field("name")

result := db.Table(users)
    .Select(name, age)
    .Where(age.Gt(25))
    .OrderBy("age", "DESC")
    .Take(3)
    .All()
```

### 3. 原生链式注释

```go
age := Field("age")

result := db.Table(users)
    .Select(name, age)
    .Where(age.Gt(25))
    .Hint("UseIndex", "idx_age")
    .Prefer("FastScan")
    .Budget(100)
    .All()
```

### 4. 性能指标

```go
result := db.Table(users)
    .Where(age.Gt(25))
    .All()

metrics := result.Metrics()
fmt.Printf("Query Time: %v\n", metrics.QueryTime)
fmt.Printf("Memory Usage: %v bytes\n", metrics.MemoryUsage)
fmt.Printf("Rows Scanned: %v\n", metrics.RowsScanned)
fmt.Printf("Rows Returned: %v\n", metrics.RowsReturned)
```

### 5. 流式处理API

```go
age := Field("age")

err := db.Table(users)
    .Where(age.Gt(20))
    .Stream(func(row map[string]interface{}) error {
        fmt.Println(row)
        return nil
    })
```

### 6. 并发安全API

```go
// 每次操作返回新对象，避免状态污染
age := Field("age")
result1 := db.Table(users).Where(age.Gt(25)).All()
result2 := db.Table(users).Where(age.Lt(30)).All()
```

## DSL API文档

### 字段定义

```go
// 创建字段
Field(name string)        // 创建字段引用
```

### 字段操作符

```go
// 比较操作
field.Eq(value)           // 等于
field.Ne(value)           // 不等于
field.Gt(value)           // 大于
field.Ge(value)           // 大于等于
field.Lt(value)           // 小于
field.Le(value)           // 小于等于

// 集合操作
field.In(values...)       // IN操作
field.Between(min, max)   // BETWEEN操作
field.Like(pattern)       // LIKE操作
field.IsNull()            // IS NULL
field.IsNotNull()         // IS NOT NULL
```

### 逻辑操作

```go
// 表达式操作
expr.And(expr)            // 逻辑与
expr.Or(expr)             // 逻辑或
expr.Not()                // 逻辑非

// 全局函数
And(left, right)          // 逻辑与
Or(left, right)           // 逻辑或
Not(expr)                 // 逻辑非
```

## 子查询语法（新特性）

WQL v3 引入了革命性的子查询语法，通过表名引用实现自然的链式调用。

### 基本语法

```wql
db.users
  .Select(name, Sum(amount) as total_amount)
  .Join(orders, id, user_id)
  .GroupBy(id, name)
  .Having(total_amount > orders
  .Where(user_id=id)
  .Avg(amount))
  .OrderBy(total_amount, DESC)
  .All()
```

### 语法优势

- **无嵌套括号** - 不需要SQL的嵌套SELECT语句
- **表名引用** - 直接使用表名开启子查询
- **延续调用** - 子查询方法自然延续
- **参数替换** - 自动替换条件中的列名

### 支持的子查询方法

| 方法 | 说明 | 示例 |
|------|------|------|
| `Table(tableName).Where(condition)` | WHERE条件 | `orders.Where(user_id=id)` |
| `Avg(column)` | 平均值 | `orders.Avg(amount)` |
| `Sum(column)` | 总和 | `orders.Sum(amount)` |
| `Count(column)` | 数量 | `orders.Count(id)` |
| `Max(column)` | 最大值 | `orders.Max(amount)` |
| `Min(column)` | 最小值 | `orders.Min(amount)` |

### 子查询示例

#### 简单子查询

```wql
// 查询消费总额高于平均水平的用户
db.users
  .Select(name, Sum(amount) as total_amount)
  .Join(orders, id, user_id)
  .GroupBy(id, name)
  .Having(total_amount > orders
  .Where(user_id=id)
  .Avg(amount))
  .All()
```

#### 多层条件

```wql
// 查询每个城市中，消费超过城市平均水平的用户
db.users
  .Select(city, name, Sum(amount) as user_total)
  .Join(orders, id, user_id)
  .GroupBy(id, name, city)
  .Having(user_total > orders
  .Where(user_id=id)
  .And(city=users.city)
  .Avg(amount))
  .OrderBy(user_total, DESC)
  .All()
```

### SQL vs WQL对比

#### SQL语法

```sql
SELECT u.name, SUM(o.amount) as total_amount
FROM users u
JOIN orders o ON u.id = o.user_id
GROUP BY u.id, u.name
HAVING total_amount > (
  SELECT AVG(amount)
  FROM orders
  WHERE user_id = u.id
)
ORDER BY total_amount DESC;
```

#### WQL语法

```wql
db.users
  .Select(name, Sum(amount) as total_amount)
  .Join(orders, id, user_id)
  .GroupBy(id, name)
  .Having(total_amount > orders
  .Where(user_id=id)
  .Avg(amount))
  .OrderBy(total_amount, DESC)
  .All()
```

**对比优势**：
- 语法更简洁，无需嵌套括号
- 更接近自然语言表达
- 自动处理参数替换
- 链式调用保持一致性

### 查询方法

```go
// 基础方法
Table(tableName)          // 指定表
Select(columns...)         // 选择列
Where(expr/string)        // WHERE条件（支持DSL或字符串）
AndWhere(expr/string)     // AND条件
OrWhere(expr/string)      // OR条件
OrderBy(column, dir)      // 排序
Take(count)              // 限制行数
Skip(offset)             // 跳过行数
All()                     // 执行查询（返回所有结果）
First()                   // 执行查询（返回第一个结果）
Count()                   // 执行查询（返回结果数量）
Stream(handler)           // 流式处理
```

### 功能化注释

```go
Hint(name, value)         // 查询提示
Prefer(preference)        // 执行偏好
Budget(maxRows)           // 查询预算
BudgetWithMemory(maxRows, maxMemory)  // 带内存预算
Assume(expr)              // 查询假设
AssumeWithConfidence(expr, confidence)  // 带置信度的假设
```

### 数据操作

```go
// 插入
insert := wqlv3.NewInsertBuilder(adapter, tableName)
insert.Values(row).Execute()

// 更新
update := wqlv3.NewUpdateBuilder(adapter, tableName)
update.Set(column, value).Where(expr).Execute()

// 删除
del := wqlv3.NewDeleteBuilder(adapter, tableName)
del.Where(expr).Execute()
```

### 聚合操作

```go
agg := wqlv3.NewAggregationBuilder(adapter, tableName)
agg.Count()                // 计数
agg.Sum(column)            // 求和
agg.Avg(column)            // 平均值
agg.Min(column)            // 最小值
agg.Max(column)            // 最大值
```

### 事务操作

```go
tx, err := db.BeginTx(ctx, &wqlv3.TxOptions{
    ReadOnly: false,
    Timeout:  10 * time.Second,
})

txResult := tx.Table(users).Where(expr).All()
err = tx.Commit()
// 或
err = tx.Rollback()
```

## WQL字符串语法

WQL字符串支持以下语法：

| 操作符 | 示例 | 说明 |
|--------|------|------|
| 大于 | `age>25` | 大于 |
| 小于 | `age<40` | 小于 |
| 大于等于 | `age>=25` | 大于等于 |
| 小于等于 | `age<=40` | 小于等于 |
| 等于 | `age==25` 或 `age=25` | 等于 |
| 不等于 | `age!=25` 或 `age<>25` | 不等于 |
| 逻辑与 | `age>25 && age<40` 或 `age>25 and age<40` | 逻辑与 |
| 逻辑或 | `age>60 || age<18` 或 `age>60 or age<18` | 逻辑或 |
| 字符串 | `name="Alice"` 或 `name='Alice'` | 字符串比较 |
| NULL | `name==null` 或 `name=null` | NULL比较 |

## 完全独立于SQL

WQL v3与SQL的对比：

| 特性 | SQL | WQL v3 |
|------|-----|--------|
| 查询方式 | 字符串 | DSL或字符串 |
| 标识符 | 需要引号 | DSL无需引号，字符串可选择性使用 |
| 参数绑定 | `?` 或 `$1` | 类型安全表达式 |
| 依赖 | database/sql | 直接使用数据库API |
| 执行 | SQL解析器 | 解释器 |
| 安全 | SQL注入风险 | 类型安全 |
| 可扩展性 | 固定语法 | 独立语言，可扩展到多数据库 |

## 架构设计

```
┌─────────────────────────────────────┐
│         WQL v3 API                  │
│  (DSL语法、WQL字符串、功能化注释)    │
└──────────────┬──────────────────────┘
               │
┌──────────────▼──────────────────────┐
│     WQL 解析器和解释器              │
│  (DSL求值、字符串解析、查询优化)     │
└──────────────┬──────────────────────┘
               │
┌──────────────▼──────────────────────┐
│     数据库适配层                     │
│  (适配WeDB等数据库API)               │
└──────────────┬──────────────────────┘
               │
┌──────────────▼──────────────────────┐
│     WeDB 存储引擎                    │
│  (B-Tree、页面、事务、MVCC)          │
└─────────────────────────────────────┘
```

## 性能指标

QueryMetrics包含以下指标：

| 指标 | 类型 | 说明 |
|------|------|------|
| QueryTime | time.Duration | 查询耗时 |
| MemoryUsage | int64 | 内存使用（字节） |
| RowsScanned | int64 | 扫描行数 |
| RowsReturned | int64 | 返回行数 |
| IndexUsed | string | 使用的索引 |
| ExecutionPlan | string | 执行计划 |
| CacheHit | bool | 缓存命中 |

## 功能化注释说明

### Hint - 查询提示

```go
Hint("UseIndex", "idx_age")   // 使用指定索引
Hint("FastScan", "true")      // 快速扫描
Hint("SlowScan", "true")      // 慢速扫描
Hint("Cache", "true")         // 启用缓存
Hint("NoCache", "true")       // 禁用缓存
```

### Prefer - 执行偏好

```go
Prefer("FastScan")   // 偏好快速扫描
Prefer("SlowScan")   // 偏慢速扫描（更准确）
Prefer("Parallel")   // 偏好并行执行
Prefer("Sequential") // 偏好顺序执行
```

### Budget - 查询预算

```go
Budget(100)                            // 最多返回100行
BudgetWithMemory(100, 1024*1024)      // 最多100行，1MB内存
```

### Assume - 查询假设

```go
age := Field("age")
Assume(age.Gt(20))                        // 假设age > 20
AssumeWithConfidence(age.Gt(20), 0.9)     // 假设age > 20，置信度90%
```

## 并发安全

所有QueryBuilder方法都返回新的QueryBuilder实例，确保并发安全：

```go
// 安全的并发查询
go func() {
    age := Field("age")
    result := db.Table(users).Where(age.Gt(25)).All()
    // 处理result
}()

go func() {
    age := Field("age")
    result := db.Table(users).Where(age.Lt(30)).All()
    // 处理result
}()
```

## 示例

完整的示例代码请参考：
- `WQL/examples/wqlv3_dsl_demo.go` - DSL语法示例
- `WQL/examples/wqlv3_true_syntax.go` - WQL字符串语法示例

## WQL作为独立查询语言

### 当前状态

- ✅ 在WeDB上进行试点
- ✅ 作为WeDB的唯一查询语言
- ✅ 支持DSL和字符串两种语法
- ✅ 完全独立于SQL
- ✅ 可扩展架构

### 未来规划

- 扩展到其他数据库
- 实现更多WQL特性
- 优化查询性能
- 完善工具链

## 技术实现

### 关键设计决策

| 决策 | 说明 | 优势 |
|------|------|------|
| 双语法支持 | DSL+字符串 | 灵活选择 |
| 不使用database/sql | 直接使用数据库API | 完全独立 |
| 表达式求值引擎 | 独立的DSL求值 | 类型安全 |
| 解析器+解释器 | 双引擎架构 | 性能优化 |
| 适配层模式 | 支持多数据库 | 可扩展 |

## 与旧版WQL的区别

| 特性 | WQL v2 | WQL v3 |
|------|--------|--------|
| SQL依赖 | 使用sql.DB | 不使用SQL |
| 查询方式 | 部分SQL兼容 | DSL+字符串双语法 |
| 引号设计 | 需要引号 | DSL无需引号 |
| 性能指标 | 无 | 内置Metrics |
| 功能化注释 | 无 | 完整支持 |
| 流式处理 | 基础支持 | 完整支持 |
| 并发安全 | 部分支持 | 完全支持 |
| 可扩展性 | 仅WeDB | 设计为多数据库 |

## 总结

WQL v3是一个**完全独立的查询语言**，具有以下特点：

✅ 完全链式查询  
✅ DSL语法（类型安全）  
✅ WQL字符串语法（简洁）  
✅ 独立于SQL（不使用database/sql）  
✅ WeDB的唯一查询语言  
✅ 原生链式注释  
✅ 性能指标内置  
✅ 流式处理  
✅ 并发安全  
✅ 可扩展到多数据库  

**语法选择**:
- 复杂查询：使用DSL语法（类型安全）
- 简单查询：使用WQL字符串语法（简洁）

**状态**: 已完成实现，可在WeDB上使用，未来可扩展到其他数据库。

---

**版本**: 3.2  
**日期**: 2026年2月18日  
**作者**: iFlow CLI