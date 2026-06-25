package bot

import (
	"fmt"
	"sync"
	"time"
)

// Manager manages multiple platform adapters.
type Manager struct {
	adapters map[string]Adapter // key: adapter name
	handler  MessageHandler
	dedup    *Deduplicator
	mu       sync.RWMutex
}

// NewManager creates a new bot manager.
func NewManager() *Manager {
	return &Manager{
		adapters: make(map[string]Adapter),
		dedup:    NewDeduplicator(5 * time.Minute),
	}
}

// SetMessageHandler sets the global message handler for all adapters.
func (m *Manager) SetMessageHandler(handler MessageHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handler = handler

	// Update existing adapters
	for _, adapter := range m.adapters {
		adapter.SetMessageHandler(m.wrapHandler(adapter.Name()))
	}
}

// AddAdapter adds a platform adapter to the manager.
func (m *Manager) AddAdapter(config AdapterConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if adapter already exists
	if _, exists := m.adapters[config.Name]; exists {
		return fmt.Errorf("adapter %s already exists", config.Name)
	}

	// Create adapter using default registry
	adapter, err := DefaultRegistry.Create(config)
	if err != nil {
		return err
	}

	// Set message handler
	adapter.SetMessageHandler(m.wrapHandler(config.Name))

	m.adapters[config.Name] = adapter
	return nil
}

// Connect connects an adapter by name.
func (m *Manager) Connect(name string) error {
	m.mu.RLock()
	adapter, exists := m.adapters[name]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("adapter %s not found", name)
	}

	return adapter.Connect()
}

// Disconnect disconnects an adapter by name.
func (m *Manager) Disconnect(name string) error {
	m.mu.RLock()
	adapter, exists := m.adapters[name]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("adapter %s not found", name)
	}

	return adapter.Disconnect()
}

// ConnectAll connects all enabled adapters.
func (m *Manager) ConnectAll() []error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var errors []error
	for name, adapter := range m.adapters {
		if err := adapter.Connect(); err != nil {
			errors = append(errors, fmt.Errorf("failed to connect %s: %w", name, err))
		}
	}
	return errors
}

// DisconnectAll disconnects all adapters.
func (m *Manager) DisconnectAll() {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, adapter := range m.adapters {
		adapter.Disconnect()
	}
}

// RemoveAdapter removes an adapter by name.
func (m *Manager) RemoveAdapter(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	adapter, exists := m.adapters[name]
	if !exists {
		return fmt.Errorf("adapter %s not found", name)
	}

	adapter.Disconnect()
	delete(m.adapters, name)
	return nil
}

// GetAdapter returns an adapter by name.
func (m *Manager) GetAdapter(name string) (Adapter, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	adapter, exists := m.adapters[name]
	if !exists {
		return nil, fmt.Errorf("adapter %s not found", name)
	}

	return adapter, nil
}

// ListAdapters returns all adapters.
func (m *Manager) ListAdapters() []AdapterConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()

	configs := make([]AdapterConfig, 0, len(m.adapters))
	for _, adapter := range m.adapters {
		configs = append(configs, AdapterConfig{
			Platform: adapter.Platform(),
			Name:     adapter.Name(),
		})
	}
	return configs
}

// ListConnectedAdapters returns all connected adapters.
func (m *Manager) ListConnectedAdapters() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var names []string
	for name, adapter := range m.adapters {
		if adapter.IsConnected() {
			names = append(names, name)
		}
	}
	return names
}

// IsConnected checks if an adapter is connected.
func (m *Manager) IsConnected(name string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	adapter, exists := m.adapters[name]
	if !exists {
		return false
	}

	return adapter.IsConnected()
}

// wrapHandler wraps the global handler with deduplication.
func (m *Manager) wrapHandler(adapterName string) MessageHandler {
	return func(event *MessageEvent) {
		// Deduplicate
		if event.MessageID != "" && m.dedup.IsDuplicate(event.MessageID) {
			return
		}

		// Call global handler
		m.mu.RLock()
		handler := m.handler
		m.mu.RUnlock()

		if handler != nil {
			handler(event)
		}
	}
}
