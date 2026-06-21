package pkg

import (
	"context"
	"testing"
	"time"
)

func TestNewAgent(t *testing.T) {
	agent, err := NewAgent()
	if err != nil {
		t.Fatalf("NewAgent failed: %v", err)
	}
	if agent == nil {
		t.Fatal("NewAgent returned nil")
	}
	if agent.options == nil {
		t.Fatal("agent.options is nil")
	}
}

func TestNewAgentWithOptions(t *testing.T) {
	dir := t.TempDir()
	agent, err := NewAgent(
		WithMaxLoops(10),
		WithMaxTokens(100000),
		WithMaxCost(5.0),
		WithMaxWallTime(5*time.Minute),
		WithVerbose(true),
		WithModel("deepseek-chat"),
		WithDataDir(dir),
	)
	if err != nil {
		t.Fatalf("NewAgent failed: %v", err)
	}

	if agent.options.MaxLoops != 10 {
		t.Errorf("expected MaxLoops 10, got %d", agent.options.MaxLoops)
	}
	if agent.options.MaxTokens != 100000 {
		t.Errorf("expected MaxTokens 100000, got %d", agent.options.MaxTokens)
	}
	if agent.options.MaxCost != 5.0 {
		t.Errorf("expected MaxCost 5.0, got %f", agent.options.MaxCost)
	}
	if agent.options.MaxWallTime != 5*time.Minute {
		t.Errorf("expected MaxWallTime 5m, got %s", agent.options.MaxWallTime)
	}
	if !agent.options.Verbose {
		t.Error("expected Verbose to be true")
	}
	if agent.options.Model != "deepseek-chat" {
		t.Errorf("expected Model deepseek-chat, got %s", agent.options.Model)
	}
	if agent.options.DataDir != dir {
		t.Errorf("expected DataDir %s, got %s", dir, agent.options.DataDir)
	}
}

func TestAgentSetGoal(t *testing.T) {
	agent, err := NewAgent()
	if err != nil {
		t.Fatalf("NewAgent failed: %v", err)
	}

	agent.SetGoal("test goal")
	if agent.goal != "test goal" {
		t.Errorf("expected goal test goal, got %s", agent.goal)
	}
}

func TestAgentSetProvider(t *testing.T) {
	agent, err := NewAgent()
	if err != nil {
		t.Fatalf("NewAgent failed: %v", err)
	}

	provider := NewHeuristicProvider("test")
	agent.SetProvider(provider)
	if agent.provider == nil {
		t.Error("expected provider to be set")
	}
}

func TestAgentRunWithoutGoal(t *testing.T) {
	agent, err := NewAgent()
	if err != nil {
		t.Fatalf("NewAgent failed: %v", err)
	}

	_, err = agent.Run()
	if err == nil {
		t.Error("expected error when running without goal")
	}
}

func TestAgentRunWithHeuristicProvider(t *testing.T) {
	agent, err := NewAgent(
		WithMaxWallTime(10*time.Second),
		WithDataDir(t.TempDir()),
	)
	if err != nil {
		t.Fatalf("NewAgent failed: %v", err)
	}
	agent.SetGoal("test goal")

	result, err := agent.Run()
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if result == nil {
		t.Fatal("Run returned nil result")
	}

	if result.Status != StatusCompleted {
		t.Errorf("expected status completed, got %s", result.Status)
	}

	if result.Answer == "" {
		t.Error("expected non-empty answer")
	}

	if result.Loops == 0 {
		t.Error("expected loops > 0")
	}

	if result.Duration == 0 {
		t.Error("expected duration > 0")
	}
}

func TestHeuristicProviderChat(t *testing.T) {
	provider := NewHeuristicProvider("test")
	ctx := context.Background()

	// Test planner prompt
	system := "You are a task planner"
	user := "分析当前项目"
	response, err := provider.Chat(ctx, system, user)
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}
	if response == "" {
		t.Error("expected non-empty response for planner prompt")
	}

	// Test reflector prompt
	system = "You are a task reflector"
	user = "test goal"
	response, err = provider.Chat(ctx, system, user)
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}
	if response == "" {
		t.Error("expected non-empty response for reflector prompt")
	}

	// Test generic prompt
	system = "You are a helpful assistant"
	user = "hello"
	response, err = provider.Chat(ctx, system, user)
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}
	if response == "" {
		t.Error("expected non-empty response for generic prompt")
	}
}

func TestHeuristicProviderChatCancelled(t *testing.T) {
	provider := NewHeuristicProvider("test")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := provider.Chat(ctx, "system", "user")
	if err == nil {
		t.Error("expected error for cancelled context")
	}
}

func TestErrString(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{"nil", nil, ""},
		{"error", context.DeadlineExceeded, "context deadline exceeded"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ErrString(tt.err)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestTruncateStr(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		n        int
		expected string
	}{
		{"short", "hello", 10, "hello"},
		{"exact", "hello", 5, "hello"},
		{"long", "hello world", 5, "hello…"},
		{"empty", "", 5, ""},
		{"unicode", "你好世界", 2, "你好…"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TruncateStr(tt.s, tt.n)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestAgentStatusConstants(t *testing.T) {
	if StatusCompleted != "completed" {
		t.Errorf("expected StatusCompleted to be 'completed', got %s", StatusCompleted)
	}
	if StatusFailed != "failed" {
		t.Errorf("expected StatusFailed to be 'failed', got %s", StatusFailed)
	}
	if StatusCancelled != "cancelled" {
		t.Errorf("expected StatusCancelled to be 'cancelled', got %s", StatusCancelled)
	}
}

func TestWithGoal(t *testing.T) {
	opt := WithGoal("test")
	if opt == nil {
		t.Fatal("WithGoal returned nil")
	}
}
