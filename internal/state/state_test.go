package state

import (
	"os"
	"path/filepath"
	"testing"
)

func setupTestStore(t *testing.T) *Store {
	dir := t.TempDir()
	return New(dir)
}

func TestNewStore(t *testing.T) {
	s := setupTestStore(t)
	if s == nil {
		t.Fatal("expected non-nil store")
	}
}

func TestGetHot_Initial(t *testing.T) {
	s := setupTestStore(t)
	hot := s.GetHot()
	if hot.Phase != "design" {
		t.Errorf("expected phase=design, got %s", hot.Phase)
	}
	if hot.Mode != "full" {
		t.Errorf("expected mode=full, got %s", hot.Mode)
	}
}

func TestUpdateHot(t *testing.T) {
	s := setupTestStore(t)

	err := s.UpdateHot(func(h *HotState) {
		h.Mode = "hotfix"
		h.Phase = "fix"
	})
	if err != nil {
		t.Fatalf("UpdateHot failed: %v", err)
	}

	hot := s.GetHot()
	if hot.Mode != "hotfix" {
		t.Errorf("expected mode=hotfix, got %s", hot.Mode)
	}
	if hot.Phase != "fix" {
		t.Errorf("expected phase=fix, got %s", hot.Phase)
	}
}

func TestAppendCold(t *testing.T) {
	s := setupTestStore(t)

	err := s.AppendCold(ColdLog{
		Type:      "gc_scan",
		Summary:   "scan completed",
		Score:     0.85,
		Timestamp: "now",
	})
	if err != nil {
		t.Fatalf("AppendCold failed: %v", err)
	}

	logs := s.GetCold(1)
	if len(logs) != 1 {
		t.Errorf("expected 1 log, got %d", len(logs))
	}
	if logs[0].Type != "gc_scan" {
		t.Errorf("expected type=gc_scan, got %s", logs[0].Type)
	}
}

func TestPersistence(t *testing.T) {
	dir := t.TempDir()

	// 写入
	s1 := New(dir)
	s1.UpdateHot(func(h *HotState) { h.Mode = "tweak" })
	s1.AppendCold(ColdLog{Type: "test", Summary: "test", Score: 1.0, Timestamp: "now"})

	// 在新实例中读取
	s2 := New(dir)
	hot := s2.GetHot()
	if hot.Mode != "tweak" {
		t.Errorf("expected mode=tweak after reload, got %s", hot.Mode)
	}
	logs := s2.GetCold(1)
	if len(logs) != 1 {
		t.Errorf("expected 1 log after reload, got %d", len(logs))
	}
}

func TestCold_MaxLogs(t *testing.T) {
	s := setupTestStore(t)

	for i := 0; i < 150; i++ {
		s.AppendCold(ColdLog{Type: "test", Summary: "test", Score: float64(i), Timestamp: "now"})
	}

	logs := s.GetCold(200)
	if len(logs) > 100 {
		t.Errorf("expected at most 100 logs, got %d", len(logs))
	}
}

func TestFileCreation(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)

	s.UpdateHot(func(h *HotState) { h.Mode = "hotfix" })

	// 检查文件是否存在
	hotFile := filepath.Join(dir, ".zhulong-state")
	if _, err := os.Stat(hotFile); os.IsNotExist(err) {
		t.Error("expected hot state file to exist")
	}
}

func TestConcurrentAccess(t *testing.T) {
	s := setupTestStore(t)

	// 并发读写
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			s.UpdateHot(func(h *HotState) { h.SessionChanges++ })
			s.AppendCold(ColdLog{Type: "test", Summary: "concurrent", Score: 0.5, Timestamp: "now"})
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	hot := s.GetHot()
	if hot.SessionChanges != 10 {
		t.Errorf("expected 10 session changes, got %d", hot.SessionChanges)
	}
}
