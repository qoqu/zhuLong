// 审查报告系统（参考Harness-Starter session-review.mjs）
package review

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Severity 风险等级
type Severity string

const (
	SeverityLow    Severity = "low"
	SeverityMedium Severity = "medium"
	SeverityHigh   Severity = "high"
	SeverityCritical Severity = "critical"
)

// Change 变更记录
type Change struct {
	File    string `json:"file"`
	Type    string `json:"type"` // added/modified/deleted
	Summary string `json:"summary"`
}

// Report 审查报告
type Report struct {
	ID         string   `json:"id"`
	SessionID  string   `json:"session_id"`
	CreatedAt  string   `json:"created_at"`
	Changes    []Change `json:"changes"`
	Severity   Severity `json:"severity"`
	Score      float64  `json:"score"`
	Findings   []string `json:"findings"`
	Suggestions []string `json:"suggestions"`
	BreakerState string `json:"breaker_state,omitempty"`
}

// Recorder 审查记录器
type Recorder struct {
	mu       sync.Mutex
	reportsDir string
}

// New 创建审查记录器
func New(reportsDir string) *Recorder {
	os.MkdirAll(reportsDir, 0755)
	return &Recorder{reportsDir: reportsDir}
}

// Record 记录审查报告
func (r *Recorder) Record(report *Report) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	report.ID = fmt.Sprintf("R-%d-%d", time.Now().Unix(), time.Now().UnixNano()%1000000)
	report.CreatedAt = time.Now().Format(time.RFC3339)

	// 评估风险等级
	report.Severity = r.evaluateSeverity(report)

	// 计算评分
	report.Score = r.calculateScore(report)

	data := []byte(r.formatReport(report))
	path := filepath.Join(r.reportsDir, fmt.Sprintf("review-%s.md", report.ID))
	return os.WriteFile(path, data, 0644)
}

// ListRecent 获取最近的审查报告
func (r *Recorder) ListRecent(n int) ([]string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	entries, err := os.ReadDir(r.reportsDir)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasPrefix(e.Name(), "review-") {
			files = append(files, e.Name())
		}
	}

	sort.Sort(sort.Reverse(sort.StringSlice(files)))

	if n > 0 && n < len(files) {
		files = files[:n]
	}

	result := make([]string, 0, len(files))
	for _, f := range files {
		data, err := os.ReadFile(filepath.Join(r.reportsDir, f))
		if err == nil {
			result = append(result, string(data))
		}
	}

	return result, nil
}

func (r *Recorder) evaluateSeverity(report *Report) Severity {
	highCount := 0
	criticalCount := 0

	for _, f := range report.Findings {
		if strings.Contains(f, "critical") || strings.Contains(f, "dangerous") {
			criticalCount++
		} else if strings.Contains(f, "high") {
			highCount++
		}
	}

	if criticalCount > 0 {
		return SeverityCritical
	}
	if highCount > 0 || len(report.Findings) > 3 {
		return SeverityHigh
	}
	if len(report.Findings) > 0 {
		return SeverityMedium
	}
	return SeverityLow
}

func (r *Recorder) calculateScore(report *Report) float64 {
	score := 1.0
	score -= float64(len(report.Findings)) * 0.1
	score -= float64(len(report.Changes)) * 0.05

	if report.Severity == SeverityCritical {
		score -= 0.3
	} else if report.Severity == SeverityHigh {
		score -= 0.15
	}

	if score < 0 {
		score = 0
	}
	return score
}

func (r *Recorder) formatReport(report *Report) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("# Review Report: %s\n\n", report.ID))
	b.WriteString(fmt.Sprintf("- **Session**: %s\n", report.SessionID))
	b.WriteString(fmt.Sprintf("- **Time**: %s\n", report.CreatedAt))
	b.WriteString(fmt.Sprintf("- **Severity**: %s\n", report.Severity))
	b.WriteString(fmt.Sprintf("- **Score**: %.2f\n\n", report.Score))

	if len(report.Changes) > 0 {
		b.WriteString("## Changes\n\n")
		b.WriteString("| File | Type | Summary |\n")
		b.WriteString("|------|------|--------|\n")
		for _, c := range report.Changes {
			b.WriteString(fmt.Sprintf("| %s | %s | %s |\n", c.File, c.Type, c.Summary))
		}
		b.WriteString("\n")
	}

	if len(report.Findings) > 0 {
		b.WriteString("## Findings\n\n")
		for _, f := range report.Findings {
			b.WriteString(fmt.Sprintf("- %s\n", f))
		}
		b.WriteString("\n")
	}

	if len(report.Suggestions) > 0 {
		b.WriteString("## Suggestions\n\n")
		for _, s := range report.Suggestions {
			b.WriteString(fmt.Sprintf("- %s\n", s))
		}
		b.WriteString("\n")
	}

	return b.String()
}
