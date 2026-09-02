package util

import "sync"

// Pool 是对象池的通用接口
type Pool interface {
	Get() interface{}
	Put(x interface{})
}

// ValuePool 是 Value 对象池
var ValuePool = sync.Pool{
	New: func() interface{} {
		return &struct{}{} // 预留 Value 结构
	},
}

// PagePool 是页面数据缓冲区池
var PagePool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 4096) // 默认页面大小
	},
}

// GetPageBuffer 从池中获取页面缓冲区
func GetPageBuffer(size int) []byte {
	buf := PagePool.Get().([]byte)
	if cap(buf) < size {
		// 如果缓冲区太小，创建新的
		return make([]byte, size)
	}
	return buf[:size]
}

// PutPageBuffer 将页面缓冲区归还到池中
func PutPageBuffer(buf []byte) {
	if cap(buf) == 4096 { // 只回收标准大小的缓冲区
		PagePool.Put(buf[:cap(buf)])
	}
}

// CursorPool 是游标对象池
var CursorPool = sync.Pool{
	New: func() interface{} {
		return &struct{}{} // 预留 Cursor 结构
	},
}

// BtCursorPool 是 B-Tree 游标池
var BtCursorPool = sync.Pool{
	New: func() interface{} {
		return &struct{}{} // 预留 BtCursor 结构
	},
}

// MemPool 是通用内存池
type MemPool struct {
	pool sync.Pool
	size int
}

// NewMemPool 创建新的内存池
func NewMemPool(size int) *MemPool {
	return &MemPool{
		size: size,
		pool: sync.Pool{
			New: func() interface{} {
				return make([]byte, size)
			},
		},
	}
}

// Get 从池中获取内存
func (p *MemPool) Get() []byte {
	return p.pool.Get().([]byte)
}

// Put 将内存归还到池中
func (p *MemPool) Put(buf []byte) {
	if cap(buf) == p.size {
		p.pool.Put(buf[:p.size])
	}
}

// Size 返回池中对象的大小
func (p *MemPool) Size() int {
	return p.size
}