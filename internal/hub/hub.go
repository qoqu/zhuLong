// Skills Hub - 技能市场生态
package hub

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// TrustLevel 信任等级
type TrustLevel string

const (
	TrustBuiltin   TrustLevel = "builtin"   // 内置：始终信任
	TrustOfficial  TrustLevel = "official"  // 官方：始终信任
	TrustTrusted   TrustLevel = "trusted"   // 可信：宽松审查
	TrustCommunity TrustLevel = "community" // 社区：严格审查
)

// SourceType 来源类型
type SourceType string

const (
	SourceOfficial SourceType = "official"  // 官方市场
	SourceGitHub   SourceType = "github"    // GitHub仓库
	SourceClawhub  SourceType = "clawhub"   // OpenClaw生态
	SourceHermes   SourceType = "hermes"    // Hermes生态
	SourceLocal    SourceType = "local"     // 本地
)

// SkillEntry 技能条目
type SkillEntry struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Version     string     `json:"version"`
	Source      SourceType `json:"source"`
	Trust       TrustLevel `json:"trust"`
	Author      string     `json:"author,omitempty"`
	Tags        []string   `json:"tags,omitempty"`
	Score       float64    `json:"score"`    // 评分 0-5
	Reviews     int        `json:"reviews"`  // 评价数
	Installs    int        `json:"installs"` // 安装次数
}

// SecurityReport 安全扫描报告
type SecurityReport struct {
	Passed      bool     `json:"passed"`
	Issues      []string `json:"issues"`
	RiskLevel   string   `json:"risk_level"` // low / medium / high / critical
	TrustLevel  TrustLevel `json:"trust_level"`
}

// Scanner 安全扫描器
type Scanner struct {
	blockedPatterns []string
	requiredFields  []string
}

// NewScanner 创建安全扫描器
func NewScanner() *Scanner {
	return &Scanner{
		blockedPatterns: []string{
			"ignore previous instructions",
			"rm -rf /", "rm -rf /*",
			"eval(", "exec(",
			"os.system(", "subprocess.call(",
		},
		requiredFields: []string{"name", "description"},
	}
}

// Scan 扫描技能安全性
func (s *Scanner) Scan(skill SkillEntry, content string) *SecurityReport {
	report := &SecurityReport{Passed: true, RiskLevel: "low"}
	lower := strings.ToLower(content)

	for _, pattern := range s.blockedPatterns {
		if strings.Contains(lower, pattern) {
			report.Issues = append(report.Issues, fmt.Sprintf("blocked pattern: %s", pattern))
			report.RiskLevel = "critical"
		}
	}

	if skill.Name == "" {
		report.Issues = append(report.Issues, "missing required field: name")
	}
	if skill.Description == "" {
		report.Issues = append(report.Issues, "missing required field: description")
	}

	if len(report.Issues) > 0 {
		report.Passed = false
	}

	// 根据来源自动确定信任等级
	switch skill.Source {
	case SourceOfficial:
		report.TrustLevel = TrustOfficial
	case SourceGitHub:
		report.TrustLevel = TrustCommunity
	default:
		report.TrustLevel = TrustCommunity
	}

	return report
}

// Hub 技能市场
type Hub struct {
	mu       sync.RWMutex
	entries  map[string]*SkillEntry
	scanner  *Scanner
	installer InstallFn
}

// InstallFn 安装函数
type InstallFn func(name string, source SourceType) error

// New 创建技能市场
func New(installer InstallFn) *Hub {
	return &Hub{
		entries:   make(map[string]*SkillEntry),
		scanner:   NewScanner(),
		installer: installer,
	}
}

// Register 注册技能到市场
func (h *Hub) Register(entry *SkillEntry) (*SecurityReport, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// 安全检查
	report := h.scanner.Scan(*entry, "")
	if !report.Passed && entry.Source != SourceOfficial {
		return report, fmt.Errorf("security check failed for %s: %v", entry.Name, report.Issues)
	}

	entry.Trust = report.TrustLevel
	h.entries[entry.Name] = entry
	return report, nil
}

// Search 搜索技能
func (h *Hub) Search(query string) []*SkillEntry {
	h.mu.RLock()
	defer h.mu.RUnlock()

	query = strings.ToLower(query)
	var results []*SkillEntry

	for _, entry := range h.entries {
		if strings.Contains(strings.ToLower(entry.Name), query) ||
			strings.Contains(strings.ToLower(entry.Description), query) {
			results = append(results, entry)
		}
	}

	// 按评分排序
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	return results
}

// Install 安装技能
func (h *Hub) Install(name string) error {
	h.mu.RLock()
	entry, ok := h.entries[name]
	h.mu.RUnlock()

	if !ok {
		return fmt.Errorf("skill %s not found in hub", name)
	}

	if h.installer == nil {
		return fmt.Errorf("no installer configured")
	}

	if err := h.installer(name, entry.Source); err != nil {
		return err
	}

	h.mu.Lock()
	entry.Installs++
	h.mu.Unlock()

	return nil
}

// ListByTrust 按信任等级列出
func (h *Hub) ListByTrust(level TrustLevel) []*SkillEntry {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var results []*SkillEntry
	for _, entry := range h.entries {
		if entry.Trust == level {
			results = append(results, entry)
		}
	}
	return results
}

// Stats 统计信息
func (h *Hub) Stats() map[string]interface{} {
	h.mu.RLock()
	defer h.mu.RUnlock()

	byTrust := make(map[TrustLevel]int)
	for _, e := range h.entries {
		byTrust[e.Trust]++
	}

	return map[string]interface{}{
		"total":        len(h.entries),
		"by_trust":     byTrust,
		"sources":      []string{"official", "github", "clawhub", "hermes", "local"},
	}
}
