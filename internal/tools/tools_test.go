package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestNewRegistry(t *testing.T) {
	registry := NewRegistry()

	if registry == nil {
		t.Error("NewRegistry() should not return nil")
	}
}

func TestRegistry_Register(t *testing.T) {
	registry := NewRegistry()
	tool := &ReadFileTool{}

	registry.Register(tool)

	if len(registry.List()) != 1 {
		t.Errorf("Registry.List() length = %v, want 1", len(registry.List()))
	}
}

func TestRegistry_Get(t *testing.T) {
	registry := NewRegistry()
	tool := &ReadFileTool{}

	registry.Register(tool)

	// Get existing tool
	got, err := registry.Get("read_file")
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

func TestRegistry_List(t *testing.T) {
	registry := NewRegistry()

	registry.Register(&ReadFileTool{})
	registry.Register(&WriteFileTool{})
	registry.Register(&SearchFileTool{})

	list := registry.List()

	if len(list) != 3 {
		t.Errorf("Registry.List() length = %v, want 3", len(list))
	}
}

func TestReadFileTool(t *testing.T) {
	tool := &ReadFileTool{}

	if tool.Name() != "read_file" {
		t.Errorf("ReadFileTool.Name() = %v, want read_file", tool.Name())
	}
	if tool.Description() == "" {
		t.Error("ReadFileTool.Description() should not be empty")
	}
}

func TestReadFileTool_Call(t *testing.T) {
	tool := &ReadFileTool{}

	// Create a temporary file
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.txt")
	content := "Hello, World!"
	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Read the file
	result, err := tool.Call(context.Background(), map[string]interface{}{
		"path": filePath,
	})
	if err != nil {
		t.Errorf("ReadFileTool.Call() error = %v", err)
	}
	if result != content {
		t.Errorf("ReadFileTool.Call() = %v, want %v", result, content)
	}
}

func TestReadFileTool_Call_MissingPath(t *testing.T) {
	tool := &ReadFileTool{}

	_, err := tool.Call(context.Background(), map[string]interface{}{})
	if err == nil {
		t.Error("ReadFileTool.Call() should return error when path is missing")
	}
}

func TestReadFileTool_Call_FileNotFound(t *testing.T) {
	tool := &ReadFileTool{}

	_, err := tool.Call(context.Background(), map[string]interface{}{
		"path": "/nonexistent/file.txt",
	})
	if err == nil {
		t.Error("ReadFileTool.Call() should return error when file doesn't exist")
	}
}

func TestWriteFileTool(t *testing.T) {
	tool := &WriteFileTool{}

	if tool.Name() != "write_file" {
		t.Errorf("WriteFileTool.Name() = %v, want write_file", tool.Name())
	}
	if tool.Description() == "" {
		t.Error("WriteFileTool.Description() should not be empty")
	}
}

func TestWriteFileTool_Call(t *testing.T) {
	tool := &WriteFileTool{}

	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.txt")
	content := "Hello, World!"

	result, err := tool.Call(context.Background(), map[string]interface{}{
		"path":    filePath,
		"content": content,
	})
	if err != nil {
		t.Errorf("WriteFileTool.Call() error = %v", err)
	}
	if result == "" {
		t.Error("WriteFileTool.Call() should return a message")
	}

	// Verify file was written
	readContent, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read written file: %v", err)
	}
	if string(readContent) != content {
		t.Errorf("Written content = %v, want %v", string(readContent), content)
	}
}

func TestWriteFileTool_Call_MissingParams(t *testing.T) {
	tool := &WriteFileTool{}

	// Missing path
	_, err := tool.Call(context.Background(), map[string]interface{}{
		"content": "test",
	})
	if err == nil {
		t.Error("WriteFileTool.Call() should return error when path is missing")
	}

	// Missing content
	_, err = tool.Call(context.Background(), map[string]interface{}{
		"path": "/tmp/test.txt",
	})
	if err == nil {
		t.Error("WriteFileTool.Call() should return error when content is missing")
	}
}

func TestSearchFileTool(t *testing.T) {
	tool := &SearchFileTool{}

	if tool.Name() != "search_file" {
		t.Errorf("SearchFileTool.Name() = %v, want search_file", tool.Name())
	}
	if tool.Description() == "" {
		t.Error("SearchFileTool.Description() should not be empty")
	}
}

func TestSearchFileTool_Call(t *testing.T) {
	tool := &SearchFileTool{}

	dir := t.TempDir()

	// Create test files
	os.WriteFile(filepath.Join(dir, "test1.txt"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(dir, "test2.txt"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(dir, "other.log"), []byte("test"), 0644)

	// Search for .txt files
	result, err := tool.Call(context.Background(), map[string]interface{}{
		"pattern": "*.txt",
		"dir":     dir,
	})
	if err != nil {
		t.Errorf("SearchFileTool.Call() error = %v", err)
	}
	if result == "" {
		t.Error("SearchFileTool.Call() should return results")
	}
}

func TestSearchFileTool_Call_NoMatches(t *testing.T) {
	tool := &SearchFileTool{}

	dir := t.TempDir()

	result, err := tool.Call(context.Background(), map[string]interface{}{
		"pattern": "*.xyz",
		"dir":     dir,
	})
	if err != nil {
		t.Errorf("SearchFileTool.Call() error = %v", err)
	}
	if result != "No files found" {
		t.Errorf("SearchFileTool.Call() = %v, want 'No files found'", result)
	}
}

func TestExecuteCommandTool(t *testing.T) {
	tool := &ExecuteCommandTool{}

	if tool.Name() != "execute_command" {
		t.Errorf("ExecuteCommandTool.Name() = %v, want execute_command", tool.Name())
	}
	if tool.Description() == "" {
		t.Error("ExecuteCommandTool.Description() should not be empty")
	}
}

func TestExecuteCommandTool_Call(t *testing.T) {
	tool := &ExecuteCommandTool{}

	result, err := tool.Call(context.Background(), map[string]interface{}{
		"command": "echo hello",
	})
	if err != nil {
		t.Errorf("ExecuteCommandTool.Call() error = %v", err)
	}
	if result == "" {
		t.Error("ExecuteCommandTool.Call() should return output")
	}
}

func TestExecuteCommandTool_Call_MissingCommand(t *testing.T) {
	tool := &ExecuteCommandTool{}

	_, err := tool.Call(context.Background(), map[string]interface{}{})
	if err == nil {
		t.Error("ExecuteCommandTool.Call() should return error when command is missing")
	}
}

func TestRegisterBuiltinTools(t *testing.T) {
	registry := NewRegistry()

	RegisterBuiltinTools(registry)

	if len(registry.List()) != 5 {
		t.Errorf("RegisterBuiltinTools() registered %v tools, want 5", len(registry.List()))
	}

	// Verify all tools are registered
	toolNames := []string{"read_file", "write_file", "search_file", "execute_command", "web_search"}
	for _, name := range toolNames {
		_, err := registry.Get(name)
		if err != nil {
			t.Errorf("Registry.Get(%v) error = %v", name, err)
		}
	}
}
