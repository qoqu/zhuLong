package controller

import (
	"testing"
	"time"
)

func TestLoopState_String(t *testing.T) {
	tests := []struct {
		state    LoopState
		expected string
	}{
		{StateIdle, "idle"},
		{StatePlanning, "planning"},
		{StateExecuting, "executing"},
		{StateReflecting, "reflecting"},
		{StateReplanning, "replanning"},
		{StateWaitingHuman, "waiting_human"},
		{StateDone, "done"},
		{StateError, "error"},
		{StateCancelled, "cancelled"},
		{LoopState(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.state.String(); got != tt.expected {
				t.Errorf("LoopState.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestNewSession(t *testing.T) {
	id := "test-session-1"
	goal := "Test goal"

	session := NewSession(id, goal)

	if session.ID != id {
		t.Errorf("Session.ID = %v, want %v", session.ID, id)
	}
	if session.Goal != goal {
		t.Errorf("Session.Goal = %v, want %v", session.Goal, goal)
	}
	if session.State != StateIdle {
		t.Errorf("Session.State = %v, want %v", session.State, StateIdle)
	}
	if session.LoopCount != 0 {
		t.Errorf("Session.LoopCount = %v, want 0", session.LoopCount)
	}
	if session.TokensUsed != 0 {
		t.Errorf("Session.TokensUsed = %v, want 0", session.TokensUsed)
	}
	if session.Cost != 0 {
		t.Errorf("Session.Cost = %v, want 0", session.Cost)
	}
}

func TestSession_UpdateState(t *testing.T) {
	session := NewSession("test", "goal")
	oldTime := session.LastActivity

	// Small delay to ensure time difference
	time.Sleep(10 * time.Millisecond)

	session.UpdateState(StatePlanning)

	if session.State != StatePlanning {
		t.Errorf("Session.State = %v, want %v", session.State, StatePlanning)
	}
	if !session.LastActivity.After(oldTime) {
		t.Error("LastActivity should be updated")
	}
}

func TestSession_IncrementLoop(t *testing.T) {
	session := NewSession("test", "goal")

	session.IncrementLoop()
	if session.LoopCount != 1 {
		t.Errorf("Session.LoopCount = %v, want 1", session.LoopCount)
	}

	session.IncrementLoop()
	if session.LoopCount != 2 {
		t.Errorf("Session.LoopCount = %v, want 2", session.LoopCount)
	}
}

func TestSession_AddTokens(t *testing.T) {
	session := NewSession("test", "goal")

	session.AddTokens(100)
	if session.TokensUsed != 100 {
		t.Errorf("Session.TokensUsed = %v, want 100", session.TokensUsed)
	}

	session.AddTokens(200)
	if session.TokensUsed != 300 {
		t.Errorf("Session.TokensUsed = %v, want 300", session.TokensUsed)
	}
}

func TestSession_AddCost(t *testing.T) {
	session := NewSession("test", "goal")

	session.AddCost(1.5)
	if session.Cost != 1.5 {
		t.Errorf("Session.Cost = %v, want 1.5", session.Cost)
	}

	session.AddCost(2.5)
	if session.Cost != 4.0 {
		t.Errorf("Session.Cost = %v, want 4.0", session.Cost)
	}
}
