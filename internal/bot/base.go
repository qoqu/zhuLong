package bot

import (
	"sync"
)

// BaseAdapter provides common functionality for all platform adapters.
// Platform-specific adapters should embed this struct and implement the Adapter interface.
type BaseAdapter struct {
	config    AdapterConfig
	handler   MessageHandler
	connected bool
	mu        sync.RWMutex
}

// NewBaseAdapter creates a new BaseAdapter.
func NewBaseAdapter(config AdapterConfig) *BaseAdapter {
	return &BaseAdapter{
		config: config,
	}
}

// Platform returns the platform type.
func (b *BaseAdapter) Platform() Platform {
	return b.config.Platform
}

// Name returns the adapter name.
func (b *BaseAdapter) Name() string {
	return b.config.Name
}

// IsConnected returns whether the adapter is connected.
func (b *BaseAdapter) IsConnected() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.connected
}

// SetConnected sets the connection status.
func (b *BaseAdapter) SetConnected(connected bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.connected = connected
}

// SetMessageHandler sets the handler for incoming messages.
func (b *BaseAdapter) SetMessageHandler(handler MessageHandler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handler = handler
}

// HandleMessage calls the registered message handler.
func (b *BaseAdapter) HandleMessage(event *MessageEvent) {
	b.mu.RLock()
	handler := b.handler
	b.mu.RUnlock()

	if handler != nil {
		handler(event)
	}
}

// Config returns the adapter configuration.
func (b *BaseAdapter) Config() AdapterConfig {
	return b.config
}
