package util

import (
	"testing"
)

func TestNewLookaside(t *testing.T) {
	la := NewLookaside(100, 10)

	if la == nil {
		t.Fatal("NewLookaside returned nil")
	}

	if la.Size() != 100 {
		t.Errorf("Expected size 100, got %d", la.Size())
	}

	if la.SlotCount() != 10 {
		t.Errorf("Expected slot count 10, got %d", la.SlotCount())
	}
}

func TestLookaside_EnableDisable(t *testing.T) {
	la := NewLookaside(100, 10)

	if !la.IsEnabled() {
		t.Error("Should be enabled by default")
	}

	la.Disable()
	if la.IsEnabled() {
		t.Error("Should be disabled after Disable()")
	}

	la.Enable()
	if !la.IsEnabled() {
		t.Error("Should be enabled after Enable()")
	}
}

func TestNewMemPool(t *testing.T) {
	pool := NewMemPool(100)

	if pool == nil {
		t.Fatal("NewMemPool returned nil")
	}

	if pool.Size() != 100 {
		t.Errorf("Expected size 100, got %d", pool.Size())
	}

	buf := pool.Get()
	if buf == nil {
		t.Fatal("Get returned nil")
	}

	if len(buf) != 100 {
		t.Errorf("Expected buffer size 100, got %d", len(buf))
	}

	pool.Put(buf)
}

func TestGetPageBuffer(t *testing.T) {
	// 测试标准页面大小
	buf := GetPageBuffer(4096)
	if len(buf) != 4096 {
		t.Errorf("Expected buffer size 4096, got %d", len(buf))
	}

	PutPageBuffer(buf)

	// 测试非标准页面大小
	buf2 := GetPageBuffer(8192)
	if len(buf2) != 8192 {
		t.Errorf("Expected buffer size 8192, got %d", len(buf2))
	}
	PutPageBuffer(buf2)
}

func TestNewMemoryPool(t *testing.T) {
	mp := NewMemoryPool()

	if mp == nil {
		t.Fatal("NewMemoryPool returned nil")
	}

	buf := mp.Alloc(100)
	if len(buf) != 100 {
		t.Errorf("Expected buffer size 100, got %d", len(buf))
	}

	mp.Free(buf)
}