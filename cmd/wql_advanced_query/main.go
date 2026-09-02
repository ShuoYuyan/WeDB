package main

import (
	"fmt"
	"log"

	"github.com/wedb/wedb/WQL/pkg/wql"
	"github.com/wedb/wedb/internal/api"
	"github.com/wedb/wedb/internal/storage"
)

// WQLAdvancedQuery WQL 高级查询示例
func WQLAdvancedQuery() {
	fmt.Println("=== WQL 高级查询示例 ===")

	// 创建数据库
	db, err := storage.NewWeDBDatabase("wql_advanced_test.db", 4096)
	if err != nil {
		log.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// 创建产品表
	schema := &api.TableSchema{
		TableName: "products",
		Columns: []api.ColumnSchema{
			{Name: "id", Type: api.TypeInteger, PrimaryKey: true},
			{Name: "name", Type: api.TypeText},
			{Name: "category", Type: api.TypeText},
			{Name: "price", Type: api.TypeInteger},
			{Name: "stock", Type: api.TypeInteger},
			{Name: "discount", Type: api.TypeInteger},
		},
		PrimaryKey: "id",
	}
	if err := db.CreateTable(schema); err != nil {
		log.Fatalf("Failed to create table: %v", err)
	}

	// 插入产品数据
	products := []map[string]interface{}{
		{"id": int64(1), "name": "Laptop", "category": "Electronics", "price": int64(1000), "stock": int64(50), "discount": int64(10)},
		{"id": int64(2), "name": "Mouse", "category": "Electronics", "price": int64(50), "stock": int64(200), "discount": int64(0)},
		{"id": int64(3), "name": "Keyboard", "category": "Electronics", "price": int64(150), "stock": int64(100), "discount": int64(5)},
		{"id": int64(4), "name": "Monitor", "category": "Electronics", "price": int64(300), "stock": int64(30), "discount": int64(15)},
		{"id": int64(5), "name": "Desk", "category": "Furniture", "price": int64(500), "stock": int64(20), "discount": int64(0)},
		{"id": int64(6), "name": "Chair", "category": "Furniture", "price": int64(200), "stock": int64(40), "discount": int64(10)},
		{"id": int64(7), "name": "Book", "category": "Books", "price": int64(30), "stock": int64(500), "discount": int64(0)},
		{"id": int64(8), "name": "Pen", "category": "Books", "price": int64(5), "stock": int64(1000), "discount": int64(0)},
	}

	for _, product := range products {
		if err := db.InsertRow("products", product); err != nil {
			log.Fatalf("Failed to insert product: %v", err)
		}
	}
	fmt.Printf("✓ 插入了 %d 个产品\n\n", len(products))

	// 示例 1: 简单 WHERE 查询
	fmt.Println("示例 1: 简单 WHERE 查询")
	fmt.Println("------------------------")
	queryAndPrint(db, "products", "category = 'Electronics'")

	// 示例 2: 比较运算符
	fmt.Println("\n示例 2: 比较运算符")
	fmt.Println("------------------------")
	queryAndPrint(db, "products", "price > 100 AND price < 500")

	// 示例 3: 多条件 OR
	fmt.Println("\n示例 3: 多条件 OR")
	fmt.Println("------------------------")
	queryAndPrint(db, "products", "category = 'Electronics' OR category = 'Furniture'")

	// 示例 4: 使用函数
	fmt.Println("\n示例 4: 使用函数")
	fmt.Println("------------------------")
	queryAndPrint(db, "products", "UPPER(category) = 'BOOKS'")

	// 示例 5: 复杂条件
	fmt.Println("\n示例 5: 复杂条件")
	fmt.Println("------------------------")
	queryAndPrint(db, "products", "(category = 'Electronics' AND price < 200) OR (stock > 100)")

	// 示例 6: 算术表达式
	fmt.Println("\n示例 6: 算术表达式")
	fmt.Println("------------------------")
	queryAndPrint(db, "products", "price * (100 - discount) / 100 > 100")

	// 示例 7: 组合运算
	fmt.Println("\n示例 7: 组合运算")
	fmt.Println("------------------------")
	queryAndPrint(db, "products", "category = 'Electronics' AND (price * stock > 10000)")

	// 示例 8: NOT 运算符
	fmt.Println("\n示例 8: NOT 运算符")
	fmt.Println("------------------------")
	queryAndPrint(db, "products", "NOT (discount = 0)")

	// 示例 9: 字符串函数
	fmt.Println("\n示例 9: 字符串函数")
	fmt.Println("------------------------")
	queryAndPrint(db, "products", "LENGTH(name) > 5")

	// 示例 10: 复杂嵌套条件
	fmt.Println("\n示例 10: 复杂嵌套条件")
	fmt.Println("------------------------")
	queryAndPrint(db, "products", "((category = 'Electronics' OR category = 'Furniture') AND discount > 0) OR (stock > 500)")

	fmt.Println("\n=== 所有查询示例完成 ===")
}

// queryAndPrint 查询并打印结果
func queryAndPrint(db *storage.WeDBDatabase, tableName string, whereExpr string) {
	fmt.Printf("WHERE: %s\n", whereExpr)
	
	// 解析 WHERE 表达式
	expr, err := wql.ParseExpressionString(whereExpr)
	if err != nil {
		fmt.Printf("  ❌ 解析失败: %v\n\n", err)
		return
	}

	// 扫描整个表
	allRows, err := db.ScanTable(tableName)
	if err != nil {
		fmt.Printf("  ❌ 查询失败: %v\n\n", err)
		return
	}

	// 使用表达式过滤行
	var filteredRows []map[string]interface{}
	for _, row := range allRows {
		ctx := wql.NewExecutionContext()
		ctx.Row = row
		
		result, err := expr.Evaluate(ctx)
		if err != nil {
			log.Printf("  ⚠️  求值失败 for row %v: %v", row, err)
			continue
		}
		
		if boolResult, ok := result.(bool); ok && boolResult {
			filteredRows = append(filteredRows, row)
		}
	}

	if len(filteredRows) == 0 {
		fmt.Println("  无匹配结果")
	} else {
		fmt.Printf("  找到 %d 条记录:\n", len(filteredRows))
		for _, row := range filteredRows {
			fmt.Printf("    %v\n", row)
		}
	}
	fmt.Println()
}

func main() {
	WQLAdvancedQuery()
}
