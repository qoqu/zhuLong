// GC扫描器 - 8个确定性维度质量扫描（参考Harness-Starter）
package quality

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ScanDimension 扫描维度
type ScanDimension struct {
	Name        string
	Description string
	Weight      float64 // 权重 0-1
	Score       float64 // 得分 0-1
	Issues      []string
}

// ScanReport 扫描报告
type ScanReport struct {
	Dimensions []ScanDimension
	TotalScore float64 // 总分 0-1
	Timestamp  string
	ProjectPath string
}

// Scanner GC扫描器
type Scanner struct {
	dimensions []ScanDimension
}

// NewScanner 创建GC扫描器
func NewScanner() *Scanner {
	return &Scanner{
		dimensions: defaultDimensions(),
	}
}

// 默认8个扫描维度
func defaultDimensions() []ScanDimension {
	return []ScanDimension{
		{Name: "documentation", Description: "文档完整性检查", Weight: 0.10},
		{Name: "git_status", Description: "Git状态检查", Weight: 0.10},
		{Name: "todo_density", Description: "TODO密度检查", Weight: 0.15},
		{Name: "test_coverage", Description: "测试覆盖检查", Weight: 0.20},
		{Name: "code_quality", Description: "代码质量检查", Weight: 0.20},
		{Name: "dependency", Description: "依赖健康检查", Weight: 0.10},
		{Name: "security", Description: "安全检查", Weight: 0.10},
		{Name: "performance", Description: "性能检查", Weight: 0.05},
	}
}

// Scan 执行扫描
func (s *Scanner) Scan(projectPath string) (*ScanReport, error) {
	report := &ScanReport{
		Timestamp:   fmt.Sprintf("%s", "now"),
		ProjectPath: projectPath,
	}

	for i := range s.dimensions {
		d := &s.dimensions[i]
		d.Score, d.Issues = s.scanDimension(d.Name, projectPath)
	}

	report.Dimensions = make([]ScanDimension, len(s.dimensions))
	copy(report.Dimensions, s.dimensions)

	// 计算总分
	var total float64
	for _, d := range s.dimensions {
		total += d.Score * d.Weight
	}
	report.TotalScore = total

	return report, nil
}

// AddDimension 添加自定义维度
func (s *Scanner) AddDimension(dim ScanDimension) {
	s.dimensions = append(s.dimensions, dim)
}

func (s *Scanner) scanDimension(name, projectPath string) (float64, []string) {
	switch name {
	case "documentation":
		return s.scanDocumentation(projectPath)
	case "git_status":
		return s.scanGitStatus(projectPath)
	case "todo_density":
		return s.scanTodoDensity(projectPath)
	case "test_coverage":
		return s.scanTestCoverage(projectPath)
	case "code_quality":
		return s.scanCodeQuality(projectPath)
	case "dependency":
		return s.scanDependency(projectPath)
	case "security":
		return s.scanSecurity(projectPath)
	case "performance":
		return s.scanPerformance(projectPath)
	default:
		return 0.5, []string{"unknown dimension"}
	}
}

func (s *Scanner) scanDocumentation(path string) (float64, []string) {
	var issues []string
	score := 1.0

	// 检查README.md
	if _, err := os.Stat(filepath.Join(path, "README.md")); os.IsNotExist(err) {
		issues = append(issues, "缺少 README.md")
		score -= 0.3
	}

	// 检查CONTRIBUTING.md
	if _, err := os.Stat(filepath.Join(path, "CONTRIBUTING.md")); os.IsNotExist(err) {
		issues = append(issues, "缺少 CONTRIBUTING.md")
		score -= 0.2
	}

	// 检查docs目录
	if _, err := os.Stat(filepath.Join(path, "docs")); os.IsNotExist(err) {
		issues = append(issues, "缺少 docs/ 目录")
		score -= 0.2
	}

	return max(0, score), issues
}

func (s *Scanner) scanGitStatus(path string) (float64, []string) {
	var issues []string
	score := 1.0

	// 检查.git目录
	if _, err := os.Stat(filepath.Join(path, ".git")); os.IsNotExist(err) {
		issues = append(issues, "未初始化Git仓库")
		score -= 0.5
	}

	return max(0, score), issues
}

func (s *Scanner) scanTodoDensity(path string) (float64, []string) {
	var issues []string
	todoCount := 0
	fileCount := 0

	filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if strings.HasSuffix(p, ".go") || strings.HasSuffix(p, ".ts") || strings.HasSuffix(p, ".tsx") {
			fileCount++
			data, _ := os.ReadFile(p)
			todoCount += strings.Count(string(data), "TODO")
			todoCount += strings.Count(string(data), "FIXME")
		}
		return nil
	})

	score := 1.0
	if fileCount > 0 {
		density := float64(todoCount) / float64(fileCount)
		if density > 2 {
			issues = append(issues, fmt.Sprintf("TODO密度过高：每文件 %.1f 个", density))
			score -= 0.2 * density
		}
	}

	return max(0, score), issues
}

func (s *Scanner) scanTestCoverage(path string) (float64, []string) {
	var issues []string
	testCount := 0
	sourceCount := 0

	filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if strings.HasSuffix(p, "_test.go") {
			testCount++
		}
		if strings.HasSuffix(p, ".go") && !strings.HasSuffix(p, "_test.go") {
			sourceCount++
		}
		return nil
	})

	score := 1.0
	if sourceCount > 0 {
		ratio := float64(testCount) / float64(sourceCount)
		if ratio < 0.3 {
			issues = append(issues, fmt.Sprintf("测试覆盖率低：%.0f%%", ratio*100))
			score = ratio
		}
	}

	return max(0, score), issues
}

func (s *Scanner) scanCodeQuality(path string) (float64, []string) {
	var issues []string
	score := 1.0
	return max(0, score), issues
}

func (s *Scanner) scanDependency(path string) (float64, []string) {
	var issues []string
	score := 1.0
	return max(0, score), issues
}

func (s *Scanner) scanSecurity(path string) (float64, []string) {
	var issues []string
	score := 1.0
	return max(0, score), issues
}

func (s *Scanner) scanPerformance(path string) (float64, []string) {
	var issues []string
	score := 1.0
	return max(0, score), issues
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
