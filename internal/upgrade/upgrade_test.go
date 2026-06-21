package upgrade

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUpgrader_RegisterTemplate(t *testing.T) {
	u := New("1.0.0")
	u.RegisterTemplate("README.md", "abc123")

	if len(u.templateFiles) != 1 {
		t.Errorf("expected 1 template file, got %d", len(u.templateFiles))
	}
}

func TestUpgrader_ScanFileOrigin_User(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "main.go")
	os.WriteFile(file, []byte("package main"), 0644)

	u := New("1.0.0")
	origin := u.ScanFileOrigin(file)

	if origin != OriginUser {
		t.Errorf("expected user origin, got %v", origin)
	}
}

func TestUpgrader_ScanFileOrigin_Template(t *testing.T) {
	u := New("1.0.0")
	u.RegisterTemplate("README.md", "abc123")

	origin := u.ScanFileOrigin("README.md")
	if origin != OriginTemplate {
		t.Errorf("expected template origin, got %v", origin)
	}
}

func TestUpgrader_ScanFileOrigin_Unknown(t *testing.T) {
	u := New("1.0.0")
	origin := u.ScanFileOrigin("/nonexistent/path")
	if origin != OriginUnknown {
		t.Errorf("expected unknown origin, got %v", origin)
	}
}

func TestUpgrader_PlanUpgrade_DryRun(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("readme"), 0644)
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), 0644)

	u := New("1.0.0")
	plan, err := u.PlanUpgrade(dir, true)
	if err != nil {
		t.Fatalf("PlanUpgrade failed: %v", err)
	}

	if !plan.DryRun {
		t.Error("expected dry run")
	}
}

func TestUpgrader_ApplyUpgrade_DryRun(t *testing.T) {
	dir := t.TempDir()
	u := New("1.0.0")

	plan, err := u.PlanUpgrade(dir, true)
	if err != nil {
		t.Fatalf("PlanUpgrade failed: %v", err)
	}

	err = u.ApplyUpgrade(plan)
	if err != nil {
		t.Errorf("ApplyUpgrade failed: %v", err)
	}
}

func TestUpgrader_Version(t *testing.T) {
	u := New("2.0.0")
	dir := t.TempDir()

	plan, _ := u.PlanUpgrade(dir, true)
	if plan.Version != "2.0.0" {
		t.Errorf("expected version 2.0.0, got %s", plan.Version)
	}
}
