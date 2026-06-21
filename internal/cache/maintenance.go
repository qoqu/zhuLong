package cache

import (
	"fmt"
	"time"
)

// MaintenanceConfig contains configuration for cache maintenance
type MaintenanceConfig struct {
	// CleanupInterval is the interval between cleanup runs
	CleanupInterval time.Duration

	// PruneOnColdResume enables pruning during cold resume
	PruneOnColdResume bool

	// MaxPruneTokens is the maximum tokens to prune per maintenance run
	MaxPruneTokens int

	// Verbose enables verbose logging
	Verbose bool
}

// DefaultMaintenanceConfig returns default maintenance configuration
func DefaultMaintenanceConfig() *MaintenanceConfig {
	return &MaintenanceConfig{
		CleanupInterval:   1 * time.Hour,
		PruneOnColdResume: true,
		MaxPruneTokens:    10000,
		Verbose:           false,
	}
}

// MaintenanceEvent represents a maintenance event
type MaintenanceEvent struct {
	Type      string
	Key       string
	Timestamp time.Time
	Details   string
}

// MaintenanceCallback is called when a maintenance event occurs
type MaintenanceCallback func(event MaintenanceEvent)

// CacheMaintenance handles cache maintenance operations
type CacheMaintenance struct {
	cache    *PrefixCache
	config   *MaintenanceConfig
	callback MaintenanceCallback
	stopCh   chan struct{}
}

// NewCacheMaintenance creates a new cache maintenance
func NewCacheMaintenance(cache *PrefixCache, config *MaintenanceConfig, callback MaintenanceCallback) *CacheMaintenance {
	if config == nil {
		config = DefaultMaintenanceConfig()
	}

	return &CacheMaintenance{
		cache:    cache,
		config:   config,
		callback: callback,
		stopCh:   make(chan struct{}),
	}
}

// Start starts the maintenance routine
func (cm *CacheMaintenance) Start() {
	go cm.run()
}

// Stop stops the maintenance routine
func (cm *CacheMaintenance) Stop() {
	close(cm.stopCh)
}

// run runs the maintenance routine
func (cm *CacheMaintenance) run() {
	ticker := time.NewTicker(cm.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			cm.runMaintenance()
		case <-cm.stopCh:
			return
		}
	}
}

// runMaintenance runs a single maintenance cycle
func (cm *CacheMaintenance) runMaintenance() {
	// Cleanup expired entries
	removed := cm.cache.Cleanup()
	if removed > 0 && cm.config.Verbose {
		cm.emitEvent(MaintenanceEvent{
			Type:      "cleanup",
			Timestamp: time.Now(),
			Details:   fmt.Sprintf("Removed %d expired cache entries", removed),
		})
	}

	// Check for cold entries that should be pruned
	if cm.config.PruneOnColdResume {
		cm.pruneColdEntries()
	}
}

// pruneColdEntries prunes cold cache entries
func (cm *CacheMaintenance) pruneColdEntries() {
	keys := cm.cache.ListKeys()
	pruned := 0

	for _, key := range keys {
		if cm.cache.ShouldPrune(key) {
			// In a real implementation, this would prune the prefix
			// For now, we just delete the entry
			cm.cache.Delete(key)
			pruned++

			cm.emitEvent(MaintenanceEvent{
				Type:      "prune",
				Key:       key,
				Timestamp: time.Now(),
				Details:   "Pruned cold cache entry",
			})
		}
	}

	if pruned > 0 && cm.config.Verbose {
		cm.emitEvent(MaintenanceEvent{
			Type:      "prune_summary",
			Timestamp: time.Now(),
			Details:   fmt.Sprintf("Pruned %d cold cache entries", pruned),
		})
	}
}

// emitEvent emits a maintenance event
func (cm *CacheMaintenance) emitEvent(event MaintenanceEvent) {
	if cm.callback != nil {
		cm.callback(event)
	}
}

// HandleColdResume handles a cold resume scenario
// This implements the cold resume maintenance from Reasonix:
// "when cache is detected as cold, perform maintenance to minimize
// the next full-price request's token consumption"
func (cm *CacheMaintenance) HandleColdResume(key string, tokens int) (bool, int) {
	state := cm.cache.GetState(key)

	if state != CacheStateCold {
		return false, 0
	}

	// Check if we should prune
	if !cm.cache.ShouldPrune(key) {
		return false, 0
	}

	// Calculate tokens to prune
	// Reasonix uses risk-asymmetric default: "pruning a still-cached
	// session costs ~4x missing tokens vs keeping it"
	// So we only prune if we're confident the cache is truly cold
	pruneTokens := tokens / 2 // Conservative: prune half

	if pruneTokens > cm.config.MaxPruneTokens {
		pruneTokens = cm.config.MaxPruneTokens
	}

	cm.emitEvent(MaintenanceEvent{
		Type:      "cold_resume",
		Key:       key,
		Timestamp: time.Now(),
		Details:   fmt.Sprintf("Cold resume maintenance: pruning %d tokens", pruneTokens),
	})

	return true, pruneTokens
}
