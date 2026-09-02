package storage

// Go 1.10 兼容垫片：以传统 sync/atomic 实现与 atomic.Xxx 等价的方法集。
// 由 tools/sync_go110.ps1 复制进镜像包，源模块不使用本文件。

import "sync/atomic"

// AtomicInt64 int64 原子包装
type AtomicInt64 struct{ v int64 }

func (a *AtomicInt64) Load() int64          { return atomic.LoadInt64(&a.v) }
func (a *AtomicInt64) Store(v int64)        { atomic.StoreInt64(&a.v, v) }
func (a *AtomicInt64) Add(n int64) int64    { return atomic.AddInt64(&a.v, n) }
func (a *AtomicInt64) Swap(n int64) int64   { return atomic.SwapInt64(&a.v, n) }
func (a *AtomicInt64) CAS(old, new int64) bool {
	return atomic.CompareAndSwapInt64(&a.v, old, new)
}

// AtomicUint64 uint64 原子包装
type AtomicUint64 struct{ v uint64 }

func (a *AtomicUint64) Load() uint64       { return atomic.LoadUint64(&a.v) }
func (a *AtomicUint64) Store(v uint64)     { atomic.StoreUint64(&a.v, v) }
func (a *AtomicUint64) Add(n uint64) uint64 { return atomic.AddUint64(&a.v, n) }

// AtomicInt32 int32 原子包装
type AtomicInt32 struct{ v int32 }

func (a *AtomicInt32) Load() int32       { return atomic.LoadInt32(&a.v) }
func (a *AtomicInt32) Store(v int32)     { atomic.StoreInt32(&a.v, v) }
func (a *AtomicInt32) Add(n int32) int32 { return atomic.AddInt32(&a.v, n) }

// AtomicUint32 uint32 原子包装
type AtomicUint32 struct{ v uint32 }

func (a *AtomicUint32) Load() uint32   { return atomic.LoadUint32(&a.v) }
func (a *AtomicUint32) Store(v uint32) { atomic.StoreUint32(&a.v, v) }

// AtomicBool bool 原子包装
type AtomicBool struct{ v uint32 }

func (b *AtomicBool) Load() bool {
	return atomic.LoadUint32(&b.v) != 0
}
func (b *AtomicBool) CompareAndSwap(old, new bool) bool {
	var nv uint32
	if new {
		nv = 1
	}
	var ov uint32
	if old {
		ov = 1
	}
	return atomic.CompareAndSwapUint32(&b.v, ov, nv)
}
func (b *AtomicBool) Store(x bool) {
	var v uint32
	if x {
		v = 1
	}
	atomic.StoreUint32(&b.v, v)
}
