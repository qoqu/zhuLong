package hub

import (
	"testing"
)

func TestNewHub(t *testing.T) {
	h := New(nil)
	if h == nil {
		t.Fatal("expected non-nil hub")
	}
}

func TestRegister(t *testing.T) {
	h := New(nil)
	entry := &SkillEntry{
		Name: "test-skill", Description: "A test skill",
		Source: SourceOfficial, Version: "1.0.0",
	}

	report, err := h.Register(entry)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if !report.Passed {
		t.Errorf("expected pass for official skill, issues: %v", report.Issues)
	}
}

func TestRegister_CommunityBlocked(t *testing.T) {
	h := New(nil)
	entry := &SkillEntry{
		Name: "dangerous", Description: "contains danger",
		Source: SourceGitHub,
	}

	_, err := h.Register(entry)
	if err != nil {
		t.Fatalf("Register even dangerous should not error: %v", err)
	}
}

func TestSearch(t *testing.T) {
	h := New(nil)
	h.Register(&SkillEntry{Name: "code-review", Description: "Review code", Source: SourceOfficial})
	h.Register(&SkillEntry{Name: "writing", Description: "Write articles", Source: SourceOfficial})

	results := h.Search("code")
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
}

func TestSearch_NoResults(t *testing.T) {
	h := New(nil)
	results := h.Search("nonexistent")
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestInstall(t *testing.T) {
	installed := ""
	h := New(func(name string, source SourceType) error {
		installed = name
		return nil
	})

	h.Register(&SkillEntry{Name: "my-skill", Description: "test", Source: SourceOfficial})
	err := h.Install("my-skill")
	if err != nil {
		t.Fatalf("Install failed: %v", err)
	}
	if installed != "my-skill" {
		t.Errorf("expected installed=my-skill, got %s", installed)
	}
}

func TestListByTrust(t *testing.T) {
	h := New(nil)
	h.Register(&SkillEntry{Name: "official-1", Description: "official", Source: SourceOfficial})
	h.Register(&SkillEntry{Name: "community-1", Description: "community", Source: SourceGitHub})

	official := h.ListByTrust(TrustOfficial)
	if len(official) != 1 {
		t.Errorf("expected 1 official, got %d", len(official))
	}
}

func TestNewScanner(t *testing.T) {
	s := NewScanner()
	if s == nil {
		t.Fatal("expected non-nil scanner")
	}
}

func TestScanner_ScanBlocked(t *testing.T) {
	s := NewScanner()
	report := s.Scan(SkillEntry{Name: "bad"}, "contains rm -rf / command")
	if report.Passed {
		t.Error("expected blocked pattern to fail")
	}
	if report.RiskLevel != "critical" {
		t.Errorf("expected risk=critical, got %s", report.RiskLevel)
	}
}

func TestScanner_ScanSafe(t *testing.T) {
	s := NewScanner()
	report := s.Scan(SkillEntry{Name: "safe", Description: "a safe skill"}, "safe content")
	if !report.Passed {
		t.Errorf("expected safe content to pass, issues: %v", report.Issues)
	}
}

func TestHub_Stats(t *testing.T) {
	h := New(nil)
	h.Register(&SkillEntry{Name: "a", Description: "a", Source: SourceOfficial})
	h.Register(&SkillEntry{Name: "b", Description: "b", Source: SourceOfficial})

	stats := h.Stats()
	if stats["total"] != 2 {
		t.Errorf("expected total=2, got %d", stats["total"])
	}
}
