package qa

import (
	"testing"
)

func TestNewLab(t *testing.T) {
	l := NewLab()
	if l == nil {
		t.Fatal("expected non-nil lab")
	}
}

func TestAddAndRun(t *testing.T) {
	l := NewLab()
	l.Add(TestCase{
		Name: "echo", Input: "hello",
		Expected: "hello",
		Run: func(input string) (string, error) {
			return input, nil
		},
	})

	results := l.RunAll()
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].Passed {
		t.Errorf("expected passed, actual=%s", results[0].Actual)
	}
}

func TestFailed(t *testing.T) {
	l := NewLab()
	l.Add(TestCase{
		Name: "fail", Input: "x", Expected: "y",
		Run: func(input string) (string, error) { return "z", nil },
	})

	results := l.RunAll()
	if results[0].Passed {
		t.Error("expected failure")
	}
}

func TestSummary(t *testing.T) {
	results := []Result{
		{Name: "a", Passed: true},
		{Name: "b", Passed: false},
	}
	s := Summary(results)
	if !contains(s, "1/2") {
		t.Errorf("unexpected summary: %s", s)
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
