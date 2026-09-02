package storage

import (
	"crypto/rand"
	"fmt"
	"sync"
)

// Pager 事务管理器
// 对应 SQLite 的 Pager 结构
type Pager struct {
	filePath   string         // 文件路径
	pageSize   int            // 页面大小
	file       *File          // 文件句柄
	nextPage   int            // 下一个可用的页面号
	journal    *Journal       // 回滚日志
	inWriteTxn bool           // 是否在写事务中
	mu         sync.RWMutex   // 读写锁
	journalPages map[int]bool // 已记录到日志的页面
	freePages  map[int]bool   // 空闲页面集合
}

// NewPager 创建新 Pager
func NewPager(filePath string, pageSize int) (*Pager, error) {
	return newPagerInternal(filePath, pageSize, nil)
}

// NewPagerSecure 创建带加密的 Pager（passphrase 为空则退化为明文模式）
func NewPagerSecure(filePath string, pageSize int, passphrase []byte) (*Pager, error) {
	return newPagerInternal(filePath, pageSize, passphrase)
}

func newPagerInternal(filePath string, pageSize int, passphrase []byte) (*Pager, error) {
	if pageSize%16 != 0 || pageSize <= 0 {
		return nil, fmt.Errorf("page size %d must be a positive multiple of 16", pageSize)
	}

	var file *File
	var err error
	if len(passphrase) > 0 || fileExists(filePath+xkeySuffix) {
		dataKey, tweakKey, kerr := loadOrCreateXKey(filePath, passphrase, rand.Reader)
		if kerr != nil {
			return nil, kerr
		}
		crypter, cerr := newXtsCrypter(append(append([]byte{}, dataKey...), tweakKey...), pageSize)
		if cerr != nil {
			return nil, cerr
		}
		file, err = OpenFileSecure(filePath, crypter, pageSize)
	} else {
		file, err = OpenFile(filePath)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	if file == nil {
		return nil, fmt.Errorf("internal: file handle is nil")
	}

	pager := &Pager{
		filePath:     filePath,
		pageSize:     pageSize,
		file:         file,
		nextPage:     1, // 页面 1 是根页面
		journal:      NewJournal(filePath),
		journalPages: make(map[int]bool),
		freePages:    make(map[int]bool),
	}
	pager.journal.dbFile = file // 回滚恢复经主文件句柄（过加密层）

	return pager, nil
}

// BeginTransaction 开始事务
func (p *Pager) BeginTransaction(readOnly bool) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !readOnly && p.inWriteTxn {
		return fmt.Errorf("write transaction already in progress")
	}

	if !readOnly {
		p.inWriteTxn = true
		// 创建回滚日志
		if err := p.journal.Begin(); err != nil {
			return fmt.Errorf("failed to begin journal: %w", err)
		}
	}

	return nil
}

// Commit 提交事务
func (p *Pager) Commit() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.inWriteTxn {
		return fmt.Errorf("no write transaction in progress")
	}

	// 提交回滚日志
	if err := p.journal.Commit(); err != nil {
		return fmt.Errorf("failed to commit journal: %w", err)
	}

	// 同步文件到磁盘
	if err := p.file.Sync(); err != nil {
		return fmt.Errorf("failed to sync file: %w", err)
	}

	// 清除已记录的页面映射
	p.journalPages = make(map[int]bool)
	p.inWriteTxn = false
	return nil
}

// Rollback 回滚事务
func (p *Pager) Rollback() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.rollbackLocked()
}

// rollbackLocked is Rollback without locking; caller must hold p.mu.
func (p *Pager) rollbackLocked() error {
	if !p.inWriteTxn {
		return fmt.Errorf("no write transaction in progress")
	}

	// 回滚回滚日志
	if err := p.journal.Rollback(); err != nil {
		return fmt.Errorf("failed to rollback journal: %w", err)
	}

	// 清除已记录的页面映射
	p.journalPages = make(map[int]bool)
	p.inWriteTxn = false
	return nil
}

// ReadPage 读取页面
func (p *Pager) ReadPage(pageNum int) ([]byte, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if pageNum < 1 {
		return nil, fmt.Errorf("invalid page number: %d", pageNum)
	}

	// 从文件读取
	offset := int64(pageNum-1) * int64(p.pageSize)
	data := make([]byte, p.pageSize)
	n, err := p.file.ReadAt(data, offset)
	if err != nil && n == 0 {
		// 如果读取到文件末尾，返回零填充的页面
		return data, nil
	}
	if n != p.pageSize {
		// 如果读取的字节数不足，返回零填充的页面
		return data, nil
	}

	return data, nil
}

// WritePage 写入页面
func (p *Pager) WritePage(pageNum int, data []byte) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if pageNum < 1 {
		return fmt.Errorf("invalid page number: %d", pageNum)
	}

	if len(data) != p.pageSize {
		return fmt.Errorf("invalid page size: %d, expected %d", len(data), p.pageSize)
	}

	// 写入回滚日志（只记录每个页面的第一次修改）
	if p.inWriteTxn {
		if !p.journalPages[pageNum] {
			// 读取原始页面数据
			originalData := make([]byte, p.pageSize)
			_, err := p.file.ReadAt(originalData, int64(pageNum-1)*int64(p.pageSize))
			if err != nil && err.Error() != "EOF" {
				return fmt.Errorf("failed to read original page data: %w", err)
			}
			
			// 记录原始页面数据到日志
			if err := p.journal.WritePage(pageNum, originalData); err != nil {
				return fmt.Errorf("failed to write to journal: %w", err)
			}
			
			// 标记页面已记录
			p.journalPages[pageNum] = true
		}
	}

	// 写入文件
	offset := int64(pageNum-1) * int64(p.pageSize)
	if _, err := p.file.WriteAt(data, offset); err != nil {
		return fmt.Errorf("failed to write page %d: %w", pageNum, err)
	}

	return nil
}

// ReadPageNoLock 读取页面（不带锁）
func (p *Pager) ReadPageNoLock(pageNum int) ([]byte, error) {
	if pageNum < 1 {
		return nil, fmt.Errorf("invalid page number: %d", pageNum)
	}

	// 从文件读取
	offset := int64(pageNum-1) * int64(p.pageSize)
	data := make([]byte, p.pageSize)
	n, err := p.file.ReadAt(data, offset)
	if err != nil && n == 0 {
		// 如果读取到文件末尾，返回零填充的页面
		return data, nil
	}
	if n != p.pageSize {
		// 如果读取的字节数不足，返回零填充的页面
		return data, nil
	}

	return data, nil
}

// SetNextPage 设置下一个可用的页面号
func (p *Pager) SetNextPage(pageNum int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if pageNum > p.nextPage {
		p.nextPage = pageNum
	}
}

// FreePage 释放页面
func (p *Pager) FreePage(pageNum int) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 不能释放页面0（无效页面）或当前正在使用的页面
	if pageNum <= 0 {
		return fmt.Errorf("invalid page number: %d", pageNum)
	}

	// 将页面标记为空闲
	p.freePages[pageNum] = true

	// 简化实现：不立即从磁盘删除页面，只是标记为空闲
	// 在下次分配时，优先使用空闲页面

	return nil
}

// AllocPage 分配新页面（优先使用空闲页面）
func (p *Pager) AllocPage() int {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 优先使用空闲页面
	for pageNum := range p.freePages {
		delete(p.freePages, pageNum)
		return pageNum
	}

	// 没有空闲页面，分配新页面
	pageNum := p.nextPage
	p.nextPage++
	return pageNum
}

// Sync 同步文件到磁盘
func (p *Pager) Sync() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if err := p.file.Sync(); err != nil {
		return fmt.Errorf("failed to sync file: %w", err)
	}

	return nil
}

// Close 关闭 Pager
func (p *Pager) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 回滚未提交的事务（已持有写锁，使用无锁版本）
	if p.inWriteTxn {
		if err := p.rollbackLocked(); err != nil {
			return err
		}
	}

	// 关闭文件
	if err := p.file.Close(); err != nil {
		return fmt.Errorf("failed to close file: %w", err)
	}

	return nil
}

// CheckIntegrity 检查文件完整性
func (p *Pager) CheckIntegrity() error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// 检查文件大小是否合理（至少包含头部）
	fileSize := p.file.Size()

	if fileSize < int64(p.pageSize) {
		return fmt.Errorf("file size %d is too small", fileSize)
	}

	return nil
}

// PageSize 返回页面大小
func (p *Pager) PageSize() int {
	return p.pageSize
}

// FileSize 返回文件大小
func (p *Pager) FileSize() int64 {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.file.Size()
}

// InTransaction 检查是否在事务中
func (p *Pager) InTransaction() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.inWriteTxn
}