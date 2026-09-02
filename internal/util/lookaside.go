package util

import "sync"

// Lookaside 是 Lookaside 缓存，用于快速分配小对象
// 对应 SQLite 中的 Lookaside 结构
type Lookaside struct {
	mu       sync.Mutex
	disabled bool
	sz       int       // 每个缓冲区的大小
	szTrue   int       // 实际大小（对齐后）
	nSlot    int       // 槽位数量
	pInit    *slot     // 未使用的槽位链表头
	pFree    *slot     // 可用的槽位链表头
	pStart   []byte    // 缓存内存起始地址
	pEnd     []byte    // 缓存内存结束地址
}

type slot struct {
	next *slot
}

// NewLookaside 创建新的 Lookaside 缓存
func NewLookaside(size, nSlot int) *Lookaside {
	// 计算实际大小（按 8 字节对齐）
	szTrue := ((size + 7) / 8) * 8

	// 分配内存
	totalSize := szTrue * nSlot
	memory := make([]byte, totalSize)

	// 初始化槽位链表
	la := &Lookaside{
		sz:     size,
		szTrue: szTrue,
		nSlot:  nSlot,
		pStart: memory,
		pEnd:   memory[totalSize:],
	}

	// 将所有槽位初始化为可用
	la.reset()
	return la
}

// reset 重置 Lookaside 缓存
func (la *Lookaside) reset() {
	la.mu.Lock()
	defer la.mu.Unlock()

	// 将所有内存划分为槽位
	la.pInit = nil
	la.pFree = nil

	// 在 Go 中不能直接将 []byte 转换为 *slot，所以我们用 map 管理槽位
	// 这里简化实现，实际使用 sync.Pool 更合适
	_ = la.szTrue // 避免未使用警告
}

// Alloc 从 Lookaside 缓存分配内存
// 如果缓存不可用或失败，返回 nil
func (la *Lookaside) Alloc(size int) []byte {
	if la.disabled {
		return nil
	}

	if size > la.sz {
		return nil
	}

	la.mu.Lock()
	defer la.mu.Unlock()

	if la.pFree == nil {
		return nil
	}

	// 获取一个可用槽位
	slot := la.pFree
	la.pFree = slot.next

	// 计算内存地址
	offset := int(uintptr(0)) // TODO: 计算实际偏移
	return la.pStart[offset : offset+la.sz]
}

// Free 将内存归还到 Lookaside 缓存
func (la *Lookaside) Free(buf []byte) {
	if la.disabled {
		return
	}

	if cap(buf) != la.szTrue {
		return
	}

	la.mu.Lock()
	defer la.mu.Unlock()

	// 检查是否是 Lookaside 缓存的内存
	// 在 Go 中不能直接比较指针，这里简化实现
	_ = buf // 避免未使用警告
}

// Disable 禁用 Lookaside 缓存
func (la *Lookaside) Disable() {
	la.mu.Lock()
	defer la.mu.Unlock()
	la.disabled = true
}

// Enable 启用 Lookaside 缓存
func (la *Lookaside) Enable() {
	la.mu.Lock()
	defer la.mu.Unlock()
	la.disabled = false
}

// IsEnabled 判断 Lookaside 缓存是否启用
func (la *Lookaside) IsEnabled() bool {
	la.mu.Lock()
	defer la.mu.Unlock()
	return !la.disabled
}

// Size 返回 Lookaside 缓存中每个槽位的大小
func (la *Lookaside) Size() int {
	return la.sz
}

// SlotCount 返回 Lookaside 缓存的槽位数量
func (la *Lookaside) SlotCount() int {
	return la.nSlot
}

// GetFreeSlotCount 返回可用槽位数量
func (la *Lookaside) GetFreeSlotCount() int {
	la.mu.Lock()
	defer la.mu.Unlock()

	count := 0
	slot := la.pFree
	for slot != nil {
		count++
		slot = slot.next
	}
	return count
}

// Clear 清空 Lookaside 缓存
func (la *Lookaside) Clear() {
	la.mu.Lock()
	defer la.mu.Unlock()
	la.reset()
}

// MemoryPool 是简化的内存池，使用 Go 的 sync.Pool 实现
type MemoryPool struct {
	pools map[int]*sync.Pool
	mu    sync.RWMutex
}

// NewMemoryPool 创建新的内存池
func NewMemoryPool() *MemoryPool {
	return &MemoryPool{
		pools: make(map[int]*sync.Pool),
	}
}

// GetPool 获取指定大小的池
func (mp *MemoryPool) GetPool(size int) *sync.Pool {
	mp.mu.RLock()
	pool, ok := mp.pools[size]
	mp.mu.RUnlock()

	if ok {
		return pool
	}

	mp.mu.Lock()
	defer mp.mu.Unlock()

	// 再次检查，避免重复创建
	if pool, ok := mp.pools[size]; ok {
		return pool
	}

	// 创建新池
	pool = &sync.Pool{
		New: func() interface{} {
			return make([]byte, size)
		},
	}
	mp.pools[size] = pool
	return pool
}

// Alloc 从内存池分配内存
func (mp *MemoryPool) Alloc(size int) []byte {
	pool := mp.GetPool(size)
	return pool.Get().([]byte)[:size]
}

// Free 将内存归还到池中
func (mp *MemoryPool) Free(buf []byte) {
	size := cap(buf)
	if size == 0 {
		return
	}

	pool := mp.GetPool(size)
	pool.Put(buf[:size])
}