// Package cache implements prefix-cache optimization for Zhulong.
//
// This module implements cache TTL awareness, cold resume maintenance,
// and risk-asymmetric defaults inspired by DeepSeek-Reasonix.
//
// Key principles:
// 1. Prefix only append, never modify (system prompt + skeleton never rewritten)
// 2. History only compress, never reorder (old loops compressed as summary appended)
// 3. Pruning only in dynamic interval (tool result pruning only in current loop)
package cache

import (
	"sync"
	"time"
)

// CacheState represents the state of the prefix cache
type CacheState int

const (
	CacheStateHot  CacheState = iota // Cache is hot (recently used)
	CacheStateWarm                    // Cache is warm (used within TTL)
	CacheStateCold                    // Cache is cold (expired)
)

// String returns the string representation
func (s CacheState) String() string {
	switch s {
	case CacheStateHot:
		return "hot"
	case CacheStateWarm:
		return "warm"
	case CacheStateCold:
		return "cold"
	default:
		return "unknown"
	}
}

// CacheConfig contains configuration for cache optimization
type CacheConfig struct {
	// TTL is the time-to-live for the cache
	TTL time.Duration

	// HotThreshold is the time threshold for hot state
	HotThreshold time.Duration

	// ColdThreshold is the time threshold for cold state
	ColdThreshold time.Duration

	// PruneThreshold is the threshold for pruning during cold resume
	// Reasonix uses 24 hours as default because "pruning a still-cached
	// session costs ~4x missing tokens vs keeping it"
	PruneThreshold time.Duration

	// MaxCacheAge is the maximum age of cache entries
	MaxCacheAge time.Duration
}

// DefaultConfig returns default cache configuration
func DefaultConfig() *CacheConfig {
	return &CacheConfig{
		TTL:            1 * time.Hour,
		HotThreshold:   5 * time.Minute,
		ColdThreshold:  30 * time.Minute,
		PruneThreshold: 24 * time.Hour,
		MaxCacheAge:    7 * 24 * time.Hour,
	}
}

// CacheEntry represents a cached prefix
type CacheEntry struct {
	// Key is the cache key (e.g., session ID)
	Key string

	// Prefix is the cached prefix content
	Prefix string

	// Tokens is the number of tokens in the prefix
	Tokens int

	// CreatedAt is when the cache was created
	CreatedAt time.Time

	// LastAccessedAt is when the cache was last accessed
	LastAccessedAt time.Time

	// HitCount is the number of cache hits
	HitCount int

	// MissCount is the number of cache misses
	MissCount int
}

// CacheStats contains cache statistics
type CacheStats struct {
	// TotalEntries is the total number of cache entries
	TotalEntries int

	// HotEntries is the number of hot cache entries
	HotEntries int

	// WarmEntries is the number of warm cache entries
	WarmEntries int

	// ColdEntries is the number of cold cache entries
	ColdEntries int

	// TotalTokens is the total number of cached tokens
	TotalTokens int

	// HitRate is the cache hit rate
	HitRate float64

	// AvgHitTime is the average cache hit time
	AvgHitTime time.Duration
}

// PrefixCache manages prefix cache optimization
type PrefixCache struct {
	config  *CacheConfig
	entries map[string]*CacheEntry
	mu      sync.RWMutex
}

// NewPrefixCache creates a new prefix cache
func NewPrefixCache(config *CacheConfig) *PrefixCache {
	if config == nil {
		config = DefaultConfig()
	}

	return &PrefixCache{
		config:  config,
		entries: make(map[string]*CacheEntry),
	}
}

// Get gets a cache entry by key
func (pc *PrefixCache) Get(key string) (*CacheEntry, bool) {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	entry, exists := pc.entries[key]
	if !exists {
		return nil, false
	}

	// Update access time
	entry.LastAccessedAt = time.Now()
	entry.HitCount++

	return entry, true
}

// Set sets a cache entry
func (pc *PrefixCache) Set(key string, prefix string, tokens int) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	now := time.Now()
	pc.entries[key] = &CacheEntry{
		Key:            key,
		Prefix:         prefix,
		Tokens:         tokens,
		CreatedAt:      now,
		LastAccessedAt: now,
		HitCount:       0,
		MissCount:      0,
	}
}

// Delete deletes a cache entry
func (pc *PrefixCache) Delete(key string) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	delete(pc.entries, key)
}

// GetState returns the state of a cache entry
func (pc *PrefixCache) GetState(key string) CacheState {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	entry, exists := pc.entries[key]
	if !exists {
		return CacheStateCold
	}

	return pc.getState(entry)
}

// getState returns the state of a cache entry (internal)
func (pc *PrefixCache) getState(entry *CacheEntry) CacheState {
	age := time.Since(entry.LastAccessedAt)

	if age <= pc.config.HotThreshold {
		return CacheStateHot
	} else if age <= pc.config.ColdThreshold {
		return CacheStateWarm
	} else {
		return CacheStateCold
	}
}

// ShouldPrune returns true if the cache entry should be pruned
// This implements the risk-asymmetric default from Reasonix:
// "pruning a still-cached session costs ~4x missing tokens vs keeping it"
func (pc *PrefixCache) ShouldPrune(key string) bool {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	entry, exists := pc.entries[key]
	if !exists {
		return false
	}

	// Only prune if the entry is cold and older than prune threshold
	state := pc.getState(entry)
	if state != CacheStateCold {
		return false
	}

	return time.Since(entry.LastAccessedAt) > pc.config.PruneThreshold
}

// Cleanup removes expired cache entries
func (pc *PrefixCache) Cleanup() int {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	removed := 0
	for key, entry := range pc.entries {
		if time.Since(entry.CreatedAt) > pc.config.MaxCacheAge {
			delete(pc.entries, key)
			removed++
		}
	}

	return removed
}

// GetStats returns cache statistics
func (pc *PrefixCache) GetStats() CacheStats {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	stats := CacheStats{
		TotalEntries: len(pc.entries),
	}

	totalHits := 0
	totalMisses := 0

	for _, entry := range pc.entries {
		state := pc.getState(entry)
		switch state {
		case CacheStateHot:
			stats.HotEntries++
		case CacheStateWarm:
			stats.WarmEntries++
		case CacheStateCold:
			stats.ColdEntries++
		}

		stats.TotalTokens += entry.Tokens
		totalHits += entry.HitCount
		totalMisses += entry.MissCount
	}

	total := totalHits + totalMisses
	if total > 0 {
		stats.HitRate = float64(totalHits) / float64(total)
	}

	return stats
}

// ListKeys returns all cache keys
func (pc *PrefixCache) ListKeys() []string {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	keys := make([]string, 0, len(pc.entries))
	for key := range pc.entries {
		keys = append(keys, key)
	}

	return keys
}

// Clear clears all cache entries
func (pc *PrefixCache) Clear() {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	pc.entries = make(map[string]*CacheEntry)
}
