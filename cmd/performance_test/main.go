package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/wedb/wedb/internal/api"
	"github.com/wedb/wedb/internal/storage"
	"github.com/wedb/wedb/pkg/adapter"
)

type PerformanceResult struct {
	Operation string
	Count     int
	Duration  time.Duration
	Success   bool
	Error     string
}

func main() {
	testSizes := []int{10000, 100000}
	testRuns := 5

	fmt.Println("=== WeDB 性能测试 ===")

	allResults := make(map[int][]PerformanceResult)

	for _, size := range testSizes {
		fmt.Printf("========== 测试数据量: %d ==========\n\n", size)
		sizeResults := make([]PerformanceResult, 0, testRuns*4) // 增删改查各5次

		for run := 1; run <= testRuns; run++ {
			fmt.Printf("--- 第 %d 次测试 ---\n", run)
			dbFile := fmt.Sprintf("perf_test_%d_run%d.db", size, run)

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
					{Name: "data", Type: api.TypeText},
				},
			}

			err = db.CreateTable(schema)
			if err != nil {
				log.Fatalf("Failed to create table: %v", err)
			}

			// 测试插入
			insertResult := testInsert(db, size, dbFile)
			sizeResults = append(sizeResults, insertResult)

			// 测试查询
			if insertResult.Success {
				readResult := testRead(db, size, dbFile)
				sizeResults = append(sizeResults, readResult)
			}

			// 测试更新
			if insertResult.Success {
				updateResult := testUpdate(db, size, dbFile)
				sizeResults = append(sizeResults, updateResult)
			}

			// 测试删除
			if insertResult.Success {
				deleteResult := testDelete(db, size, dbFile)
				sizeResults = append(sizeResults, deleteResult)
			}

			// 关闭数据库
			wedbDB.Close()

			// 清理
			os.Remove(dbFile)
			os.Remove(dbFile + ".metadata")

			fmt.Println()
		}

		allResults[size] = sizeResults
	}

	// 生成报告
	generateReport(allResults, testSizes, testRuns)
}

func testInsert(db *adapter.WeDBAdapter, count int, dbFile string) PerformanceResult {
	fmt.Printf("  插入 %d 条数据...", count)

	rows := make([]map[string]interface{}, count)
	for i := 0; i < count; i++ {
		rows[i] = map[string]interface{}{
			"name":  fmt.Sprintf("Item%d", i),
			"value": i * 10,
			"data":  fmt.Sprintf("Data for item %d", i),
		}
	}

	start := time.Now()
	err := db.InsertRows("test_table", rows)
	duration := time.Since(start)

	result := PerformanceResult{
		Operation: "INSERT",
		Count:     count,
		Duration:  duration,
		Success:   err == nil,
	}

	if err != nil {
		result.Error = err.Error()
		fmt.Printf(" 失败: %v\n", err)
	} else {
		fmt.Printf(" 成功 (耗时: %v, 速度: %.2f 条/秒)\n", duration, float64(count)/duration.Seconds())
	}

	return result
}

func testRead(db *adapter.WeDBAdapter, count int, dbFile string) PerformanceResult {
	fmt.Printf("  查询 %d 条数据...", count)

	start := time.Now()
	rows, err := db.ScanTable("test_table")
	duration := time.Since(start)

	result := PerformanceResult{
		Operation: "SELECT",
		Count:     count,
		Duration:  duration,
		Success:   err == nil,
	}

	if err != nil {
		result.Error = err.Error()
		fmt.Printf(" 失败: %v\n", err)
	} else {
		if len(rows) != count {
			result.Success = false
			result.Error = fmt.Sprintf("查询结果数量不匹配: 期望 %d, 实际 %d", count, len(rows))
			fmt.Printf(" 失败: %v\n", result.Error)
		} else {
			fmt.Printf(" 成功 (耗时: %v, 速度: %.2f 条/秒)\n", duration, float64(count)/duration.Seconds())
		}
	}

	return result
}

func testUpdate(db *adapter.WeDBAdapter, count int, dbFile string) PerformanceResult {
	updateCount := count / 10 // 更新10%的数据
	if updateCount < 1 {
		updateCount = 1
	}

	fmt.Printf("  更新 %d 条数据...", updateCount)

	start := time.Now()
	for i := 0; i < updateCount; i++ {
		row := map[string]interface{}{
			"value": i * 100,
			"data":  fmt.Sprintf("Updated data for item %d", i),
		}
		err := db.UpdateRow("test_table", row, "id = ?")
		if err != nil {
			duration := time.Since(start)
			return PerformanceResult{
				Operation: "UPDATE",
				Count:     updateCount,
				Duration:  duration,
				Success:   false,
				Error:     err.Error(),
			}
		}
	}
	duration := time.Since(start)

	result := PerformanceResult{
		Operation: "UPDATE",
		Count:     updateCount,
		Duration:  duration,
		Success:   true,
	}

	fmt.Printf(" 成功 (耗时: %v, 速度: %.2f 条/秒)\n", duration, float64(updateCount)/duration.Seconds())

	return result
}

func testDelete(db *adapter.WeDBAdapter, count int, dbFile string) PerformanceResult {
	deleteCount := count / 10 // 删除10%的数据
	if deleteCount < 1 {
		deleteCount = 1
	}

	fmt.Printf("  删除 %d 条数据...", deleteCount)

	start := time.Now()
	for i := 0; i < deleteCount; i++ {
		err := db.DeleteRow("test_table", "id = ?")
		if err != nil {
			duration := time.Since(start)
			return PerformanceResult{
				Operation: "DELETE",
				Count:     deleteCount,
				Duration:  duration,
				Success:   false,
				Error:     err.Error(),
			}
		}
	}
	duration := time.Since(start)

	result := PerformanceResult{
		Operation: "DELETE",
		Count:     deleteCount,
		Duration:  duration,
		Success:   true,
	}

	fmt.Printf(" 成功 (耗时: %v, 速度: %.2f 条/秒)\n", duration, float64(deleteCount)/duration.Seconds())

	return result
}

func generateReport(results map[int][]PerformanceResult, sizes []int, runs int) {
	fmt.Println("\n========== 性能测试报告 ==========")

	for _, size := range sizes {
		sizeResults := results[size]
		if len(sizeResults) == 0 {
			continue
		}

		fmt.Printf("## 数据量: %d 条\n\n", size)

		// 按操作类型分组
		operations := map[string][]PerformanceResult{
			"INSERT": make([]PerformanceResult, 0),
			"SELECT": make([]PerformanceResult, 0),
			"UPDATE": make([]PerformanceResult, 0),
			"DELETE": make([]PerformanceResult, 0),
		}

		for _, result := range sizeResults {
			operations[result.Operation] = append(operations[result.Operation], result)
		}

		// 生成每个操作的报告
		for _, op := range []string{"INSERT", "SELECT", "UPDATE", "DELETE"} {
			opResults := operations[op]
			if len(opResults) == 0 {
				continue
			}

			fmt.Printf("### %s\n\n", op)

			var totalDuration time.Duration
			var successCount int
			var failures []string

			for _, result := range opResults {
				if result.Success {
					totalDuration += result.Duration
					successCount++
				} else {
					failures = append(failures, result.Error)
				}
			}

			if successCount > 0 {
				avgDuration := totalDuration / time.Duration(successCount)
				avgSpeed := float64(opResults[0].Count) / avgDuration.Seconds()

				fmt.Printf("- 测试次数: %d\n", len(opResults))
				fmt.Printf("- 成功次数: %d\n", successCount)
				fmt.Printf("- 平均耗时: %v\n", avgDuration)
				fmt.Printf("- 平均速度: %.2f 条/秒\n", avgSpeed)
				fmt.Printf("- 吞吐量: %.2f 条/秒\n", avgSpeed)

				// 最小和最大耗时
				minDuration := opResults[0].Duration
				maxDuration := opResults[0].Duration
				for _, result := range opResults {
					if result.Success {
						if result.Duration < minDuration {
							minDuration = result.Duration
						}
						if result.Duration > maxDuration {
							maxDuration = result.Duration
						}
					}
				}
				fmt.Printf("- 最小耗时: %v\n", minDuration)
				fmt.Printf("- 最大耗时: %v\n", maxDuration)
			}

			if len(failures) > 0 {
				fmt.Printf("- 失败次数: %d\n", len(failures))
				for i, failure := range failures {
					if i < 3 { // 只显示前3个错误
						fmt.Printf("  - %s\n", failure)
					}
				}
				if len(failures) > 3 {
					fmt.Printf("  - ... 还有 %d 个错误\n", len(failures)-3)
				}
			}

			fmt.Println()
		}

		fmt.Println("---")
	}

	fmt.Println("测试完成！")
}
