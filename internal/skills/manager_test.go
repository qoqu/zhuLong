package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewSkillManager(t *testing.T) {
	sm := NewSkillManager([]string{"/tmp/skills"})

	if sm == nil {
		t.Error("NewSkillManager() should not return nil")
	}
	if len(sm.searchPaths) != 1 {
		t.Errorf("searchPaths length = %v, want 1", len(sm.searchPaths))
	}
}

func TestSkillManager_Discover(t *testing.T) {
	// Create temporary skill directory
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "test-skill")
	os.MkdirAll(skillDir, 0755)

	// Create SKILL.md
	skillContent := `---
name: test-skill
description: A test skill for testing
---

# Test Skill

This is a test skill.
`
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillContent), 0644)

	sm := NewSkillManager([]string{dir})
	err := sm.Discover()

	if err != nil {
		t.Errorf("Discover() error = %v", err)
	}

	skills := sm.GetAllSkills()
	if len(skills) != 1 {
		t.Errorf("GetAllSkills() length = %v, want 1", len(skills))
	}

	if skills[0].Name != "test-skill" {
		t.Errorf("Skill name = %v, want 'test-skill'", skills[0].Name)
	}
}

func TestSkillManager_Discover_NoSkills(t *testing.T) {
	dir := t.TempDir()
	sm := NewSkillManager([]string{dir})
	err := sm.Discover()

	if err != nil {
		t.Errorf("Discover() error = %v", err)
	}

	skills := sm.GetAllSkills()
	if len(skills) != 0 {
		t.Errorf("GetAllSkills() length = %v, want 0", len(skills))
	}
}

func TestSkillManager_ViewSkill(t *testing.T) {
	// Create temporary skill directory
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "test-skill")
	os.MkdirAll(skillDir, 0755)

	// Create SKILL.md
	skillContent := `---
name: test-skill
description: A test skill for testing
---

# Test Skill

This is a test skill.
`
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillContent), 0644)

	sm := NewSkillManager([]string{dir})
	sm.Discover()

	// View skill
	skill, err := sm.ViewSkill("test-skill")
	if err != nil {
		t.Errorf("ViewSkill() error = %v", err)
	}

	if !skill.Loaded {
		t.Error("Skill should be loaded after ViewSkill()")
	}

	if skill.Content != skillContent {
		t.Error("Skill content should match original content")
	}
}

func TestSkillManager_ViewSkill_NotFound(t *testing.T) {
	sm := NewSkillManager([]string{"/tmp/skills"})

	_, err := sm.ViewSkill("nonexistent")
	if err == nil {
		t.Error("ViewSkill() should return error for nonexistent skill")
	}
}

func TestSkillManager_SearchSkills(t *testing.T) {
	// Create temporary skill directory
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "test-skill")
	os.MkdirAll(skillDir, 0755)

	// Create SKILL.md
	skillContent := `---
name: test-skill
description: A test skill for testing
---

# Test Skill
`
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillContent), 0644)

	sm := NewSkillManager([]string{dir})
	sm.Discover()

	// Search by name
	results := sm.SearchSkills("test")
	if len(results) != 1 {
		t.Errorf("SearchSkills('test') returned %v results, want 1", len(results))
	}

	// Search by description
	results = sm.SearchSkills("testing")
	if len(results) != 1 {
		t.Errorf("SearchSkills('testing') returned %v results, want 1", len(results))
	}

	// Search with no results
	results = sm.SearchSkills("nonexistent")
	if len(results) != 0 {
		t.Errorf("SearchSkills('nonexistent') returned %v results, want 0", len(results))
	}
}

func TestSkillManager_GetSkillList(t *testing.T) {
	// Create temporary skill directory
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "test-skill")
	os.MkdirAll(skillDir, 0755)

	// Create SKILL.md
	skillContent := `---
name: test-skill
description: A test skill for testing
---

# Test Skill
`
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillContent), 0644)

	sm := NewSkillManager([]string{dir})
	sm.Discover()

	list := sm.GetSkillList()
	if len(list) != 1 {
		t.Errorf("GetSkillList() length = %v, want 1", len(list))
	}

	if list[0].Name != "test-skill" {
		t.Errorf("SkillInfo.Name = %v, want 'test-skill'", list[0].Name)
	}
}

func TestSkillManager_Reload(t *testing.T) {
	// Create temporary skill directory
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "test-skill")
	os.MkdirAll(skillDir, 0755)

	// Create SKILL.md
	skillContent := `---
name: test-skill
description: A test skill for testing
---

# Test Skill
`
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillContent), 0644)

	sm := NewSkillManager([]string{dir})
	sm.Discover()

	// Add another skill
	skillDir2 := filepath.Join(dir, "another-skill")
	os.MkdirAll(skillDir2, 0755)
	skillContent2 := `---
name: another-skill
description: Another test skill
---

# Another Skill
`
	os.WriteFile(filepath.Join(skillDir2, "SKILL.md"), []byte(skillContent2), 0644)

	// Reload
	err := sm.Reload()
	if err != nil {
		t.Errorf("Reload() error = %v", err)
	}

	skills := sm.GetAllSkills()
	if len(skills) != 2 {
		t.Errorf("GetAllSkills() length after reload = %v, want 2", len(skills))
	}
}

// ========== 新增测试 ==========

func TestSkillManager_Evict(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "test-skill")
	os.MkdirAll(skillDir, 0755)
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: test-skill\ndescription: test\n---\n\n# Content"), 0644)

	sm := NewSkillManager([]string{dir})
	sm.Discover()
	sm.ViewSkill("test-skill")

	err := sm.Evict("test-skill")
	if err != nil {
		t.Fatalf("Evict failed: %v", err)
	}

	skill, _ := sm.GetSkill("test-skill")
	if skill.Loaded {
		t.Error("skill should not be loaded after evict")
	}
	if skill.LoadedTier != Tier1Name {
		t.Errorf("expected Tier1 after evict, got %d", skill.LoadedTier)
	}
}

func TestSkillManager_Pin(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "test-skill")
	os.MkdirAll(skillDir, 0755)
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: test-skill\ndescription: test\n---\n\n# Content"), 0644)

	sm := NewSkillManager([]string{dir})
	sm.Discover()

	err := sm.Pin("test-skill")
	if err != nil {
		t.Fatalf("Pin failed: %v", err)
	}

	// Pinned skill should not be evictable
	sm.ViewSkill("test-skill")
	err = sm.Evict("test-skill")
	if err == nil {
		t.Error("expected error evicting pinned skill")
	}
}

func TestSkillManager_EvictStale(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < 3; i++ {
		name := fmt.Sprintf("skill-%d", i)
		sd := filepath.Join(dir, name)
		os.MkdirAll(sd, 0755)
		os.WriteFile(filepath.Join(sd, "SKILL.md"), []byte(fmt.Sprintf("---\nname: %s\ndescription: test\n---\n\n# Content", name)), 0644)
	}

	sm := NewSkillManager([]string{dir})
	sm.Discover()
	sm.ViewSkill("skill-0")
	sm.ViewSkill("skill-1")

	// Set evictAfter to negative so all loaded skills are stale
	sm.evictAfter = -1 * time.Second
	time.Sleep(10 * time.Millisecond)
	count := sm.EvictStale()
	if count != 2 {
		t.Errorf("expected 2 evictions, got %d", count)
	}
}

func TestSkillManager_ListByTag(t *testing.T) {
	dir := t.TempDir()
	sd := filepath.Join(dir, "code-skill")
	os.MkdirAll(sd, 0755)
	os.WriteFile(filepath.Join(sd, "SKILL.md"), []byte("---\nname: code-skill\ndescription: coding\ntags: [coding, go]\n---\n\n# Content"), 0644)

	sm := NewSkillManager([]string{dir})
	sm.Discover()

	results := sm.ListByTag("coding")
	if len(results) != 1 {
		t.Errorf("expected 1 skill with coding tag, got %d", len(results))
	}
}

func TestSkillManager_Stats(t *testing.T) {
	dir := t.TempDir()
	sd := filepath.Join(dir, "test")
	os.MkdirAll(sd, 0755)
	os.WriteFile(filepath.Join(sd, "SKILL.md"), []byte("---\nname: test\ndescription: test\n---\n\n# Content"), 0644)

	sm := NewSkillManager([]string{dir})
	sm.Discover()
	sm.ViewSkill("test")

	stats := sm.Stats()
	if stats["total"] != 1 {
		t.Errorf("expected total=1, got %d", stats["total"])
	}
	if stats["loaded"] != 1 {
		t.Errorf("expected loaded=1, got %d", stats["loaded"])
	}
}

func TestSkillManager_Tier3Full(t *testing.T) {
	dir := t.TempDir()
	sd := filepath.Join(dir, "full-skill")
	os.MkdirAll(sd, 0755)
	os.WriteFile(filepath.Join(sd, "SKILL.md"), []byte("---\nname: full-skill\ndescription: full\n---\n\n# Full"), 0644)

	// 创建支持文件
	os.MkdirAll(filepath.Join(sd, "references"), 0755)
	os.WriteFile(filepath.Join(sd, "references", "guide.md"), []byte("# Reference Guide"), 0644)
	os.MkdirAll(filepath.Join(sd, "templates"), 0755)
	os.WriteFile(filepath.Join(sd, "templates", "template.txt"), []byte("Template content"), 0644)

	sm := NewSkillManager([]string{dir})
	sm.Discover()

	skill, err := sm.ViewSkillFull("full-skill")
	if err != nil {
		t.Fatalf("ViewSkillFull failed: %v", err)
	}

	if skill.LoadedTier != Tier3Full {
		t.Errorf("expected Tier3Full, got %d", skill.LoadedTier)
	}
	if len(skill.SupportFiles) != 2 {
		t.Errorf("expected 2 support files, got %d", len(skill.SupportFiles))
	}
}

func TestSkillManager_GetTier2Content(t *testing.T) {
	dir := t.TempDir()
	sd := filepath.Join(dir, "tier2")
	os.MkdirAll(sd, 0755)
	os.WriteFile(filepath.Join(sd, "SKILL.md"), []byte("---\nname: tier2\ndescription: tier2 test\n---\n\n# Core Content\n\nThis is the skill body."), 0644)

	sm := NewSkillManager([]string{dir})
	sm.Discover()

	content, err := sm.GetTier2Content("tier2")
	if err != nil {
		t.Fatalf("GetTier2Content failed: %v", err)
	}

	if !strings.Contains(content, "Core Content") {
		t.Error("expected Core Content in tier2 output")
	}
}

func TestSkillManager_UseCount(t *testing.T) {
	dir := t.TempDir()
	sd := filepath.Join(dir, "test")
	os.MkdirAll(sd, 0755)
	os.WriteFile(filepath.Join(sd, "SKILL.md"), []byte("---\nname: test\ndescription: test\n---\n\n# Content"), 0644)

	sm := NewSkillManager([]string{dir})
	sm.Discover()

	sm.ViewSkill("test")
	sm.ViewSkill("test")
	sm.ViewSkill("test")

	skill, _ := sm.GetSkill("test")
	if skill.UseCount != 3 {
		t.Errorf("expected UseCount=3, got %d", skill.UseCount)
	}
}
