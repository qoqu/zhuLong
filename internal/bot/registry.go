package bot

import (
	"fmt"
	"sync"
)

// AdapterFactory creates an adapter from a config.
type AdapterFactory func(config AdapterConfig) (Adapter, error)

// PlatformEntry represents a registered platform.
type PlatformEntry struct {
	Platform      Platform
	Label         string
	Factory       AdapterFactory
	Description   string
	RequiredFields []string
}

// Registry manages platform adapter registrations.
type Registry struct {
	entries map[Platform]*PlatformEntry
	mu      sync.RWMutex
}

// NewRegistry creates a new platform registry.
func NewRegistry() *Registry {
	return &Registry{
		entries: make(map[Platform]*PlatformEntry),
	}
}

// Register registers a platform adapter factory.
func (r *Registry) Register(entry *PlatformEntry) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.entries[entry.Platform] = entry
}

// Create creates an adapter for the given platform.
func (r *Registry) Create(config AdapterConfig) (Adapter, error) {
	r.mu.RLock()
	entry, exists := r.entries[config.Platform]
	r.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("unsupported platform: %s", config.Platform)
	}

	return entry.Factory(config)
}

// GetEntry returns the registry entry for a platform.
func (r *Registry) GetEntry(platform Platform) (*PlatformEntry, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, exists := r.entries[platform]
	if !exists {
		return nil, fmt.Errorf("unsupported platform: %s", platform)
	}

	return entry, nil
}

// ListPlatforms returns all registered platforms.
func (r *Registry) ListPlatforms() []*PlatformEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entries := make([]*PlatformEntry, 0, len(r.entries))
	for _, entry := range r.entries {
		entries = append(entries, entry)
	}
	return entries
}

// DefaultRegistry is the global platform registry.
var DefaultRegistry = NewRegistry()
