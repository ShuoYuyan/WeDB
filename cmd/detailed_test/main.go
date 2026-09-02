package main

import (
	"fmt"
	"log"
	"os"

	"github.com/wedb/wedb/internal/api"
	"github.com/wedb/wedb/internal/storage"
	"github.com/wedb/wedb/pkg/adapter"
)

func main() {
	testSizes := []int{1000, 10000}
	testRuns := 5

	fmt.Println("=== 详细数据验证测试 ===")

	for _, size := range testSizes {
		fmt.Printf("========== 测试数据量: %d ==========\n\n", size)

		for run := 1; run <= testRuns; run++ {
			fmt.Printf("--- 第 %d 次测试 ---\n", run)
			dbFile := fmt.Sprintf("detailed_test_%d_run%d.db", size, run)

			// 清理旧数据
			os.Remove(dbFile)
			os.Remove(dbFile + ".metadata")

			// 创建数据库
			wedbDB, err := storage.NewWeDBDatabase(dbFile, 4096)
			if err != nil {
				log.Fatalf("Failed to create database: %v", err)
			}
			db := adapter.NewWeDBAdapter(wedbDB)

			// 创建表
			schema := &api.TableSchema{
				TableName: "test_table",
				Columns: []api.ColumnSchema{
					{Name: "id", Type: api.TypeInteger, PrimaryKey: true, AutoIncrement: true},
					{Name: "name", Type: api.TypeText},
					{Name: "value", Type: api.TypeInteger},
				},
			}

			err = db.CreateTable(schema)
			if err != nil {
				log.Fatalf("Failed to create table: %v", err)
			}

			// 测试插入
			inserted := testInsertWithVerify(db, size, dbFile)
			if !inserted {
				fmt.Printf("  ❌ 插入测试失败\n")
				wedbDB.Close()
				continue
			}

			// 测试查询
			if !testReadWithVerify(db, size, dbFile) {
				fmt.Printf("  ❌ 查询测试失败\n")
				wedbDB.Close()
				continue
			}

			// 测试更新
			if !testUpdateWithVerify(db, size, dbFile) {
				fmt.Printf("  ❌ 更新测试失败\n")
				wedbDB.Close()
				continue
			}

			// 测试删除
			if !testDeleteWithVerify(db, size, dbFile) {
				fmt.Printf("  ❌ 删除测试失败\n")
				wedbDB.Close()
				continue
			}

			// 关闭数据库
			wedbDB.Close()

			// 清理
			os.Remove(dbFile)
			os.Remove(dbFile + ".metadata")

			fmt.Println("  ✓ 所有测试通过")
		}
	}

	fmt.Println("=== 测试完成 ===")
}

func testInsertWithVerify(db *adapter.WeDBAdapter, count int, dbFile string) bool {
	fmt.Printf("  1. 插入 %d 条数据...", count)

	rows := make([]map[string]interface{}, count)
	for i := 0; i < count; i++ {
		rows[i] = map[string]interface{}{
			"name":  fmt.Sprintf("Item%d", i),
			"value": i,
		}
	}

	err := db.InsertRows("test_table", rows)
	if err != nil {
		fmt.Printf(" 失败: %v\n", err)
		return false
	}

	// 验证插入的数据
	scanRows, err := db.ScanTable("test_table")
	if err != nil {
		fmt.Printf(" 验证失败: %v\n", err)
		return false
	}

	if len(scanRows) != count {
		fmt.Printf(" 验证失败: 期望 %d 条，实际 %d 条\n", count, len(scanRows))
		return false
	}

	fmt.Printf(" 成功\n")
	return true
}

func testReadWithVerify(db *adapter.WeDBAdapter, count int, dbFile string) bool {
	fmt.Printf("  2. 查询 %d 条数据...", count)

	rows, err := db.ScanTable("test_table")
	if err != nil {
		fmt.Printf(" 失败: %v\n", err)
		return false
	}

	if len(rows) != count {
		fmt.Printf(" 验证失败: 期望 %d 条，实际 %d 条\n", count, len(rows))
		return false
	}

	// 验证数据内容
	for i := 0; i < count && i < 10; i++ {
		if rows[i]["name"] != fmt.Sprintf("Item%d", i) {
			fmt.Printf(" 验证失败: 数据不正确\n")
			return false
		}
	}

	fmt.Printf(" 成功\n")
	return true
}

func testUpdateWithVerify(db *adapter.WeDBAdapter, count int, dbFile string) bool {
	updateCount := count / 10
	if updateCount < 1 {
		updateCount = 1
	}

	fmt.Printf("  3. 更新 %d 条数据...", updateCount)

	for i := 0; i < updateCount; i++ {
		row := map[string]interface{}{
			"value": i * 100,
		}
		err := db.UpdateRow("test_table", row, fmt.Sprintf("id = %d", i+1))
		if err != nil {
			fmt.Printf(" 失败: %v\n", err)
			return false
		}
	}

	// 验证更新后的数据
	rows, err := db.ScanTable("test_table")
	if err != nil {
		fmt.Printf(" 验证失败: %v\n", err)
		return false
	}

	for i := 0; i < updateCount; i++ {
		expectedValue := int64(i * 100)
		if rows[i]["value"] != expectedValue {
			fmt.Printf(" 验证失败: 更新不正确\n")
			return false
		}
	}

	fmt.Printf(" 成功\n")
	return true
}

func testDeleteWithVerify(db *adapter.WeDBAdapter, count int, dbFile string) bool {
	deleteCount := count / 10
	if deleteCount < 1 {
		deleteCount = 1
	}

	fmt.Printf("  4. 删除 %d 条数据...", deleteCount)

	for i := 0; i < deleteCount; i++ {
		err := db.DeleteRow("test_table", fmt.Sprintf("id = %d", i+1))
		if err != nil {
			fmt.Printf(" 失败: %v\n", err)
			return false
		}
	}

	// 验证删除后的数据
	rows, err := db.ScanTable("test_table")
	if err != nil {
		fmt.Printf(" 验证失败: %v\n", err)
		return false
	}

	expectedCount := count - deleteCount
	if len(rows) != expectedCount {
		fmt.Printf(" 验证失败: 期望 %d 条，实际 %d 条\n", expectedCount, len(rows))
		return false
	}

	fmt.Printf(" 成功\n")
	return true
}
