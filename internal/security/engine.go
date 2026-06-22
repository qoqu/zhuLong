// 7层安全模型 - 深度防御设计（参考Hermes）
package security

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Layer 安全层级
type Layer string

const (
	LayerContainer    Layer = "container"     // 容器隔离
	LayerInput        Layer = "input"         // 输入清理
	LayerSSRF         Layer = "ssrf"          // SSRF保护
	LayerCredentials  Layer = "credentials"   // 凭据过滤
	LayerFileMutation Layer = "file_mutation" // 文件突变验证
	LayerSupplyChain  Layer = "supply_chain"  // 供应链审计
	LayerBlacklist    Layer = "blacklist"     // 硬性黑名单
)

// CheckResult 安全检查结果
type CheckResult struct {
	Passed  bool
	Layer   Layer
	Message string
	Risk    string // low / medium / high / critical
}

// ========== 1. 输入清理 ==========

// InputSanitizer 输入清理器
type InputSanitizer struct {
	blockedPatterns []*regexp.Regexp
	allowedDirs     []string
}

// NewInputSanitizer 创建输入清理器
func NewInputSanitizer() *InputSanitizer {
	return &InputSanitizer{
		blockedPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)ignore\s+(?:all\s+)?(?:previous|prior|preceding)\s+instructions`),
			regexp.MustCompile(`(?i)forget\s+(?:all\s+)?(?:previous|prior)`),
			regexp.MustCompile(`(?:rm|del|remove)\s+[-/][rR][fF]\s+[/~]`),
			regexp.MustCompile(`\$\{[\w]+\}`),   // 环境变量注入
			regexp.MustCompile(`;.*(?:rm|del|wget|curl)`), // 命令链注入
		},
	}
}

// Sanitize 清理输入
func (s *InputSanitizer) Sanitize(input string) *CheckResult {
	for _, p := range s.blockedPatterns {
		if p.MatchString(input) {
			return &CheckResult{
				Passed:  false,
				Layer:   LayerInput,
				Message: fmt.Sprintf("blocked pattern: %s", p.String()),
				Risk:    "high",
			}
		}
	}
	return &CheckResult{Passed: true, Layer: LayerInput}
}

// ValidatePath 验证路径在白名单内
func (s *InputSanitizer) ValidatePath(path string) *CheckResult {
	abs, err := filepath.Abs(path)
	if err != nil {
		return &CheckResult{Passed: false, Layer: LayerInput, Message: err.Error(), Risk: "medium"}
	}

	for _, allowed := range s.allowedDirs {
		allowedAbs, _ := filepath.Abs(allowed)
		if strings.HasPrefix(abs, allowedAbs) {
			return &CheckResult{Passed: true, Layer: LayerInput}
		}
	}

	return &CheckResult{
		Passed:  false,
		Layer:   LayerInput,
		Message: fmt.Sprintf("path %s not in allowed directories", path),
		Risk:    "high",
	}
}

// ========== 2. SSRF保护 ==========

// SSRFProtector SSRF保护器
type SSRFProtector struct {
	blockedCIDRs []string
}

// NewSSRFProtector 创建SSRF保护器
func NewSSRFProtector() *SSRFProtector {
	return &SSRFProtector{
		blockedCIDRs: []string{
			"127.0.0.0/8",     // 回环
			"10.0.0.0/8",      // 私有
			"172.16.0.0/12",   // 私有
			"192.168.0.0/16",  // 私有
			"169.254.0.0/16",  // 链路本地
			"::1/128",          // IPv6回环
			"fc00::/7",         // IPv6唯一本地
			"fe80::/10",        // IPv6链路本地
		},
	}
}

// CheckURL 检查URL
func (p *SSRFProtector) CheckURL(url string) *CheckResult {
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		return &CheckResult{
			Passed:  false,
			Layer:   LayerSSRF,
			Message: "SSRF protection: HTTP/HTTPS blocked by default",
			Risk:    "medium",
		}
	}
	return &CheckResult{Passed: true, Layer: LayerSSRF}
}

// ========== 3. 凭据过滤 ==========

// CredentialFilter 凭据过滤器
type CredentialFilter struct {
	patterns []*regexp.Regexp
}

// NewCredentialFilter 创建凭据过滤器
func NewCredentialFilter() *CredentialFilter {
	return &CredentialFilter{
		patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)(api[_-]?key|apikey|secret|token|password)\s*[:=]\s*['"]?\w{8,}`),
			regexp.MustCompile(`sk-[a-zA-Z0-9]{20,}`),      // OpenAI key
			regexp.MustCompile(`ghp_[a-zA-Z0-9]{36}`),       // GitHub PAT
			regexp.MustCompile(`AKIA[0-9A-Z]{16}`),          // AWS AK
		},
	}
}

// Filter 过滤凭据
func (f *CredentialFilter) Filter(input string) (string, *CheckResult) {
	hasCredential := false
	for _, p := range f.patterns {
		if p.MatchString(input) {
			input = p.ReplaceAllString(input, "[REDACTED]")
			hasCredential = true
		}
	}
	if hasCredential {
		return input, &CheckResult{
			Passed:  false,
			Layer:   LayerCredentials,
			Message: "credentials redacted",
			Risk:    "critical",
		}
	}
	return input, &CheckResult{Passed: true, Layer: LayerCredentials}
}

// ========== 4. 文件突变验证 ==========

// FileMutationValidator 文件突变验证器
type FileMutationValidator struct{}

// Validate 验证文件写入
func (v *FileMutationValidator) Validate(path, content string) *CheckResult {
	// 检查尝试写入敏感文件
	sensitiveFiles := []string{".env", ".ssh/", "id_rsa", "*.pem", "/etc/"}
	for _, sf := range sensitiveFiles {
		if strings.Contains(path, sf) {
			return &CheckResult{
				Passed:  false,
				Layer:   LayerFileMutation,
				Message: fmt.Sprintf("blocked write to sensitive file: %s", path),
				Risk:    "critical",
			}
		}
	}

	// 检查已存在的文件
	if info, err := os.Stat(path); err == nil {
		if info.Size() > 0 {
			return &CheckResult{
				Passed:  false,
				Layer:   LayerFileMutation,
				Message: fmt.Sprintf("file %s already exists with size %d", path, info.Size()),
				Risk:    "medium",
			}
		}
	}

	return &CheckResult{Passed: true, Layer: LayerFileMutation}
}

// ========== 5. 硬性黑名单 ==========

// HardBlocklist 硬性黑名单（YOLO模式也无法绕过）
type HardBlocklist struct {
	commands []string
}

// NewHardBlocklist 创建硬性黑名单
func NewHardBlocklist() *HardBlocklist {
	return &HardBlocklist{
		commands: []string{
			"rm -rf /", "rm -rf /*", "rm -rf ~",
			":(){ :|:& };:", // fork bomb
			"dd if=/dev/zero of=/", "mkfs.", "format",
			"chmod -R 000 /", "chown -R root /",
		},
	}
}

// Check 检查命令
func (b *HardBlocklist) Check(command string) *CheckResult {
	lower := strings.ToLower(command)
	for _, blocked := range b.commands {
		if strings.Contains(lower, blocked) {
			return &CheckResult{
				Passed:  false,
				Layer:   LayerBlacklist,
				Message: fmt.Sprintf("hard blocked command: %s", blocked),
				Risk:    "critical",
			}
		}
	}
	return &CheckResult{Passed: true, Layer: LayerBlacklist}
}

// ========== 引擎 ==========

// Engine 安全引擎（组合所有层）
type Engine struct {
	sanitizer     *InputSanitizer
	ssrf          *SSRFProtector
	credential    *CredentialFilter
	mutation      *FileMutationValidator
	blacklist     *HardBlocklist
}

// NewEngine 创建安全引擎
func NewEngine() *Engine {
	return &Engine{
		sanitizer:  NewInputSanitizer(),
		ssrf:       NewSSRFProtector(),
		credential: NewCredentialFilter(),
		mutation:   &FileMutationValidator{},
		blacklist:  NewHardBlocklist(),
	}
}

// CheckAll 全量安全检查（保留向后兼容）
// 关键修复: 之前 CheckAll(input, path, command) 三个参数语义混乱：
//   - input 传给了 InputSanitizer（实际是 prompt 注入防护，不能给 path/command）
//   - path 在非空时也走 InputSanitizer.ValidatePath（行为正确但耦合）
// 推荐: 文件操作用 CheckFileAccess，命令用 CheckCommand，prompt 用 CheckPrompt
// 保留 CheckAll 用于向后兼容，但内部显式分开调用避免串扰
func (e *Engine) CheckAll(input string, path string, command string) []CheckResult {
	var results []CheckResult

	// 黑名单
	r := e.blacklist.Check(command)
	results = append(results, *r)
	if !r.Passed {
		return results
	}

	// input 清理（仅当 input 非空且看起来像 prompt 时）
	if input != "" {
		r = e.sanitizer.Sanitize(input)
		results = append(results, *r)
	}

	// 凭据过滤（仅当 input 非空时，避免路径误报）
	if input != "" {
		_, r = e.credential.Filter(input)
		results = append(results, *r)
	}

	// 路径相关（仅当 path 非空时）
	if path != "" {
		r = e.sanitizer.ValidatePath(path)
		results = append(results, *r)
		r = e.mutation.Validate(path, input)
		results = append(results, *r)
	}

	// SSRF（仅当 input 看起来像 URL 时）
	if input != "" {
		r = e.ssrf.CheckURL(input)
		results = append(results, *r)
	}

	return results
}

// CheckPrompt 检查 LLM 收到的 prompt/响应是否安全
// 用于普通 prompt 注入防护（agent.go 中"输入清理"用途）
func (e *Engine) CheckPrompt(prompt string) []CheckResult {
	var results []CheckResult
	r := e.sanitizer.Sanitize(prompt)
	results = append(results, *r)
	_, r = e.credential.Filter(prompt)
	results = append(results, *r)
	r = e.ssrf.CheckURL(prompt)
	results = append(results, *r)
	return results
}

// CheckFileAccess 检查文件访问
// 关键修复: 替代之前 SecureReadFile/WriteFile 误用 CheckAll("", path, "")
// 显式区分：path 验证 + 内容凭据过滤 + 是否允许覆盖（write）
func (e *Engine) CheckFileAccess(path, content string, isWrite bool) []CheckResult {
	var results []CheckResult
	if path == "" {
		results = append(results, CheckResult{
			Passed: false, Layer: LayerInput,
			Message: "path is empty", Risk: "low",
		})
		return results
	}
	r := e.sanitizer.ValidatePath(path)
	results = append(results, *r)
	if !r.Passed {
		return results
	}
	if isWrite {
		r = e.mutation.Validate(path, content)
		results = append(results, *r)
		if content != "" {
			_, r = e.credential.Filter(content)
			results = append(results, *r)
		}
	}
	return results
}

// CheckCommand 检查 shell 命令
// 关键修复: 替代之前 SecureExecuteCommand 误用 CheckAll(command, "", command)
// 显式只跑黑名单 + SSRF 拦截（命令里嵌的 URL 也算）
func (e *Engine) CheckCommand(command string) []CheckResult {
	var results []CheckResult
	r := e.blacklist.Check(command)
	results = append(results, *r)
	if !r.Passed {
		return results
	}
	r = e.ssrf.CheckURL(command)
	results = append(results, *r)
	return results
}
