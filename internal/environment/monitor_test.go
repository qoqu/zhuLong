package environment

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestChangeType_String(t *testing.T) {
	tests := []struct {
		changeType ChangeType
		expected   string
	}{
		{ChangeModified, "modified"},
		{ChangeAdded, "added"},
		{ChangeDeleted, "deleted"},
		{ChangeType(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.changeType.String(); got != tt.expected {
				t.Errorf("ChangeType.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestNewMonitor(t *testing.T) {
	m := NewMonitor()

	if m == nil {
		t.Error("NewMonitor() should not return nil")
	}
	if m.running {
		t.Error("Monitor should not be running initially")
	}
}

func TestMonitor_AddWatcher(t *testing.T) {
	m := NewMonitor()

	config := WatcherConfig{
		Path:         "/tmp/test",
		Recursive:    true,
		PollInterval: 1 * time.Second,
	}

	m.AddWatcher(config)

	if len(m.configs) != 1 {
		t.Errorf("configs length = %v, want 1", len(m.configs))
	}
}

func TestMonitor_StartStop(t *testing.T) {
	m := NewMonitor()

	config := WatcherConfig{
		Path:         "/tmp/test",
		Recursive:    true,
		PollInterval: 1 * time.Second,
	}

	m.AddWatcher(config)

	err := m.Start()
	if err != nil {
		t.Errorf("Start() error = %v", err)
	}

	if !m.running {
		t.Error("Monitor should be running after Start()")
	}

	// Start again should be idempotent
	err = m.Start()
	if err != nil {
		t.Errorf("Start() error = %v", err)
	}

	m.Stop()

	if m.running {
		t.Error("Monitor should not be running after Stop()")
	}
}

func TestMonitor_DetectChanges(t *testing.T) {
	// Create temporary directory
	dir := t.TempDir()

	m := NewMonitor()

	config := WatcherConfig{
		Path:         dir,
		Recursive:    false,
		PollInterval: 100 * time.Millisecond,
	}

	m.AddWatcher(config)

	// Start monitor
	m.Start()
	defer m.Stop()

	// Wait for initial scan
	time.Sleep(200 * time.Millisecond)

	// Create a new file
	filePath := filepath.Join(dir, "test.txt")
	os.WriteFile(filePath, []byte("hello"), 0644)

	// Wait for change detection
	select {
	case change := <-m.GetChanges():
		if change.Type != ChangeAdded {
			t.Errorf("Change type = %v, want ChangeAdded", change.Type)
		}
		if change.Path != filePath {
			t.Errorf("Change path = %v, want %v", change.Path, filePath)
		}
	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for change")
	}

	// Modify the file
	os.WriteFile(filePath, []byte("world"), 0644)

	// Wait for change detection
	select {
	case change := <-m.GetChanges():
		if change.Type != ChangeModified {
			t.Errorf("Change type = %v, want ChangeModified", change.Type)
		}
	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for change")
	}

	// Delete the file
	os.Remove(filePath)

	// Wait for change detection
	select {
	case change := <-m.GetChanges():
		if change.Type != ChangeDeleted {
			t.Errorf("Change type = %v, want ChangeDeleted", change.Type)
		}
	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for change")
	}
}

func TestMonitor_IgnorePatterns(t *testing.T) {
	// Create temporary directory
	dir := t.TempDir()

	m := NewMonitor()

	config := WatcherConfig{
		Path:         dir,
		Recursive:    false,
		PollInterval: 100 * time.Millisecond,
		IgnorePatterns: []string{"*.tmp"},
	}

	m.AddWatcher(config)

	// Start monitor
	m.Start()
	defer m.Stop()

	// Wait for initial scan
	time.Sleep(200 * time.Millisecond)

	// Create a file that should be detected first
	txtPath := filepath.Join(dir, "test.txt")
	os.WriteFile(txtPath, []byte("text"), 0644)

	// Wait for change detection
	select {
	case change := <-m.GetChanges():
		if change.Path != txtPath {
			t.Errorf("Change path = %v, want %v", change.Path, txtPath)
		}
	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for change")
	}

	// Create a file that should be ignored
	tmpPath := filepath.Join(dir, "test.tmp")
	os.WriteFile(tmpPath, []byte("temp"), 0644)

	// Wait a bit and check that no change is received for .tmp file
	select {
	case change := <-m.GetChanges():
		if change.Path == tmpPath {
			t.Error("Should not receive change for ignored .tmp file")
		}
	case <-time.After(500 * time.Millisecond):
		// Expected: no change for .tmp file
	}
}
