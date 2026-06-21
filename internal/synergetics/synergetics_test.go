package synergetics

import (
	"testing"
	"time"
)

func TestIdentifyOrderParameter(t *testing.T) {
	session := &Session{
		Goal:            "Read and analyze code",
		CurrentStrategy: "Sequential execution",
		StrategyChanges: 0,
		StartTime:       time.Now(),
	}

	op := IdentifyOrderParameter(session)

	if op.Goal != session.Goal {
		t.Errorf("OrderParameter.Goal = %v, want %v", op.Goal, session.Goal)
	}
	if op.Strategy != session.CurrentStrategy {
		t.Errorf("OrderParameter.Strategy = %v, want %v", op.Strategy, session.CurrentStrategy)
	}
	if op.Priority != 1.0 {
		t.Errorf("OrderParameter.Priority = %v, want 1.0", op.Priority)
	}
	if op.Stability != 1.0 {
		t.Errorf("OrderParameter.Stability = %v, want 1.0", op.Stability)
	}
}

func TestIdentifyOrderParameter_NilSession(t *testing.T) {
	op := IdentifyOrderParameter(nil)

	if op.Goal != "" {
		t.Errorf("OrderParameter.Goal = %v, want empty", op.Goal)
	}
	if op.Stability != 0 {
		t.Errorf("OrderParameter.Stability = %v, want 0", op.Stability)
	}
}

func TestIdentifyOrderParameter_WithStrategyChanges(t *testing.T) {
	session := &Session{
		Goal:            "Test goal",
		CurrentStrategy: "Strategy",
		StrategyChanges: 2,
		StartTime:       time.Now(),
	}

	op := IdentifyOrderParameter(session)

	// Stability = 1 / (2 + 1) = 0.333...
	if op.Stability >= 1.0 {
		t.Errorf("OrderParameter.Stability = %v, should be < 1.0", op.Stability)
	}
}

func TestOrderParameter_UpdateStrategy(t *testing.T) {
	op := &OrderParameter{
		Goal:      "Test",
		Strategy:  "Old strategy",
		Priority:  1.0,
		Stability: 1.0,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	oldUpdatedAt := op.UpdatedAt
	time.Sleep(10 * time.Millisecond)

	op.UpdateStrategy("New strategy")

	if op.Strategy != "New strategy" {
		t.Errorf("OrderParameter.Strategy = %v, want 'New strategy'", op.Strategy)
	}
	if !op.UpdatedAt.After(oldUpdatedAt) {
		t.Error("UpdatedAt should be updated")
	}
}

func TestOrderParameter_UpdateGoal(t *testing.T) {
	op := &OrderParameter{
		Goal:      "Old goal",
		Strategy:  "Strategy",
		Priority:  1.0,
		Stability: 1.0,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	op.UpdateGoal("New goal")

	if op.Goal != "New goal" {
		t.Errorf("OrderParameter.Goal = %v, want 'New goal'", op.Goal)
	}
}

func TestOrderParameter_IsStable(t *testing.T) {
	tests := []struct {
		stability float64
		expected  bool
	}{
		{0.9, true},
		{0.7, true},
		{0.5, false},
		{0.3, false},
	}

	for _, tt := range tests {
		op := &OrderParameter{Stability: tt.stability}
		if got := op.IsStable(); got != tt.expected {
			t.Errorf("IsStable() with stability %v = %v, want %v", tt.stability, got, tt.expected)
		}
	}
}

func TestOrderParameter_GetStabilityLevel(t *testing.T) {
	tests := []struct {
		stability float64
		expected  string
	}{
		{0.9, "high"},
		{0.7, "medium"},
		{0.3, "low"},
	}

	for _, tt := range tests {
		op := &OrderParameter{Stability: tt.stability}
		if got := op.GetStabilityLevel(); got != tt.expected {
			t.Errorf("GetStabilityLevel() with stability %v = %v, want %v", tt.stability, got, tt.expected)
		}
	}
}

func TestNewSlavingPrinciple(t *testing.T) {
	op := &OrderParameter{Goal: "Test goal"}
	sp := NewSlavingPrinciple(op)

	if sp == nil {
		t.Error("NewSlavingPrinciple() should not return nil")
	}
	if sp.GetOrderParameter() != op {
		t.Error("SlavingPrinciple should store the order parameter")
	}
}

func TestSlavingPrinciple_EnforceSlaving(t *testing.T) {
	op := &OrderParameter{Goal: "Read file contents"}
	sp := NewSlavingPrinciple(op)

	action := &Action{
		Description: "Read the main.go file",
		Type:        "tool_call",
		Tool:        "read_file",
		Priority:    1.0,
	}

	enforced, err := sp.EnforceSlaving(action)

	if err != nil {
		t.Errorf("EnforceSlaving() error = %v", err)
	}
	if enforced == nil {
		t.Error("EnforceSlaving() should not return nil")
	}
	if enforced.Priority <= 0 {
		t.Errorf("EnforcedAction.Priority = %v, should be > 0", enforced.Priority)
	}
}

func TestSlavingPrinciple_EnforceSlaving_NilAction(t *testing.T) {
	op := &OrderParameter{Goal: "Test"}
	sp := NewSlavingPrinciple(op)

	_, err := sp.EnforceSlaving(nil)

	if err == nil {
		t.Error("EnforceSlaving() should return error for nil action")
	}
}

func TestSlavingPrinciple_UpdateOrderParameter(t *testing.T) {
	op1 := &OrderParameter{Goal: "Old goal"}
	sp := NewSlavingPrinciple(op1)

	op2 := &OrderParameter{Goal: "New goal"}
	sp.UpdateOrderParameter(op2)

	if sp.GetOrderParameter().Goal != "New goal" {
		t.Errorf("OrderParameter.Goal = %v, want 'New goal'", sp.GetOrderParameter().Goal)
	}
}
