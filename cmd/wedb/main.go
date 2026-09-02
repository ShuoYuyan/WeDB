package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/wedb/wedb/internal/api"
	"github.com/wedb/wedb/pkg/adapter"
)

func main() {
	fmt.Println("WeDB - We Database")
	fmt.Println("==================")
	fmt.Println()

	// 打开数据库
	dbFile := "test.db"
	adapter := adapter.NewWeDBAdapter(nil)

	if err := adapter.OpenDatabase(dbFile); err != nil {
		fmt.Printf("Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer adapter.CloseDatabase()

	fmt.Printf("Database opened: %s\n", dbFile)
	fmt.Println("Type 'help' for available commands")
	fmt.Println()

	// 交互式命令行
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("wedb> ")
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("Error reading input: %v\n", err)
			continue
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		if input == "exit" || input == "quit" {
			fmt.Println("Goodbye!")
			break
		}

		// 执行命令
		if err := executeCommand(adapter, input); err != nil {
			fmt.Printf("Error: %v\n", err)
		}
	}
}

func executeCommand(adapter *adapter.WeDBAdapter, input string) error {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return nil
	}

	command := strings.ToLower(parts[0])

	switch command {
	case "help":
		printHelp()
	case "tables":
		return listTables(adapter)
	case "create":
		if len(parts) < 2 {
			return fmt.Errorf("usage: create <table_name>")
		}
		return createTable(adapter, parts[1:])
	case "describe":
		if len(parts) < 2 {
			return fmt.Errorf("usage: describe <table_name>")
		}
		return describeTable(adapter, parts[1])
	case "insert":
		return executeInsert(adapter, parts[1:])
	case "select":
		return executeSelect(adapter, parts[1:])
	case "update":
		return executeUpdate(adapter, parts[1:])
	case "delete":
		return executeDelete(adapter, parts[1:])
	case "count":
		if len(parts) < 2 {
			return fmt.Errorf("usage: count <table_name>")
		}
		return countRows(adapter, parts[1])
	case "stats":
		if len(parts) < 2 {
			return fmt.Errorf("usage: stats <table_name>")
		}
		return showStats(adapter, parts[1])
	case "index":
		return executeIndex(adapter, parts[1:])
	default:
		fmt.Printf("Unknown command: %s\n", command)
		fmt.Println("Type 'help' for available commands")
	}

	return nil
}

func printHelp() {
	fmt.Println("Available commands:")
	fmt.Println("  help                  - Show this help message")
	fmt.Println("  tables                - List all tables")
	fmt.Println("  create <table>        - Create a table")
	fmt.Println("  describe <table>      - Describe table structure")
	fmt.Println("  insert <table> ...    - Insert data into table")
	fmt.Println("  select <table>        - Select data from table")
	fmt.Println("  update <table> ...    - Update data in table")
	fmt.Println("  delete <table> ...    - Delete data from table")
	fmt.Println("  count <table>         - Count rows in table")
	fmt.Println("  stats <table>         - Show table statistics")
	fmt.Println("  index <table> ...     - Index operations")
	fmt.Println("  exit / quit           - Exit the program")
}

func listTables(adapter *adapter.WeDBAdapter) error {
	tables := adapter.ListTables()
	if len(tables) == 0 {
		fmt.Println("No tables found")
		return nil
	}

	fmt.Println("Tables:")
	for _, table := range tables {
		fmt.Printf("  - %s\n", table)
	}
	return nil
}

func createTable(adapter *adapter.WeDBAdapter, parts []string) error {
	tableName := parts[0]

	schema := &api.TableSchema{
		TableName: tableName,
		Columns: []api.ColumnSchema{
			{
				Name: "id",
				Type: api.TypeInteger,
			},
			{
				Name: "name",
				Type: api.TypeText,
			},
			{
				Name: "value",
				Type: api.TypeInteger,
			},
		},
		PrimaryKey: "id",
	}

	if err := adapter.CreateTable(schema); err != nil {
		return err
	}

	fmt.Printf("Table '%s' created successfully\n", tableName)
	return nil
}

func describeTable(adapter *adapter.WeDBAdapter, tableName string) error {
	schema, err := adapter.GetTableSchema(tableName)
	if err != nil {
		return err
	}

	fmt.Printf("Table: %s\n", schema.TableName)
	fmt.Println("Columns:")
	for _, col := range schema.Columns {
		primaryKey := ""
		if col.Name == schema.PrimaryKey {
			primaryKey = " (PRIMARY KEY)"
		}
		fmt.Printf("  - %s: %s%s\n", col.Name, col.Type, primaryKey)
	}

	return nil
}

func executeInsert(adapter *adapter.WeDBAdapter, parts []string) error {
	if len(parts) < 3 {
		return fmt.Errorf("usage: insert <table> <name> <value>")
	}

	tableName := parts[0]
	name := parts[1]
	value := parts[2]

	row := map[string]interface{}{
		"id":    0,
		"name":  name,
		"value": value,
	}

	if err := adapter.InsertRow(tableName, row); err != nil {
		return err
	}

	fmt.Printf("Row inserted into '%s'\n", tableName)
	return nil
}

func executeSelect(adapter *adapter.WeDBAdapter, parts []string) error {
	if len(parts) < 1 {
		return fmt.Errorf("usage: select <table>")
	}

	tableName := parts[0]

	rows, err := adapter.ScanTable(tableName)
	if err != nil {
		return err
	}

	if len(rows) == 0 {
		fmt.Println("No rows found")
		return nil
	}

	fmt.Printf("Results from '%s':\n", tableName)
	for i, row := range rows {
		if i == 0 {
			headers := make([]string, 0, len(row))
			for key := range row {
				headers = append(headers, key)
			}
			fmt.Println(headers)
			fmt.Println(strings.Repeat("-", 50))
		}
		values := make([]string, 0, len(row))
		for _, val := range row {
			values = append(values, fmt.Sprintf("%v", val))
		}
		fmt.Println(values)
	}

	fmt.Printf("\n%d row(s) returned\n", len(rows))
	return nil
}

func executeUpdate(adapter *adapter.WeDBAdapter, parts []string) error {
	if len(parts) < 3 {
		return fmt.Errorf("usage: update <table> <name> <new_value>")
	}

	tableName := parts[0]
	value := parts[2]

	row := map[string]interface{}{
		"value": value,
	}

	if err := adapter.UpdateRow(tableName, row, "*"); err != nil {
		return err
	}

	fmt.Printf("Row(s) updated in '%s'\n", tableName)
	return nil
}

func executeDelete(adapter *adapter.WeDBAdapter, parts []string) error {
	if len(parts) < 1 {
		return fmt.Errorf("usage: delete <table>")
	}

	tableName := parts[0]

	if err := adapter.DeleteRow(tableName, "*"); err != nil {
		return err
	}

	fmt.Printf("Row(s) deleted from '%s'\n", tableName)
	return nil
}

func countRows(adapter *adapter.WeDBAdapter, tableName string) error {
	count, err := adapter.Count(tableName, "")
	if err != nil {
		return err
	}

	fmt.Printf("Table '%s' has %d row(s)\n", tableName, count)
	return nil
}

func showStats(adapter *adapter.WeDBAdapter, tableName string) error {
	stats, err := adapter.GetTableStats(tableName)
	if err != nil {
		return err
	}

	fmt.Printf("Statistics for '%s':\n", tableName)
	fmt.Printf("  Row Count: %d\n", stats.RowCount)
	fmt.Printf("  Index Count: %d\n", stats.IndexCount)
	fmt.Printf("  Column Count: %d\n", stats.ColumnCount)
	fmt.Printf("  Table Size: %d bytes\n", stats.TableSize)
	fmt.Printf("  Last Modified: %s\n", stats.LastModified)
	fmt.Printf("  Created: %s\n", stats.Created)

	return nil
}

func executeIndex(adapter *adapter.WeDBAdapter, parts []string) error {
	if len(parts) < 2 {
		return fmt.Errorf("usage: index <table> <create|list|drop> [...]")
	}

	tableName := parts[0]
	action := strings.ToLower(parts[1])

	switch action {
	case "create":
		if len(parts) < 3 {
			return fmt.Errorf("usage: index <table> create <index_name>")
		}
		return createIndex(adapter, tableName, parts[2:])
	case "list":
		return listIndexes(adapter, tableName)
	case "drop":
		if len(parts) < 3 {
			return fmt.Errorf("usage: index <table> drop <index_name>")
		}
		return dropIndex(adapter, tableName, parts[2:])
	default:
		return fmt.Errorf("unknown index action: %s", action)
	}
}

func createIndex(adapter *adapter.WeDBAdapter, tableName string, parts []string) error {
	indexName := parts[0]

	index := &api.IndexInfo{
		IndexName: indexName,
		Columns:   []string{"name"},
		Unique:    false,
		Type:      api.TypeBTree,
	}

	if err := adapter.CreateIndex(tableName, index); err != nil {
		return err
	}

	fmt.Printf("Index '%s' created on table '%s'\n", indexName, tableName)
	return nil
}

func listIndexes(adapter *adapter.WeDBAdapter, tableName string) error {
	indexes, err := adapter.GetIndexInfo(tableName)
	if err != nil {
		return err
	}

	if len(indexes) == 0 {
		fmt.Printf("No indexes found on table '%s'\n", tableName)
		return nil
	}

	fmt.Printf("Indexes on '%s':\n", tableName)
	for _, idx := range indexes {
		fmt.Printf("  - %s (%s) on columns: %v\n", idx.IndexName, idx.Type, idx.Columns)
	}

	return nil
}

func dropIndex(adapter *adapter.WeDBAdapter, tableName string, parts []string) error {
	indexName := parts[0]

	if err := adapter.DropIndex(tableName, indexName); err != nil {
		return err
	}

	fmt.Printf("Index '%s' dropped from table '%s'\n", indexName, tableName)
	return nil
}