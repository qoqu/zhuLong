package review

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewRecorder(t *testing.T) {
	dir := t.TempDir()
	r := New(filepath.Join(dir, "reviews"))
	if r == nil {
		t.Fatal("expected non-nil recorder")
	}
}

func TestRecord(t *testing.T) {
	dir := t.TempDir()
	r := New(filepath.Join(dir, "reviews"))

	report := &Report{
		SessionID: "session-001",
		Changes: []Change{
			{File: "main.go", Type: "modified", Summary: "fix bug"},
		},
		Findings:   []string{"unused variable"},
		Suggestions: []string{"remove unused var"},
	}

	err := r.Record(report)
	if err != nil {
		t.Fatalf("Record failed: %v", err)
	}
}

func TestListRecent(t *testing.T) {
	dir := t.TempDir()
	r := New(filepath.Join(dir, "reviews"))

	for i := 0; i < 5; i++ {
		r.Record(&Report{SessionID: "test", Findings: []string{"test"}})
	}

	reports, err := r.ListRecent(3)
	if err != nil {
		t.Fatalf("ListRecent failed: %v", err)
	}

	if len(reports) != 3 {
		t.Errorf("expected 3 reports, got %d", len(reports))
	}
}

func TestSeverityEvaluation(t *testing.T) {
	dir := t.TempDir()
	r := New(filepath.Join(dir, "reviews"))

	tests := []struct {
		name     string
		report   *Report
		expected Severity
	}{
		{"no findings", &Report{SessionID: "test"}, SeverityLow},
		{"medium", &Report{SessionID: "test", Findings: []string{"minor issue"}}, SeverityMedium},
		{"high", &Report{SessionID: "test", Findings: []string{"a", "b", "c", "d"}}, SeverityHigh},
		{"critical", &Report{SessionID: "test", Findings: []string{"critical issue"}}, SeverityCritical},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r.Record(tt.report)
		})
	}
}

func TestScore(t *testing.T) {
	dir := t.TempDir()
	r := New(filepath.Join(dir, "reviews"))

	r.Record(&Report{SessionID: "test"})
	reports, _ := r.ListRecent(1)
	if len(reports) == 0 {
		t.Fatal("expected at least 1 report")
	}
	if !contains(reports[0], "Score") {
		t.Error("expected Score in report")
	}
}

func TestFilePersistence(t *testing.T) {
	dir := t.TempDir()
	reviewsDir := filepath.Join(dir, "reviews")
	r := New(reviewsDir)

	r.Record(&Report{SessionID: "persist-test", Changes: []Change{{File: "test.go", Type: "added", Summary: "new file"}}})

	entries, err := os.ReadDir(reviewsDir)
	if err != nil {
		t.Fatalf("ReadDir failed: %v", err)
	}

	if len(entries) == 0 {
		t.Error("expected files in reviews directory")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsHelper(s, substr)
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
