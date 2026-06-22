package backup

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewManager(t *testing.T) {
	dir := t.TempDir()
	m := NewManager(filepath.Join(dir, "backups"))
	if m == nil {
		t.Fatal("expected non-nil manager")
	}
}

func TestCreateAndList(t *testing.T) {
	dir := t.TempDir()
	m := NewManager(filepath.Join(dir, "backups"))

	// 创建测试文件
	srcDir := filepath.Join(dir, "src")
	os.MkdirAll(srcDir, 0755)
	os.WriteFile(filepath.Join(srcDir, "test.txt"), []byte("hello"), 0644)

	snap, err := m.Create("test-backup", []string{srcDir})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if snap.Size == 0 {
		t.Error("expected non-zero size")
	}

	snapshots, err := m.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(snapshots) != 1 {
		t.Errorf("expected 1 snapshot, got %d", len(snapshots))
	}
}

func TestRestore(t *testing.T) {
	dir := t.TempDir()
	m := NewManager(filepath.Join(dir, "backups"))

	srcDir := filepath.Join(dir, "src")
	os.MkdirAll(srcDir, 0755)
	os.WriteFile(filepath.Join(srcDir, "data.txt"), []byte("content"), 0644)

	snap, _ := m.Create("restore-test", []string{srcDir})

	restoreDir := filepath.Join(dir, "restored")
	err := m.Restore(snap, restoreDir)
	if err != nil {
		t.Fatalf("Restore failed: %v", err)
	}

	// data.txt 会在 restored/src/data.txt 因为rel路径以src为base
	files, _ := os.ReadDir(restoreDir)
	if len(files) == 0 {
		t.Fatal("no files restored")
	}

	found := false
	for _, f := range files {
		if f.Name() == "data.txt" || f.Name() == "src" {
			found = true
		}
	}
	if !found {
		t.Errorf("unexpected restored structure")
	}
}
