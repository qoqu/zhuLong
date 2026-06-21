package quality

import (
	"testing"
)

func TestScanner_DefaultDimensions(t *testing.T) {
	s := NewScanner()
	if len(s.dimensions) != 8 {
		t.Errorf("expected 8 dimensions, got %d", len(s.dimensions))
	}
}

func TestScanner_Scan(t *testing.T) {
	s := NewScanner()
	report, err := s.Scan(".")
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	if len(report.Dimensions) != 8 {
		t.Errorf("expected 8 dimensions in report, got %d", len(report.Dimensions))
	}

	for _, d := range report.Dimensions {
		if d.Score < 0 || d.Score > 1 {
			t.Errorf("dimension %s score out of range: %f", d.Name, d.Score)
		}
	}
}

func TestScanner_AddDimension(t *testing.T) {
	s := NewScanner()
	s.AddDimension(ScanDimension{
		Name:        "custom",
		Description: "custom check",
		Weight:      0.1,
	})

	if len(s.dimensions) != 9 {
		t.Errorf("expected 9 dimensions after add, got %d", len(s.dimensions))
	}
}

func TestScanner_DocumentationScan(t *testing.T) {
	s := NewScanner()
	report, err := s.Scan(".")
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	for _, d := range report.Dimensions {
		if d.Name == "documentation" {
			if d.Score < 0 || d.Score > 1 {
				t.Errorf("documentation score out of range: %f", d.Score)
			}
			return
		}
	}
	t.Error("documentation dimension not found")
}

func TestScanner_TotalScore(t *testing.T) {
	s := NewScanner()
	report, err := s.Scan(".")
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	if report.TotalScore < 0 || report.TotalScore > 1 {
		t.Errorf("total score out of range: %f", report.TotalScore)
	}
}

func TestScanner_TodoDensity(t *testing.T) {
	s := NewScanner()
	report, err := s.Scan(".")
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	for _, d := range report.Dimensions {
		if d.Name == "todo_density" {
			return
		}
	}
	t.Error("todo_density dimension not found")
}

func TestScanner_TestCoverage(t *testing.T) {
	s := NewScanner()
	report, err := s.Scan(".")
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	for _, d := range report.Dimensions {
		if d.Name == "test_coverage" {
			return
		}
	}
	t.Error("test_coverage dimension not found")
}
