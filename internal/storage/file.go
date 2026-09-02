package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// File 文件封装
// 提供跨平台的文件操作；可选扇区加密（加密静态存储）
type File struct {
	file   *os.File
	mu     sync.Mutex
	path   string
	crypt  sectorCipher // 可为 nil（明文模式）
	secSz  int          // 扇区大小（= 页大小）
}

// OpenFile 打开文件
func OpenFile(path string) (*File, error) {
	return openFileInternal(path, nil, 0)
}

// OpenFileSecure 以扇区加密模式打开文件
// sectorSize 必须是 16 的倍数；同一偏移的读写按 (off/sectorSize) 作为 tweak。
func OpenFileSecure(path string, c sectorCipher, sectorSize int) (*File, error) {
	return openFileInternal(path, c, sectorSize)
}

func openFileInternal(path string, c sectorCipher, sectorSize int) (*File, error) {
	// 验证路径，防止路径遍历攻击
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// 清理路径并检查是否包含危险字符
	cleanPath := filepath.Clean(absPath)

	// 检查路径是否包含危险字符
	if strings.Contains(cleanPath, "..") {
		return nil, fmt.Errorf("invalid path: contains parent directory reference")
	}

	// 检查路径中是否包含空字节（防止空字节注入）
	if strings.Contains(cleanPath, "\x00") {
		return nil, fmt.Errorf("invalid path: contains null byte")
	}

	// 注：允许绝对路径打开数据库（测试与多实例部署需要）；
	// 路径遍历已由上面的 .. 与空字节检查拦截。

	// 打开文件
	file, err := os.OpenFile(cleanPath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}

	return &File{
		file:  file,
		path:  cleanPath,
		crypt: c,
		secSz: sectorSize,
	}, nil
}

// ReadAt 读取数据
func (f *File) ReadAt(p []byte, off int64) (n int, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	n, err = f.file.ReadAt(p, off)
	if err == nil && f.crypt != nil && len(p) > 0 {
		if err := f.decryptRange(int64(off), p); err != nil {
			return n, err
		}
	}
	return n, err
}

// WriteAt 写入数据
func (f *File) WriteAt(p []byte, off int64) (n int, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := p
	if f.crypt != nil && len(p) > 0 {
		cp := make([]byte, len(p))
		copy(cp, p)
		if err := f.encryptRange(int64(off), cp); err != nil {
			return 0, err
		}
		out = cp
	}
	return f.file.WriteAt(out, off)
}

// encryptRange / decryptRange 按扇区处理；要求对齐且长度不跨扇区越界
func (f *File) encryptRange(off int64, buf []byte) error { return f.applyRange(off, buf, true) }
func (f *File) decryptRange(off int64, buf []byte) error { return f.applyRange(off, buf, false) }

func (f *File) applyRange(off int64, buf []byte, enc bool) error {
	sz := f.secSz
	if sz <= 0 || off%int64(sz) != 0 || int64(len(buf))%int64(sz) != 0 {
		// 非整页访问不应出现在加密数据文件上
		return fmt.Errorf("crypto: unaligned page access off=%d len=%d sector=%d", off, len(buf), sz)
	}
	first := uint64(off / int64(sz))
	for i := 0; i*sz < len(buf); i++ {
		seg := buf[i*sz : (i+1)*sz]
		var err error
		if enc {
			err = f.crypt.EncryptSector(first+uint64(i), seg)
		} else {
			err = f.crypt.DecryptSector(first+uint64(i), seg)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// Sync 同步文件到磁盘
func (f *File) Sync() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.file.Sync()
}

// Close 关闭文件
func (f *File) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.file.Close()
}

// Size 返回文件大小
func (f *File) Size() int64 {
	f.mu.Lock()
	defer f.mu.Unlock()

	info, err := f.file.Stat()
	if err != nil {
		return 0
	}
	return info.Size()
}

// Journal 回滚日志
// 对应 SQLite 的 rollback journal
type Journal struct {
	file     *File
	dbFile   *File // 数据库主文件（回滚恢复时经由它写入，保证经过加密层）
	filePath string
	active   bool
	mu       sync.Mutex
}

// NewJournal 创建新回滚日志
func NewJournal(filePath string) *Journal {
	// 使用filepath.Join正确拼接路径
	journalPath := filepath.Join(filePath + "-journal")
	return &Journal{
		filePath: journalPath,
		active:   false,
	}
}

// Begin 开始日志
func (j *Journal) Begin() error {
	j.mu.Lock()
	defer j.mu.Unlock()

	if j.active {
		return nil // 已经激活
	}

	file, err := OpenFile(j.filePath)
	if err != nil {
		return err
	}

	j.file = file
	j.active = true

	return nil
}

// WritePage 写入页面到日志
func (j *Journal) WritePage(pageNum int, data []byte) error {
	j.mu.Lock()
	defer j.mu.Unlock()

	if !j.active {
		return nil // 日志未激活
	}

	// 写入页面号
	off := int64(pageNum * 4)
	if _, err := j.file.WriteAt([]byte{byte(pageNum >> 24), byte(pageNum >> 16), byte(pageNum >> 8), byte(pageNum)}, off); err != nil {
		return err
	}

	// 写入页面数据
	dataOff := int64(pageNum * 4096 + 4) // 假设页面大小为 4096
	if _, err := j.file.WriteAt(data, dataOff); err != nil {
		return err
	}

	return nil
}

// Commit 提交日志
func (j *Journal) Commit() error {
	j.mu.Lock()
	defer j.mu.Unlock()

	if !j.active {
		return nil // 日志未激活
	}

	// 删除日志文件
	if j.file != nil {
		j.file.Close()
		j.file = nil
	}

	os.Remove(j.filePath)
	j.active = false

	return nil
}

// Rollback 回滚日志
func (j *Journal) Rollback() error {
	j.mu.Lock()
	defer j.mu.Unlock()

	if !j.active {
		return nil // 日志未激活
	}

	// TODO: 实现回滚逻辑
	// 从日志文件中读取所有页面并恢复到数据库文件
	// 日志文件格式：
	// - 前 4KB: 页面号列表（每 4 字节一个页面号）
	// - 之后: 页面数据（每个 4KB）

	// 读取所有页面号
	pageNums := make([]int, 0)
	buf := make([]byte, 4096)
	n, err := j.file.ReadAt(buf, 0)
	if err != nil && n == 0 {
		// 日志为空
	} else if err != nil {
		return fmt.Errorf("failed to read journal: %w", err)
	}

	// 解析页面号
	for i := 0; i < n-3; i += 4 {
		pageNum := int(buf[i])<<24 | int(buf[i+1])<<16 | int(buf[i+2])<<8 | int(buf[i+3])
		if pageNum > 0 {
			pageNums = append(pageNums, pageNum)
		}
	}

	// 恢复每个页面的原始数据（经由数据库主文件句柄，保证经过加密层与锁）
	for _, pageNum := range pageNums {
		// 从日志中读取原始页面数据
		data := make([]byte, 4096)
		off := int64(pageNum*4096 + 4)
		if _, err := j.file.ReadAt(data, off); err != nil {
			return fmt.Errorf("failed to read page %d from journal: %w", pageNum, err)
		}

		if j.dbFile == nil {
			return fmt.Errorf("journal has no database file reference")
		}
		// 写入到数据库文件
		dbOff := int64(pageNum-1) * 4096
		if _, err := j.dbFile.WriteAt(data, dbOff); err != nil {
			return fmt.Errorf("failed to restore page %d: %w", pageNum, err)
		}

		if err := j.dbFile.Sync(); err != nil {
			return fmt.Errorf("failed to sync database file: %w", err)
		}
	}

	// 删除日志文件
	if j.file != nil {
		j.file.Close()
		j.file = nil
	}

	os.Remove(j.filePath)
	j.active = false

	return nil
}