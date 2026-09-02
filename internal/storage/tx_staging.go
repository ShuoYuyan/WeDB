package storage

import (
	"fmt"

	"github.com/wedb/wedb/internal/api"
)

// 本文件实现“会话级事务”语义：
//   - 手动写事务（writeGate 持有者）期间，所有 DML 被暂存（staged），
//     提交时按序重放到 B-Tree/索引；回滚则直接丢弃。
//   - 读事务按隔离级别取视图：
//       READ UNCOMMITTED -> 可见未提交数据（实时叠加暂存）
//       READ COMMITTED   -> 只见已提交数据
//       REPEATABLE READ / SNAPSHOT -> 首次读取建立快照并复用
//
// 约束：被暂存的表必须有主键（可见性过滤依赖主键签名）。

type rowUpdate struct {
	key     int64
	oldData []byte
	newData []byte
	oldRow  map[string]interface{}
	newRow  map[string]interface{}
}

type rowDelete struct {
	key int64
	row map[string]interface{}
}

type stagedOp struct {
	kind       byte // 'I', 'U', 'D'
	table      string
	row        map[string]interface{} // 'I'
	updates    []rowUpdate            // 'U'
	deletes    []rowDelete            // 'D'
	removedSig map[string]bool        // 可见性隐藏的主键签名
	addedRow   map[string]interface{} // 可见性新增的行
}

// ---------------------------------------------------------------- helpers

// pkOfTable 返回表的主键列名。
func (db *WeDBDatabase) pkOfTable(tableName string) (string, bool) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.pkOfTableLocked(tableName)
}

func (db *WeDBDatabase) pkOfTableLocked(tableName string) (string, bool) {
	table, exists := db.schema.Tables[tableName]
	if !exists {
		return "", false
	}
	if table.PrimaryIndex != nil && len(table.PrimaryIndex.Columns) > 0 {
		return table.PrimaryIndex.Columns[0], true
	}
	for _, col := range table.Columns {
		if col.PrimaryKey {
			return col.Name, true
		}
	}
	return "", false
}

// pkSig 归一化主键值作为可见性签名。
func pkSig(row map[string]interface{}, pk string) string {
	if pk == "" {
		return ""
	}
	v, ok := row[pk]
	if !ok {
		return "\x00missing"
	}
	switch n := v.(type) {
	case int:
		return fmt.Sprintf("i%d", int64(n))
	case int8:
		return fmt.Sprintf("i%d", int64(n))
	case int16:
		return fmt.Sprintf("i%d", int64(n))
	case int32:
		return fmt.Sprintf("i%d", int64(n))
	case int64:
		return fmt.Sprintf("i%d", n)
	case uint:
		return fmt.Sprintf("u%d", uint64(n))
	case uint8:
		return fmt.Sprintf("u%d", uint64(n))
	case uint16:
		return fmt.Sprintf("u%d", uint64(n))
	case uint32:
		return fmt.Sprintf("u%d", uint64(n))
	case uint64:
		return fmt.Sprintf("u%d", n)
	case float32:
		return fmt.Sprintf("f%v", float64(n))
	case float64:
		return fmt.Sprintf("f%v", n)
	case string:
		return "s" + n
	default:
		return fmt.Sprintf("o%v", v)
	}
}

// ------------------------------------------------------------- staging

// stageInsert 将 INSERT 记入当前写事务的暂存区。
func (tx *WeDBTransaction) stageInsert(tableName string, row map[string]interface{}) error {
	pk, _ := tx.db.pkOfTableLocked(tableName)
	if pk == "" {
		return fmt.Errorf("transactional insert requires a primary key on table %s", tableName)
	}
	sig := pkSig(row, pk)
	if sig == "" || sig == "\x00missing" {
		return fmt.Errorf("missing primary key column '%s' in transactional insert", pk)
	}
	if tx.stagedPKs == nil {
		tx.stagedPKs = make(map[string]map[string]bool)
	}
	if tx.stagedPKs[tableName] == nil {
		tx.stagedPKs[tableName] = make(map[string]bool)
	}
	if tx.stagedPKs[tableName][sig] {
		return fmt.Errorf("duplicate primary key %v within transaction", row[pk])
	}
	tx.stagedPKs[tableName][sig] = true

	cp := make(map[string]interface{}, len(row))
	for k, v := range row {
		cp[k] = v
	}
	tx.staged = append(tx.staged, stagedOp{
		kind:     'I',
		table:    tableName,
		row:      cp,
		addedRow: cp,
	})
	return nil
}

// stageUpdates 将 UPDATE 的预计算产物记入暂存区。
func (tx *WeDBTransaction) stageUpdates(tableName string, pk string, ups []rowUpdate) error {
	op := stagedOp{kind: 'U', table: tableName, removedSig: make(map[string]bool)}
	for _, u := range ups {
		op.updates = append(op.updates, u)
		if pk != "" {
			op.removedSig[pkSig(u.oldRow, pk)] = true
		}
	}
	tx.staged = append(tx.staged, op)
	return nil
}

// stageDeletes 将 DELETE 的预计算产物记入暂存区。
func (tx *WeDBTransaction) stageDeletes(tableName string, pk string, dels []rowDelete) error {
	op := stagedOp{kind: 'D', table: tableName, removedSig: make(map[string]bool)}
	for _, d := range dels {
		op.deletes = append(op.deletes, d)
		if pk != "" {
			op.removedSig[pkSig(d.row, pk)] = true
		}
	}
	tx.staged = append(tx.staged, op)
	return nil
}

// ------------------------------------------------------- commit replay

// replayStaged 将暂存操作依序落到已提交存储。任何失败即中止（调用方负责回滚）。
// 调用方持有 db.mu 写锁；执行期间暂时摘除 curWriteTx 防止 DML 再次路由进暂存。
func (tx *WeDBTransaction) replayStaged() error {
	saved := tx.db.curWriteTx
	tx.db.curWriteTx = nil
	defer func() { tx.db.curWriteTx = saved }()

	for _, op := range tx.staged {
		var err error
		switch op.kind {
		case 'I':
			err = tx.db.insertRowCommitted(op.table, op.row)
		case 'U':
			err = tx.db.applyUpdateArtifacts(op.table, op.updates)
		case 'D':
			err = tx.db.applyDeleteArtifacts(op.table, op.deletes)
		default:
			err = fmt.Errorf("unknown staged op kind %c", op.kind)
		}
		if err != nil {
			return fmt.Errorf("replay failed (%c on %s): %w", op.kind, op.table, err)
		}
	}
	return nil
}

// insertRowCommitted 是 InsertRow 的落盘实现（含全部校验）。
// 调用方必须已持有 db.mu 写锁。
func (db *WeDBDatabase) insertRowCommitted(tableName string, row map[string]interface{}) error {
	table, exists := db.schema.Tables[tableName]
	if !exists {
		return fmt.Errorf("table not found: %s", tableName)
	}
	if err := db.validateRow(tableName, row); err != nil {
		return fmt.Errorf("row validation failed: %w", err)
	}

	// 自增列处理
	autoIncrementCol := ""
	for _, col := range table.Columns {
		if col.AutoIncrement {
			autoIncrementCol = col.Name
			break
		}
	}
	for _, col := range table.Columns {
		if col.PrimaryKey {
			if _, ok := row[col.Name]; !ok && !col.AutoIncrement {
				return fmt.Errorf("missing required primary key column: %s", col.Name)
			}
		}
	}
	if autoIncrementCol != "" {
		if _, ok := row[autoIncrementCol]; !ok {
			row[autoIncrementCol] = db.getTableNextRowID(tableName)
		} else if val, ok := row[autoIncrementCol].(int64); ok {
			if val > db.tableRowIDs[tableName] {
				db.tableRowIDs[tableName] = val
			}
		}
	}

	// 列存在性检查
	for colName := range row {
		found := false
		for _, col := range table.Columns {
			if col.Name == colName {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("column '%s' does not exist in table '%s'", colName, tableName)
		}
	}

	rootPage, exists := db.tables[tableName]
	if !exists || rootPage <= 0 {
		return fmt.Errorf("table root page not found or invalid: %s", tableName)
	}
	if db.nextRowID >= 0x7FFFFFFFFFFFFFFF {
		return fmt.Errorf("row ID overflow, cannot insert more rows")
	}
	db.nextRowID++
	rowID := db.nextRowID

	data, err := serializeRow(row, table)
	if err != nil {
		return fmt.Errorf("failed to serialize row: %w", err)
	}
	if len(data) == 0 {
		return fmt.Errorf("serialized row data is empty")
	}
	if len(data) > db.pageSize-20 {
		return fmt.Errorf("row data too large: %d bytes exceeds page limit", len(data))
	}

	tempBTree := db.tableBTreeByRoot(rootPage, tableName)
	if tempBTree == nil {
		return fmt.Errorf("failed to create B-Tree")
	}
	if err := tempBTree.Insert(rowID, data); err != nil {
		return fmt.Errorf("failed to insert row: %w", err)
	}
	if err := db.indexManager.UpdateIndex(tableName, row, "insert"); err != nil {
		if rbErr := tempBTree.Delete(rowID); rbErr != nil {
			return fmt.Errorf("failed to rollback after index update error: %v / %v", err, rbErr)
		}
		return fmt.Errorf("failed to update index: %w", err)
	}
	return nil
}

// applyUpdateArtifacts 将预计算的更新产物落到 B-Tree 与索引。
// 调用方持有 db.mu 写锁。
func (db *WeDBDatabase) applyUpdateArtifacts(tableName string, ups []rowUpdate) error {
	rootPage, exists := db.tables[tableName]
	if !exists || rootPage <= 0 {
		return fmt.Errorf("table root page not found or invalid: %s", tableName)
	}
	schemaTable, exists := db.schema.Tables[tableName]
	if !exists {
		return fmt.Errorf("table not found: %s", tableName)
	}

	tempBTree := db.tableBTreeByRoot(rootPage, tableName)
	if tempBTree == nil {
		return fmt.Errorf("failed to create B-Tree")
	}

	for _, u := range ups {
		if err := tempBTree.Delete(u.key); err != nil {
			return fmt.Errorf("failed to delete old data: %w", err)
		}
	}
	for _, u := range ups {
		if err := tempBTree.Insert(u.key, u.newData); err != nil {
			return fmt.Errorf("failed to insert updated row: %w", err)
		}
	}
	for _, u := range ups {
		newRow := DeserializeRow(u.newData, schemaTable)
		if err := db.indexManager.UpdateIndexWithOldRow(tableName, u.oldRow, newRow, "update"); err != nil {
			return fmt.Errorf("failed to update index: %w", err)
		}
	}
	return nil
}

// applyDeleteArtifacts 将预计算的删除产物落到 B-Tree 与索引。
// 调用方持有 db.mu 写锁。
func (db *WeDBDatabase) applyDeleteArtifacts(tableName string, dels []rowDelete) error {
	rootPage, exists := db.tables[tableName]
	if !exists || rootPage <= 0 {
		return fmt.Errorf("table root page not found or invalid: %s", tableName)
	}
	tempBTree := db.tableBTreeByRoot(rootPage, tableName)
	if tempBTree == nil {
		return fmt.Errorf("failed to create B-Tree")
	}
	for _, d := range dels {
		if err := tempBTree.Delete(d.key); err != nil {
			return fmt.Errorf("failed to delete row: %w", err)
		}
		if err := db.indexManager.UpdateIndex(tableName, d.row, "delete"); err != nil {
			return fmt.Errorf("failed to update index: %w", err)
		}
	}
	return nil
}

// ---------------------------------------------------------- read views

// applyReadViewLocked 根据活动读事务的隔离级别调整扫描结果。
// 调用方持有 db.mu 读锁。
func (db *WeDBDatabase) applyReadViewLocked(tableName string, rows []map[string]interface{}) []map[string]interface{} {
	rt := db.activeReadTx
	if rt == nil {
		return rows
	}

	// REPEATABLE READ / SNAPSHOT：首次读取建立快照，后续复用
	if rt.snapshottable() {
		if cached, ok := rt.cachedSnapshot(tableName); ok {
			return cached
		}
		rt.cacheSnapshot(tableName, rows)
		return rows
	}

	// READ UNCOMMITTED：叠加当前写事务的暂存变更
	if rt.isoLevel == api.LevelReadUncommitted {
		if wtx := db.curWriteTx; wtx != nil {
			rows = wtx.overlayStaged(tableName, rows)
		}
	}
	return rows
}

func (tx *WeDBTransaction) snapshottable() bool {
	return tx.snapshotIso
}

func (tx *WeDBTransaction) isolationLevel() api.IsolationLevel { return tx.isoLevel }

func (tx *WeDBTransaction) cachedSnapshot(table string) ([]map[string]interface{}, bool) {
	tx.snapMu.RLock()
	defer tx.snapMu.RUnlock()
	rows, ok := tx.snapCache[table]
	return rows, ok
}

func (tx *WeDBTransaction) cacheSnapshot(table string, rows []map[string]interface{}) {
	cp := make([]map[string]interface{}, len(rows))
	for i, r := range rows {
		cp[i] = r
	}
	tx.snapMu.Lock()
	if tx.snapCache == nil {
		tx.snapCache = make(map[string][]map[string]interface{})
	}
	tx.snapCache[table] = cp
	tx.snapMu.Unlock()
}

// overlayStaged 把写事务的暂存变更叠加到已提交行集合上（脏读）。
func (tx *WeDBTransaction) overlayStaged(tableName string, rows []map[string]interface{}) []map[string]interface{} {
	pk, _ := tx.db.pkOfTableLocked(tableName)
	out := make([]map[string]interface{}, 0, len(rows))
	for _, r := range rows {
		sig := pkSig(r, pk)
		hidden := false
		for _, op := range tx.staged {
			if op.table != tableName {
				continue
			}
			if op.removedSig[sig] {
				hidden = true
				break
			}
		}
		if !hidden {
			out = append(out, r)
		}
	}
	for _, op := range tx.staged {
		if op.table == tableName && op.kind == 'I' && op.addedRow != nil {
			out = append(out, op.addedRow)
		}
	}
	return out
}
