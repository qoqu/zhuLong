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

	dir, _ := params["path"].(string)
	if dir == "" {
		dir, _ = params["dir"].(string)
	}
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
		// Windows: 自动转换常见 Unix 命令为 Windows 等价命令
		command = convertUnixToWindows(command)
		// 使用 PowerShell 执行，原生支持 UTF-8
		psCmd := fmt.Sprintf("[Console]::OutputEncoding = [System.Text.Encoding]::UTF8; %s", command)
		cmd := exec.CommandContext(ctx, "powershell", "-NoProfile", "-Command", psCmd)
		output, err := cmd.CombinedOutput()
		result := strings.TrimSpace(string(output))
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

// convertUnixToWindows 将常见 Unix 命令转换为 Windows 等价命令
func convertUnixToWindows(cmd string) string {
	cmd = strings.TrimSpace(cmd)

	// 处理纯命令名（无参数）的情况（PowerShell 语法）
	singleCmds := map[string]string{
		"ls":      "Get-ChildItem -Name",
		"dir":     "Get-ChildItem -Name",
		"cat":     "Write-Host 'Please provide file path'",
		"head":    "Write-Host 'Please provide file path'",
		"tail":    "Write-Host 'Please provide file path'",
		"grep":    "Write-Host 'Please provide search pattern'",
		"find":    "Write-Host 'Please provide search target'",
		"wc":      "Write-Host 'Please provide file path'",
		"pwd":     "Get-Location",
		"whoami":  "$env:USERNAME",
		"date":    "Get-Date -Format 'yyyy-MM-dd HH:mm:ss'",
		"echo":    "Write-Host",
	}
	if replacement, ok := singleCmds[cmd]; ok {
		return replacement
	}

	// grep pattern file → findstr /i "pattern" file
	if strings.HasPrefix(cmd, "grep ") {
		// grep -r pattern dir → findstr /s /i "pattern" dir\*
		// grep pattern file → findstr /i "pattern" file
		parts := strings.SplitN(cmd, " ", 4)
		if len(parts) >= 3 {
			flags := ""
			pattern := ""
			target := ""
			idx := 1
			// 跳过 -r, -n 等 grep 标志
			for idx < len(parts) && strings.HasPrefix(parts[idx], "-") {
				flags += parts[idx][1:]
				idx++
			}
			if idx < len(parts) {
				pattern = strings.Trim(parts[idx], "\"'")
				idx++
			}
			if idx < len(parts) {
				target = parts[idx]
			}

			findstrFlags := "/i"
			if strings.Contains(flags, "r") || strings.Contains(flags, "R") {
				findstrFlags += " /s" // 递归搜索
			}
			if strings.Contains(flags, "n") {
				findstrFlags += " /n" // 显示行号
			}

			if target != "" {
				return fmt.Sprintf("findstr %s \"%s\" %s\\*", findstrFlags, pattern, target)
			}
			return fmt.Sprintf("findstr %s \"%s\" *", findstrFlags, pattern)
		}
	}

	// head -n N file 或 head -N file → Get-Content file -Head N
	if strings.HasPrefix(cmd, "head ") {
		parts := strings.Fields(cmd)
		n := "10"
		file := ""
		for i := 1; i < len(parts); i++ {
			if parts[i] == "-n" && i+1 < len(parts) {
				n = parts[i+1]
				i++
			} else if strings.HasPrefix(parts[i], "-") && len(parts[i]) > 1 {
				// 处理 -10 格式（无 -n 前缀）
				n = parts[i][1:]
			} else if !strings.HasPrefix(parts[i], "-") {
				file = parts[i]
			}
		}
		if file != "" {
			return fmt.Sprintf("Get-Content '%s' -Head %s", file, n)
		}
	}

	// tail -n N file → Get-Content file -Tail N
	if strings.HasPrefix(cmd, "tail ") {
		parts := strings.Fields(cmd)
		n := "10"
		file := ""
		for i := 1; i < len(parts); i++ {
			if parts[i] == "-n" && i+1 < len(parts) {
				n = parts[i+1]
				i++
			} else if !strings.HasPrefix(parts[i], "-") {
				file = parts[i]
			}
		}
		if file != "" {
			return fmt.Sprintf("Get-Content '%s' -Tail %s", file, n)
		}
	}

	// ls → dir /b
	if cmd == "ls" || strings.HasPrefix(cmd, "ls ") {
		return "dir /b"
	}

	// cat file → type file
	if strings.HasPrefix(cmd, "cat ") {
		file := strings.TrimPrefix(cmd, "cat ")
		return "type " + file
	}

	// wc -l file → (Get-Content file).Count
	if strings.HasPrefix(cmd, "wc ") {
		parts := strings.Fields(cmd)
		file := ""
		for _, p := range parts {
			if !strings.HasPrefix(p, "-") {
				file = p
			}
		}
		if file != "" {
			return fmt.Sprintf("(Get-Content '%s').Count", file)
		}
	}

	// find file → where file
	if strings.HasPrefix(cmd, "find ") {
		return "where " + strings.TrimPrefix(cmd, "find ")
	}

	// curl url → Invoke-WebRequest
	if strings.HasPrefix(cmd, "curl ") {
		url := strings.TrimPrefix(cmd, "curl ")
		url = strings.Trim(url, "\"'")
		return fmt.Sprintf("(Invoke-WebRequest -Uri '%s').Content", url)
	}

	// echo text → Write-Output
	if strings.HasPrefix(cmd, "echo ") {
		text := strings.TrimPrefix(cmd, "echo ")
		return fmt.Sprintf("Write-Host '%s'", text)
	}

	// Windows 上遇到 bash 语法时，返回 ASCII 提示信息
	if runtime.GOOS == "windows" {
		// 只检测明确的 bash 语法结构，避免误判
		if strings.Contains(cmd, "for ") && strings.Contains(cmd, "; do ") {
			return fmt.Sprintf("echo [ERROR] bash syntax detected. Use Windows commands instead. Command: %s", cmd)
		}
		if strings.HasPrefix(cmd, "for ") && strings.Contains(cmd, " in ") && strings.Contains(cmd, "; do") {
			return fmt.Sprintf("echo [ERROR] bash syntax detected. Use Windows commands instead. Command: %s", cmd)
		}
	}

	return cmd
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
