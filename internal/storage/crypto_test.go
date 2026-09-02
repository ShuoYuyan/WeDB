package storage

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wedb/wedb/internal/api"
	"github.com/wedb/wedb/internal/util"
)

// TestMain 之外，加密测试额外释放全局日志句柄以便 TempDir 清理
func releaseLogs() { _ = util.CloseLogging() }

// newTempDir 双工具链兼容的临时目录（Go1.10 无 t.TempDir/t.Cleanup）
func newTempDir(t *testing.T) (string, func()) {
	d, err := ioutil.TempDir("", "wedb-enc-")
	if err != nil {
		t.Fatal(err)
	}
	return d, func() { os.RemoveAll(d) }
}
// TestEncryptionRoundtrip 加密库写入→关闭→正确口令重开→数据完整
func TestEncryptionRoundtrip(t *testing.T) {
	dir, dirCleanup := newTempDir(t)
	defer func() { dirCleanup() }()
	dbPath := filepath.Join(dir, "enc.db")
	pass := []byte("correct-horse-battery-staple")

	marker := "TOP-SECRET-公司机密数据-1234567890"
	func() {
		db, err := NewWeDBDatabaseSecure(dbPath, 4096, pass)
		if err != nil {
			t.Fatalf("open secure: %v", err)
		}
		defer db.Close()
		if err := db.CreateTable(&api.TableSchema{
			TableName: "secrets",
			Columns: []api.ColumnSchema{
				{Name: "id", Type: api.TypeInteger, PrimaryKey: true},
				{Name: "payload", Type: api.TypeText},
			},
		}); err != nil {
			t.Fatalf("create table: %v", err)
		}
		for i := 1; i <= 50; i++ {
			row := map[string]interface{}{
				"id":      int64(i),
				"payload": marker + strings.Repeat("x", i*10),
			}
			if err := db.InsertRow("secrets", row); err != nil {
				t.Fatalf("insert %d: %v", i, err)
			}
		}
	}()

	// 原始文件不应包含明文标记
	raw, err := os.ReadFile(dbPath)
	if err != nil {
		t.Fatalf("read raw: %v", err)
	}
	if bytes.Contains(raw, []byte(marker)) {
		t.Fatal("plaintext marker found in encrypted database file")
	}

	// 错误口令必须失败
	if _, err := NewWeDBDatabaseSecure(dbPath, 4096, []byte("wrong")); err == nil {
		t.Fatal("wrong passphrase should fail")
	} else if !IsWrongPassphrase(err) {
		t.Fatalf("expected ErrWrongPassphrase, got: %v", err)
	}

	// 无口令打开已加密库必须失败
	if _, err := NewWeDBDatabase(dbPath, 4096); err == nil {
		t.Fatal("opening encrypted db without passphrase should fail")
	}

	// 正确口令重开，数据完整
	releaseLogs()
	db2, err := NewWeDBDatabaseSecure(dbPath, 4096, pass)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer db2.Close()

	rows, err := db2.ScanTable("secrets")
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(rows) != 50 {
		t.Fatalf("expected 50 rows, got %d", len(rows))
	}
	first := rows[0]["payload"].(string)
	if !strings.HasPrefix(first, marker) {
		t.Fatalf("payload corrupted: %q", first[:min(len(first), 40)])
	}
}

// TestEncryptionPlaintextStillWorks 明文模式不受影响
func TestEncryptionPlaintextStillWorks(t *testing.T) {
	dir, dirCleanup := newTempDir(t)
	defer func() { dirCleanup() }()
	dbPath := filepath.Join(dir, "plain.db")

	db, err := NewWeDBDatabase(dbPath, 4096)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if _, err := os.Stat(xkeyPathFor(dbPath)); !os.IsNotExist(err) {
		t.Fatal("plaintext mode must not create xkey file")
	}
	if err := db.CreateTable(&api.TableSchema{
		TableName: "t",
		Columns:   []api.ColumnSchema{{Name: "id", Type: api.TypeInteger, PrimaryKey: true}},
	}); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := db.InsertRow("t", map[string]interface{}{"id": int64(1)}); err != nil {
		t.Fatalf("insert: %v", err)
	}
	db.Close()
}

// TestXTSCrypterUnit XTS 单元：同密钥往返一致、异密钥不一致、扇区号影响密文
func TestXTSCrypterUnit(t *testing.T) {
	key := bytes.Repeat([]byte{7}, 64)
	c, err := newXtsCrypter(key, 512)
	if err != nil {
		t.Fatal(err)
	}

	page := make([]byte, 512)
	for i := range page {
		page[i] = byte(i)
	}
	orig := append([]byte{}, page...)

	if err := c.EncryptSector(3, page); err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(page, orig) {
		t.Fatal("encryption produced no change")
	}
	if err := c.DecryptSector(3, page); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(page, orig) {
		t.Fatal("roundtrip mismatch")
	}

	// 不同扇区号的 tweak 必须产生不同密文
	p2 := append([]byte{}, orig...)
	c.EncryptSector(4, p2)
	if bytes.Equal(p2, func() []byte { x := append([]byte{}, orig...); c.EncryptSector(3, x); return x }()) {
		t.Fatal("same plaintext at different sectors produced identical ciphertext")
	}

	// 非法尺寸
	if err := c.EncryptSector(1, make([]byte, 10)); err == nil {
		t.Fatal("non-multiple block size must fail")
	}
}

// TestEncryptionJournalRollback 加密下的事务回滚恢复（经加密层）
func TestEncryptionJournalRollback(t *testing.T) {
	dir, dirCleanup := newTempDir(t)
	defer func() { dirCleanup() }()
	dbPath := filepath.Join(dir, "enc_rb.db")
	pass := []byte("rb-passphrase-01")

	db, err := NewWeDBDatabaseSecure(dbPath, 4096, pass)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	if err := db.CreateTable(&api.TableSchema{
		TableName: "rb",
		Columns: []api.ColumnSchema{
			{Name: "id", Type: api.TypeInteger, PrimaryKey: true},
			{Name: "v", Type: api.TypeInteger},
		},
	}); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := db.InsertRow("rb", map[string]interface{}{"id": int64(1), "v": int64(11)}); err != nil {
		t.Fatalf("seed insert: %v", err)
	}

	// 开启写事务并修改，然后回滚 —— pager journal 记录旧页镜像，
	// 回滚经加密层把原始页写回
	tx, err := db.BeginTx(nil, nil)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	if err := db.UpdateRow("rb", map[string]interface{}{"v": int64(999)}, "id = 1"); err != nil {
		t.Fatalf("update: %v", err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatalf("rollback: %v", err)
	}

	rows, err := db.ScanTable("rb")
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(rows) != 1 || rows[0]["v"].(int64) != 11 {
		t.Fatalf("rollback did not restore data: %+v", rows)
	}
}

func IsWrongPassphrase(err error) bool {
	type causer interface{ Unwrap() error }
	for err != nil {
		if err == ErrWrongPassphrase {
			return true
		}
		if u, ok := err.(causer); ok {
			err = u.Unwrap()
			continue
		}
		break
	}
	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
