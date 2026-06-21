package human

import (
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	if config == nil {
		t.Fatal("DefaultConfig returned nil")
	}
	if !config.Enabled {
		t.Error("expected Enabled to be true")
	}
	if !config.Interactive {
		t.Error("expected Interactive to be true")
	}
	if len(config.Breakpoints) != 2 {
		t.Errorf("expected 2 breakpoints, got %d", len(config.Breakpoints))
	}
}

func TestNewManager(t *testing.T) {
	manager := NewManager(nil)
	if manager == nil {
		t.Fatal("NewManager returned nil")
	}
	if manager.config == nil {
		t.Fatal("manager.config is nil")
	}
	if manager.reader == nil {
		t.Fatal("manager.reader is nil")
	}
}

func TestNewManagerWithConfig(t *testing.T) {
	config := &Config{
		Enabled:     false,
		Interactive: false,
		Breakpoints: []BreakpointType{BreakpointOnError},
	}
	manager := NewManager(config)
	if manager.config != config {
		t.Error("expected manager.config to be the provided config")
	}
}

func TestShouldPause(t *testing.T) {
	config := &Config{
		Enabled: true,
		Breakpoints: []BreakpointType{
			BreakpointOnError,
			BreakpointOnReplan,
			BreakpointCostExceed,
		},
	}
	manager := NewManager(config)

	tests := []struct {
		event    string
		expected bool
	}{
		{"error", true},
		{"replan", true},
		{"cost_exceed", true},
		{"plan_start", false},
		{"step_done", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.event, func(t *testing.T) {
			result := manager.ShouldPause(tt.event, nil)
			if result != tt.expected {
				t.Errorf("ShouldPause(%s) = %v, want %v", tt.event, result, tt.expected)
			}
		})
	}
}

func TestShouldPauseDisabled(t *testing.T) {
	config := &Config{
		Enabled: false,
		Breakpoints: []BreakpointType{
			BreakpointOnError,
		},
	}
	manager := NewManager(config)

	result := manager.ShouldPause("error", nil)
	if result {
		t.Error("expected ShouldPause to return false when disabled")
	}
}

func TestWaitForInputNonInteractive(t *testing.T) {
	config := &Config{
		Enabled:     true,
		Interactive: false,
	}
	manager := NewManager(config)

	input, err := manager.WaitForInput("test prompt")
	if err != nil {
		t.Fatalf("WaitForInput failed: %v", err)
	}

	if input.Action != "continue" {
		t.Errorf("expected action continue, got %s", input.Action)
	}
	if input.Content != "" {
		t.Errorf("expected empty content, got %s", input.Content)
	}
}

func TestConfirmNonInteractive(t *testing.T) {
	config := &Config{
		Enabled:     true,
		Interactive: false,
	}
	manager := NewManager(config)

	result, err := manager.Confirm("test prompt")
	if err != nil {
		t.Fatalf("Confirm failed: %v", err)
	}

	if !result {
		t.Error("expected Confirm to return true in non-interactive mode")
	}
}

func TestBreakpointTypeConstants(t *testing.T) {
	if BreakpointOnError != 0 {
		t.Errorf("expected BreakpointOnError to be 0, got %d", BreakpointOnError)
	}
	if BreakpointOnReplan != 1 {
		t.Errorf("expected BreakpointOnReplan to be 1, got %d", BreakpointOnReplan)
	}
	if BreakpointCostExceed != 2 {
		t.Errorf("expected BreakpointCostExceed to be 2, got %d", BreakpointCostExceed)
	}
	if BreakpointCustom != 3 {
		t.Errorf("expected BreakpointCustom to be 3, got %d", BreakpointCustom)
	}
}

func TestDisplayStatus(t *testing.T) {
	config := &Config{
		Enabled:     true,
		Interactive: false, // won't print anything
	}
	manager := NewManager(config)

	// Should not panic
	manager.DisplayStatus("test status")
}

func TestDisplayProgress(t *testing.T) {
	config := &Config{
		Enabled:     true,
		Interactive: false, // won't print anything
	}
	manager := NewManager(config)

	// Should not panic
	manager.DisplayProgress(1, 10, "test progress")
}
