package storage

import (
	"sync"
	"sync/atomic"
	"time"
)

// BatchWriteBuffer 批量写入缓冲
// 用于批量写入页面数据，减少磁盘IO次数
type BatchWriteBuffer struct {
	// 配置
	config *BatchConfig

	// 缓冲区
	buffer *WriteBuffer

	// 页面号到数据的映射
	pageMap map[uint32][]byte

	// 当前缓冲区大小
	currentSize atomic.Int64

	// 统计信息
	stats *BufferStats

	// 写入回调
	writeCallback WriteCallback

	// 刷新计时器
	flushTimer *time.Timer

	// 锁
	mu sync.RWMutex

	// 停止通道
	stopChan chan struct{}

	// 关闭标志
	closed atomic.Bool
}

// BatchConfig 批量写入配置
type BatchConfig struct {
	// 缓冲区大小（字节数）
	BufferSize int64

	// 批量大小（页数）
	BatchSize int

	// 刷新间隔
	FlushInterval time.Duration

	// 是否启用自动刷新
	EnableAutoFlush bool
}

// DefaultBatchConfig 返回默认配置
func DefaultBatchConfig() *BatchConfig {
	return &BatchConfig{
		BufferSize:       4 * 1024 * 1024, // 4MB
		BatchSize:        100,             // 100页
		FlushInterval:    5 * time.Second,
		EnableAutoFlush:  true,
	}
}

// WriteBuffer 写入缓冲
type WriteBuffer struct {
	entries []*WriteEntry
}

// WriteEntry 写入条目
type WriteEntry struct {
	PageNum uint32
	Data    []byte
	Offset  int64
}

// BufferStats 缓冲统计
type BufferStats struct {
	TotalWrites     atomic.Uint64
	BufferedWrites  atomic.Uint64
	FlushedWrites   atomic.Uint64
	TotalBytesWritten atomic.Uint64
	IOOperationsSaved atomic.Uint64
}

// WriteCallback 写入回调函数
type WriteCallback func(pageNum uint32, data []byte, offset int64) error

// NewBatchWriteBuffer 创建批量写入缓冲
func NewBatchWriteBuffer(config *BatchConfig, writeCallback WriteCallback) *BatchWriteBuffer {
	if config == nil {
		config = DefaultBatchConfig()
	}

	bwb := &BatchWriteBuffer{
		config:        config,
		buffer:        &WriteBuffer{entries: make([]*WriteEntry, 0, config.BatchSize)},
		pageMap:       make(map[uint32][]byte),
		stats:         &BufferStats{},
		writeCallback: writeCallback,
		stopChan:      make(chan struct{}),
	}

	// 启动自动刷新
	if config.EnableAutoFlush {
		bwb.startAutoFlush()
	}

	return bwb
}

// Write 写入数据到缓冲
func (bwb *BatchWriteBuffer) Write(pageNum uint32, data []byte, offset int64) error {
	if bwb.closed.Load() {
		return nil
	}

	bwb.mu.Lock()
	defer bwb.mu.Unlock()

	// 添加到缓冲
	bwb.buffer.entries = append(bwb.buffer.entries, &WriteEntry{
		PageNum: pageNum,
		Data:    data,
		Offset:  offset,
	})
	bwb.pageMap[pageNum] = data

	// 更新统计
	bwb.stats.BufferedWrites.Add(1)
	bwb.stats.TotalBytesWritten.Add(uint64(len(data)))
	bwb.currentSize.Add(int64(len(data)))

	// 检查是否需要刷新
	if bwb.shouldFlush() {
		bwb.flush()
	}

	return nil
}

// Flush 刷新缓冲
func (bwb *BatchWriteBuffer) Flush() error {
	bwb.mu.Lock()
	defer bwb.mu.Unlock()

	return bwb.flush()
}

// flush 内部刷新方法
func (bwb *BatchWriteBuffer) flush() error {
	if len(bwb.buffer.entries) == 0 {
		return nil
	}

	// 保存当前缓冲区状态，用于回滚
	savedEntries := make([]*WriteEntry, len(bwb.buffer.entries))
	copy(savedEntries, bwb.buffer.entries)
	savedPageMap := make(map[uint32][]byte)
	for k, v := range bwb.pageMap {
		savedPageMap[k] = make([]byte, len(v))
		copy(savedPageMap[k], v)
	}
	savedSize := bwb.currentSize.Load()

	// 批量写入
	for _, entry := range bwb.buffer.entries {
		if bwb.writeCallback != nil {
			if err := bwb.writeCallback(entry.PageNum, entry.Data, entry.Offset); err != nil {
				// 写入失败，回滚缓冲区状态
				bwb.buffer.entries = savedEntries
				bwb.pageMap = savedPageMap
				bwb.currentSize.Store(savedSize)
				return err
			}
		}
	}

	// 更新统计
	bwb.stats.FlushedWrites.Add(uint64(len(bwb.buffer.entries)))
	bwb.stats.IOOperationsSaved.Add(uint64(len(bwb.buffer.entries) - 1))

	// 清空缓冲
	bwb.buffer.entries = bwb.buffer.entries[:0]
	bwb.pageMap = make(map[uint32][]byte)
	bwb.currentSize.Store(0)

	return nil
}

// shouldFlush 检查是否应该刷新
func (bwb *BatchWriteBuffer) shouldFlush() bool {
	// 检查缓冲区大小
	if bwb.currentSize.Load() >= bwb.config.BufferSize {
		return true
	}

	// 检查批量大小
	if len(bwb.buffer.entries) >= bwb.config.BatchSize {
		return true
	}

	return false
}

// startAutoFlush 启动自动刷新
func (bwb *BatchWriteBuffer) startAutoFlush() {
	bwb.flushTimer = time.AfterFunc(bwb.config.FlushInterval, func() {
		if !bwb.closed.Load() {
			bwb.Flush()
			bwb.startAutoFlush()
		}
	})
}

// Close 关闭缓冲
func (bwb *BatchWriteBuffer) Close() error {
	if bwb.closed.CompareAndSwap(false, true) {
		// 停止自动刷新
		if bwb.flushTimer != nil {
			bwb.flushTimer.Stop()
		}

		// 刷新剩余数据
		bwb.Flush()

		close(bwb.stopChan)
	}
	return nil
}

// GetStats 获取统计信息
func (bwb *BatchWriteBuffer) GetStats() *BufferStats {
	return bwb.stats
}