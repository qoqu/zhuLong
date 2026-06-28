package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
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

// EditFileTool edits a file by replacing old_string with new_string
type EditFileTool struct{}

// Name returns the tool name
func (t *EditFileTool) Name() string {
	return "edit_file"
}

// Description returns the tool description
func (t *EditFileTool) Description() string {
	return "Edit a file by replacing old_string with new_string. More precise than write_file - only changes the specified portion."
}

// Call edits a file
func (t *EditFileTool) Call(ctx context.Context, params map[string]interface{}) (string, error) {
	path, ok := params["path"].(string)
	if !ok || path == "" {
		return "", fmt.Errorf("missing parameter: path")
	}
	oldStr, ok := params["old_string"].(string)
	if !ok {
		return "", fmt.Errorf("missing parameter: old_string")
	}
	newStr, ok := params["new_string"].(string)
	if !ok {
		return "", fmt.Errorf("missing parameter: new_string")
	}
	// replace_all 参数：是否替换所有匹配
	replaceAll := false
	if ra, ok := params["replace_all"].(bool); ok {
		replaceAll = ra
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	content := string(data)
	if oldStr == "" {
		// 空 old_string 表示在文件开头插入
		content = newStr + content
	} else if !strings.Contains(content, oldStr) {
		return "", fmt.Errorf("old_string not found in file: %s", path)
	} else {
		count := strings.Count(content, oldStr)
		if !replaceAll && count > 1 {
			return "", fmt.Errorf("old_string is not unique (%d occurrences). Use replace_all=true or provide more context.", count)
		}
		if replaceAll {
			content = strings.ReplaceAll(content, oldStr, newStr)
		} else {
			content = strings.Replace(content, oldStr, newStr, 1)
		}
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return fmt.Sprintf("File %s edited successfully", path), nil
}

// GrepContentTool searches for content within files
type GrepContentTool struct{}

// Name returns the tool name
func (t *GrepContentTool) Name() string {
	return "grep_content"
}

// Description returns the tool description
func (t *GrepContentTool) Description() string {
	return "Search for text content within files (like grep). Returns matching lines with file paths and line numbers."
}

// Call searches file contents
func (t *GrepContentTool) Call(ctx context.Context, params map[string]interface{}) (string, error) {
	keyword, ok := params["keyword"].(string)
	if !ok || keyword == "" {
		return "", fmt.Errorf("missing parameter: keyword")
	}

	dir, _ := params["path"].(string)
	if dir == "" {
		dir = "."
	}

	filePattern, _ := params["pattern"].(string)
	if filePattern == "" {
		filePattern = "*"
	}

	// 尝试编译正则表达式，失败则用简单字符串匹配
	useRegex := false
	var regexPat *regexp.Regexp
	if r, err := regexp.Compile(keyword); err == nil {
		regexPat = r
		useRegex = true
	}

	// 默认排除目录
	excludeDirs := map[string]bool{
		".git": true, "node_modules": true, ".workbuddy": true,
		"vendor": true, "__pycache__": true, ".vscode": true,
		"dist": true, "build": true, ".next": true,
	}

	var results []string
	fileCount := 0
	matchCount := 0

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
		// 检查文件扩展名
		if filePattern != "*" {
			matched, _ := filepath.Match(filePattern, info.Name())
			if !matched {
				return nil
			}
		}
		// 跳过二进制文件（简单判断：大小超过 1MB 跳过）
		if info.Size() > 1024*1024 {
			return nil
		}
		fileCount++

		// 读取文件内容并搜索
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		content := string(data)
		lines := strings.Split(content, "\n")
		for lineNum, line := range lines {
			matched := false
			if useRegex {
				matched = regexPat.MatchString(line)
			} else {
				matched = strings.Contains(strings.ToLower(line), strings.ToLower(keyword))
			}
			if matched {
				relPath, _ := filepath.Rel(dir, path)
				results = append(results, fmt.Sprintf("%s:%d: %s", relPath, lineNum+1, strings.TrimSpace(line)))
				matchCount++
				if matchCount >= 50 {
					return filepath.SkipDir
				}
			}
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("search failed: %w", err)
	}

	if len(results) == 0 {
		return fmt.Sprintf("No matches found for '%s' in %d files", keyword, fileCount), nil
	}

	header := fmt.Sprintf("Found %d matches in %d files:\n\n", matchCount, fileCount)
	return header + strings.Join(results, "\n"), nil
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

// GitTool executes git commands
type GitTool struct{}

// Name returns the tool name
func (t *GitTool) Name() string {
	return "git"
}

// Description returns the tool description
func (t *GitTool) Description() string {
	return "Execute git commands. Supports: diff, log, status, blame, show. Example: git diff, git log --oneline -5, git status"
}

// Call executes a git command
func (t *GitTool) Call(ctx context.Context, params map[string]interface{}) (string, error) {
	args, ok := params["args"].(string)
	if !ok || args == "" {
		return "", fmt.Errorf("missing parameter: args")
	}

	// 安全检查：只允许特定 git 命令
	allowedCommands := map[string]bool{
		"diff": true, "log": true, "status": true,
		"blame": true, "show": true, "branch": true,
		"remote": true, "tag": true, "stash": true,
		"ls-files": true, "rev-parse": true,
	}

	parts := strings.Fields(args)
	if len(parts) == 0 {
		return "", fmt.Errorf("empty git command")
	}

	cmdName := parts[0]
	if !allowedCommands[cmdName] {
		return "", fmt.Errorf("git command '%s' not allowed. Allowed: diff, log, status, blame, show, branch, remote, tag, stash, ls-files, rev-parse", cmdName)
	}

	// 执行 git 命令
	cmd := exec.Command("git", parts...)
	output, err := cmd.CombinedOutput()
	result := strings.TrimSpace(string(output))

	if err != nil {
		if result != "" {
			return result, nil
		}
		return "", fmt.Errorf("git %s failed: %w", cmdName, err)
	}

	if result == "" {
		return fmt.Sprintf("git %s: no output", cmdName), nil
	}
	return result, nil
}

// ParseJSONTool extracts values from JSON files
type ParseJSONTool struct{}

// Name returns the tool name
func (t *ParseJSONTool) Name() string {
	return "parse_json"
}

// Description returns the tool description
func (t *ParseJSONTool) Description() string {
	return "Extract values from a JSON file using a dot-notation path. Example: parse_json path=config.json query=settings.model"
}

// Call parses JSON and extracts a value
func (t *ParseJSONTool) Call(ctx context.Context, params map[string]interface{}) (string, error) {
	path, ok := params["path"].(string)
	if !ok || path == "" {
		return "", fmt.Errorf("missing parameter: path")
	}
	query, _ := params["query"].(string)

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// 尝试解析 JSON
	var result interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return "", fmt.Errorf("invalid JSON: %w", err)
	}

	// 如果没有 query，返回整个 JSON（格式化）
	if query == "" {
		out, _ := json.MarshalIndent(result, "", "  ")
		return string(out), nil
	}

	// 按 dot-notation 路径提取值
	parts := strings.Split(query, ".")
	current := result
	for _, part := range parts {
		switch v := current.(type) {
		case map[string]interface{}:
			val, exists := v[part]
			if !exists {
				return "", fmt.Errorf("key '%s' not found", part)
			}
			current = val
		case []interface{}:
			// 尝试将 part 解析为索引
			idx := 0
			fmt.Sscanf(part, "%d", &idx)
			if idx < 0 || idx >= len(v) {
				return "", fmt.Errorf("index %d out of range (len=%d)", idx, len(v))
			}
			current = v[idx]
		default:
			return "", fmt.Errorf("cannot navigate into %T at key '%s'", current, part)
		}
	}

	// 格式化输出
	switch v := current.(type) {
	case string:
		return v, nil
	case float64:
		return fmt.Sprintf("%g", v), nil
	case bool:
		return fmt.Sprintf("%t", v), nil
	case nil:
		return "null", nil
	default:
		out, _ := json.MarshalIndent(v, "", "  ")
		return string(out), nil
	}
}

// DiffFilesTool compares two files
type DiffFilesTool struct{}

// Name returns the tool name
func (t *DiffFilesTool) Name() string {
	return "diff_files"
}

// Description returns the tool description
func (t *DiffFilesTool) Description() string {
	return "Compare two files and show differences. Returns line-by-line diff."
}

// Call compares two files
func (t *DiffFilesTool) Call(ctx context.Context, params map[string]interface{}) (string, error) {
	file1, ok := params["file1"].(string)
	if !ok || file1 == "" {
		return "", fmt.Errorf("missing parameter: file1")
	}
	file2, ok := params["file2"].(string)
	if !ok || file2 == "" {
		return "", fmt.Errorf("missing parameter: file2")
	}

	data1, err := os.ReadFile(file1)
	if err != nil {
		return "", fmt.Errorf("failed to read %s: %w", file1, err)
	}
	data2, err := os.ReadFile(file2)
	if err != nil {
		return "", fmt.Errorf("failed to read %s: %w", file2, err)
	}

	lines1 := strings.Split(string(data1), "\n")
	lines2 := strings.Split(string(data2), "\n")

	var diff []string
	maxLines := len(lines1)
	if len(lines2) > maxLines {
		maxLines = len(lines2)
	}

	added := 0
	removed := 0
	for i := 0; i < maxLines; i++ {
		l1 := ""
		l2 := ""
		if i < len(lines1) {
			l1 = lines1[i]
		}
		if i < len(lines2) {
			l2 = lines2[i]
		}
		if l1 != l2 {
			if l1 != "" {
				diff = append(diff, fmt.Sprintf("- %d: %s", i+1, l1))
				removed++
			}
			if l2 != "" {
				diff = append(diff, fmt.Sprintf("+ %d: %s", i+1, l2))
				added++
			}
		}
	}

	if len(diff) == 0 {
		return "Files are identical", nil
	}

	header := fmt.Sprintf("Files differ: %d lines added, %d lines removed\n\n", added, removed)
	return header + strings.Join(diff, "\n"), nil
}

// BatchEditTool edits multiple files in one operation
type BatchEditTool struct{}

// Name returns the tool name
func (t *BatchEditTool) Name() string {
	return "batch_edit"
}

// Description returns the tool description
func (t *BatchEditTool) Description() string {
	return "Edit multiple files at once. Each edit has path, old_string, new_string. More efficient than multiple edit_file calls."
}

// Call performs batch edits
func (t *BatchEditTool) Call(ctx context.Context, params map[string]interface{}) (string, error) {
	editsRaw, ok := params["edits"].([]interface{})
	if !ok || len(editsRaw) == 0 {
		return "", fmt.Errorf("missing parameter: edits (array of {path, old_string, new_string})")
	}

	var results []string
	errors := []string{}

	for i, editRaw := range editsRaw {
		edit, ok := editRaw.(map[string]interface{})
		if !ok {
			errors = append(errors, fmt.Sprintf("edit[%d]: invalid format", i))
			continue
		}

		path, _ := edit["path"].(string)
		oldStr, _ := edit["old_string"].(string)
		newStr, _ := edit["new_string"].(string)

		if path == "" {
			errors = append(errors, fmt.Sprintf("edit[%d]: missing path", i))
			continue
		}

		data, err := os.ReadFile(path)
		if err != nil {
			errors = append(errors, fmt.Sprintf("edit[%d]: failed to read %s: %v", i, path, err))
			continue
		}

		content := string(data)
		if oldStr == "" {
			content = newStr + content
		} else if !strings.Contains(content, oldStr) {
			errors = append(errors, fmt.Sprintf("edit[%d]: old_string not found in %s", i, path))
			continue
		} else {
			content = strings.Replace(content, oldStr, newStr, 1)
		}

		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			errors = append(errors, fmt.Sprintf("edit[%d]: failed to write %s: %v", i, path, err))
			continue
		}
		results = append(results, fmt.Sprintf("edit[%d]: %s OK", i, path))
	}

	if len(errors) > 0 {
		return strings.Join(results, "\n") + "\n\nErrors:\n" + strings.Join(errors, "\n"), nil
	}
	return strings.Join(results, "\n"), nil
}

// ReadFileRangeTool reads specific line ranges from a file
type ReadFileRangeTool struct{}

// Name returns the tool name
func (t *ReadFileRangeTool) Name() string {
	return "read_file_range"
}

// Description returns the tool description
func (t *ReadFileRangeTool) Description() string {
	return "Read specific line range from a file. Useful for reading large files in portions."
}

// Call reads a file range
func (t *ReadFileRangeTool) Call(ctx context.Context, params map[string]interface{}) (string, error) {
	path, ok := params["path"].(string)
	if !ok || path == "" {
		return "", fmt.Errorf("missing parameter: path")
	}

	start := 1
	if s, ok := params["start"].(float64); ok && s > 0 {
		start = int(s)
	}

	limit := 100
	if l, ok := params["limit"].(float64); ok && l > 0 {
		limit = int(l)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	if start > len(lines) {
		return "", fmt.Errorf("start line %d exceeds file length %d", start, len(lines))
	}

	end := start + limit - 1
	if end > len(lines) {
		end = len(lines)
	}

	var result []string
	for i := start - 1; i < end; i++ {
		result = append(result, fmt.Sprintf("%d\t%s", i+1, lines[i]))
	}

	return strings.Join(result, "\n"), nil
}

// HTTPRequestTool makes HTTP requests
type HTTPRequestTool struct{}

// Name returns the tool name
func (t *HTTPRequestTool) Name() string {
	return "http_request"
}

// Description returns the tool description
func (t *HTTPRequestTool) Description() string {
	return "Make HTTP requests (GET/POST/PUT/DELETE). Useful for API testing and web scraping."
}

// Call makes an HTTP request
func (t *HTTPRequestTool) Call(ctx context.Context, params map[string]interface{}) (string, error) {
	url, ok := params["url"].(string)
	if !ok || url == "" {
		return "", fmt.Errorf("missing parameter: url")
	}

	method, _ := params["method"].(string)
	if method == "" {
		method = "GET"
	}

	body, _ := params["body"].(string)

	// 创建请求
	var reqBody *strings.Reader
	if body != "" {
		reqBody = strings.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// 设置默认 Content-Type
	if body != "" && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	// 执行请求
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	result := fmt.Sprintf("Status: %d %s\n\n", resp.StatusCode, resp.Status)
	if len(respBody) > 5000 {
		result += string(respBody[:5000]) + "\n... (truncated)"
	} else {
		result += string(respBody)
	}
	return result, nil
}

// MakeDirTool creates directories
type MakeDirTool struct{}

// Name returns the tool name
func (t *MakeDirTool) Name() string {
	return "make_dir"
}

// Description returns the tool description
func (t *MakeDirTool) Description() string {
	return "Create directories recursively."
}

// Call creates a directory
func (t *MakeDirTool) Call(ctx context.Context, params map[string]interface{}) (string, error) {
	path, ok := params["path"].(string)
	if !ok || path == "" {
		return "", fmt.Errorf("missing parameter: path")
	}

	if err := os.MkdirAll(path, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}
	return fmt.Sprintf("Directory created: %s", path), nil
}

// ParseYAMLTool extracts values from YAML-like files (simple key-value)
type ParseYAMLTool struct{}

// Name returns the tool name
func (t *ParseYAMLTool) Name() string {
	return "parse_yaml"
}

// Description returns the tool description
func (t *ParseYAMLTool) Description() string {
	return "Extract values from YAML files using dot-notation path. Simple YAML only (no complex nesting)."
}

// Call parses YAML
func (t *ParseYAMLTool) Call(ctx context.Context, params map[string]interface{}) (string, error) {
	path, ok := params["path"].(string)
	if !ok || path == "" {
		return "", fmt.Errorf("missing parameter: path")
	}
	query, _ := params["query"].(string)

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	content := string(data)

	// 简单 YAML 解析：处理 key: value 格式
	lines := strings.Split(content, "\n")
	result := make(map[string]interface{})

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// 处理 key: value
		if idx := strings.Index(trimmed, ":"); idx > 0 {
			key := strings.TrimSpace(trimmed[:idx])
			value := strings.TrimSpace(trimmed[idx+1:])
			if value != "" {
				// 去除引号
				value = strings.Trim(value, "\"'")
				result[key] = value
			}
		}
	}

	// 如果没有 query，返回整个结果
	if query == "" {
		out, _ := json.MarshalIndent(result, "", "  ")
		return string(out), nil
	}

	// 按 dot-notation 路径提取值
	parts := strings.Split(query, ".")
	var current interface{} = result
	for _, part := range parts {
		switch v := current.(type) {
		case map[string]interface{}:
			val, exists := v[part]
			if !exists {
				return "", fmt.Errorf("key '%s' not found", part)
			}
			current = val
		default:
			return "", fmt.Errorf("cannot navigate into %T at key '%s'", current, part)
		}
	}

	switch v := current.(type) {
	case string:
		return v, nil
	case nil:
		return "null", nil
	default:
		out, _ := json.MarshalIndent(v, "", "  ")
		return string(out), nil
	}
}

// RegisterBuiltinTools registers all built-in tools
func RegisterBuiltinTools(registry *Registry) {
	registry.Register(&ReadFileTool{})
	registry.Register(&ReadFileRangeTool{})
	registry.Register(&WriteFileTool{})
	registry.Register(&EditFileTool{})
	registry.Register(&SearchFileTool{})
	registry.Register(&GrepContentTool{})
	registry.Register(&ExecuteCommandTool{})
	registry.Register(&GitTool{})
	registry.Register(&ParseJSONTool{})
	registry.Register(&ParseYAMLTool{})
	registry.Register(&DiffFilesTool{})
	registry.Register(&BatchEditTool{})
	registry.Register(&HTTPRequestTool{})
	registry.Register(&MakeDirTool{})
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
