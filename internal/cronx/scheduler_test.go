package cronx

import (
	"testing"
)

func TestNewScheduler(t *testing.T) {
	s := NewScheduler()
	if s == nil {
		t.Fatal("expected non-nil scheduler")
	}
}

func TestAddJob(t *testing.T) {
	s := NewScheduler()
	s.AddJob(Job{ID: "test", Prompt: "hello", NoAgent: true})
	due := s.GetDueJobs()
	if len(due) != 1 {
		t.Errorf("expected 1 due job, got %d", len(due))
	}
}

func TestWakeGate(t *testing.T) {
	s := NewScheduler()
	s.AddJob(Job{ID: "gated", Prompt: "test", WakeGate: &WakeGate{Type: "file_change", Config: map[string]string{"path": "/tmp/test"}}})

	due := s.GetDueJobs()
	if len(due) != 1 {
		t.Errorf("expected 1 due job, got %d", len(due))
	}
}

func TestNoAgentJob(t *testing.T) {
	s := NewScheduler()
	s.AddJob(Job{ID: "script", NoAgent: true, ScriptPath: "backup.sh"})

	due := s.GetDueJobs()
	if len(due) != 1 || !due[0].NoAgent {
		t.Error("expected no-agent job")
	}
}

func TestDescribeJob(t *testing.T) {
	j := Job{ID: "daily", NoAgent: true, WakeGate: &WakeGate{Type: "file_change"}, Deliver: "silent"}
	desc := DescribeJob(j)
	if !contains(desc, "无Agent") || !contains(desc, "静默") {
		t.Errorf("unexpected description: %s", desc)
	}
}

func TestContextChain(t *testing.T) {
	s := NewScheduler()
	s.AddJob(Job{ID: "a", Prompt: "first"})
	s.AddJob(Job{ID: "b", Prompt: "second", ContextFrom: "a"})

	due := s.GetDueJobs()
	if len(due) != 2 {
		t.Errorf("expected 2 due jobs, got %d", len(due))
	}
}

func TestNewGateChecker(t *testing.T) {
	g := &GateChecker{}
	r := g.Check(nil)
	if !r.WakeAgent {
		t.Error("expected wake when no gate")
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
