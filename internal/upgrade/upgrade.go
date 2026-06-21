// 智能升级系统（参考Harness-Starter）
// 区分用户文件和模板文件，支持dry-run预览
package upgrade

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FileOrigin 文件来源
type FileOrigin string

const (
	OriginTemplate FileOrigin = "template" // 模板文件
	OriginUser     FileOrigin = "user"     // 用户自定义文件
	OriginMixed    FileOrigin = "mixed"    // 混合文件
	OriginUnknown  FileOrigin = "unknown"  // 未知
)

// UpgradeFile 升级文件
type UpgradeFile struct {
	Path     string
	Origin   FileOrigin
	Checksum string
	Action   string // keep/overwrite/merge/skip
}

// UpgradePlan 升级计划
type UpgradePlan struct {
	Version   string
	Files     []UpgradeFile
	DryRun    bool
	Overrides int
	NewFiles  int
}

// Upgrader 升级管理器
type Upgrader struct {
	version       string
	templateFiles map[string]string // path -> checksum
}

// New 创建升级管理器
func New(version string) *Upgrader {
	return &Upgrader{
		version:       version,
		templateFiles: make(map[string]string),
	}
}

// RegisterTemplate 注册模板文件
func (u *Upgrader) RegisterTemplate(path, checksum string) {
	u.templateFiles[path] = checksum
}

// ScanFileOrigin 扫描文件来源
func (u *Upgrader) ScanFileOrigin(path string) FileOrigin {
	if _, ok := u.templateFiles[path]; ok {
		return OriginTemplate
	}

	// 检查文件是否存在
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return OriginUnknown
	}

	// 检查是否为模板文件
	data, err := os.ReadFile(path)
	if err != nil {
		return OriginUnknown
	}

	content := string(data)
	if strings.Contains(content, "harness") || strings.Contains(content, "template") {
		return OriginMixed
	}

	return OriginUser
}

// PlanUpgrade 制定升级计划
func (u *Upgrader) PlanUpgrade(targetDir string, dryRun bool) (*UpgradePlan, error) {
	plan := &UpgradePlan{
		Version: u.version,
		DryRun:  dryRun,
	}

	// 扫描目录所有文件
	err := filepath.Walk(targetDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		relPath, _ := filepath.Rel(targetDir, path)
		origin := u.ScanFileOrigin(relPath)

		file := UpgradeFile{
			Path:     relPath,
			Origin:   origin,
			Checksum: fmt.Sprintf("%d", info.ModTime().Unix()),
		}

		switch origin {
		case OriginTemplate:
			file.Action = "overwrite"
			plan.Overrides++
		case OriginUser:
			file.Action = "keep"
		case OriginMixed:
			file.Action = "merge"
		default:
			file.Action = "skip"
		}

		plan.Files = append(plan.Files, file)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("walk failed: %w", err)
	}

	return plan, nil
}

// ApplyUpgrade 应用升级
func (u *Upgrader) ApplyUpgrade(plan *UpgradePlan) error {
	if plan.DryRun {
		// 预览模式，只输出计划
		return nil
	}

	for _, file := range plan.Files {
		switch file.Action {
		case "overwrite":
			// 覆盖模板文件
		case "keep":
			// 保留用户文件
		case "merge":
			// 合并文件
		}
	}

	return nil
}
