// wql-demo - WQL v3 No-Quotes Fluent API end-to-end demonstration.
//
// This program exercises every WQL v3 capability against a real WeDB
// database: DDL, DML, SELECT with WHERE, JOIN, GROUP BY / HAVING,
// ORDER BY, LIMIT, OFFSET, aggregates, EXPLAIN, and ANSI highlighting.
//
// Run it:
//
//	go run ./cmd/wql-demo
//	go run ./cmd/wql-demo -db /tmp/demo.db
//
// The program is intentionally self-contained: it creates a fresh
// database, runs the scenarios, prints results, and tears the database
// down on exit. It also acts as living documentation of the WQL API.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/wedb/wedb/WQL/pkg/wqlv3"
	"github.com/wedb/wedb/internal/storage"
)

var dbPath = flag.String("db", "wql_demo.db", "WeDB database file path")
var color = flag.Bool("color", true, "enable ANSI color output")

func main() {
	flag.Parse()
	wqlv3.SetColorEnabled(*color)

	// --- Setup ---
	_ = os.Remove(*dbPath) // start clean
	wedb, err := storage.NewWeDBDatabase(*dbPath, 4096)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer wedb.Close()
	defer os.Remove(*dbPath) // clean up after demo

	adapter := wqlv3.NewWeDBAdapter(wedb)
	db := wqlv3.NewDatabase(adapter)

	header("WQL v3 - WeDB Native Query Language Demo")
	fmt.Println("Database:", *dbPath)
	fmt.Println()

	// --- 1. DDL: Create tables ---
	header("1. DDL - CREATE TABLE (no-quotes)")
	run(db, `db.Table(users).Create(id INTEGER PRIMARY KEY, name TEXT, age INTEGER, dept TEXT).Execute()`)
	run(db, `db.Table(orders).Create(id INTEGER PRIMARY KEY, user_id INTEGER, amount REAL, product TEXT).Execute()`)
	run(db, `db.Table(departments).Create(id INTEGER PRIMARY KEY, name TEXT).Execute()`)

	// --- 2. DML: Insert rows ---
	header("2. DML - INSERT (no-quotes, object-literal VALUES)")
	run(db, `db.Table(departments).Insert({id: 1, name: "Engineering"}).Execute()`)
	run(db, `db.Table(departments).Insert({id: 2, name: "Sales"}).Execute()`)
	run(db, `db.Table(departments).Insert({id: 3, name: "Marketing"}).Execute()`)

	users := []string{
		`db.Table(users).Insert({id: 1,  name: "alice",   age: 30, dept: "Engineering"}).Execute()`,
		`db.Table(users).Insert({id: 2,  name: "bob",     age: 25, dept: "Sales"}).Execute()`,
		`db.Table(users).Insert({id: 3,  name: "carol",   age: 35, dept: "Engineering"}).Execute()`,
		`db.Table(users).Insert({id: 4,  name: "dave",    age: 17, dept: "Sales"}).Execute()`,
		`db.Table(users).Insert({id: 5,  name: "eve",     age: 45, dept: "Marketing"}).Execute()`,
		`db.Table(users).Insert({id: 6,  name: "frank",   age: 28, dept: "Engineering"}).Execute()`,
		`db.Table(users).Insert({id: 7,  name: "grace",   age: 22, dept: "Marketing"}).Execute()`,
		`db.Table(users).Insert({id: 8,  name: "heidi",   age: 40, dept: "Sales"}).Execute()`,
		`db.Table(users).Insert({id: 9,  name: "ivan",    age: 33, dept: "Engineering"}).Execute()`,
		`db.Table(users).Insert({id: 10, name: "judy",    age: 19, dept: "Marketing"}).Execute()`,
	}
	for _, q := range users {
		run(db, q)
	}

	orders := []string{
		`db.Table(orders).Insert({id: 100, user_id: 1, amount: 50.0,  product: "book"}).Execute()`,
		`db.Table(orders).Insert({id: 101, user_id: 1, amount: 200.0, product: "laptop"}).Execute()`,
		`db.Table(orders).Insert({id: 102, user_id: 2, amount: 30.0,  product: "pen"}).Execute()`,
		`db.Table(orders).Insert({id: 103, user_id: 3, amount: 800.0, product: "monitor"}).Execute()`,
		`db.Table(orders).Insert({id: 104, user_id: 3, amount: 25.0,  product: "cable"}).Execute()`,
		`db.Table(orders).Insert({id: 105, user_id: 5, amount: 120.0, product: "chair"}).Execute()`,
		`db.Table(orders).Insert({id: 106, user_id: 6, amount: 75.0,  product: "lamp"}).Execute()`,
		`db.Table(orders).Insert({id: 107, user_id: 8, amount: 1000.0, product: "desk"}).Execute()`,
		`db.Table(orders).Insert({id: 108, user_id: 9, amount: 60.0,  product: "phone"}).Execute()`,
		`db.Table(orders).Insert({id: 109, user_id: 10, amount: 15.0, product: "sticker"}).Execute()`,
	}
	for _, q := range orders {
		run(db, q)
	}

	// --- 3. SELECT: basic ---
	header("3. SELECT - basic columns")
	run(db, `db.Table(users).Select(id, name, age).Take(5).All()`)

	// --- 4. WHERE: simple predicate ---
	header("4. WHERE - simple predicate (age >= 25)")
	run(db, `db.Table(users).Where(age >= 25).Select(name, age).OrderBy(age, ASC).All()`)

	// --- 5. WHERE: compound predicate ---
	header("5. WHERE - compound predicate (age > 20 AND dept = \"Sales\")")
	run(db, `db.Table(users).Where(age > 20 AND dept = "Sales").Select(name, age).All()`)

	// --- 6. Aggregate: Count ---
	header("6. Aggregate - Count all users")
	run(db, `db.Table(users).Count()`)

	// --- 7. Aggregate: Avg ---
	header("7. Aggregate - Avg(age)")
	run(db, `db.Table(users).Select(Avg(age)).All()`)

	// --- 8. Aggregate: Sum(amount) ---
	header("8. Aggregate - Sum(amount) over all orders")
	run(db, `db.Table(orders).Select(Sum(amount)).All()`)

	// --- 9. GROUP BY + HAVING ---
	header("9. GROUP BY + HAVING (orders grouped by user, total > 100)")
	run(db, `db.Table(orders).GroupBy(user_id).Select(user_id, Sum(amount)).Having(Sum(amount) > 100).All()`)

	// --- 10. JOIN: INNER with ON clause ---
	header("10. JOIN - INNER (orders JOIN users ON orders.user_id = users.id)")
	run(db, `db.Table(orders).Join(users, ON orders.user_id = users.id).Select(users.name, orders.amount).OrderBy(orders.amount, DESC).All()`)

	// --- 11. JOIN: LEFT ---
	header("11. JOIN - LEFT (all users, even those without orders)")
	run(db, `db.Table(users).LeftJoin(orders, ON users.id = orders.user_id).Select(users.name, orders.amount).Take(8).All()`)

	// --- 12. ORDER BY + LIMIT + OFFSET ---
	header("12. ORDER BY + LIMIT + OFFSET (top 3 by age, skip 2)")
	run(db, `db.Table(users).OrderBy(age, DESC).Skip(2).Take(3).Select(name, age).All()`)

	// --- 13. UPDATE + DELETE ---
	header("13. UPDATE - give alice a 5-unit raise")
	run(db, `db.Table(users).Set(age, 35).Where(name = "alice").Execute()`)
	run(db, `db.Table(users).Where(name = "alice").Select(name, age).All()`)

	header("14. DELETE - remove users under 18")
	run(db, `db.Table(users).Where(age < 18).Delete().Execute()`)
	run(db, `db.Table(users).Count()`)

	// --- 15. EXPLAIN (no execution) ---
	header("15. EXPLAIN - show query plan (no execution)")
	runPlan(`db.Table(orders).Join(users, ON orders.user_id = users.id).Where(amount > 50).OrderBy(amount, DESC).Take(5).All()`)

	runPlan(`db.Table(users).Where(age > 30).All()`)

	runPlan(`db.Table(orders).GroupBy(user_id).Select(Sum(amount)).Having(Sum(amount) > 100).All()`)

	// --- 16. ANSI syntax highlight (format) ---
	header("16. FORMAT - ANSI syntax highlighting")
	queries := []string{
		`db.Table(orders).Join(users, ON orders.user_id = users.id).Where(amount > 50).OrderBy(amount, DESC).Take(10).All()`,
		`db.Table(users).Where(age > 18 AND dept = "Sales").Select(name, age).All()`,
		`db.Table(orders).GroupBy(user_id).Select(Sum(amount)).Having(Sum(amount) > 100).All()`,
	}
	for _, q := range queries {
		fmt.Println(wqlv3.HighlightSimple(q))
	}

	// --- 17. DROP TABLE ---
	header("17. DDL - DROP TABLE")
	run(db, `db.Table(orders).Drop().Execute()`)
	run(db, `db.Table(users).Drop().Execute()`)
	run(db, `db.Table(departments).Drop().Execute()`)

	fmt.Println()
	header("Demo complete")
	fmt.Println("All WQL v3 features exercised. Database cleaned up.")
}

// run executes a WQL query and pretty-prints its result.
func run(db *wqlv3.Database, query string) {
	fmt.Println(">>", wqlv3.HighlightSimple(query))
	res, err := wqlv3.EvaluateQueryNoQuotes(db, query)
	if err != nil {
		fmt.Println("  ERROR:", err)
		return
	}
	wqlv3.PrintResult(res)
	fmt.Println()
}

// runPlan prints a query plan via EXPLAIN (no execution).
func runPlan(query string) {
	fmt.Println(">>", wqlv3.HighlightSimple("explain "+query))
	plan, err := wqlv3.Explain(query)
	if err != nil {
		fmt.Println("  ERROR:", err)
		return
	}
	fmt.Print(plan.String())
}

func header(s string) {
	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════════════════════")
	fmt.Println(" ", s)
	fmt.Println("═══════════════════════════════════════════════════════════════════")
}
