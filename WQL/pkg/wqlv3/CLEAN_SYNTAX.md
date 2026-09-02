# WQL v3 DSL语法指南

## 理念：接近自然语言的查询

WQL v3的设计理念是让查询代码尽可能接近自然语言，通过DSL（领域特定语言）实现简洁的语法。

**最新特性：表名引用+延续的子查询语法**

WQL v3 现在支持人性化的子查询语法，通过表名引用实现自然的链式调用：

```wql
db.users
  .Select(name, email, Sum(amount) as total_amount)
  .Join(orders, id, user_id)
  .GroupBy(id, name, email)
  .Having(total_amount > orders
  .Where(user_id=id)
  .Avg(amount))
  .OrderBy(total_amount, DESC)
  .All()
```

这种语法的优势：
- 无需嵌套括号或函数调用
- 子查询通过表名引用自然嵌入
- 延续链式调用，保持流畅性
- 完全符合"无双引号"哲学

## 语法方式

WQL v3支持两种查询方式：

### 方式1: DSL语法（推荐）

```go
// 定义字段
age := Field("age")
name := Field("name")

// 使用DSL语法
result := db.Table(users)
    .Select(name, age)
    .Where(age.Gt(25))
    .All()
```

### 方式2: WQL字符串

```go
// 直接使用WQL字符串
result := db.Table(users)
    .Select(name, age)
    .Where("age>25")
    .All()
```

## DSL语法示例

### 1. 基础查询

```go
age := Field("age")
name := Field("name")

// 查询年龄大于25的用户
result := db.Table(users)
    .Select(name, age)
    .Where(age.Gt(25))
    .All()
```

### 2. 复合条件

```go
age := Field("age")

// 查询年龄在25到40之间的用户
result := db.Table(users)
    .Select(name, age)
    .Where(age.Gt(25).And(age.Lt(40)))
    .All()
```

### 3. 字符串比较

```go
name := Field("name")

// 查询名字为Alice的用户
result := db.Table(users)
    .Select(name, age)
    .Where(name.Eq("Alice"))
    .All()
```

### 4. 复杂条件

```go
age := Field("age")
name := Field("name")

// 查询年龄在25到40之间，或者名字为Eve的用户
result := db.Table(users)
    .Select(name, age)
    .Where(age.Gt(25).And(age.Lt(40)).Or(name.Eq("Eve")))
    .All()
```

### 5. IN操作

```go
age := Field("age")

// 查询年龄为25、30、35的用户
result := db.Table(users)
    .Select(name, age)
    .Where(age.In(25, 30, 35))
    .All()
```

### 6. BETWEEN操作

```go
age := Field("age")

// 查询年龄在25到35之间的用户
result := db.Table(users)
    .Select(name, age)
    .Where(age.Between(25, 35))
    .All()
```

### 7. LIKE操作

```go
name := Field("name")

// 查询名字以A开头的用户
result := db.Table(users)
    .Select(name, age)
    .Where(name.Like("A%"))
    .All()
```

### 8. NULL操作

```go
email := Field("email")

// 查询邮箱不为null的用户
result := db.Table(users)
    .Select(name, email)
    .Where(email.IsNull().Not())
    .All()
```

### 9. 功能化注释

```go
age := Field("age")

// 使用提示优化查询
result := db.Table(users)
    .Select(name, age)
    .Where(age.Gt(25))
    .Hint("UseIndex", "idx_age")
    .Prefer("FastScan")
    .Budget(100)
    .All()
```

### 10. 流式处理

```go
age := Field("age")

// 流式处理大量数据
err := db.Table(users)
    .Where(age.Gt(20))
    .Stream(func(row map[string]interface{}) error {
        fmt.Println(row)
        return nil
    })
```

### 11. 性能指标

```go
age := Field("age")

// 获取性能指标
result := db.Table(users)
    .Where(age.Gt(25))
    .All()

metrics := result.Metrics()
fmt.Printf("Query Time: %v\n", metrics.QueryTime)
fmt.Printf("Memory Usage: %v bytes\n", metrics.MemoryUsage)
fmt.Printf("Rows Scanned: %v\n", metrics.RowsScanned)
fmt.Printf("Rows Returned: %v\n", metrics.RowsReturned)
```

## DSL操作符

| 操作符 | 方法 | 示例 |
|--------|------|------|
| 等于 | `Eq(value)` | `age.Eq(25)` |
| 不等于 | `Ne(value)` | `age.Ne(25)` |
| 大于 | `Gt(value)` | `age.Gt(25)` |
| 大于等于 | `Ge(value)` | `age.Ge(25)` |
| 小于 | `Lt(value)` | `age.Lt(25)` |
| 小于等于 | `Le(value)` | `age.Le(25)` |
| IN | `In(values...)` | `age.In(25, 30, 35)` |
| BETWEEN | `Between(min, max)` | `age.Between(25, 35)` |
| LIKE | `Like(pattern)` | `name.Like("A%")` |
| IS NULL | `IsNull()` | `email.IsNull()` |
| IS NOT NULL | `IsNotNull()` | `email.IsNotNull()` |

## 逻辑操作

| 操作符 | 方法 | 示例 |
|--------|------|------|
| 与 | `.And(expr)` | `age.Gt(25).And(age.Lt(40))` |
| 或 | `.Or(expr)` | `age.Gt(60).Or(age.Lt(18))` |
| 非 | `.Not()` | `email.IsNull().Not()` |

## WQL字符串语法

WQL字符串语法支持以下操作符：

| 操作符 | 示例 | 说明 |
|--------|------|------|
| 大于 | `age>25` | 大于 |
| 小于 | `age<40` | 小于 |
| 大于等于 | `age>=25` | 大于等于 |
| 小于等于 | `age<=40` | 小于等于 |
| 等于 | `age==25` 或 `age=25` | 等于 |
| 不等于 | `age!=25` 或 `age<>25` | 不等于 |
| 逻辑与 | `age>25 and age<40` | 逻辑与 |
| 逻辑或 | `age>60 or age<18` | 逻辑或 |
| 字符串 | `name="Alice"` | 字符串比较 |

## 子查询语法（新特性）

WQL v3 引入了革命性的子查询语法，通过表名引用实现自然的链式调用。

### 基本语法

```wql
db.users
  .Select(name, email, Sum(amount) as total_amount)
  .Join(orders, id, user_id)
  .GroupBy(id, name, email)
  .Having(total_amount > orders
  .Where(user_id=id)
  .Avg(amount))
  .OrderBy(total_amount, DESC)
  .All()
```

### 子查询语法特点

1. **表名引用** - 在条件中直接引用表名：`orders`
2. **延续调用** - 后续方法自动作用于引用的表：`.Where()`, `.Avg()`
3. **无嵌套** - 不需要嵌套括号，语法自然流畅
4. **参数替换** - 自动替换条件中的列名：`user_id=id` 中的 `id` 会被替换为当前行的值

### 支持的子查询方法

| 方法 | 说明 | 示例 |
|------|------|------|
| `Where(condition)` | 添加WHERE条件 | `orders.Where(user_id=id)` |
| `Avg(column)` | 计算平均值 | `orders.Avg(amount)` |
| `Sum(column)` | 计算总和 | `orders.Sum(amount)` |
| `Count(column)` | 计算数量 | `orders.Count(id)` |
| `Max(column)` | 计算最大值 | `orders.Max(amount)` |
| `Min(column)` | 计算最小值 | `orders.Min(amount)` |

### 子查询示例

#### 示例1：简单比较

```wql
// 查询消费总额高于平均消费水平的用户
db.users
  .Select(name, Sum(amount) as total_amount)
  .Join(orders, id, user_id)
  .GroupBy(id, name)
  .Having(total_amount > orders
  .Where(user_id=id)
  .Avg(amount))
  .All()
```

#### 示例2：多层条件

```wql
// 查询每个城市中，消费金额超过该城市平均水平的用户
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

#### 示例3：综合分析

```wql
// 全方位分析用户购买行为
db.users
  .Select(city, name, Count(order_id) as order_count, Sum(amount) as total_amount, Avg(amount) as avg_amount)
  .Join(orders, id, user_id)
  .Where(status="completed")
  .GroupBy(id, name, city)
  .Having(total_amount > orders
  .Where(user_id=id)
  .Avg(amount)
  .And(total_amount > orders
  .Where(city=users.city)
  .Avg(amount)))
  .OrderBy(total_amount, DESC)
  .All()
```

### SQL vs WQL子查询对比

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
- WQL无需嵌套括号
- WQL无需SELECT子查询前缀
- WQL语法更接近自然语言
- WQL参数替换自动处理

## DSL语法特点

### 1. 接近自然语言

```go
// 人类语言：查询年龄大于25的用户
// DSL代码：
age := Field("age")
result := db.Table(users).Where(age.Gt(25)).All()
```

### 2. 类型安全

```go
// 编译时类型检查
age := Field("age")  // age字段
age.Gt(25)  // 25是数字，编译器可以检查类型

name := Field("name")
name.Eq("Alice")  // "Alice"是字符串，类型匹配
```

### 3. IDE支持

```go
// IDE自动完成
age := Field("age")
age.  // IDE会提示：Gt, Lt, Ge, Le, Eq, Ne, In, Between, Like, IsNull, IsNotNull
```

### 4. 链式调用

```go
// 流畅的链式调用
db.Table(users)
    .Select(name, age)
    .Where(age.Gt(25))
    .OrderBy("age", "DESC")
    .Take(10)
    .All()
```

## SQL vs WQL对比

### SQL语法

```sql
-- 需要引号，字符串拼接
SELECT name, age 
FROM users 
WHERE age > 25 
AND age < 40
ORDER BY age DESC 
LIMIT 10;
```

### WQL DSL语法

```go
// 类型安全，链式调用
age := Field("age")
name := Field("name")

result := db.Table(users)
    .Select(name, age)
    .Where(age.Gt(25).And(age.Lt(40)))
    .OrderBy("age", "DESC")
    .Take(10)
    .All()
```

### WQL字符串语法

```go
// 简洁的字符串语法
result := db.Table(users)
    .Select(name, age)
    .Where("age>25 and age<40")
    .OrderBy("age", "DESC")
    .Take(10)
    .All()
```

## 最佳实践

### 1. 在函数开头定义字段

```go
func queryUsers(db *wqlv3.Database) {
    // 在函数开头定义所有使用的字段
    age := wqlv3.Field("age")
    name := wqlv3.Field("name")
    email := wqlv3.Field("email")
    
    // 使用字段进行查询
    result := db.Table(users)
        .Select(name, age, email)
        .Where(age.Gt(18).And(age.Lt(65)))
        .All()
}
```

### 2. 包级别定义常用字段

```go
// 包级别定义
var (
    Age  = Field("age")
    Name = Field("name")
    Email = Field("email")
)

// 使用
result := db.Table(users).Where(Age.Gt(25)).All()
```

### 3. 使用有意义的变量名

```go
// ✅ 好的变量名
userAge := Field("age")
userName := Field("name")

// ❌ 不好的变量名
a := Field("age")
n := Field("name")
```

### 4. 复杂条件分行书写

```go
// ✅ 清晰的格式
.Where(age.Gt(25)
    .And(age.Lt(40))
    .Or(name.Eq("Eve")))

// ❌ 难以阅读
.Where(age.Gt(25).And(age.Lt(40)).Or(name.Eq("Eve")))
```

### 5. 利用性能指标

```go
result := db.Table(users)
    .Where(age.Gt(25))
    .All()

// 检查性能
metrics := result.Metrics()
if metrics.QueryTime > 100*time.Millisecond {
    log.Printf("查询耗时较长: %v", metrics.QueryTime)
}
```

## 常见问题

### Q: DSL语法和WQL字符串语法有什么区别？

A: DSL语法使用方法调用，提供类型安全和IDE支持；WQL字符串语法更简洁，适合快速查询。两者可以混合使用。

### Q: 字段需要每次都定义吗？

A: 可以在包级别定义常用字段，或者在函数开头定义一次，然后重复使用。

### Q: 性能如何？

A: WQL的性能与直接使用WeDB API相当，因为：
- 表达式求值是编译时优化的
- 查询执行直接调用WeDB API
- 没有SQL解析开销

### Q: 支持哪些数据库？

A: WQL v3设计为独立查询语言，当前在WeDB上试点，未来可扩展到其他数据库。

## 总结

WQL v3支持两种语法：

### DSL语法特点

✅ 类型安全  
✅ IDE自动完成  
✅ 链式调用  
✅ 易于阅读  
✅ 易于维护  

### WQL字符串语法特点

✅ 简洁  
✅ 接近自然语言  
✅ 无需预定义字段  
✅ 快速编写  

**选择建议**:
- 复杂查询：使用DSL语法
- 简单查询：使用WQL字符串语法

---

**版本**: 3.2  
**日期**: 2026年2月18日  
**作者**: iFlow CLI