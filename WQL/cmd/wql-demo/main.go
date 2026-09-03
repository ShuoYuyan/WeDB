// WQL v3 No-Quotes Demonstration.
//
// Exercises every WQL v3 capability against a real WeDB database:
// DDL, DML, SELECT with WHERE, JOIN, GROUP BY / HAVING, ORDER BY,
// LIMIT, OFFSET, aggregates, EXPLAIN, ANSI highlighting,
// DISTINCT, UNION/INTERSECT/EXCEPT, transactions, and IS NULL.
//
// Run it:
//
//	go run ./cmd/wql-demo
//	go run ./cmd/wql-demo -db /tmp/demo.db
//	go run ./cmd/wql-demo -color=false
//
// Self-contained: creates a fresh database, runs the scenarios,
// prints results, and tears the database down on exit.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/wedb/wedb/WQL/pkg/wqlv3"
	"github.com/wedb/wedb/internal/storage"
)

var (
	dbPath = flag.String("db", "wql-demo.db", "WeDB database file path")
	color  = flag.Bool("color", true, "enable ANSI color in output")
)

func main() {
	flag.Parse()
	wqlv3.SetColorEnabled(*color)

	_ = os.Remove(*dbPath)
	_ = os.Remove(*dbPath + ".metadata")
	defer os.Remove(*dbPath)
	defer os.Remove(*dbPath + ".metadata")

	wedb, err := storage.NewWeDBDatabase(*dbPath, 4096)
	if err != nil {
		fmt.Fprintln(os.Stderr, "open WeDB:", err)
		os.Exit(1)
	}
	defer wedb.Close()

	db := wqlv3.NewDatabase(wqlv3.NewWeDBAdapter(wedb))

	header("WQL v3 No-Quotes Demo")
	fmt.Println("Database file:", *dbPath)
	fmt.Println("Color:", *color)

	// 1. DDL
	header("1. DDL - Create Tables")
	run(db, `db.Table(departments).Create(id INTEGER PRIMARY KEY, name TEXT).Execute()`)
	run(db, `db.Table(users).Create(id INTEGER PRIMARY KEY, name TEXT, age INTEGER, dept TEXT).Execute()`)
	run(db, `db.Table(orders).Create(id INTEGER PRIMARY KEY, user_id INTEGER, amount REAL, product TEXT).Execute()`)

	// 2. DML Insert
	header("2. DML - Insert Data (no quotes on string values)")
	run(db, `db.Table(departments).Insert({id: 1, name: Engineering}).Execute()`)
	run(db, `db.Table(departments).Insert({id: 2, name: Sales}).Execute()`)
	run(db, `db.Table(departments).Insert({id: 3, name: Marketing}).Execute()`)
	users := []string{
		`db.Table(users).Insert({id: 1,  name: alice,   age: 30, dept: Engineering}).Execute()`,
		`db.Table(users).Insert({id: 2,  name: bob,     age: 25, dept: Sales}).Execute()`,
		`db.Table(users).Insert({id: 3,  name: carol,   age: 35, dept: Engineering}).Execute()`,
		`db.Table(users).Insert({id: 4,  name: dave,    age: 17, dept: Sales}).Execute()`,
		`db.Table(users).Insert({id: 5,  name: eve,     age: 45, dept: Marketing}).Execute()`,
		`db.Table(users).Insert({id: 6,  name: frank,   age: 28, dept: Engineering}).Execute()`,
		`db.Table(users).Insert({id: 7,  name: grace,   age: 22, dept: Marketing}).Execute()`,
		`db.Table(users).Insert({id: 8,  name: heidi,   age: 40, dept: Sales}).Execute()`,
		`db.Table(users).Insert({id: 9,  name: ivan,    age: 33, dept: Engineering}).Execute()`,
		`db.Table(users).Insert({id: 10, name: judy,    age: 19, dept: Marketing}).Execute()`,
	}
	for _, q := range users {
		run(db, q)
	}
	orders := []string{
		`db.Table(orders).Insert({id: 100, user_id: 1, amount: 50.0,   product: book}).Execute()`,
		`db.Table(orders).Insert({id: 101, user_id: 1, amount: 200.0,  product: laptop}).Execute()`,
		`db.Table(orders).Insert({id: 102, user_id: 2, amount: 30.0,   product: pen}).Execute()`,
		`db.Table(orders).Insert({id: 103, user_id: 3, amount: 800.0,  product: monitor}).Execute()`,
		`db.Table(orders).Insert({id: 104, user_id: 3, amount: 25.0,   product: cable}).Execute()`,
		`db.Table(orders).Insert({id: 105, user_id: 5, amount: 120.0,  product: chair}).Execute()`,
		`db.Table(orders).Insert({id: 106, user_id: 6, amount: 75.0,   product: lamp}).Execute()`,
		`db.Table(orders).Insert({id: 107, user_id: 8, amount: 1000.0, product: desk}).Execute()`,
		`db.Table(orders).Insert({id: 108, user_id: 9, amount: 60.0,   product: phone}).Execute()`,
		`db.Table(orders).Insert({id: 109, user_id: 10, amount: 15.0,  product: sticker}).Execute()`,
	}
	for _, q := range orders {
		run(db, q)
	}

	// 3. SELECT / WHERE
	header("3. SELECT with WHERE (no-quote strings)")
	run(db, `db.Table(users).Where(age > 20 AND dept = Sales).Select(name, age).All()`)
	run(db, `db.Table(users).Set(age, 35).Where(name = alice).Execute()`)
	run(db, `db.Table(users).Where(name = alice).Select(name, age).All()`)

	// 4. IN / BETWEEN / LIKE / IS NULL
	header("4. Advanced WHERE operators (no-quote)")
	run(db, `db.Table(users).Where(id IN (1, 3, 5, 7)).Select(id, name).All()`)
	run(db, `db.Table(users).Where(age BETWEEN 25 AND 35).Select(name, age).All()`)
	run(db, `db.Table(users).Where(name LIKE a%).Select(name).All()`)
	run(db, `db.Table(users).Where(dept IS NOT NULL).Count()`)
	run(db, `db.Table(users).Where(dept IS NULL).Count()`)
	run(db, `db.Table(users).Where(NOT (age > 18)).Select(name, age).All()`)

	// 5. ORDER BY / LIMIT
	header("5. ORDER BY + LIMIT + OFFSET")
	run(db, `db.Table(users).OrderBy(age, DESC).Select(name, age).Take(3).All()`)
	run(db, `db.Table(users).OrderBy(age, ASC).Skip(5).Take(3).Select(name, age).All()`)

	// 6. Aggregates
	header("6. Aggregate Functions")
	run(db, `db.Table(users).Count()`)
	run(db, `db.Table(orders).Sum(amount)`)
	run(db, `db.Table(orders).Avg(amount)`)
	run(db, `db.Table(users).Min(age)`)
	run(db, `db.Table(users).Max(age)`)

	// 7. GROUP BY / HAVING
	header("7. GROUP BY + HAVING")
	run(db, `db.Table(users).GroupBy(dept).Select(dept, Count()).All()`)
	run(db, `db.Table(users).GroupBy(dept).Having(Count() > 2).Select(dept, Count()).All()`)

	// 8. JOIN
	header("8. JOIN")
	run(db, `db.Table(users).Join(orders, ON users.id = orders.user_id).Select(name, product, amount).OrderBy(amount, DESC).All()`)
	run(db, `db.Table(users).LeftJoin(orders, ON users.id = orders.user_id).Select(name, product).All()`)

	// 9. DISTINCT
	header("9. DISTINCT")
	run(db, `db.Table(users).Select(dept).Distinct().All()`)
	run(db, `db.Table(orders).Select(user_id).Distinct(user_id).All()`)

	// 10. UNION / INTERSECT / EXCEPT
	header("10. Set Operations (UNION/INTERSECT/EXCEPT)")
	run(db, `db.Table(employeesA).Create(id INTEGER PRIMARY KEY, name TEXT).Execute()`)
	run(db, `db.Table(employeesB).Create(id INTEGER PRIMARY KEY, name TEXT).Execute()`)
	run(db, `db.Table(employeesA).Insert({id: 1, name: alice}).Execute()`)
	run(db, `db.Table(employeesA).Insert({id: 2, name: bob}).Execute()`)
	run(db, `db.Table(employeesB).Insert({id: 2, name: bob}).Execute()`)
	run(db, `db.Table(employeesB).Insert({id: 3, name: carol}).Execute()`)
	run(db, `db.Table(employeesA).Select(id, name).Union(db.Table(employeesB).Select(id, name)).All()`)
	run(db, `db.Table(employeesA).Select(id, name).Intersect(db.Table(employeesB).Select(id, name)).All()`)
	run(db, `db.Table(employeesA).Select(id, name).Except(db.Table(employeesB).Select(id, name)).All()`)

	// 11. Transactions
	header("11. Transactions (DML inside tx)")
	run(db, `db.Table(accounts).Create(id INTEGER PRIMARY KEY, balance INTEGER).Execute()`)
	run(db, `db.Table(accounts).Insert({id: 1, balance: 100}).Execute()`)
	run(db, `db.Table(accounts).Begin()`)
	run(db, `db.Table(accounts).Set(balance, 200).Where(id = 1).Execute()`)
	run(db, `db.Table(accounts).Where(id = 1).First()`)
	run(db, `db.Table(accounts).Commit()`)
	run(db, `db.Table(accounts).Where(id = 1).First()`)

	// 12. Rollback
	header("12. Transaction Rollback")
	run(db, `db.Table(accounts).Begin()`)
	run(db, `db.Table(accounts).Set(balance, 999).Where(id = 1).Execute()`)
	run(db, `db.Table(accounts).Rollback()`)
	run(db, `db.Table(accounts).Where(id = 1).First()`)

	// 13. EXPLAIN
	header("13. EXPLAIN")
	plan, err := wqlv3.Explain(`db.Table(orders).Where(amount > 100).OrderBy(amount, DESC).Take(5).All()`)
	if err != nil {
		fmt.Println("EXPLAIN error:", err)
	} else {
		fmt.Println(plan)
	}

	// 14. Highlight
	header("14. Syntax Highlighting")
	q := `db.Table(orders).Where(amount > 100).OrderBy(amount, DESC).Take(5).All()`
	fmt.Println(wqlv3.HighlightSimple(q))

	// 15. CASE WHEN / COALESCE / NULLIF / CAST (no-quote expressions)
	header("15. CASE WHEN / COALESCE / NULLIF / CAST (no-quote)")
	run(db, `db.Table(orders).Select(id, CASE WHEN amount > 100 THEN high WHEN amount > 50 THEN mid ELSE low END).All()`)
	run(db, `db.Table(orders).Select(id, COALESCE(status, unknown)).All()`)
	run(db, `db.Table(orders).Select(id, NULLIF(status, cancelled)).All()`)
	run(db, `db.Table(orders).Select(id, CAST(amount AS INTEGER)).All()`)

	// 16. UPSERT (ON CONFLICT)
	header("16. UPSERT / ON CONFLICT")
	run(db, `db.Table(orders).Insert({id: 100, amount: 999, status: first}).Execute()`)
	run(db, `db.Table(orders).Insert({id: 100, amount: 1000, status: second}).OnConflict(UPDATE, id).Execute()`)
	run(db, `db.Table(orders).Where(id = 100).All()`)

	// 17. No-quote keyword as value (WQL design principle)
	header("17. No-quote keyword as value")
	// 以下字面量值都是 WQL 关键字；无双引号设计下应自动视为字符串
	run(db, `db.Table(orders).Select(id, CASE WHEN status = first THEN primary WHEN status = second THEN backup ELSE other END).All()`)

	header("Done")
}

func run(db *wqlv3.Database, query string) {
	fmt.Println(">>", wqlv3.HighlightSimple(query))
	res, err := wqlv3.EvaluateQueryNoQuotes(db, query)
	if err != nil {
		fmt.Println("   ERROR:", err)
		return
	}
	if res.Value != nil {
		fmt.Printf("   = %v\n", res.Value)
		return
	}
	if len(res.Rows) == 0 {
		fmt.Println("   (0 rows)")
		return
	}
	printRows(res.Rows, 8)
}

func printRows(rows []map[string]interface{}, max int) {
	if len(rows) > max {
		fmt.Printf("   (%d rows, showing first %d)\n", len(rows), max)
		rows = rows[:max]
	}
	for _, r := range rows {
		keys := make([]string, 0, len(r))
		for k := range r {
			keys = append(keys, k)
		}
		pairs := make([]string, 0, len(r))
		// 简单排序
		for _, k := range sortedKeys(keys) {
			pairs = append(pairs, fmt.Sprintf("%s=%v", k, r[k]))
		}
		fmt.Println("   " + strings.Join(pairs, ", "))
	}
}

func sortedKeys(in []string) []string {
	out := make([]string, len(in))
	copy(out, in)
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j-1] > out[j]; j-- {
			out[j-1], out[j] = out[j], out[j-1]
		}
	}
	return out
}

func header(s string) {
	fmt.Println()
	fmt.Println("=== " + s + " ===")
}
