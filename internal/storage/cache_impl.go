package storage

import (
	"container/list"
	"fmt"
	"sync"
)

// PageCache 页面缓存
// 使用 LRU 策略管理缓存页面
type PageCache struct {
	maxSize    int           // 最大缓存页面数
	maxMemory  int64         // 最大内存使用量（字节）
	usedMemory int64         // 当前内存使用量（字节）
	pages      map[int]*Page // 页面映射
	lru        *list.List    // LRU 链表
	dirtyPages *list.List    // 脏页面链表
	mu         sync.RWMutex  // 读写锁
	pager      *Pager        // 关联的 Pager
}

// cacheEntry 缓存条目
type cacheEntry struct {
	pageNum int
	element *list.Element
}

// NewPageCache 创建新页面缓存
func NewPageCache(maxSize int, pager *Pager) *PageCache {
	return &PageCache{
		maxSize:    maxSize,
		maxMemory:  100 * 1024 * 1024, // 默认100MB
		usedMemory: 0,
		pages:      make(map[int]*Page),
		lru:        list.New(),
		dirtyPages: list.New(),
		pager:      pager,
	}
}

// Get 获取页面
func (pc *PageCache) Get(pageNum int) (*Page, error) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	// 检查缓存中是否存在
	if page, ok := pc.pages[pageNum]; ok {
		// 移动到 LRU 链表头部
		for e := pc.lru.Front(); e != nil; e = e.Next() {
			if e.Value.(*Page).PageNum == pageNum {
				pc.lru.MoveToFront(e)
				break
			}
		}
		return page, nil
	}

	// 从磁盘读取页面
	data, err := pc.pager.ReadPage(pageNum)
	if err != nil {
		return nil, fmt.Errorf("failed to read page %d: %w", pageNum, err)
	}

	// 反序列化页面
	page, err := DeserializePage(pageNum, data)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize page %d: %w", pageNum, err)
	}

	// 估算页面内存使用（页面大小 + 数据大小）
	estimatedSize := int64(pc.pager.pageSize + len(data))

	// 检查内存使用，如果超过限制则先淘汰页面
	if pc.usedMemory+estimatedSize > pc.maxMemory {
		pc.evictUntil(pc.maxMemory * 80 / 100) // 淘汰到80%以下
	}

	// 添加到缓存
	pc.addToCache(page)

	return page, nil
}

// Put 放入页面
func (pc *PageCache) Put(page *Page) error {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	// 如果页面已存在，先移除
	if existingPage, ok := pc.pages[page.PageNum]; ok {
		pc.removeFromCache(existingPage)
	}

	// 如果缓存已满，淘汰一个页面
	if len(pc.pages) >= pc.maxSize {
		pc.evict()
	}

	// 添加到缓存
	pc.addToCache(page)

	return nil
}

// MarkDirty 标记页面为脏页面
func (pc *PageCache) MarkDirty(page *Page) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	page.Dirty = true

	// 添加到脏页面链表
	for e := pc.dirtyPages.Front(); e != nil; e = e.Next() {
		if e.Value.(*Page).PageNum == page.PageNum {
			return // 已经在脏页面链表中
		}
	}
	pc.dirtyPages.PushBack(page)
}

// Flush 刷新所有脏页面到磁盘
func (pc *PageCache) Flush() error {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	// 遍历所有脏页面并刷新
	for e := pc.dirtyPages.Front(); e != nil; e = e.Next() {
		page := e.Value.(*Page)
		if page.Dirty {
			if err := pc.flushPage(page); err != nil {
				return fmt.Errorf("failed to flush page %d: %w", page.PageNum, err)
			}
		}
	}

	pc.dirtyPages.Init()

	return nil
}

// Clear 清空缓存
func (pc *PageCache) Clear() error {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	// 遍历所有脏页面并刷新
	for e := pc.dirtyPages.Front(); e != nil; e = e.Next() {
		page := e.Value.(*Page)
		if page.Dirty {
			if err := pc.flushPage(page); err != nil {
				return fmt.Errorf("failed to flush page %d: %w", page.PageNum, err)
			}
		}
	}

	// 清空缓存
	pc.pages = make(map[int]*Page)
	pc.lru.Init()
	pc.dirtyPages.Init()

	return nil
}

// Close 关闭缓存
func (pc *PageCache) Close() error {
	return pc.Clear()
}

// Size 返回缓存大小
func (pc *PageCache) Size() int {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	return len(pc.pages)
}

// DirtyCount 返回脏页面数量
func (pc *PageCache) DirtyCount() int {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	return pc.dirtyPages.Len()
}

// addToCache 添加页面到缓存
func (pc *PageCache) addToCache(page *Page) {
	pc.pages[page.PageNum] = page

	// 估算页面内存大小并更新统计
	estimatedSize := int64(pc.pager.pageSize + len(page.Data))
	pc.usedMemory += estimatedSize

	// 添加到 LRU 链表
	pc.lru.PushFront(page)
}

// Remove 从缓存中移除页面（公共接口）
func (pc *PageCache) Remove(pageNum int) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	page, exists := pc.pages[pageNum]
	if !exists {
		return
	}

	delete(pc.pages, pageNum)

	// 减少内存使用统计
	estimatedSize := int64(pc.pager.pageSize + len(page.Data))
	pc.usedMemory -= estimatedSize

	// 从 LRU 链表中移除
	for e := pc.lru.Front(); e != nil; e = e.Next() {
		if e.Value.(*Page).PageNum == pageNum {
			pc.lru.Remove(e)
			break
		}
	}

	// 从脏页面链表中移除
	for e := pc.dirtyPages.Front(); e != nil; e = e.Next() {
		if e.Value.(*Page).PageNum == pageNum {
			pc.dirtyPages.Remove(e)
			break
		}
	}
}

// removeFromCache 从缓存中移除页面（内部方法）
func (pc *PageCache) removeFromCache(page *Page) {
	delete(pc.pages, page.PageNum)

	// 减少内存使用统计
	estimatedSize := int64(pc.pager.pageSize + len(page.Data))
	pc.usedMemory -= estimatedSize

	// 从 LRU 链表中移除
	for e := pc.lru.Front(); e != nil; e = e.Next() {
		if e.Value.(*Page).PageNum == page.PageNum {
			pc.lru.Remove(e)
			break
		}
	}

	// 从脏页面链表中移除
	for e := pc.dirtyPages.Front(); e != nil; e = e.Next() {
		if e.Value.(*Page).PageNum == page.PageNum {
			pc.dirtyPages.Remove(e)
			break
		}
	}
}

// evict 淘汰一个页面
func (pc *PageCache) evict() {
	// 获取 LRU 链表末尾的页面
	element := pc.lru.Back()
	if element == nil {
		return
	}

	page := element.Value.(*Page)

	// 估算页面内存大小
	estimatedSize := int64(pc.pager.pageSize + len(page.Data))

	// 如果是脏页面，先刷新
	if page.Dirty {
		if err := pc.flushPage(page); err != nil {
			// 刷新失败，不淘汰
			return
		}
	}

	// 从缓存中移除
	pc.removeFromCache(page)

	// 减少内存使用统计
	pc.usedMemory -= estimatedSize
}

// evictUntil 淘汰页面直到内存使用低于指定阈值
func (pc *PageCache) evictUntil(threshold int64) {
	for pc.usedMemory > threshold && len(pc.pages) > 0 {
		pc.evict()
	}
}

// flushPage 刷新单个页面到磁盘
func (pc *PageCache) flushPage(page *Page) error {
	if !page.Dirty {
		return nil
	}

	// 序列化页面
	data, err := page.Serialize()
	if err != nil {
		return fmt.Errorf("failed to serialize page: %w", err)
	}

	// 写入磁盘
	if err := pc.pager.WritePage(page.PageNum, data); err != nil {
		return fmt.Errorf("failed to write page %d: %w", page.PageNum, err)
	}

	// 标记为干净
	page.Dirty = false

	return nil
}

// Invalidate 失效指定页面
func (pc *PageCache) Invalidate(pageNum int) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	if page, ok := pc.pages[pageNum]; ok {
		pc.removeFromCache(page)
	}
}

// GetStats 获取缓存统计信息
func (pc *PageCache) GetStats() CacheStats {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	return CacheStats{
		Size:       len(pc.pages),
		MaxSize:    pc.maxSize,
		DirtyCount: pc.dirtyPages.Len(),
		UsedMemory:  pc.usedMemory,
		MaxMemory:  pc.maxMemory,
		HitRatio:   0, // TODO: 实现命中率统计
	}
}

// CacheStats 缓存统计信息
type CacheStats struct {
	Size       int     // 当前缓存页面数
	MaxSize    int     // 最大缓存页面数
	DirtyCount int     // 脏页面数量
	UsedMemory int64   // 当前内存使用量（字节）
	MaxMemory  int64   // 最大内存限制（字节）
	HitRatio   float64 // 命中率
}