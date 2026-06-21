package exploration

import (
	"testing"
)

func TestNewTrigger(t *testing.T) {
	trigger := NewTrigger(0.7, 1.5, []string{"read_file", "write_file"})

	if trigger == nil {
		t.Error("NewTrigger() should not return nil")
	}
	if trigger.baseTemperature != 0.7 {
		t.Errorf("Trigger baseTemperature = %v, want 0.7", trigger.baseTemperature)
	}
	if trigger.maxTemperature != 1.5 {
		t.Errorf("Trigger maxTemperature = %v, want 1.5", trigger.maxTemperature)
	}
}

func TestNewTrigger_Defaults(t *testing.T) {
	trigger := NewTrigger(0, 0, nil)

	if trigger.baseTemperature != 0.7 {
		t.Errorf("Default baseTemperature = %v, want 0.7", trigger.baseTemperature)
	}
	if trigger.maxTemperature != 1.5 {
		t.Errorf("Default maxTemperature = %v, want 1.5", trigger.maxTemperature)
	}
}

func TestTrigger_ShouldExplore(t *testing.T) {
	trigger := NewTrigger(0.7, 1.5, nil)

	if trigger.ShouldExplore("none") {
		t.Error("Should not explore when stagnation type is 'none'")
	}
	if !trigger.ShouldExplore("info_starved") {
		t.Error("Should explore when stagnation type is 'info_starved'")
	}
	if !trigger.ShouldExplore("looping") {
		t.Error("Should explore when stagnation type is 'looping'")
	}
	if !trigger.ShouldExplore("blocked") {
		t.Error("Should explore when stagnation type is 'blocked'")
	}
}

func TestTrigger_GenerateExploration(t *testing.T) {
	trigger := NewTrigger(0.7, 1.5, []string{"read_file", "write_file"})

	// Test info_starved
	action := trigger.GenerateExploration("info_starved")
	if action.Type != "increase_temperature" {
		t.Errorf("info_starved action type = %v, want increase_temperature", action.Type)
	}
	if action.Temperature <= 0.7 {
		t.Errorf("Temperature = %v, should be > 0.7", action.Temperature)
	}

	// Test looping
	action = trigger.GenerateExploration("looping")
	if action.Type != "try_new_tool" {
		t.Errorf("looping action type = %v, want try_new_tool", action.Type)
	}

	// Test blocked
	action = trigger.GenerateExploration("blocked")
	if action.Type != "change_strategy" {
		t.Errorf("blocked action type = %v, want change_strategy", action.Type)
	}
}

func TestTrigger_RecordToolUsage(t *testing.T) {
	trigger := NewTrigger(0.7, 1.5, nil)

	trigger.RecordToolUsage("read_file")
	trigger.RecordToolUsage("read_file")
	trigger.RecordToolUsage("write_file")

	usage := trigger.GetToolUsage()

	if usage["read_file"] != 2 {
		t.Errorf("read_file usage = %v, want 2", usage["read_file"])
	}
	if usage["write_file"] != 1 {
		t.Errorf("write_file usage = %v, want 1", usage["write_file"])
	}
}

func TestTrigger_Reset(t *testing.T) {
	trigger := NewTrigger(0.7, 1.5, nil)

	trigger.RecordToolUsage("read_file")
	trigger.Reset()

	usage := trigger.GetToolUsage()
	if len(usage) != 0 {
		t.Errorf("Usage after reset = %v, want empty", usage)
	}
}
