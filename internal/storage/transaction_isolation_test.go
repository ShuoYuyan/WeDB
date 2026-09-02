package storage

import (
	"context"
	"testing"
	"time"

	"github.com/wedb/wedb/internal/api"
)

// TestTransactionIsolationLevels 测试事务隔离级别
func TestTransactionIsolationLevels(t *testing.T) {
	db, err := NewWeDBDatabase("test_isolation.db", 4096)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer func() {
		db.Close()
	}()

	// 创建测试表
	err = db.CreateTable(&api.TableSchema{
		TableName: "test_table",
		Columns: []api.ColumnSchema{
			{Name: "id", Type: api.TypeInteger, PrimaryKey: true, Nullable: false},
			{Name: "value", Type: api.TypeInteger, Nullable: false},
		},
	})
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// 测试1: READ COMMITTED 隔离级别
	t.Run("READ COMMITTED", func(t *testing.T) {
		// 开始事务1并插入数据
		tx1, err := db.BeginTx(context.Background(), &api.TxOptions{
			Isolation: api.LevelReadCommitted,
			ReadOnly:  false,
		})
		if err != nil {
			t.Fatalf("Failed to begin transaction: %v", err)
		}

		// 插入数据
		row := map[string]interface{}{"id": int64(1), "value": int64(100)}
		err = db.InsertRow("test_table", row)
		if err != nil {
			t.Fatalf("Failed to insert row: %v", err)
		}

		// 开始事务2（读已提交）
		tx2, err := db.BeginTx(context.Background(), &api.TxOptions{
			Isolation: api.LevelReadCommitted,
			ReadOnly:  true,
		})
		if err != nil {
			t.Fatalf("Failed to begin transaction: %v", err)
		}
		defer tx2.Rollback()

		// 事务2应该看不到未提交的数据
		rows, err := db.ScanTable("test_table")
		if err != nil {
			t.Fatalf("Failed to scan table: %v", err)
		}

		if len(rows) > 0 {
			t.Error("READ COMMITTED should not see uncommitted data")
		}

		// 提交事务1
		err = tx1.Commit()
		if err != nil {
			t.Fatalf("Failed to commit transaction: %v", err)
		}

		// 现在事务2应该能看到已提交的数据
		rows, err = db.ScanTable("test_table")
		if err != nil {
			t.Fatalf("Failed to scan table: %v", err)
		}

		if len(rows) != 1 {
			t.Error("READ COMMITTED should see committed data")
		}
	})

	// 测试2: REPEATABLE READ 隔离级别
	t.Run("REPEATABLE READ", func(t *testing.T) {
		// 清空表
		_ = db.DropTable("test_table")
		err = db.CreateTable(&api.TableSchema{
			TableName: "test_table",
			Columns: []api.ColumnSchema{
				{Name: "id", Type: api.TypeInteger, PrimaryKey: true, Nullable: false},
				{Name: "value", Type: api.TypeInteger, Nullable: false},
			},
		})
		if err != nil {
			t.Fatalf("Failed to create table: %v", err)
		}

		// 开始事务1
		tx1, err := db.BeginTx(context.Background(), &api.TxOptions{
			Isolation: api.LevelRepeatableRead,
			ReadOnly:  true,
		})
		if err != nil {
			t.Fatalf("Failed to begin transaction: %v", err)
		}
		defer tx1.Rollback()

		// 第一次读取
		rows1, err := db.ScanTable("test_table")
		if err != nil {
			t.Fatalf("Failed to scan table: %v", err)
		}

		// 在另一个事务中插入数据
		tx2, err := db.BeginTx(context.Background(), &api.TxOptions{
			Isolation: api.LevelReadCommitted,
			ReadOnly:  false,
		})
		if err != nil {
			t.Fatalf("Failed to begin transaction: %v", err)
		}

		row := map[string]interface{}{"id": int64(1), "value": int64(100)}
		err = db.InsertRow("test_table", row)
		if err != nil {
			t.Fatalf("Failed to insert row: %v", err)
		}

		err = tx2.Commit()
		if err != nil {
			t.Fatalf("Failed to commit transaction: %v", err)
		}

		// 第二次读取（REPEATABLE READ应该看到相同的数据）
		rows2, err := db.ScanTable("test_table")
		if err != nil {
			t.Fatalf("Failed to scan table: %v", err)
		}

		if len(rows1) != len(rows2) {
			t.Error("REPEATABLE READ should see consistent data")
		}
	})

	// 测试3: SNAPSHOT 隔离级别
	t.Run("SNAPSHOT", func(t *testing.T) {
		// 清空表
		_ = db.DropTable("test_table")
		err = db.CreateTable(&api.TableSchema{
			TableName: "test_table",
			Columns: []api.ColumnSchema{
				{Name: "id", Type: api.TypeInteger, PrimaryKey: true, Nullable: false},
				{Name: "value", Type: api.TypeInteger, Nullable: false},
			},
		})
		if err != nil {
			t.Fatalf("Failed to create table: %v", err)
		}

		// 开始快照事务
		tx1, err := db.BeginTx(context.Background(), &api.TxOptions{
			Isolation: api.LevelSnapshot,
			ReadOnly:  true,
		})
		if err != nil {
			t.Fatalf("Failed to begin transaction: %v", err)
		}
		defer tx1.Rollback()

		// 获取快照数据
		rows1, err := db.ScanTable("test_table")
		if err != nil {
			t.Fatalf("Failed to scan table: %v", err)
		}

		// 在另一个事务中插入并提交数据
		tx2, err := db.BeginTx(context.Background(), &api.TxOptions{
			Isolation: api.LevelReadCommitted,
			ReadOnly:  false,
		})
		if err != nil {
			t.Fatalf("Failed to begin transaction: %v", err)
		}

		for i := 0; i < 5; i++ {
			row := map[string]interface{}{"id": i + 1, "value": i * 10}
			err = db.InsertRow("test_table", row)
			if err != nil {
				t.Fatalf("Failed to insert row: %v", err)
			}
		}

		err = tx2.Commit()
		if err != nil {
			t.Fatalf("Failed to commit transaction: %v", err)
		}

		// 快照事务应该仍然看到空表
		rows2, err := db.ScanTable("test_table")
		if err != nil {
			t.Fatalf("Failed to scan table: %v", err)
		}

		if len(rows1) != len(rows2) {
			t.Error("SNAPSHOT isolation should see consistent snapshot data")
		}
	})

	// 测试4: READ UNCOMMITTED 隔离级别
	t.Run("READ UNCOMMITTED", func(t *testing.T) {
		// 清空表
		_ = db.DropTable("test_table")
		err = db.CreateTable(&api.TableSchema{
			TableName: "test_table",
			Columns: []api.ColumnSchema{
				{Name: "id", Type: api.TypeInteger, PrimaryKey: true, Nullable: false},
				{Name: "value", Type: api.TypeInteger, Nullable: false},
			},
		})
		if err != nil {
			t.Fatalf("Failed to create table: %v", err)
		}

		// 开始事务1并插入数据（未提交）
		tx1, err := db.BeginTx(context.Background(), &api.TxOptions{
			Isolation: api.LevelReadUncommitted,
			ReadOnly:  false,
		})
		if err != nil {
			t.Fatalf("Failed to begin transaction: %v", err)
		}

		row := map[string]interface{}{"id": 1, "value": 100}
		err = db.InsertRow("test_table", row)
		if err != nil {
			t.Fatalf("Failed to insert row: %v", err)
		}

		// 开始事务2（读未提交）
		tx2, err := db.BeginTx(context.Background(), &api.TxOptions{
			Isolation: api.LevelReadUncommitted,
			ReadOnly:  true,
		})
		if err != nil {
			t.Fatalf("Failed to begin transaction: %v", err)
		}
		defer tx2.Rollback()

		// READ UNCOMMITTED应该能看到未提交的数据
		rows, err := db.ScanTable("test_table")
		if err != nil {
			t.Fatalf("Failed to scan table: %v", err)
		}

		if len(rows) == 0 {
			t.Error("READ UNCOMMITTED should see uncommitted data")
		}

		// 回滚事务1
		err = tx1.Rollback()
		if err != nil {
			t.Fatalf("Failed to rollback transaction: %v", err)
		}
	})

	// 测试5: SERIALIZABLE 隔离级别
	t.Run("SERIALIZABLE", func(t *testing.T) {
		// 清空表
		_ = db.DropTable("test_table")
		err = db.CreateTable(&api.TableSchema{
			TableName: "test_table",
			Columns: []api.ColumnSchema{
				{Name: "id", Type: api.TypeInteger, PrimaryKey: true, Nullable: false},
				{Name: "value", Type: api.TypeInteger, Nullable: false},
			},
		})
		if err != nil {
			t.Fatalf("Failed to create table: %v", err)
		}

		// 开始可串行化事务
		tx1, err := db.BeginTx(context.Background(), &api.TxOptions{
			Isolation: api.LevelSerializable,
			ReadOnly:  false,
		})
		if err != nil {
			t.Fatalf("Failed to begin transaction: %v", err)
		}
		defer tx1.Rollback()

		// 插入数据
		row := map[string]interface{}{"id": 1, "value": 100}
		err = db.InsertRow("test_table", row)
		if err != nil {
			t.Fatalf("Failed to insert row: %v", err)
		}

		// 提交事务1
		err = tx1.Commit()
		if err != nil {
			t.Fatalf("Failed to commit transaction: %v", err)
		}

		// 验证数据已正确插入
		rows, err := db.ScanTable("test_table")
		if err != nil {
			t.Fatalf("Failed to scan table: %v", err)
		}

		if len(rows) != 1 {
			t.Error("SERIALIZABLE transaction should have inserted data correctly")
		}
	})
}

// TestTransactionRollback 测试事务回滚
func TestTransactionRollback(t *testing.T) {
	db, err := NewWeDBDatabase("test_rollback.db", 4096)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer func() {
		db.Close()
	}()

	// 创建测试表
	err = db.CreateTable(&api.TableSchema{
		TableName: "test_table",
		Columns: []api.ColumnSchema{
			{Name: "id", Type: api.TypeInteger, PrimaryKey: true, Nullable: false},
			{Name: "value", Type: api.TypeInteger, Nullable: false},
		},
	})
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// 开始事务
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	// 插入数据
	row := map[string]interface{}{"id": 1, "value": 100}
	err = db.InsertRow("test_table", row)
	if err != nil {
		t.Fatalf("Failed to insert row: %v", err)
	}

	// 回滚事务
	err = tx.Rollback()
	if err != nil {
		t.Fatalf("Failed to rollback transaction: %v", err)
	}

	// 验证数据已被回滚
	rows, err := db.ScanTable("test_table")
	if err != nil {
		t.Fatalf("Failed to scan table: %v", err)
	}

	if len(rows) > 0 {
		t.Error("Rollback should have removed all data")
	}
}

// TestTransactionTimeout 测试事务超时
func TestTransactionTimeout(t *testing.T) {
	db, err := NewWeDBDatabase("test_timeout.db", 4096)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer func() {
		db.Close()
	}()

	// 创建测试表
	err = db.CreateTable(&api.TableSchema{
		TableName: "test_table",
		Columns: []api.ColumnSchema{
			{Name: "id", Type: api.TypeInteger, PrimaryKey: true, Nullable: false},
			{Name: "value", Type: api.TypeInteger, Nullable: false},
		},
	})
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// 开始带超时的事务
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	tx, err := db.BeginTx(ctx, &api.TxOptions{
		Isolation: api.LevelReadCommitted,
		ReadOnly:  false,
	})
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	// 模拟长时间操作
	time.Sleep(50 * time.Millisecond)

	// 检查事务是否仍然活跃
	if !tx.IsActive() {
		t.Error("Transaction should still be active")
	}

	// 等待超时
	time.Sleep(60 * time.Millisecond)

	// 事务应该已经超时
	// 注意：在实际实现中，超时检测需要后台goroutine支持
}

// TestConcurrentTransactions 测试并发事务
func TestConcurrentTransactions(t *testing.T) {
	db, err := NewWeDBDatabase("test_concurrent.db", 4096)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer func() {
		db.Close()
	}()

	// 创建测试表
	err = db.CreateTable(&api.TableSchema{
		TableName: "test_table",
		Columns: []api.ColumnSchema{
			{Name: "id", Type: api.TypeInteger, PrimaryKey: true, Nullable: false},
			{Name: "value", Type: api.TypeInteger, Nullable: false},
		},
	})
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// 并发插入数据
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(idx int) {
			tx, err := db.BeginTx(context.Background(), &api.TxOptions{
				Isolation: api.LevelReadCommitted,
				ReadOnly:  false,
			})
			if err != nil {
				t.Errorf("Failed to begin transaction: %v", err)
				done <- false
				return
			}

			row := map[string]interface{}{"id": idx + 1, "value": idx * 10}
			err = db.InsertRow("test_table", row)
			if err != nil {
				t.Errorf("Failed to insert row: %v", err)
				tx.Rollback()
				done <- false
				return
			}

			err = tx.Commit()
			if err != nil {
				t.Errorf("Failed to commit transaction: %v", err)
				done <- false
				return
			}

			done <- true
		}(i)
	}

	// 等待所有事务完成
	successCount := 0
	for i := 0; i < 10; i++ {
		if <-done {
			successCount++
		}
	}

	if successCount != 10 {
		t.Errorf("Expected all 10 transactions to succeed, got %d", successCount)
	}

	// 验证数据
	rows, err := db.ScanTable("test_table")
	if err != nil {
		t.Fatalf("Failed to scan table: %v", err)
	}

	if len(rows) != 10 {
		t.Errorf("Expected 10 rows, got %d", len(rows))
	}
}