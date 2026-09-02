// WQL CLI - WeDB 原生查询语言命令行工具
// 这是 WQL 项目的官方 CLI 入口，使用 wqlv3 包（基于 WeDB API，完全不依赖 SQL）。
//
// 用法:
//   wql-cli <database.db>              # 打开数据库进入交互模式
//   wql-cli <database.db> <wql-query>  # 执行单条查询并退出
//   wql-cli --help                      # 显示帮助
//   wql-cli --version                   # 显示版本
//
// WQL 语法示例 (链式方法调用, 零引号设计):
//   users.Select(id, name, age)
//   users.Select(id, name).Where(age > 18).OrderBy(age, DESC).Take(10)
//   users.Where(age > 18 AND name = "alice").Count()
//   users.Sum(age)
//
// 注意: 这里的 WQL 是 Go 原生 API 的方法链，**不是 SQL 文本**。
//       内部通过 WeDBAdapter 直接调用 WeDB 的 Go API，
//       不生成任何 SQL 字符串。
package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/wedb/wedb/WQL/pkg/wqlv3"
	"github.com/wedb/wedb/internal/storage"
)

const version = "0.1.0"

const banner = `
 ╦ ╦╔═╗╦  ╔═╗
 ║║║║╣ ║  ╠═╣
 ╚╩╝╚═╝╩═╝╩ ╩ WeDB Native Query Language

 v%s  •  backed by WeDB pure-Go storage engine
 type 'help' for commands, 'quit' to exit
`

const helpText = `
Available commands:
  tables              - List all tables
  schema <table>      - Show table schema
  ddl <wql>           - Execute a DDL statement (CREATE/DROP TABLE)
  dml <wql>           - Execute a DML statement (INSERT/UPDATE/DELETE)
  query <wql>         - Execute a SELECT-like query
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
`

func main() {
	if len(os.Args) < 2 {
		printGlobalHelp()
		os.Exit(1)
	}

	// 解析命令行参数
	args := os.Args[1:]
	if len(args) > 0 {
		switch args[0] {
		case "-h", "--help", "help":
			printGlobalHelp()
			return
		case "-v", "--version", "version":
			fmt.Printf("wql-cli version %s\n", version)
			fmt.Println("  backed by WeDB pure-Go storage engine")
			fmt.Println("  uses WQLv3 (fluent Go API, no SQL generation)")
			return
		}
	}

	dbPath := args[0]
	query := ""
	if len(args) >= 2 {
		query = strings.Join(args[1:], " ")
	}

	// 打开 WeDB 数据库
	wedb, err := storage.NewWeDBDatabase(dbPath, 4096)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer wedb.Close()

	// 创建 WQL 适配器（将 WeDB 包装为 wqlv3.Adapter）
	adapter := wqlv3.NewWeDBAdapter(wedb)
	wdb := wqlv3.NewDatabase(adapter)
	defer wdb.Close()

	if query != "" {
		// 单条查询模式
		runQuery(wdb, query)
		return
	}

	// 交互 REPL 模式
	runREPL(wdb, dbPath)
}

func printGlobalHelp() {
	fmt.Printf("wql-cli v%s - WeDB Native Query Language CLI\n\n", version)
	fmt.Println("Usage:")
	fmt.Println("  wql-cli <database.db>              Interactive REPL")
	fmt.Println("  wql-cli <database.db> <wql-query>  Execute one query and exit")
	fmt.Println("  wql-cli --help                      Show this help")
	fmt.Println("  wql-cli --version                   Show version")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println(`  wql-cli test.db 'users.Where(age>18).Select(name).All()'`)
	fmt.Println(`  wql-cli test.db 'users.Count()'`)
	fmt.Println(`  wql-cli test.db`)
	fmt.Println()
	fmt.Println("WQL is a fluent Go API for WeDB. It does NOT generate SQL strings.")
	fmt.Println("All operations call WeDB's native storage API directly via wqlv3.WeDBAdapter.")
}

// runQuery 执行单条 WQL 查询并打印结果
// 优先使用 WQL 无双引号解析器（真正的 WQL 语法）：
//   db.Table(users).Select(name, age).Where(age > 18).All()
//   db.Table(orders).Sum(amount)
//   db.Table(users).Where(name = "alice").First()
//
// 失败时回退到旧的字符串解析器（向后兼容）：
//   T("users").Where("age > 18").All()
func runQuery(wdb *wqlv3.Database, query string) {
	var result wqlv3.QueryResult
	var err error

	// 优先尝试 WQL 无双引号解析器（标准 WQL 语法）
	if strings.HasPrefix(query, "db.") {
		result, err = wqlv3.EvaluateQueryNoQuotes(wdb, query)
	} else {
		// 旧语法: T("table").Where(...).All()
		result, err = wqlv3.EvaluateQuery(wdb, query)
	}

	if err != nil {
		// 回退尝试：如果标准语法失败，尝试旧的字符串接口
		if !strings.HasPrefix(query, "db.") {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		result, err = wqlv3.EvaluateQuery(wdb, query)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}
	wqlv3.PrintResult(result)
}

// runREPL 启动交互式 REPL
func runREPL(wdb *wqlv3.Database, dbPath string) {
	fmt.Printf(banner, version)
	fmt.Printf("  database: %s\n", dbPath)
	fmt.Printf("  backend:  wqlv3 + WeDB native Go storage\n\n")

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 允许 1MB 行

	for {
		fmt.Print("wql> ")
		if !scanner.Scan() {
			fmt.Println()
			break
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		lower := strings.ToLower(line)
		switch {
		case lower == "quit" || lower == "exit" || lower == ":q":
			fmt.Println("Bye!")
			return
		case lower == "help" || lower == "?" || lower == ":h":
			fmt.Print(helpText)
			continue
		case lower == "tables":
			handleTables(wdb)
			continue
		case strings.HasPrefix(lower, "schema "):
			handleSchema(wdb, strings.TrimSpace(line[7:]))
			continue
		case lower == "clear" || lower == "cls":
			clearScreen()
			continue
		}

		// 视为 WQL 查询
		// 优先使用 WQL 无双引号语法：db.Table(users).Select(...).All()
		// 如果不是 db. 开头，回退到旧字符串语法
		start := time.Now()
		var result wqlv3.QueryResult
		var err error
		if strings.HasPrefix(line, "db.") {
			result, err = wqlv3.EvaluateQueryNoQuotes(wdb, line)
		} else {
			result, err = wqlv3.EvaluateQuery(wdb, line)
		}
		if err != nil {
			fmt.Printf("  ERROR: %v\n", err)
			continue
		}
		wqlv3.PrintResult(result)
		fmt.Printf("  %d row(s) in %v\n", countRows(result), time.Since(start))
	}
}

func handleTables(wdb *wqlv3.Database) {
	tables, err := wqlv3.ListTables(wdb)
	if err != nil {
		fmt.Printf("  ERROR: %v\n", err)
		return
	}
	if len(tables) == 0 {
		fmt.Println("  (no tables)")
		return
	}
	fmt.Printf("  Found %d table(s):\n", len(tables))
	for _, t := range tables {
		fmt.Printf("    - %s\n", t)
	}
}

func handleSchema(wdb *wqlv3.Database, tableName string) {
	schema, err := wqlv3.GetSchema(wdb, tableName)
	if err != nil {
		fmt.Printf("  ERROR: %v\n", err)
		return
	}
	fmt.Printf("  Schema for '%s':\n", tableName)
	for _, col := range schema {
		nullable := "NOT NULL"
		if col.Nullable {
			nullable = "NULL"
		}
		fmt.Printf("    %-20s %-12s %s\n", col.Name, col.Type, nullable)
	}
}

func countRows(result wqlv3.QueryResult) int {
	if result.Rows != nil {
		return len(result.Rows)
	}
	if result.Value != nil {
		return 1
	}
	return 0
}

func clearScreen() {
	fmt.Print("\033[H\033[2J")
}
