package evolution

import (
	"testing"
)

func TestNewSuggestionEngine(t *testing.T) {
	e := NewSuggestionEngine()
	if e == nil {
		t.Fatal("expected non-nil engine")
	}
}

func TestFromPattern(t *testing.T) {
	e := NewSuggestionEngine()
	s := e.FromPattern("分析代码质量", 3)
	if s == nil {
		t.Fatal("expected suggestion")
	}
	if s.Type != SuggestionSkill {
		t.Errorf("expected SuggestionSkill, got %v", s.Type)
	}
	if s.Priority != 3 {
		t.Errorf("expected priority=3, got %d", s.Priority)
	}
}

func TestFromPattern_Duplicate(t *testing.T) {
	e := NewSuggestionEngine()
	e.FromPattern("pattern", 3)
	s := e.FromPattern("pattern", 5)
	if s != nil {
		t.Error("expected nil for duplicate pattern")
	}
}

func TestFromCatalog(t *testing.T) {
	e := NewSuggestionEngine()
	s := e.FromCatalog("自动化简报", "每天生成代码质量报告")
	if s == nil {
		t.Fatal("expected suggestion")
	}
	if s.Type != SuggestionBlueprint {
		t.Errorf("expected SuggestionBlueprint, got %v", s.Type)
	}
}

func TestAccept(t *testing.T) {
	e := NewSuggestionEngine()
	s := e.FromPattern("test", 3)

	ok := e.Accept(s.ID)
	if !ok {
		t.Fatal("expected accept to succeed")
	}

	pending := e.ListPending()
	if len(pending) != 0 {
		t.Errorf("expected 0 pending after accept, got %d", len(pending))
	}
}

func TestDismiss(t *testing.T) {
	e := NewSuggestionEngine()
	s := e.FromPattern("test", 3)

	ok := e.Dismiss(s.ID)
	if !ok {
		t.Fatal("expected dismiss to succeed")
	}

	all := e.ListAll()
	if len(all) != 0 {
		t.Errorf("expected 0 after dismiss, got %d", len(all))
	}
}

func TestMaxPending(t *testing.T) {
	e := NewSuggestionEngine()
	e.maxPending = 2

	e.FromPattern("a", 3)
	e.FromPattern("b", 3)
	s := e.FromPattern("c", 3)

	if s != nil {
		t.Error("expected nil when maxPending reached")
	}
}

func TestListPending_PriorityOrder(t *testing.T) {
	e := NewSuggestionEngine()

	e.FromPattern("low", 1)
	e.FromPattern("high", 5)
	e.FromPattern("mid", 3)

	pending := e.ListPending()
	if len(pending) < 3 {
		t.Fatalf("expected at least 3 pending, got %d", len(pending))
	}

	if pending[0].Priority < pending[1].Priority {
		t.Error("expected descending priority order")
	}
}

func TestListAll(t *testing.T) {
	e := NewSuggestionEngine()
	e.FromPattern("a", 3)
	e.FromPattern("b", 3)

	all := e.ListAll()
	if len(all) != 2 {
		t.Errorf("expected 2 suggestions, got %d", len(all))
	}
}
