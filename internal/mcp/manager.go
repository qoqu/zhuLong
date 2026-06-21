package mcp

import (
	"context"
	"fmt"
	"sync"
)

// Manager manages multiple MCP server connections
type Manager struct {
	servers map[string]*Client
	mu      sync.RWMutex
}

// NewManager creates a new MCP manager
func NewManager() *Manager {
	return &Manager{
		servers: make(map[string]*Client),
	}
}

// AddServer adds an MCP server configuration
func (m *Manager) AddServer(config ServerConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.servers[config.Name]; exists {
		return fmt.Errorf("server %s already exists", config.Name)
	}

	client := NewClient(config)
	m.servers[config.Name] = client

	return nil
}

// Connect connects to an MCP server
func (m *Manager) Connect(name string) error {
	m.mu.RLock()
	client, exists := m.servers[name]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("server %s not found", name)
	}

	return client.Connect()
}

// ConnectAll connects to all enabled MCP servers
func (m *Manager) ConnectAll() []error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var errors []error
	for name, client := range m.servers {
		if !client.config.Enabled {
			continue
		}
		if err := client.Connect(); err != nil {
			errors = append(errors, fmt.Errorf("failed to connect to %s: %w", name, err))
		}
	}

	return errors
}

// Disconnect disconnects from an MCP server
func (m *Manager) Disconnect(name string) error {
	m.mu.RLock()
	client, exists := m.servers[name]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("server %s not found", name)
	}

	return client.Close()
}

// DisconnectAll disconnects from all MCP servers
func (m *Manager) DisconnectAll() {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, client := range m.servers {
		client.Close()
	}
}

// GetClient returns the MCP client for a server
func (m *Manager) GetClient(name string) (*Client, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	client, exists := m.servers[name]
	if !exists {
		return nil, fmt.Errorf("server %s not found", name)
	}

	return client, nil
}

// ListServers returns the list of configured servers
func (m *Manager) ListServers() []ServerConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var configs []ServerConfig
	for _, client := range m.servers {
		configs = append(configs, client.config)
	}

	return configs
}

// ListConnectedServers returns the list of connected servers
func (m *Manager) ListConnectedServers() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var names []string
	for name, client := range m.servers {
		if client.IsConnected() {
			names = append(names, name)
		}
	}

	return names
}

// ListAllTools returns all tools from all connected servers
func (m *Manager) ListAllTools() map[string][]Tool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string][]Tool)
	for name, client := range m.servers {
		if client.IsConnected() {
			result[name] = client.ListTools()
		}
	}

	return result
}

// FindTool finds a tool by name across all connected servers
func (m *Manager) FindTool(name string) (*Client, *Tool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, client := range m.servers {
		if !client.IsConnected() {
			continue
		}
		tool := client.GetTool(name)
		if tool != nil {
			return client, tool
		}
	}

	return nil, nil
}

// CallTool calls a tool on the appropriate server
func (m *Manager) CallTool(ctx context.Context, name string, args map[string]interface{}) (*ToolResult, error) {
	client, tool := m.FindTool(name)
	if client == nil || tool == nil {
		return nil, fmt.Errorf("tool %s not found", name)
	}

	return client.CallTool(ctx, name, args)
}

// IsServerConnected checks if a server is connected
func (m *Manager) IsServerConnected(name string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	client, exists := m.servers[name]
	if !exists {
		return false
	}

	return client.IsConnected()
}
