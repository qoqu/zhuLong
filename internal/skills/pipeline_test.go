package skills

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewPipeline(t *testing.T) {
	p := NewPipeline([]string{"/tmp/skills"})
	if p == nil {
		t.Fatal("expected non-nil pipeline")
	}
}

func TestPipeline_Init(t *testing.T) {
	dir := t.TempDir()
	sd := filepath.Join(dir, "test-skill")
	os.MkdirAll(sd, 0755)
	os.WriteFile(filepath.Join(sd, "SKILL.md"), []byte("---\nname: test-skill\ndescription: A test skill\n---\n\n# Test Skill\n\nDo something."), 0644)

	p := NewPipeline([]string{dir})
	err := p.Init()
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	list := p.GetSkillList()
	if !contains(list, "test-skill") {
		t.Error("expected test-skill in skill list")
	}
}

func TestPipeline_LoadAndCompress(t *testing.T) {
	dir := t.TempDir()
	sd := filepath.Join(dir, "test-skill")
	os.MkdirAll(sd, 0755)
	os.WriteFile(filepath.Join(sd, "SKILL.md"), []byte("---\nname: test-skill\ndescription: A test skill for compression\n---\n\n# Test Skill\n\nThis is a test skill with some content that should be compressed by the skeleton generator."), 0644)

	p := NewPipeline([]string{dir})
	p.Init()

	result, err := p.LoadAndCompress("test-skill")
	if err != nil {
		t.Fatalf("LoadAndCompress failed: %v", err)
	}

	if !contains(result, "test-skill") {
		t.Error("expected skill name in result")
	}
	if !contains(result, "Content") {
		t.Error("expected Content section in result")
	}
}

func TestPipeline_LoadFull(t *testing.T) {
	dir := t.TempDir()
	sd := filepath.Join(dir, "full-skill")
	os.MkdirAll(sd, 0755)
	os.WriteFile(filepath.Join(sd, "SKILL.md"), []byte("---\nname: full-skill\ndescription: full\n---\n\n# Full Content"), 0644)
	os.MkdirAll(filepath.Join(sd, "references"), 0755)
	os.WriteFile(filepath.Join(sd, "references", "guide.md"), []byte("# Guide"), 0644)

	p := NewPipeline([]string{dir})
	p.Init()

	result, err := p.LoadFull("full-skill")
	if err != nil {
		t.Fatalf("LoadFull failed: %v", err)
	}

	if !contains(result, "guide.md") {
		t.Error("expected guide.md in support files")
	}
}

func TestPipeline_Search(t *testing.T) {
	dir := t.TempDir()
	sd := filepath.Join(dir, "code-helper")
	os.MkdirAll(sd, 0755)
	os.WriteFile(filepath.Join(sd, "SKILL.md"), []byte("---\nname: code-helper\ndescription: Helps with coding\n---\n\n# Code Helper"), 0644)

	p := NewPipeline([]string{dir})
	p.Init()

	result := p.Search("coding")
	if !contains(result, "code-helper") {
		t.Error("expected code-helper in search results")
	}
}

func TestPipeline_SkillViewTool(t *testing.T) {
	dir := t.TempDir()
	sd := filepath.Join(dir, "tool-skill")
	os.MkdirAll(sd, 0755)
	os.WriteFile(filepath.Join(sd, "SKILL.md"), []byte("---\nname: tool-skill\ndescription: A tool skill\n---\n\n# Tool Skill Content"), 0644)

	p := NewPipeline([]string{dir})
	p.Init()

	result, err := p.SkillViewTool("tool-skill")
	if err != nil {
		t.Fatalf("SkillViewTool failed: %v", err)
	}

	if !contains(result, "tool-skill") {
		t.Error("expected tool-skill in result")
	}
}

func TestPipeline_Stats(t *testing.T) {
	dir := t.TempDir()
	sd := filepath.Join(dir, "stat-skill")
	os.MkdirAll(sd, 0755)
	os.WriteFile(filepath.Join(sd, "SKILL.md"), []byte("---\nname: stat-skill\ndescription: stats\n---\n\n# Stats"), 0644)

	p := NewPipeline([]string{dir})
	p.Init()

	stats := p.Stats()
	if stats["total"] != 1 {
		t.Errorf("expected total=1, got %d", stats["total"])
	}
}

func TestPipeline_Refresh(t *testing.T) {
	dir := t.TempDir()
	sd := filepath.Join(dir, "skill-a")
	os.MkdirAll(sd, 0755)
	os.WriteFile(filepath.Join(sd, "SKILL.md"), []byte("---\nname: skill-a\ndescription: a\n---\n\n# A"), 0644)

	p := NewPipeline([]string{dir})
	p.Init()

	// Add a new skill
	sd2 := filepath.Join(dir, "skill-b")
	os.MkdirAll(sd2, 0755)
	os.WriteFile(filepath.Join(sd2, "SKILL.md"), []byte("---\nname: skill-b\ndescription: b\n---\n\n# B"), 0644)

	err := p.Refresh()
	if err != nil {
		t.Fatalf("Refresh failed: %v", err)
	}

	list := p.GetSkillList()
	if !contains(list, "skill-b") {
		t.Error("expected skill-b after refresh")
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
