package cache

import (
	"testing"
	"time"
)

func TestNewPrefixCache(t *testing.T) {
	cache := NewPrefixCache(nil)
	if cache == nil {
		t.Fatal("NewPrefixCache returned nil")
	}
	if cache.config == nil {
		t.Fatal("config is nil")
	}
}

func TestPrefixCacheSetGet(t *testing.T) {
	cache := NewPrefixCache(nil)

	// Set entry
	cache.Set("test-key", "test-prefix", 100)

	// Get entry
	entry, exists := cache.Get("test-key")
	if !exists {
		t.Fatal("expected entry to exist")
	}
	if entry.Key != "test-key" {
		t.Errorf("expected key test-key, got %s", entry.Key)
	}
	if entry.Prefix != "test-prefix" {
		t.Errorf("expected prefix test-prefix, got %s", entry.Prefix)
	}
	if entry.Tokens != 100 {
		t.Errorf("expected tokens 100, got %d", entry.Tokens)
	}
	if entry.HitCount != 1 {
		t.Errorf("expected hit count 1, got %d", entry.HitCount)
	}
}

func TestPrefixCacheDelete(t *testing.T) {
	cache := NewPrefixCache(nil)

	// Set entry
	cache.Set("test-key", "test-prefix", 100)

	// Delete entry
	cache.Delete("test-key")

	// Get entry
	_, exists := cache.Get("test-key")
	if exists {
		t.Error("expected entry to not exist")
	}
}

func TestPrefixCacheGetState(t *testing.T) {
	config := &CacheConfig{
		HotThreshold:  1 * time.Minute,
		ColdThreshold: 5 * time.Minute,
	}
	cache := NewPrefixCache(config)

	// Set entry
	cache.Set("test-key", "test-prefix", 100)

	// Should be hot
	state := cache.GetState("test-key")
	if state != CacheStateHot {
		t.Errorf("expected state hot, got %s", state)
	}
}

func TestPrefixCacheShouldPrune(t *testing.T) {
	config := &CacheConfig{
		HotThreshold:   1 * time.Minute,
		ColdThreshold:  5 * time.Minute,
		PruneThreshold: 10 * time.Minute,
	}
	cache := NewPrefixCache(config)

	// Set entry
	cache.Set("test-key", "test-prefix", 100)

	// Should not prune (hot)
	if cache.ShouldPrune("test-key") {
		t.Error("expected should not prune hot entry")
	}
}

func TestPrefixCacheCleanup(t *testing.T) {
	config := &CacheConfig{
		MaxCacheAge: 1 * time.Hour,
	}
	cache := NewPrefixCache(config)

	// Set entry
	cache.Set("test-key", "test-prefix", 100)

	// Cleanup should not remove anything
	removed := cache.Cleanup()
	if removed != 0 {
		t.Errorf("expected 0 removed, got %d", removed)
	}
}

func TestPrefixCacheGetStats(t *testing.T) {
	cache := NewPrefixCache(nil)

	// Set entries
	cache.Set("key1", "prefix1", 100)
	cache.Set("key2", "prefix2", 200)

	// Get stats
	stats := cache.GetStats()
	if stats.TotalEntries != 2 {
		t.Errorf("expected 2 entries, got %d", stats.TotalEntries)
	}
	if stats.TotalTokens != 300 {
		t.Errorf("expected 300 tokens, got %d", stats.TotalTokens)
	}
}

func TestPrefixCacheListKeys(t *testing.T) {
	cache := NewPrefixCache(nil)

	// Set entries
	cache.Set("key1", "prefix1", 100)
	cache.Set("key2", "prefix2", 200)

	// List keys
	keys := cache.ListKeys()
	if len(keys) != 2 {
		t.Errorf("expected 2 keys, got %d", len(keys))
	}
}

func TestPrefixCacheClear(t *testing.T) {
	cache := NewPrefixCache(nil)

	// Set entries
	cache.Set("key1", "prefix1", 100)
	cache.Set("key2", "prefix2", 200)

	// Clear
	cache.Clear()

	// Get stats
	stats := cache.GetStats()
	if stats.TotalEntries != 0 {
		t.Errorf("expected 0 entries, got %d", stats.TotalEntries)
	}
}

func TestCacheStateString(t *testing.T) {
	tests := []struct {
		state    CacheState
		expected string
	}{
		{CacheStateHot, "hot"},
		{CacheStateWarm, "warm"},
		{CacheStateCold, "cold"},
	}

	for _, tt := range tests {
		if tt.state.String() != tt.expected {
			t.Errorf("expected %s, got %s", tt.expected, tt.state.String())
		}
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	if config == nil {
		t.Fatal("DefaultConfig returned nil")
	}
	if config.TTL != 1*time.Hour {
		t.Errorf("expected TTL 1h, got %s", config.TTL)
	}
	if config.HotThreshold != 5*time.Minute {
		t.Errorf("expected HotThreshold 5m, got %s", config.HotThreshold)
	}
	if config.ColdThreshold != 30*time.Minute {
		t.Errorf("expected ColdThreshold 30m, got %s", config.ColdThreshold)
	}
	if config.PruneThreshold != 24*time.Hour {
		t.Errorf("expected PruneThreshold 24h, got %s", config.PruneThreshold)
	}
	if config.MaxCacheAge != 7*24*time.Hour {
		t.Errorf("expected MaxCacheAge 168h, got %s", config.MaxCacheAge)
	}
}
