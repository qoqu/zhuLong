package controller

import (
	"time"
)

// LoopState represents the state of the agent loop
type LoopState int

const (
	StateIdle          LoopState = iota // Waiting for goal input
	StatePlanning                        // Planning: calling Planner
	StateExecuting                       // Executing: calling Executor
	StateReflecting                      // Reflecting: calling Reflector
	StateReplanning                      // Replanning: adjusting plan
	StateWaitingHuman                    // Waiting for human input
	StateDone                            // Completed
	StateError                           // Error terminated
	StateCancelled                       // User cancelled
)

// String returns the string representation of the state
func (s LoopState) String() string {
	switch s {
	case StateIdle:
		return "idle"
	case StatePlanning:
		return "planning"
	case StateExecuting:
		return "executing"
	case StateReflecting:
		return "reflecting"
	case StateReplanning:
		return "replanning"
	case StateWaitingHuman:
		return "waiting_human"
	case StateDone:
		return "done"
	case StateError:
		return "error"
	case StateCancelled:
		return "cancelled"
	default:
		return "unknown"
	}
}

// Session contains the state of a single agent session
type Session struct {
	ID           string
	Goal         string
	State        LoopState
	Plan         *Plan
	CurrentStep  int
	LoopCount    int
	TokensUsed   int
	Cost         float64
	Error        error
	StartTime    time.Time
	LastActivity time.Time
}

// Plan contains the current plan
type Plan struct {
	ID          string
	Steps       []Step
	Rationale   string
	CreatedAt   time.Time
}

// Step represents a single step in the plan
type Step struct {
	ID          string
	Description string
	Action      Action
	DependsOn   []string
	Breakpoint  bool
}

// Action represents an action to be executed
type Action struct {
	Type    string
	Tool    string
	Params  map[string]interface{}
	Prompt  string
}

// NewSession creates a new session
func NewSession(id string, goal string) *Session {
	now := time.Now()
	return &Session{
		ID:           id,
		Goal:         goal,
		State:        StateIdle,
		StartTime:    now,
		LastActivity: now,
	}
}

// UpdateState updates the session state
func (s *Session) UpdateState(state LoopState) {
	s.State = state
	s.LastActivity = time.Now()
}

// IncrementLoop increments the loop count
func (s *Session) IncrementLoop() {
	s.LoopCount++
	s.LastActivity = time.Now()
}

// AddTokens adds tokens to the session
func (s *Session) AddTokens(tokens int) {
	s.TokensUsed += tokens
	s.LastActivity = time.Now()
}

// AddCost adds cost to the session
func (s *Session) AddCost(cost float64) {
	s.Cost += cost
	s.LastActivity = time.Now()
}
