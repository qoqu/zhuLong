package profile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewManager(t *testing.T) {
	m := NewManager(t.TempDir())
	if m == nil {
		t.Fatal("expected non-nil manager")
	}
}

func TestCreateAndSwitch(t *testing.T) {
	dir := t.TempDir()
	m := NewManager(dir)

	p, err := m.Create("dev")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if p.Name != "dev" {
		t.Errorf("expected name=dev, got %s", p.Name)
	}

	// 验证主目录已创建
	if _, err := os.Stat(filepath.Join(dir, "dev")); os.IsNotExist(err) {
		t.Error("profile home not created")
	}

	m.Create("prod")
	err = m.Switch("prod")
	if err != nil {
		t.Fatalf("Switch failed: %v", err)
	}
	if m.Current().Name != "prod" {
		t.Errorf("expected current=prod, got %s", m.Current().Name)
	}
}

func TestList(t *testing.T) {
	m := NewManager(t.TempDir())
	m.Create("a")
	m.Create("b")

	names := m.List()
	if len(names) != 2 {
		t.Errorf("expected 2 profiles, got %d", len(names))
	}
}

func TestDefaultProfile(t *testing.T) {
	m := NewManager(t.TempDir())
	if m.Current() != nil {
		t.Error("expected nil current before any profile created")
	}
}
