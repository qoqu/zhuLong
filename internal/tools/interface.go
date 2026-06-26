package tools

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Tool interface for calling tools
type Tool interface {
	Name() string
	Description() string
	Call(ctx context.Context, params map[string]interface{}) (string, error)
}

// Registry manages available tools
type Registry struct {
	tools map[string]Tool
}

// NewRegistry creates a new tool registry
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

// Register registers a tool
func (r *Registry) Register(tool Tool) {
	r.tools[tool.Name()] = tool
}

// Get gets a tool by name
func (r *Registry) Get(name string) (Tool, error) {
	tool, ok := r.tools[name]
	if !ok {
		return nil, fmt.Errorf("tool not found: %s", name)
	}
	return tool, nil
}

// List lists all registered tools
func (r *Registry) List() []string {
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}

// ReadFileTool reads a file
type ReadFileTool struct{}

// Name returns the tool name
func (t *ReadFileTool) Name() string {
	return "read_file"
}

// Description returns the tool description
func (t *ReadFileTool) Description() string {
	return "Read the contents of a file"
}

// Call reads a file
func (t *ReadFileTool) Call(ctx context.Context, params map[string]interface{}) (string, error) {
	path, ok := params["path"].(string)
	if !ok {
		return "", fmt.Errorf("missing parameter: path")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	return string(data), nil
}

// WriteFileTool writes a file
type WriteFileTool struct{}

// Name returns the tool name
func (t *WriteFileTool) Name() string {
	return "write_file"
}

// Description returns the tool description
func (t *WriteFileTool) Description() string {
	return "Write content to a file"
}

// Call writes a file
func (t *WriteFileTool) Call(ctx context.Context, params map[string]interface{}) (string, error) {
	path, ok := params["path"].(string)
	if !ok {
		return "", fmt.Errorf("missing parameter: path")
	}

	content, ok := params["content"].(string)
	if !ok {
		return "", fmt.Errorf("missing parameter: content")
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return fmt.Sprintf("File written: %s", path), nil
}

// SearchFileTool searches for files
type SearchFileTool struct{}

// Name returns the tool name
func (t *SearchFileTool) Name() string {
	return "search_file"
}

// Description returns the tool description
func (t *SearchFileTool) Description() string {
	return "Search for files matching a pattern"
}

// Call searches for files
func (t *SearchFileTool) Call(ctx context.Context, params map[string]interface{}) (string, error) {
	pattern, ok := params["pattern"].(string)
	if !ok {
		return "", fmt.Errorf("missing parameter: pattern")
	}

	dir, _ := params["dir"].(string)
	if dir == "" {
		dir = "."
	}

	// 默认排除目录
	excludeDirs := map[string]bool{
		".git": true, "node_modules": true, ".workbuddy": true,
		"vendor": true, "__pycache__": true, ".vscode": true,
		"dist": true, "build": true, ".next": true,
	}

	var matches []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		// 跳过排除目录
		if info.IsDir() && excludeDirs[info.Name()] {
			return filepath.SkipDir
		}
		if info.IsDir() {
			return nil
		}

		matched, _ := filepath.Match(pattern, info.Name())
		if matched {
			rel, _ := filepath.Rel(dir, path)
			matches = append(matches, rel)
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("search failed: %w", err)
	}

	if len(matches) == 0 {
		return "No files found.", nil
	}

	// 限制输出数量
	if len(matches) > 30 {
		result := fmt.Sprintf("Found %d files (showing first 30):\n", len(matches))
		for _, m := range matches[:30] {
			result += m + "\n"
		}
		result += fmt.Sprintf("... and %d more", len(matches)-30)
		return result, nil
	}

	return strings.Join(matches, "\n"), nil
}

// ExecuteCommandTool executes a shell command
type ExecuteCommandTool struct{}

// Name returns the tool name
func (t *ExecuteCommandTool) Name() string {
	return "execute_command"
}

// Description returns the tool description
func (t *ExecuteCommandTool) Description() string {
	return "Execute a shell command"
}

// Call executes a command
func (t *ExecuteCommandTool) Call(ctx context.Context, params map[string]interface{}) (string, error) {
	command, ok := params["command"].(string)
	if !ok {
		return "", fmt.Errorf("missing parameter: command")
	}

	// 检测操作系统，选择正确的 shell 和编码
	if runtime.GOOS == "windows" {
		// Windows: 先 chcp 65001 切换到 UTF-8 代码页，再执行用户命令
		// CHCP 是 cmd 内部命令，不能作为环境变量，必须作为命令前缀
		// 同时设置代码页到 65001 确保中文输出正常
		fullCmd := fmt.Sprintf("chcp 65001 >nul 2>&1 & %s", command)
		cmd := exec.CommandContext(ctx, "cmd", "/c", fullCmd)
		output, err := cmd.CombinedOutput()
		// 即使有错误也返回输出（可能是部分成功）
		result := string(output)
		if err != nil && result == "" {
			return result, fmt.Errorf("command failed: %w", err)
		}
		return result, nil
	}

	// Linux/macOS: 直接执行
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	output, err := cmd.CombinedOutput()
	result := string(output)
	if err != nil && result == "" {
		return result, fmt.Errorf("command failed: %w", err)
	}
	return result, nil
}

// RegisterBuiltinTools registers all built-in tools
func RegisterBuiltinTools(registry *Registry) {
	registry.Register(&ReadFileTool{})
	registry.Register(&WriteFileTool{})
	registry.Register(&SearchFileTool{})
	registry.Register(&ExecuteCommandTool{})
	registry.Register(&WebSearchTool{})
	registry.Register(&ListDirTool{})
}

// ListDirTool lists directory contents
type ListDirTool struct{}

func (t *ListDirTool) Name() string        { return "list_dir" }
func (t *ListDirTool) Description() string { return "List files and subdirectories in a directory" }

func (t *ListDirTool) Call(ctx context.Context, params map[string]interface{}) (string, error) {
	dir, _ := params["path"].(string)
	if dir == "" {
		dir = "."
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("cannot read directory: %w", err)
	}

	// 排除目录
	excludeDirs := map[string]bool{
		".git": true, "node_modules": true, ".workbuddy": true,
		"vendor": true, "__pycache__": true,
	}

	var dirs, files []string
	for _, e := range entries {
		if e.IsDir() {
			if excludeDirs[e.Name()] {
				continue
			}
			dirs = append(dirs, e.Name()+"/")
		} else {
			files = append(files, e.Name())
		}
	}

	var result strings.Builder
	if len(dirs) > 0 {
		result.WriteString("Directories:\n")
		for _, d := range dirs {
			result.WriteString("  " + d + "\n")
		}
	}
	if len(files) > 0 {
		result.WriteString("Files:\n")
		for _, f := range files {
			result.WriteString("  " + f + "\n")
		}
	}
	if result.Len() == 0 {
		return "(empty directory)", nil
	}
	return result.String(), nil
}
