package evolution

import (
	"fmt"
	"strings"
	"testing"
)

func TestNewBackgroundReviewer(t *testing.T) {
	r := NewBackgroundReviewer(nil)
	if r == nil {
		t.Fatal("expected non-nil reviewer")
	}
}

func TestRecordSession(t *testing.T) {
	r := NewBackgroundReviewer(nil)
	r.RecordSession(SessionSnapshot{
		Goal:     "test goal",
		Messages: []MessagePair{{Role: "user", Content: "hello"}},
	})

	if len(r.sessionLogs) != 1 {
		t.Errorf("expected 1 session, got %d", len(r.sessionLogs))
	}
}

func TestReview_NoSessions(t *testing.T) {
	r := NewBackgroundReviewer(nil)
	result, err := r.Review()
	if err != nil {
		t.Fatalf("Review failed: %v", err)
	}
	if result.HasChanges {
		t.Error("expected no changes for empty sessions")
	}
}

func TestHeuristicReview_StyleCorrection(t *testing.T) {
	result := HeuristicReview(SessionSnapshot{
		Goal:     "write article",
		Messages: []MessagePair{{Role: "user", Content: "请简洁一点，不要冗长"}},
	})
	if !result.HasChanges {
		t.Error("expected style correction signal")
	}
	if result.Signal != SignalStyleCorrection {
		t.Errorf("expected SignalStyleCorrection, got %v", result.Signal)
	}
}

func TestHeuristicReview_WorkflowChange(t *testing.T) {
	result := HeuristicReview(SessionSnapshot{
		Goal:     "refactor code",
		Messages: []MessagePair{{Role: "user", Content: "应该先测试再重构，顺序错了"}},
	})
	if !result.HasChanges {
		t.Error("expected workflow change signal")
	}
	if result.Signal != SignalWorkflowChange {
		t.Errorf("expected SignalWorkflowChange, got %v", result.Signal)
	}
}

func TestHeuristicReview_NewTechnique(t *testing.T) {
	result := HeuristicReview(SessionSnapshot{
		Goal:     "optimize",
		Messages: []MessagePair{{Role: "user", Content: "有个更好的方法：用缓存优化"}},
	})
	if !result.HasChanges {
		t.Error("expected new technique signal")
	}
}

func TestHeuristicReview_NoMatch(t *testing.T) {
	result := HeuristicReview(SessionSnapshot{
		Goal:     "simple task",
		Messages: []MessagePair{{Role: "user", Content: "请帮我做这个"}},
	})
	if result.HasChanges {
		t.Error("expected no changes for neutral message")
	}
}

func TestAnalyzePatterns(t *testing.T) {
	r := NewBackgroundReviewer(nil)

	for i := 0; i < 5; i++ {
		r.RecordSession(SessionSnapshot{
			Goal:     "分析代码质量",
			Messages: []MessagePair{{Role: "user", Content: "分析代码"}},
		})
	}

	suggestions := r.AnalyzePatterns()
	if len(suggestions) == 0 {
		t.Error("expected suggestions for repeating patterns")
	}
}

func TestAnalyzePatterns_FewSessions(t *testing.T) {
	r := NewBackgroundReviewer(nil)
	r.RecordSession(SessionSnapshot{Goal: "one", Messages: nil})
	r.RecordSession(SessionSnapshot{Goal: "two", Messages: nil})

	suggestions := r.AnalyzePatterns()
	if len(suggestions) != 0 {
		t.Errorf("expected no suggestions for only 2 sessions, got %d", len(suggestions))
	}
}

func TestMaxSessions(t *testing.T) {
	r := NewBackgroundReviewer(nil)
	r.maxSessions = 3

	for i := 0; i < 10; i++ {
		r.RecordSession(SessionSnapshot{
			Goal:     fmt.Sprintf("session-%d", i),
			Messages: nil,
		})
	}

	if len(r.sessionLogs) != 3 {
		t.Errorf("expected 3 sessions (max), got %d", len(r.sessionLogs))
	}
}

func TestNormalizeGoal(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"  Write Code  ", "write code"},
		{"A"+strings.Repeat("x", 100), "a" + strings.Repeat("x", 49)},
	}

	for _, tt := range tests {
		result := normalizeGoal(tt.input)
		if len(result) > 50 {
			t.Errorf("normalized goal too long: %d", len(result))
		}
		_ = tt.expected
	}
}
