package information

import (
	"testing"
)

func TestNewInformationGain(t *testing.T) {
	ig := NewInformationGain()

	if ig == nil {
		t.Error("NewInformationGain() should not return nil")
	}
}

func TestInformationGain_AddResult(t *testing.T) {
	ig := NewInformationGain()

	result := ToolResult{
		ToolName: "read_file",
		Output:   "file content",
		Tokens:   10,
	}

	ig.AddResult(result)

	usage := ig.GetToolUsage()
	if usage["read_file"] != 1 {
		t.Errorf("Tool usage = %v, want 1", usage["read_file"])
	}
}

func TestInformationGain_EstimateGain(t *testing.T) {
	ig := NewInformationGain()

	// New tool should have max gain (1.0)
	gain := ig.EstimateGain("new_tool")
	if gain != 1.0 {
		t.Errorf("New tool gain = %v, want 1.0", gain)
	}

	// Add some results
	ig.AddResult(ToolResult{ToolName: "read_file", Output: "content1", Tokens: 10})
	ig.AddResult(ToolResult{ToolName: "read_file", Output: "content2", Tokens: 10})

	// Existing tool should have lower gain
	gain = ig.EstimateGain("read_file")
	if gain >= 1.0 {
		t.Errorf("Existing tool gain = %v, should be < 1.0", gain)
	}
}

func TestInformationGain_DiminishingGain(t *testing.T) {
	ig := NewInformationGain()

	// Add multiple results
	for i := 0; i < 5; i++ {
		ig.AddResult(ToolResult{ToolName: "read_file", Output: "content", Tokens: 10})
	}

	// Gain should decrease with more usage
	gain1 := ig.EstimateGain("read_file")
	if gain1 >= 0.8 {
		t.Errorf("Gain after 5 uses = %v, should be < 0.8", gain1)
	}
}

func TestInformationGain_GetToolUsage(t *testing.T) {
	ig := NewInformationGain()

	ig.AddResult(ToolResult{ToolName: "read_file", Output: "content", Tokens: 10})
	ig.AddResult(ToolResult{ToolName: "read_file", Output: "content", Tokens: 10})
	ig.AddResult(ToolResult{ToolName: "write_file", Output: "content", Tokens: 10})

	usage := ig.GetToolUsage()

	if usage["read_file"] != 2 {
		t.Errorf("read_file usage = %v, want 2", usage["read_file"])
	}
	if usage["write_file"] != 1 {
		t.Errorf("write_file usage = %v, want 1", usage["write_file"])
	}
}

func TestInformationGain_Reset(t *testing.T) {
	ig := NewInformationGain()

	ig.AddResult(ToolResult{ToolName: "read_file", Output: "content", Tokens: 10})
	ig.Reset()

	usage := ig.GetToolUsage()
	if len(usage) != 0 {
		t.Errorf("Usage after reset = %v, want empty", usage)
	}
}

func TestNewDensityCalculator(t *testing.T) {
	counter := &SimpleTokenCounter{}
	dc := NewDensityCalculator(counter)

	if dc == nil {
		t.Error("NewDensityCalculator() should not return nil")
	}
}

func TestDensityCalculator_Density(t *testing.T) {
	counter := &SimpleTokenCounter{}
	dc := NewDensityCalculator(counter)

	messages := []Message{
		{Role: "system", Content: "You are a helpful assistant", Tokens: 5},
		{Role: "user", Content: "Hello", Tokens: 1},
		{Role: "assistant", Content: "Hi there", Tokens: 2},
	}

	density := dc.Density(messages)

	if density <= 0 {
		t.Errorf("Density = %v, should be > 0", density)
	}
}

func TestDensityCalculator_Density_Empty(t *testing.T) {
	counter := &SimpleTokenCounter{}
	dc := NewDensityCalculator(counter)

	messages := []Message{}
	density := dc.Density(messages)

	if density != 0 {
		t.Errorf("Density for empty messages = %v, want 0", density)
	}
}

func TestDensityCalculator_OptimizeDensity(t *testing.T) {
	counter := &SimpleTokenCounter{}
	dc := NewDensityCalculator(counter)

	messages := []Message{
		{Role: "system", Content: "System prompt", Tokens: 2},
		{Role: "user", Content: "User message", Tokens: 2},
		{Role: "assistant", Content: "Assistant message", Tokens: 2},
		{Role: "tool", Content: "Tool result", Tokens: 2},
	}

	// Optimize with small token limit
	optimized := dc.OptimizeDensity(messages, 4)

	if len(optimized) > len(messages) {
		t.Errorf("Optimized length = %v, should be <= %v", len(optimized), len(messages))
	}
}

func TestSimpleTokenCounter_Count(t *testing.T) {
	counter := &SimpleTokenCounter{}

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
				t.Errorf("SimpleTokenCounter.Count(%q) = %v, want %v", tt.text, count, tt.expected)
			}
		})
	}
}
