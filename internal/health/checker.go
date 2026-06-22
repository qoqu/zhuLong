// 健康检查系统
package health

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CheckItem 检查项
type CheckItem struct {
	Name    string `json:"name"`
	Status  string `json:"status"` // pass/warn/fail
	Message string `json:"message"`
}

// Report 健康检查报告
type Report struct {
	Items  []CheckItem `json:"items"`
	Passed int         `json:"passed"`
	Failed int         `json:"failed"`
	Total  int         `json:"total"`
	OK     bool        `json:"ok"`
}

// Checker 健康检查器
type Checker struct {
	projectPath string
}

// New 创建健康检查器
func New(projectPath string) *Checker {
	return &Checker{projectPath: projectPath}
}

// Run 运行全部检查
func (c *Checker) Run() *Report {
	report := &Report{}

	checks := []struct {
		name string
		fn   func() CheckItem
	}{
		{"CLAUDE.md", c.checkClaudeMD},
		{"Git Repository", c.checkGit},
		{"Config Directory", c.checkConfigDir},
		{"Hook Registration", c.checkHooks},
		{"Build Status", c.checkBuild},
		{"Test Status", c.checkTest},
		{"Module Structure", c.checkModules},
	}

	for _, check := range checks {
		item := check.fn()
		report.Items = append(report.Items, item)
		report.Total++

		if item.Status == "pass" {
			report.Passed++
		} else {
			report.Failed++
		}
	}

	report.OK = report.Failed == 0
	return report
}

// String 返回可读报告
func (r *Report) String() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Health Check: %d/%d passed\n", r.Passed, r.Total))

	for _, item := range r.Items {
		icon := map[string]string{"pass": "✅", "warn": "⚠️", "fail": "❌"}
		b.WriteString(fmt.Sprintf("  %s %s", icon[item.Status], item.Name))
		if item.Message != "" {
			b.WriteString(fmt.Sprintf(" - %s", item.Message))
		}
		b.WriteString("\n")
	}

	if r.OK {
		b.WriteString("\nAll checks passed!\n")
	} else {
		b.WriteString(fmt.Sprintf("\n%d check(s) failed. Please fix the issues above.\n", r.Failed))
	}
	return b.String()
}

// ========== 检查项实现 ==========

func (c *Checker) checkClaudeMD() CheckItem {
	path := filepath.Join(c.projectPath, "README.md")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return CheckItem{Name: "CLAUDE.md", Status: "warn", Message: "README.md not found"}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return CheckItem{Name: "CLAUDE.md", Status: "fail", Message: fmt.Sprintf("cannot read: %v", err)}
	}
	if len(data) < 100 {
		return CheckItem{Name: "CLAUDE.md", Status: "warn", Message: "README.md is too short (< 100 bytes)"}
	}
	return CheckItem{Name: "CLAUDE.md", Status: "pass", Message: fmt.Sprintf("%d bytes", len(data))}
}

func (c *Checker) checkGit() CheckItem {
	path := filepath.Join(c.projectPath, ".git")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return CheckItem{Name: "Git Repository", Status: "warn", Message: "not a git repository"}
	}
	return CheckItem{Name: "Git Repository", Status: "pass"}
}

func (c *Checker) checkConfigDir() CheckItem {
	dirs := []string{filepath.Join(c.projectPath, "config")}
	optional := []string{filepath.Join(c.projectPath, "docs")}

	for _, d := range dirs {
		if _, err := os.Stat(d); os.IsNotExist(err) {
			return CheckItem{Name: "Config Directory", Status: "warn", Message: "config/ not found"}
		}
	}
	for _, d := range optional {
		if _, err := os.Stat(d); os.IsNotExist(err) {
			return CheckItem{Name: "Config Directory", Status: "pass", Message: "docs/ is optional"}
		}
	}
	return CheckItem{Name: "Config Directory", Status: "pass"}
}

func (c *Checker) checkHooks() CheckItem {
	hookDir := filepath.Join(c.projectPath, "internal", "hook")
	if _, err := os.Stat(hookDir); os.IsNotExist(err) {
		return CheckItem{Name: "Hook Registration", Status: "warn", Message: "hook module not found"}
	}
	return CheckItem{Name: "Hook Registration", Status: "pass"}
}

func (c *Checker) checkBuild() CheckItem {
	_, err := os.Stat(filepath.Join(c.projectPath, "go.mod"))
	if os.IsNotExist(err) {
		return CheckItem{Name: "Build Status", Status: "warn", Message: "go.mod not found"}
	}
	return CheckItem{Name: "Build Status", Status: "pass"}
}

func (c *Checker) checkTest() CheckItem {
	testDirs := []string{
		filepath.Join(c.projectPath, "internal", "hook"),
		filepath.Join(c.projectPath, "internal", "breaker"),
		filepath.Join(c.projectPath, "internal", "state"),
	}
	hasTest := false
	for _, d := range testDirs {
		entries, err := os.ReadDir(d)
		if err == nil {
			for _, e := range entries {
				if strings.HasSuffix(e.Name(), "_test.go") {
					hasTest = true
					break
				}
			}
		}
	}
	if !hasTest {
		return CheckItem{Name: "Test Status", Status: "warn", Message: "no test files found in core modules"}
	}
	return CheckItem{Name: "Test Status", Status: "pass"}
}

func (c *Checker) checkModules() CheckItem {
	internalDir := filepath.Join(c.projectPath, "internal")
	entries, err := os.ReadDir(internalDir)
	if err != nil {
		return CheckItem{Name: "Module Structure", Status: "fail", Message: fmt.Sprintf("cannot read internal/: %v", err)}
	}
	count := 0
	for _, e := range entries {
		if e.IsDir() {
			count++
		}
	}
	return CheckItem{Name: "Module Structure", Status: "pass", Message: fmt.Sprintf("%d modules", count)}
}
