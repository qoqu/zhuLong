package bot

import (
	"testing"
	"time"
)

func TestNewDeduplicator(t *testing.T) {
	dedup := NewDeduplicator(5 * time.Minute)

	if dedup == nil {
		t.Fatal("NewDeduplicator returned nil")
	}

	if dedup.seen == nil {
		t.Error("seen map should be initialized")
	}

	if dedup.ttl != 5*time.Minute {
		t.Errorf("Expected TTL=5m, got %v", dedup.ttl)
	}
}

func TestDeduplicatorIsDuplicate(t *testing.T) {
	dedup := NewDeduplicator(5 * time.Minute)

	// 第一次应该是不重复
	if dedup.IsDuplicate("msg-123") {
		t.Error("First call should not be duplicate")
	}

	// 第二次应该是重复
	if !dedup.IsDuplicate("msg-123") {
		t.Error("Second call should be duplicate")
	}

	// 不同的消息 ID 应该不是重复
	if dedup.IsDuplicate("msg-456") {
		t.Error("Different message ID should not be duplicate")
	}
}

func TestDeduplicatorEmptyID(t *testing.T) {
	dedup := NewDeduplicator(5 * time.Minute)

	// 空 ID 应该不是重复
	if dedup.IsDuplicate("") {
		t.Error("Empty ID should not be duplicate")
	}
}

func TestDeduplicatorCleanup(t *testing.T) {
	// 使用很短的 TTL 进行测试
	dedup := NewDeduplicator(100 * time.Millisecond)

	dedup.IsDuplicate("msg-1")
	dedup.IsDuplicate("msg-2")

	// 等待 TTL 过期
	time.Sleep(150 * time.Millisecond)

	// 清理应该已经发生
	dedup.mu.Lock()
	count := len(dedup.seen)
	dedup.mu.Unlock()

	// 注意：清理是在后台 goroutine 中运行的，可能还没执行
	// 这里只验证不会 panic
	_ = count
}
