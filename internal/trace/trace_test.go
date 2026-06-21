package trace

import (
	"os"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	if config == nil {
		t.Fatal("DefaultConfig returned nil")
	}
	if !config.Enabled {
		t.Error("expected Enabled to be true")
	}
	if config.OutputDir != ".zhulong/traces" {
		t.Errorf("expected OutputDir .zhulong/traces, got %s", config.OutputDir)
	}
	if config.Format != "json" {
		t.Errorf("expected Format json, got %s", config.Format)
	}
	if config.Verbose {
		t.Error("expected Verbose to be false")
	}
}

func TestNewLogger(t *testing.T) {
	logger := NewLogger("test-session", nil)
	if logger == nil {
		t.Fatal("NewLogger returned nil")
	}
	if logger.sessionID != "test-session" {
		t.Errorf("expected sessionID test-session, got %s", logger.sessionID)
	}
	if len(logger.entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(logger.entries))
	}
}

func TestLoggerLog(t *testing.T) {
	config := &Config{
		Enabled:   true,
		OutputDir: t.TempDir(),
		Format:    "json",
		Verbose:   false,
	}
	logger := NewLogger("test-session", config)

	// Log entries
	logger.Log("plan", "plan_start", nil)
	logger.Log("plan", "plan_ok", map[string]int{"steps": 5})
	logger.Log("exec", "step_start", map[string]string{"tool": "read_file"})

	entries := logger.GetEntries()
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}

	// Verify first entry
	if entries[0].Phase != "plan" {
		t.Errorf("expected phase plan, got %s", entries[0].Phase)
	}
	if entries[0].Event != "plan_start" {
		t.Errorf("expected event plan_start, got %s", entries[0].Event)
	}
	if entries[0].Loop != 0 {
		t.Errorf("expected loop 0, got %d", entries[0].Loop)
	}
}

func TestLoggerLogWithLoop(t *testing.T) {
	config := &Config{
		Enabled:   true,
		OutputDir: t.TempDir(),
		Format:    "json",
		Verbose:   false,
	}
	logger := NewLogger("test-session", config)

	// Log entries with loop
	logger.LogWithLoop(1, "exec", "step_start", nil)
	logger.LogWithLoop(2, "exec", "step_done", nil)

	entries := logger.GetEntries()
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	// Verify loop numbers
	if entries[0].Loop != 1 {
		t.Errorf("expected loop 1, got %d", entries[0].Loop)
	}
	if entries[1].Loop != 2 {
		t.Errorf("expected loop 2, got %d", entries[1].Loop)
	}
}

func TestLoggerLogWithDuration(t *testing.T) {
	config := &Config{
		Enabled:   true,
		OutputDir: t.TempDir(),
		Format:    "json",
		Verbose:   false,
	}
	logger := NewLogger("test-session", config)

	// Log entry with duration
	logger.LogWithDuration("exec", "step_done", nil, 5*time.Second)

	entries := logger.GetEntries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	// Verify duration
	if entries[0].Duration != 5*time.Second {
		t.Errorf("expected duration 5s, got %s", entries[0].Duration)
	}
}

func TestLoggerDisabled(t *testing.T) {
	config := &Config{
		Enabled:   false,
		OutputDir: t.TempDir(),
		Format:    "json",
		Verbose:   false,
	}
	logger := NewLogger("test-session", config)

	// Log entries (should be ignored)
	logger.Log("plan", "plan_start", nil)
	logger.LogWithLoop(1, "exec", "step_start", nil)
	logger.LogWithDuration("exec", "step_done", nil, 5*time.Second)

	entries := logger.GetEntries()
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestLoggerSave(t *testing.T) {
	dir := t.TempDir()
	config := &Config{
		Enabled:   true,
		OutputDir: dir,
		Format:    "json",
		Verbose:   false,
	}
	logger := NewLogger("test-session", config)

	// Log entries
	logger.Log("plan", "plan_start", nil)
	logger.Log("plan", "plan_ok", nil)

	// Save
	if err := logger.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify file was created
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir failed: %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("expected 1 file, got %d", len(entries))
	}

	// Verify filename format
	filename := entries[0].Name()
	if len(filename) < 10 {
		t.Errorf("filename too short: %s", filename)
	}
}

func TestLoggerSaveDisabled(t *testing.T) {
	dir := t.TempDir()
	config := &Config{
		Enabled:   false,
		OutputDir: dir,
		Format:    "json",
		Verbose:   false,
	}
	logger := NewLogger("test-session", config)

	// Save (should be no-op)
	if err := logger.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify no file was created
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir failed: %v", err)
	}

	if len(entries) != 0 {
		t.Errorf("expected 0 files, got %d", len(entries))
	}
}

func TestFormatData(t *testing.T) {
	tests := []struct {
		name     string
		data     interface{}
		expected string
	}{
		{"nil", nil, ""},
		{"string", "hello", "hello"},
		{"error", os.ErrNotExist, "file does not exist"},
		{"int", 42, "42"},
		{"map", map[string]int{"a": 1}, "map[a:1]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatData(tt.data)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}
