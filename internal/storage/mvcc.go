package storage

import (
	"fmt"
	"sync"
	"time"

	"github.com/wedb/wedb/internal/api"
)

// PageSnapshot 页面快照
type PageSnapshot struct {
	pageNum    int
	data       []byte
	cells      []*CellSnapshot
	timestamp  time.Time
	txID       int64
}

// CellSnapshot 单元格快照
type CellSnapshot struct {
	key        int64
	value      []byte
	timestamp  time.Time
	txID       int64
	visible    bool
}

// MVCell 多版本单元格
// 支持多个版本的数据
type MVCell struct {
	key         int64
	versions    []*CellVersion
	mu          sync.RWMutex
}

// CellVersion 单元格版本
type CellVersion struct {
	value      []byte
	startTxID  int64 // 开始可见的事务ID
	endTxID    int64 // 结束可见的事务ID（0表示未结束）
	timestamp  time.Time
	isCommitted bool
}

// TransactionManager 事务管理器
type TransactionManager struct {
	nextTxID    int64                // 下一个事务ID
	transactions map[int64]*TransactionInfo // 活跃事务
	mvCells     map[int64]*MVCell   // 多版本单元格映射
	snapshots   map[int64]*dbSnapshot // 活跃快照
	mu          sync.RWMutex
}

// TransactionInfo 事务信息
type TransactionInfo struct {
	txID        int64
	isolation   api.IsolationLevel
	isReadOnly  bool
	beginTime   time.Time
	state       TxState
}

// TxState 事务状态
type TxState int

const (
	TxActive TxState = iota
	TxCommitted
	TxRolledBack
)

// NewTransactionManager 创建新的事务管理器
func NewTransactionManager() *TransactionManager {
	return &TransactionManager{
		nextTxID:     1,
		transactions: make(map[int64]*TransactionInfo),
		mvCells:      make(map[int64]*MVCell),
		snapshots:    make(map[int64]*dbSnapshot),
	}
}

// BeginTransaction 开始新事务
func (tm *TransactionManager) BeginTransaction(isolation api.IsolationLevel, readOnly bool) int64 {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	txID := tm.nextTxID
	tm.nextTxID++

	txInfo := &TransactionInfo{
		txID:       txID,
		isolation:  isolation,
		isReadOnly: readOnly,
		beginTime:  time.Now(),
		state:      TxActive,
	}

	tm.transactions[txID] = txInfo

	// 对于快照隔离级别，创建数据库快照
	if isolation == api.LevelSnapshot || isolation == api.LevelRepeatableRead {
		tm.createSnapshot(txID)
	}

	return txID
}

// createSnapshot 创建数据库快照
func (tm *TransactionManager) createSnapshot(txID int64) *dbSnapshot {
	snapshot := &dbSnapshot{
		timestamp: time.Now(),
		txID:      txID,
		pages:     make(map[int]*PageSnapshot),
		valid:     true,
	}

	tm.snapshots[txID] = snapshot
	return snapshot
}

// Commit 提交事务
func (tm *TransactionManager) Commit(txID int64) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	txInfo, exists := tm.transactions[txID]
	if !exists {
		return fmt.Errorf("transaction not found: %d", txID)
	}

	if txInfo.state != TxActive {
		return fmt.Errorf("transaction not active: %d", txID)
	}

	// 标记所有修改的数据为已提交
	for _, mvCell := range tm.mvCells {
		mvCell.mu.Lock()
		for _, version := range mvCell.versions {
			if version.startTxID == txID {
				version.isCommitted = true
			}
		}
		mvCell.mu.Unlock()
	}

	txInfo.state = TxCommitted

	// 清理快照
	delete(tm.snapshots, txID)

	// 延迟删除事务信息（用于后续的可见性检查）
	go func() {
		time.Sleep(5 * time.Second)
		tm.mu.Lock()
		delete(tm.transactions, txID)
		tm.mu.Unlock()
	}()

	return nil
}

// Rollback 回滚事务
func (tm *TransactionManager) Rollback(txID int64) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	txInfo, exists := tm.transactions[txID]
	if !exists {
		return fmt.Errorf("transaction not found: %d", txID)
	}

	if txInfo.state != TxActive {
		return fmt.Errorf("transaction not active: %d", txID)
	}

	// 标记所有修改的数据为已回滚
	for _, mvCell := range tm.mvCells {
		mvCell.mu.Lock()
		for _, version := range mvCell.versions {
			if version.startTxID == txID {
				version.endTxID = txID
			}
		}
		mvCell.mu.Unlock()
	}

	txInfo.state = TxRolledBack

	// 清理快照
	delete(tm.snapshots, txID)

	// 立即删除事务信息
	delete(tm.transactions, txID)

	return nil
}

// IsVisible 检查数据版本是否对事务可见
func (tm *TransactionManager) IsVisible(txID int64, version *CellVersion) bool {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	txInfo, exists := tm.transactions[txID]
	if !exists {
		return false
	}

	// 如果是当前事务自己的修改，总是可见
	if version.startTxID == txID {
		return true
	}

	// 如果版本未提交，不可见（除了当前事务）
	if !version.isCommitted {
		return false
	}

	// 根据隔离级别判断可见性
	switch txInfo.isolation {
	case api.LevelReadUncommitted:
		// 读未提交：可见所有版本，包括未提交的
		return true

	case api.LevelReadCommitted:
		// 读已提交：只可见已提交的版本
		return version.isCommitted

	case api.LevelRepeatableRead, api.LevelSnapshot:
		// 可重复读/快照隔离：只可见快照之前提交的版本
		snapshot, hasSnapshot := tm.snapshots[txID]
		if !hasSnapshot {
			return version.isCommitted
		}
		return version.timestamp.Before(snapshot.timestamp) || version.timestamp.Equal(snapshot.timestamp)

	case api.LevelSerializable:
		// 可串行化：最严格的隔离级别
		// 检查是否有冲突
		return tm.isSerializable(txID, version)

	default:
		return version.isCommitted
	}
}

// isSerializable 检查是否符合可串行化隔离级别
func (tm *TransactionManager) isSerializable(txID int64, version *CellVersion) bool {
	// 可串行化要求：所有事务按照某种顺序执行
	// 简化实现：检查版本是否在事务开始之前提交
	txInfo, exists := tm.transactions[txID]
	if !exists {
		return false
	}

	return version.timestamp.Before(txInfo.beginTime) || version.isCommitted && version.startTxID < txID
}

// GetVisibleVersion 获取对事务可见的数据版本
func (tm *TransactionManager) GetVisibleVersion(txID int64, mvCell *MVCell) *CellVersion {
	mvCell.mu.RLock()
	defer mvCell.mu.RUnlock()

	// 从新到旧遍历版本，找到第一个可见的版本
	for i := len(mvCell.versions) - 1; i >= 0; i-- {
		version := mvCell.versions[i]
		if tm.IsVisible(txID, version) {
			return version
		}
	}

	return nil
}

// AddVersion 添加新的数据版本
func (tm *TransactionManager) AddVersion(txID int64, key int64, value []byte) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	mvCell, exists := tm.mvCells[key]
	if !exists {
		mvCell = &MVCell{
			key:      key,
			versions: make([]*CellVersion, 0),
		}
		tm.mvCells[key] = mvCell
	}

	mvCell.mu.Lock()
	defer mvCell.mu.Unlock()

	version := &CellVersion{
		value:       value,
		startTxID:   txID,
		endTxID:     0,
		timestamp:   time.Now(),
		isCommitted: false,
	}

	mvCell.versions = append(mvCell.versions, version)
}

// CleanupOldVersions 清理旧版本的数据
func (tm *TransactionManager) CleanupOldVersions() {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// 获取最小的活跃事务ID
	minActiveTxID := tm.nextTxID
	for txID := range tm.transactions {
		if txID < minActiveTxID {
			minActiveTxID = txID
		}
	}

	// 清理不再需要的版本
	for _, mvCell := range tm.mvCells {
		mvCell.mu.Lock()
		newVersions := make([]*CellVersion, 0)
		for _, version := range mvCell.versions {
			// 保留最新的版本或活跃事务可见的版本
			if version.startTxID >= minActiveTxID || version.endTxID == 0 {
				newVersions = append(newVersions, version)
			}
		}
		mvCell.versions = newVersions
		mvCell.mu.Unlock()
	}
}