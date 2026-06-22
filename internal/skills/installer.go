// 技能安装器 - 跨生态技能兼容
package skills

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Source 技能来源类型
type Source string

const (
	SourceClawhub  Source = "clawhub"  // OpenClaw 技能生态
	SourceGitHub   Source = "github"   // 任意 GitHub 仓库
	SourceHermes   Source = "hermes"   // Hermes 技能目录
	SourceLocal    Source = "local"    // 本地路径
	SourceAny      Source = "any"      // 自动检测
)

// Installer 技能安装器
type Installer struct {
	manager    *SkillManager
	httpClient *http.Client
}

// NewInstaller 创建安装器
func NewInstaller(manager *SkillManager) *Installer {
	return &Installer{
		manager: manager,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// Install 从指定源安装技能
// source=any 时自动检测：GitHub URL→github, 本地路径→local, 否则→clawhub
func (i *Installer) Install(name string, source Source, targetDir string) error {
	switch source {
	case SourceClawhub:
		return i.installFromClawhub(name, targetDir)
	case SourceGitHub:
		return i.installFromGitHub(name, targetDir)
	case SourceHermes:
		return i.installFromHermes(name, targetDir)
	case SourceLocal:
		return i.installFromLocal(name, targetDir)
	case SourceAny:
		return i.installAuto(name, targetDir)
	default:
		return fmt.Errorf("unknown source: %s", source)
	}
}

// ListSources 列出可用来源
func (i *Installer) ListSources() []string {
	return []string{
		"clawhub  - OpenClaw 技能生态 (https://github.com/openclaw/openclaw/tree/main/skills)",
		"hermes   - Hermes Agent 技能生态",
		"github   - 任意 GitHub 仓库 (格式: owner/repo/path)",
		"local    - 本地文件路径",
		"any      - 自动检测来源",
	}
}

// InstallFromLocal 从本地路径安装（测试用导出包装）
func (i *Installer) InstallFromLocal(path, targetDir string) error {
	return i.installFromLocal(path, targetDir)
}

// InstallAuto 自动检测并安装（测试用导出包装）
func (i *Installer) InstallAuto(path, targetDir string) error {
	return i.installAuto(path, targetDir)
}

// ========== 内部实现 ==========

func (i *Installer) installFromClawhub(name, targetDir string) error {
	// OpenClaw 仓库中 skills/<name>/ 目录
	baseURL := fmt.Sprintf("https://raw.githubusercontent.com/openclaw/openclaw/main/skills/%s", name)
	skillURL := baseURL + "/SKILL.md"

	return i.downloadSkill(skillURL, name, targetDir)
}

func (i *Installer) installFromGitHub(name, targetDir string) error {
	// 格式: owner/repo 或 owner/repo/path
	parts := strings.SplitN(name, "/", 3)
	if len(parts) < 2 {
		return fmt.Errorf("invalid github format: %s (need owner/repo)", name)
	}

	owner, repo := parts[0], parts[1]
	path := "skills"
	if len(parts) >= 3 {
		path = parts[2]
	}

	skillURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/main/%s/%s/SKILL.md",
		owner, repo, path, name)

	return i.downloadSkill(skillURL, name, targetDir)
}

func (i *Installer) installFromHermes(name, targetDir string) error {
	// Hermes Agent 技能目录
	baseURL := fmt.Sprintf("https://raw.githubusercontent.com/NousResearch/hermes-agent/main/skills/%s", name)
	skillURL := baseURL + "/SKILL.md"

	return i.downloadSkill(skillURL, name, targetDir)
}

func (i *Installer) installFromLocal(name, targetDir string) error {
	// name 本身是本地路径
	skillPath := filepath.Join(name, "SKILL.md")
	if _, err := os.Stat(skillPath); err != nil {
		// 尝试作为完整路径
		if _, err := os.Stat(name); err == nil {
			skillPath = name
		} else {
			return fmt.Errorf("skill not found at %s", name)
		}
	}

	// 复制到目标目录
	targetSkillDir := filepath.Join(targetDir, filepath.Base(filepath.Dir(skillPath)))
	if err := os.MkdirAll(targetSkillDir, 0755); err != nil {
		return err
	}

	// 复制 SKILL.md
	data, err := os.ReadFile(skillPath)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(targetSkillDir, "SKILL.md"), data, 0644); err != nil {
		return err
	}

	// 尝试复制 references/templates/scripts 支持目录
	for _, sub := range []string{"references", "templates", "scripts"} {
		srcSub := filepath.Join(filepath.Dir(skillPath), sub)
		dstSub := filepath.Join(targetSkillDir, sub)
		if entries, err := os.ReadDir(srcSub); err == nil {
			os.MkdirAll(dstSub, 0755)
			for _, e := range entries {
				if !e.IsDir() {
					data, err := os.ReadFile(filepath.Join(srcSub, e.Name()))
					if err == nil {
						os.WriteFile(filepath.Join(dstSub, e.Name()), data, 0644)
					}
				}
			}
		}
	}

	return i.manager.Reload()
}

func (i *Installer) installAuto(name, targetDir string) error {
	// 检测是否为 GitHub URL
	if strings.Contains(name, "/") && strings.Count(name, "/") >= 1 {
		return i.installFromGitHub(name, targetDir)
	}

	// 检测是否为本地路径
	if strings.Contains(name, string(filepath.Separator)) || strings.HasPrefix(name, ".") {
		if _, err := os.Stat(name); err == nil {
			return i.installFromLocal(name, targetDir)
		}
	}

	// 默认从 clawhub 安装
	return i.installFromClawhub(name, targetDir)
}

func (i *Installer) downloadSkill(url, name, targetDir string) error {
	resp, err := i.httpClient.Get(url)
	if err != nil {
		return fmt.Errorf("download %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s: HTTP %d", url, resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	// 创建技能目录
	skillDir := filepath.Join(targetDir, name)
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		return err
	}

	// 写入 SKILL.md
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), data, 0644); err != nil {
		return err
	}

	// 重新扫描技能
	return i.manager.Reload()
}
