package skills

import (
	"os"
	"path/filepath"
	"testing"
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
