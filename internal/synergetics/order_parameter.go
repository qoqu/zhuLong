package synergetics

import (
	"time"
)

// OrderParameter represents the order parameter (slow variable) of the system
type OrderParameter struct {
	Goal        string
	Strategy    string
	Priority    float64
	Stability   float64
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Session represents a session for order parameter identification
type Session struct {
	Goal             string
	CurrentStrategy  string
	StrategyChanges  int
	StartTime        time.Time
}

// IdentifyOrderParameter identifies the order parameter from a session
func IdentifyOrderParameter(session *Session) *OrderParameter {
	if session == nil {
		return &OrderParameter{
			Goal:      "",
			Strategy:  "",
			Priority:  0,
			Stability: 0,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
	}

	// Calculate stability (inverse of strategy changes)
	stability := 1.0 / (float64(session.StrategyChanges) + 1.0)

	return &OrderParameter{
		Goal:      session.Goal,
		Strategy:  session.CurrentStrategy,
		Priority:  1.0,
		Stability: stability,
		CreatedAt: session.StartTime,
		UpdatedAt: time.Now(),
	}
}

// UpdateStrategy updates the strategy in the order parameter
func (op *OrderParameter) UpdateStrategy(newStrategy string) {
	op.Strategy = newStrategy
	op.UpdatedAt = time.Now()
}

// UpdateGoal updates the goal in the order parameter
func (op *OrderParameter) UpdateGoal(newGoal string) {
	op.Goal = newGoal
	op.UpdatedAt = time.Now()
}

// IsStable returns true if the order parameter is stable
func (op *OrderParameter) IsStable() bool {
	return op.Stability > 0.5
}

// GetStabilityLevel returns the stability level
func (op *OrderParameter) GetStabilityLevel() string {
	if op.Stability > 0.8 {
		return "high"
	} else if op.Stability > 0.5 {
		return "medium"
	} else {
		return "low"
	}
}
