package executor

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// MockTool implements Tool for testing
type MockTool struct {
	name        string
	description string
	output      string
	err         error
}

func (m *MockTool) Name() string        { return m.name }
func (m *MockTool) Description() string { return m.description }
func (m *MockTool) Call(ctx context.Context, params map[string]interface{}) (string, error) {
	return m.output, m.err
}

// MockMemoryReader implements MemoryReader for testing
type MockMemoryReader struct {
	sessionSummary string
}

func (m *MockMemoryReader) GetSessionSummary() string {
	return m.sessionSummary
}

// MockLLMProvider implements LLMProvider for testing
type MockLLMProvider struct {
	response string
	err      error
}

func (m *MockLLMProvider) Chat(ctx context.Context, messages []Message) (string, error) {
	return m.response, m.err
}

func TestNewToolRegistry(t *testing.T) {
	registry := NewToolRegistry()

	if registry == nil {
		t.Error("NewToolRegistry() should not return nil")
	}
}

func TestToolRegistry_Register(t *testing.T) {
	registry := NewToolRegistry()
	tool := &MockTool{name: "test-tool", description: "A test tool"}

	registry.Register(tool)

	if len(registry.List()) != 1 {
		t.Errorf("Registry.List() length = %v, want 1", len(registry.List()))
	}
}

func TestToolRegistry_Get(t *testing.T) {
	registry := NewToolRegistry()
	tool := &MockTool{name: "test-tool", description: "A test tool"}

	registry.Register(tool)

	// Get existing tool
	got, err := registry.Get("test-tool")
	if err != nil {
		t.Errorf("Registry.Get() error = %v", err)
	}
	if got != tool {
		t.Error("Registry.Get() should return the registered tool")
	}

	// Get non-existing tool
	_, err = registry.Get("nonexistent")
	if err == nil {
		t.Error("Registry.Get() should return error for non-existing tool")
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Model != "deepseek-chat" {
		t.Errorf("DefaultConfig().Model = %v, want deepseek-chat", config.Model)
	}
	if config.Temperature != 0.7 {
		t.Errorf("DefaultConfig().Temperature = %v, want 0.7", config.Temperature)
	}
	if config.ToolTimeout != 30*time.Second {
		t.Errorf("DefaultConfig().ToolTimeout = %v, want 30s", config.ToolTimeout)
	}
}

func TestLLMExecutor_Execute_ToolCall(t *testing.T) {
	provider := &MockLLMProvider{response: "test response"}
	registry := NewToolRegistry()
	tool := &MockTool{
		name:        "test-tool",
		description: "A test tool",
		output:      "tool output",
	}
	registry.Register(tool)

	executor := NewLLMExecutor(provider, registry, nil)

	step := Step{
		ID:          "step-1",
		Description: "Test step",
		Action: Action{
			Type: "tool_call",
			Tool: "test-tool",
			Params: map[string]interface{}{
				"key": "value",
			},
		},
	}

	memory := &MockMemoryReader{sessionSummary: "test"}
	result, err := executor.Execute(context.Background(), step, memory)

	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}
	if !result.Success {
		t.Error("Execute() result should be successful")
	}
	if result.Output != "tool output" {
		t.Errorf("Execute() output = %v, want 'tool output'", result.Output)
	}
}

func TestLLMExecutor_Execute_ToolNotFound(t *testing.T) {
	provider := &MockLLMProvider{response: "test response"}
	registry := NewToolRegistry()

	executor := NewLLMExecutor(provider, registry, nil)

	step := Step{
		ID:          "step-1",
		Description: "Test step",
		Action: Action{
			Type: "tool_call",
			Tool: "nonexistent",
		},
	}

	memory := &MockMemoryReader{sessionSummary: "test"}
	result, err := executor.Execute(context.Background(), step, memory)

	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}
	if result.Success {
		t.Error("Execute() result should not be successful for missing tool")
	}
}

func TestLLMExecutor_Execute_LLMGenerate(t *testing.T) {
	provider := &MockLLMProvider{response: "generated text"}
	registry := NewToolRegistry()

	executor := NewLLMExecutor(provider, registry, nil)

	step := Step{
		ID:          "step-1",
		Description: "Test step",
		Action: Action{
			Type:   "llm_generate",
			Prompt: "Generate something",
		},
	}

	memory := &MockMemoryReader{sessionSummary: "test"}
	result, err := executor.Execute(context.Background(), step, memory)

	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}
	if !result.Success {
		t.Error("Execute() result should be successful")
	}
	if result.Output != "generated text" {
		t.Errorf("Execute() output = %v, want 'generated text'", result.Output)
	}
}

func TestLLMExecutor_Execute_UnknownAction(t *testing.T) {
	provider := &MockLLMProvider{response: "test response"}
	registry := NewToolRegistry()

	executor := NewLLMExecutor(provider, registry, nil)

	step := Step{
		ID:          "step-1",
		Description: "Test step",
		Action: Action{
			Type: "unknown",
		},
	}

	memory := &MockMemoryReader{sessionSummary: "test"}
	result, err := executor.Execute(context.Background(), step, memory)

	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}
	if result.Success {
		t.Error("Execute() result should not be successful for unknown action")
	}
}

func TestLLMExecutor_Execute_ToolError(t *testing.T) {
	provider := &MockLLMProvider{response: "test response"}
	registry := NewToolRegistry()
	tool := &MockTool{
		name:   "error-tool",
		output: "",
		err:    fmt.Errorf("tool error"),
	}
	registry.Register(tool)

	executor := NewLLMExecutor(provider, registry, nil)

	step := Step{
		ID:          "step-1",
		Description: "Test step",
		Action: Action{
			Type: "tool_call",
			Tool: "error-tool",
		},
	}

	memory := &MockMemoryReader{sessionSummary: "test"}
	result, err := executor.Execute(context.Background(), step, memory)

	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}
	if result.Success {
		t.Error("Execute() result should not be successful when tool returns error")
	}
}
