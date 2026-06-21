package compressor

import (
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if !config.PruneEnabled {
		t.Error("DefaultConfig().PruneEnabled should be true")
	}
	if config.PruneMaxAge != 2 {
		t.Errorf("DefaultConfig().PruneMaxAge = %v, want 2", config.PruneMaxAge)
	}
	if !config.KeepSignature {
		t.Error("DefaultConfig().KeepSignature should be true")
	}
	if config.MaxTokens != 8000 {
		t.Errorf("DefaultConfig().MaxTokens = %v, want 8000", config.MaxTokens)
	}
}

func TestSimpleTokenCounter_Count(t *testing.T) {
	counter := &SimpleTokenCounter{}

	tests := []struct {
		text  string
		min   int
		max   int
	}{
		{"hello", 1, 2},
		{"hello world", 2, 4},
		{"", 0, 1},
		{"abcd", 1, 1},
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			count := counter.Count(tt.text)
			if count < tt.min || count > tt.max {
				t.Errorf("SimpleTokenCounter.Count(%q) = %v, want between %v and %v", tt.text, count, tt.min, tt.max)
			}
		})
	}
}

func TestStringTokenCounter_Count(t *testing.T) {
	counter := &StringTokenCounter{}

	tests := []struct {
		text     string
		expected int
	}{
		{"hello world", 2},
		{"one two three four", 4},
		{"", 0},
		{"single", 1},
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			count := counter.Count(tt.text)
			if count != tt.expected {
				t.Errorf("StringTokenCounter.Count(%q) = %v, want %v", tt.text, count, tt.expected)
			}
		})
	}
}

func TestSimpleCompressor_Prune(t *testing.T) {
	counter := &SimpleTokenCounter{}
	config := &Config{
		PruneEnabled:  true,
		PruneMaxAge:   2,
		KeepSignature: true,
	}

	compressor := NewSimpleCompressor(counter, config)

	messages := []Message{
		{Role: "system", Content: "System prompt", Tokens: 10},
		{Role: "user", Content: "User message", Tokens: 5},
		{Role: "tool", Content: "Old tool result", ToolName: "read_file", LoopNumber: 1, Tokens: 20, Prunable: true},
		{Role: "tool", Content: "Recent tool result", ToolName: "write_file", LoopNumber: 4, Tokens: 15, Prunable: true},
		{Role: "assistant", Content: "Assistant message", Tokens: 8},
	}

	// Current loop is 5, so loop 1 is old enough to prune (5-1=4 > 2)
	pruned := compressor.Prune(messages, 5)

	if len(pruned) != 5 {
		t.Errorf("Prune() returned %v messages, want 5", len(pruned))
	}

	// The old tool result should be pruned (content replaced)
	if pruned[2].Content == "Old tool result" {
		t.Error("Old tool result should have been pruned")
	}

	// The recent tool result should not be pruned
	if pruned[3].Content != "Recent tool result" {
		t.Error("Recent tool result should not have been pruned")
	}
}

func TestSimpleCompressor_PruneDisabled(t *testing.T) {
	counter := &SimpleTokenCounter{}
	config := &Config{
		PruneEnabled:  false,
		PruneMaxAge:   2,
		KeepSignature: true,
	}

	compressor := NewSimpleCompressor(counter, config)

	messages := []Message{
		{Role: "tool", Content: "Old tool result", ToolName: "read_file", LoopNumber: 1, Tokens: 20, Prunable: true},
	}

	pruned := compressor.Prune(messages, 5)

	if len(pruned) != 1 {
		t.Errorf("Prune() returned %v messages, want 1", len(pruned))
	}
	if pruned[0].Content != "Old tool result" {
		t.Error("Tool result should not be pruned when pruning is disabled")
	}
}

func TestSimpleCompressor_PruneNonPrunable(t *testing.T) {
	counter := &SimpleTokenCounter{}
	config := &Config{
		PruneEnabled:  true,
		PruneMaxAge:   2,
		KeepSignature: true,
	}

	compressor := NewSimpleCompressor(counter, config)

	messages := []Message{
		{Role: "tool", Content: "Important result", ToolName: "read_file", LoopNumber: 1, Tokens: 20, Prunable: false},
	}

	pruned := compressor.Prune(messages, 5)

	if len(pruned) != 1 {
		t.Errorf("Prune() returned %v messages, want 1", len(pruned))
	}
	if pruned[0].Content != "Important result" {
		t.Error("Non-prunable tool result should not be pruned")
	}
}

func TestSimpleCompressor_AssembleContext(t *testing.T) {
	counter := &SimpleTokenCounter{}
	config := &Config{
		MaxTokens: 1000,
	}

	compressor := NewSimpleCompressor(counter, config)

	systemPrompt := "You are a helpful assistant"
	skeleton := "project-structure"
	sessionSummary := "Session summary"
	workingMessages := []Message{
		{Role: "user", Content: "User message", Tokens: 5},
		{Role: "assistant", Content: "Assistant message", Tokens: 10},
	}
	currentInput := "Current input"

	messages := compressor.AssembleContext(
		systemPrompt,
		skeleton,
		sessionSummary,
		workingMessages,
		currentInput,
		1000,
	)

	// Should have: system prompt + skeleton + session summary + working messages + current input
	if len(messages) < 4 {
		t.Errorf("AssembleContext() returned %v messages, want at least 4", len(messages))
	}

	// First message should be system prompt
	if messages[0].Role != "system" {
		t.Errorf("First message role = %v, want system", messages[0].Role)
	}
	if messages[0].Content != systemPrompt {
		t.Errorf("First message content = %v, want %v", messages[0].Content, systemPrompt)
	}
}

func TestSimpleCompressor_AssembleContext_TokenLimit(t *testing.T) {
	counter := &StringTokenCounter{}
	config := &Config{
		MaxTokens: 20, // Very small limit
	}

	compressor := NewSimpleCompressor(counter, config)

	systemPrompt := "You are a helpful assistant"
	skeleton := ""
	sessionSummary := ""
	workingMessages := []Message{
		{Role: "user", Content: "Message 1", Tokens: 3},
		{Role: "assistant", Content: "Message 2", Tokens: 3},
		{Role: "user", Content: "Message 3", Tokens: 3},
		{Role: "assistant", Content: "Message 4", Tokens: 3},
	}
	currentInput := "Current input"

	messages := compressor.AssembleContext(
		systemPrompt,
		skeleton,
		sessionSummary,
		workingMessages,
		currentInput,
		20,
	)

	// Should have system prompt + some working messages + current input
	// The exact number depends on token counting
	if len(messages) < 2 {
		t.Errorf("AssembleContext() returned %v messages, want at least 2", len(messages))
	}
}
