package executor

import (
	"context"
	"fmt"
	"time"
)

// Executor interface for executing steps
type Executor interface {
	// Execute executes a single step
	Execute(ctx context.Context, step Step, memory MemoryReader) (*StepResult, error)
}

// MemoryReader provides read access to memory
type MemoryReader interface {
	GetSessionSummary() string
}

// Step represents a step to execute
type Step struct {
	ID          string
	Description string
	Action      Action
	DependsOn   []string
	Breakpoint  bool
}

// Action represents an action to execute
type Action struct {
	Type   string
	Tool   string
	Params map[string]interface{}
	Prompt string
}

// StepResult contains the result of executing a step
type StepResult struct {
	StepID      string
	Success     bool
	Output      string
	TokensUsed  int
	Duration    time.Duration
	Error       error
}

// Tool interface for calling tools
type Tool interface {
	Name() string
	Description() string
	Call(ctx context.Context, params map[string]interface{}) (string, error)
}

// ToolRegistry manages available tools
type ToolRegistry struct {
	tools map[string]Tool
}

// NewToolRegistry creates a new tool registry
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]Tool),
	}
}

// Register registers a tool
func (r *ToolRegistry) Register(tool Tool) {
	r.tools[tool.Name()] = tool
}

// Get gets a tool by name
func (r *ToolRegistry) Get(name string) (Tool, error) {
	tool, ok := r.tools[name]
	if !ok {
		return nil, fmt.Errorf("tool not found: %s", name)
	}
	return tool, nil
}

// List lists all registered tool names
func (r *ToolRegistry) List() []string {
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}

// LLMExecutor implements Executor using LLM and tools
type LLMExecutor struct {
	provider LLMProvider
	tools    *ToolRegistry
	config   *Config
}

// LLMProvider interface for calling LLM
type LLMProvider interface {
	Chat(ctx context.Context, messages []Message) (string, error)
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Config contains executor configuration
type Config struct {
	Model       string
	Temperature float64
	ToolTimeout time.Duration
}

// DefaultConfig returns default executor configuration
func DefaultConfig() *Config {
	return &Config{
		Model:       "deepseek-chat",
		Temperature: 0.7,
		ToolTimeout: 30 * time.Second,
	}
}

// NewLLMExecutor creates a new LLM executor
func NewLLMExecutor(provider LLMProvider, tools *ToolRegistry, config *Config) *LLMExecutor {
	if config == nil {
		config = DefaultConfig()
	}

	return &LLMExecutor{
		provider: provider,
		tools:    tools,
		config:   config,
	}
}

// Execute executes a single step
func (e *LLMExecutor) Execute(ctx context.Context, step Step, memory MemoryReader) (*StepResult, error) {
	startTime := time.Now()

	switch step.Action.Type {
	case "tool_call":
		return e.executeToolCall(ctx, step, startTime)
	case "llm_generate":
		return e.executeLLMGenerate(ctx, step, memory, startTime)
	default:
		return &StepResult{
			StepID:   step.ID,
			Success:  false,
			Error:    fmt.Errorf("unknown action type: %s", step.Action.Type),
			Duration: time.Since(startTime),
		}, nil
	}
}

// executeToolCall executes a tool call
func (e *LLMExecutor) executeToolCall(ctx context.Context, step Step, startTime time.Time) (*StepResult, error) {
	// Get tool
	tool, err := e.tools.Get(step.Action.Tool)
	if err != nil {
		return &StepResult{
			StepID:   step.ID,
			Success:  false,
			Error:    err,
			Duration: time.Since(startTime),
		}, nil
	}

	// Create timeout context
	toolCtx, cancel := context.WithTimeout(ctx, e.config.ToolTimeout)
	defer cancel()

	// Call tool
	output, err := tool.Call(toolCtx, step.Action.Params)
	if err != nil {
		return &StepResult{
			StepID:   step.ID,
			Success:  false,
			Error:    err,
			Duration: time.Since(startTime),
		}, nil
	}

	return &StepResult{
		StepID:   step.ID,
		Success:  true,
		Output:   output,
		Duration: time.Since(startTime),
	}, nil
}

// executeLLMGenerate executes an LLM generation
func (e *LLMExecutor) executeLLMGenerate(ctx context.Context, step Step, memory MemoryReader, startTime time.Time) (*StepResult, error) {
	// Build messages
	messages := []Message{
		{Role: "system", Content: "You are a helpful assistant. Complete the following task."},
		{Role: "user", Content: step.Action.Prompt},
	}

	// Call LLM
	response, err := e.provider.Chat(ctx, messages)
	if err != nil {
		return &StepResult{
			StepID:   step.ID,
			Success:  false,
			Error:    err,
			Duration: time.Since(startTime),
		}, nil
	}

	return &StepResult{
		StepID:   step.ID,
		Success:  true,
		Output:   response,
		Duration: time.Since(startTime),
	}, nil
}
