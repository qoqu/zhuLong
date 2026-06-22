package skills

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewInstaller(t *testing.T) {
	sm := NewSkillManager([]string{"/tmp"})
	i := NewInstaller(sm)
	if i == nil {
		t.Fatal("expected non-nil installer")
	}
}

func TestInstaller_ListSources(t *testing.T) {
	sm := NewSkillManager([]string{"/tmp"})
	i := NewInstaller(sm)
	sources := i.ListSources()
	if len(sources) != 5 {
		t.Errorf("expected 5 sources, got %d", len(sources))
	}
}

func TestInstaller_InstallFromLocal(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "test-skill")
	os.MkdirAll(skillDir, 0755)
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: test-skill\ndescription: test\n---\n\n# Content"), 0644)

	targetDir := t.TempDir()
	sm := NewSkillManager([]string{targetDir})
	i := NewInstaller(sm)

	err := i.InstallFromLocal(skillDir, targetDir)
	if err != nil {
		t.Fatalf("InstallFromLocal failed: %v", err)
	}

	// 验证技能文件已复制
	if _, err := os.Stat(filepath.Join(targetDir, "test-skill", "SKILL.md")); os.IsNotExist(err) {
		t.Error("SKILL.md not found in target")
	}
}

func TestInstaller_InstallFromLocal_WithSupportFiles(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "full-skill")
	os.MkdirAll(skillDir, 0755)
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: full-skill\ndescription: full\n---\n\n# Full"), 0644)
	os.MkdirAll(filepath.Join(skillDir, "references"), 0755)
	os.WriteFile(filepath.Join(skillDir, "references", "guide.md"), []byte("# Guide"), 0644)

	targetDir := t.TempDir()
	sm := NewSkillManager([]string{targetDir})
	i := NewInstaller(sm)

	err := i.InstallFromLocal(skillDir, targetDir)
	if err != nil {
		t.Fatalf("InstallFromLocal failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(targetDir, "full-skill", "references", "guide.md")); os.IsNotExist(err) {
		t.Error("support file not found in target")
	}
}

func TestInstaller_InstallFromLocal_Reload(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "reload-skill")
	os.MkdirAll(skillDir, 0755)
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: reload-skill\ndescription: reload\n---\n\n# Reload"), 0644)

	targetDir := t.TempDir()
	sm := NewSkillManager([]string{targetDir})
	i := NewInstaller(sm)

	// 安装前无技能
	before := sm.GetAllSkills()
	if len(before) != 0 {
		t.Errorf("expected 0 skills before install, got %d", len(before))
	}

	i.InstallFromLocal(skillDir, targetDir)

	// 安装后应有技能
	after := sm.GetAllSkills()
	if len(after) != 1 {
		t.Errorf("expected 1 skill after install, got %d", len(after))
	}
}

func TestInstaller_AutoDetect_LocalPath(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "auto-skill")
	os.MkdirAll(skillDir, 0755)
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: auto-skill\ndescription: auto\n---\n\n# Auto"), 0644)

	targetDir := t.TempDir()
	sm := NewSkillManager([]string{targetDir})
	i := NewInstaller(sm)

	err := i.InstallAuto(skillDir, targetDir)
	if err != nil {
		t.Fatalf("InstallAuto failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(targetDir, "auto-skill", "SKILL.md")); os.IsNotExist(err) {
		t.Error("auto-installed SKILL.md not found")
	}
}

func TestInstaller_OpenClawCompatibility(t *testing.T) {
	// 测试OpenClaw格式的YAML frontmatter能被正确解析
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "openclaw-skill")
	os.MkdirAll(skillDir, 0755)

	// OpenClaw 风格 SKILL.md
	content := `---
name: 1password
description: Manage 1Password items
requires:
  bins: [op, jq]
---

# 1Password Skill
`
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0644)

	sm := NewSkillManager([]string{dir})
	err := sm.Discover()
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	skills := sm.GetAllSkills()
	if len(skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(skills))
	}

	skill := sm.GetSkillList()[0]
	if skill.Name != "1password" {
		t.Errorf("expected name=1password, got %s", skill.Name)
	}

	// 验证依赖被解析为 req: 标签
	hasReqTag := false
	for _, tag := range skills[0].Tags {
		if tag == "req:op" {
			hasReqTag = true
			break
		}
	}
	if !hasReqTag {
		t.Error("expected req:op tag for OpenClaw requires.bins")
	}
}

func TestInstaller_HermesCompatibility(t *testing.T) {
	// 测试Hermes格式的YAML frontmatter能被正确解析
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "hermes-skill")
	os.MkdirAll(skillDir, 0755)

	// Hermes 风格 SKILL.md
	content := `---
name: hermes-skill
description: A Hermes-compatible skill
version: 1.0.0
prerequisites:
  commands: [git, curl]
tags: [hermes, compatible]
---

# Hermes Skill
`
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0644)

	sm := NewSkillManager([]string{dir})
	err := sm.Discover()
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	skills := sm.GetAllSkills()
	if len(skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(skills))
	}

	skill := skills[0]
	if skill.Name != "hermes-skill" {
		t.Errorf("expected name=hermes-skill, got %s", skill.Name)
	}

	hasReqTag := false
	for _, tag := range skill.Tags {
		if tag == "req:git" {
			hasReqTag = true
			break
		}
	}
	if !hasReqTag {
		t.Error("expected req:git tag for Hermes prerequisites.commands")
	}
}
