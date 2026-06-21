package mcp

import (
	"testing"
)

func TestNewClient(t *testing.T) {
	config := ServerConfig{
		Name: "test-server",
		Transport: TransportConfig{
			Type:    "stdio",
			Command: "echo",
			Args:    []string{"hello"},
		},
		Enabled: true,
	}

	client := NewClient(config)
	if client == nil {
		t.Fatal("NewClient returned nil")
	}
	if client.config.Name != "test-server" {
		t.Errorf("expected name test-server, got %s", client.config.Name)
	}
	if client.connected {
		t.Error("expected connected to be false")
	}
}

func TestManager(t *testing.T) {
	manager := NewManager()
	if manager == nil {
		t.Fatal("NewManager returned nil")
	}

	// Add server
	config := ServerConfig{
		Name: "test-server",
		Transport: TransportConfig{
			Type:    "stdio",
			Command: "echo",
			Args:    []string{"hello"},
		},
		Enabled: true,
	}

	if err := manager.AddServer(config); err != nil {
		t.Fatalf("AddServer failed: %v", err)
	}

	// List servers
	servers := manager.ListServers()
	if len(servers) != 1 {
		t.Errorf("expected 1 server, got %d", len(servers))
	}

	// Add duplicate server
	if err := manager.AddServer(config); err == nil {
		t.Error("expected error for duplicate server")
	}

	// Get client
	client, err := manager.GetClient("test-server")
	if err != nil {
		t.Fatalf("GetClient failed: %v", err)
	}
	if client == nil {
		t.Fatal("GetClient returned nil")
	}

	// Get non-existent client
	_, err = manager.GetClient("non-existent")
	if err == nil {
		t.Error("expected error for non-existent server")
	}
}

func TestToolTypes(t *testing.T) {
	tool := Tool{
		Name:        "test-tool",
		Description: "A test tool",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"input": map[string]interface{}{
					"type": "string",
				},
			},
		},
	}

	if tool.Name != "test-tool" {
		t.Errorf("expected name test-tool, got %s", tool.Name)
	}
	if tool.Description != "A test tool" {
		t.Errorf("expected description 'A test tool', got %s", tool.Description)
	}
}

func TestToolCall(t *testing.T) {
	call := ToolCall{
		Name: "test-tool",
		Arguments: map[string]interface{}{
			"input": "test",
		},
	}

	if call.Name != "test-tool" {
		t.Errorf("expected name test-tool, got %s", call.Name)
	}
	if call.Arguments["input"] != "test" {
		t.Errorf("expected input 'test', got %v", call.Arguments["input"])
	}
}

func TestToolResult(t *testing.T) {
	result := ToolResult{
		Content: []ToolContent{
			{
				Type: "text",
				Text: "test result",
			},
		},
		IsError: false,
	}

	if len(result.Content) != 1 {
		t.Errorf("expected 1 content, got %d", len(result.Content))
	}
	if result.Content[0].Type != "text" {
		t.Errorf("expected type text, got %s", result.Content[0].Type)
	}
	if result.Content[0].Text != "test result" {
		t.Errorf("expected text 'test result', got %s", result.Content[0].Text)
	}
	if result.IsError {
		t.Error("expected IsError to be false")
	}
}

func TestRequestResponse(t *testing.T) {
	req := Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "test",
		Params:  map[string]string{"key": "value"},
	}

	if req.JSONRPC != "2.0" {
		t.Errorf("expected jsonrpc 2.0, got %s", req.JSONRPC)
	}
	if req.ID != 1 {
		t.Errorf("expected id 1, got %d", req.ID)
	}
	if req.Method != "test" {
		t.Errorf("expected method test, got %s", req.Method)
	}

	resp := Response{
		JSONRPC: "2.0",
		ID:      1,
		Result:  []byte(`{"key": "value"}`),
	}

	if resp.JSONRPC != "2.0" {
		t.Errorf("expected jsonrpc 2.0, got %s", resp.JSONRPC)
	}
	if resp.ID != 1 {
		t.Errorf("expected id 1, got %d", resp.ID)
	}
	if resp.Error != nil {
		t.Error("expected error to be nil")
	}
}

func TestServerConfig(t *testing.T) {
	config := ServerConfig{
		Name: "test-server",
		Transport: TransportConfig{
			Type:    "stdio",
			Command: "npx",
			Args:    []string{"-y", "@modelcontextprotocol/server-everything"},
			Env: map[string]string{
				"KEY": "value",
			},
			Timeout: 30,
		},
		Enabled: true,
	}

	if config.Name != "test-server" {
		t.Errorf("expected name test-server, got %s", config.Name)
	}
	if config.Transport.Type != "stdio" {
		t.Errorf("expected type stdio, got %s", config.Transport.Type)
	}
	if config.Transport.Command != "npx" {
		t.Errorf("expected command npx, got %s", config.Transport.Command)
	}
	if len(config.Transport.Args) != 2 {
		t.Errorf("expected 2 args, got %d", len(config.Transport.Args))
	}
	if config.Transport.Env["KEY"] != "value" {
		t.Errorf("expected env KEY=value, got %s", config.Transport.Env["KEY"])
	}
	if !config.Enabled {
		t.Error("expected enabled to be true")
	}
}
