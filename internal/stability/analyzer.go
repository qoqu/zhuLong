package stability

import (
	"time"
)

// LoopSnapshot contains a snapshot of a loop's state
type LoopSnapshot struct {
	LoopNumber int
	State      string
	Progress   float64
	Timestamp  time.Time
}

// StabilityMetrics contains stability metrics
type StabilityMetrics struct {
	IsOscillating     bool
	IsDiverging       bool
	ConvergenceRate   float64
	OscillationFreq   int
	SettlingTime      time.Duration
}

// Analyzer analyzes system stability
type Analyzer struct {
	history []LoopSnapshot
	window  int
}

// NewAnalyzer creates a new stability analyzer
func NewAnalyzer(window int) *Analyzer {
	if window <= 0 {
		window = 4
	}
	return &Analyzer{
		history: make([]LoopSnapshot, 0),
		window:  window,
	}
}

// AddSnapshot adds a loop snapshot to the history
func (a *Analyzer) AddSnapshot(snapshot LoopSnapshot) {
	a.history = append(a.history, snapshot)
}

// IsOscillating detects if the system is oscillating
// Oscillation pattern: A → B → A → B
func (a *Analyzer) IsOscillating() bool {
	if len(a.history) < a.window {
		return false
	}

	n := len(a.history)
	recent := a.history[n-a.window:]

	// Check for A → B → A → B pattern
	if len(recent) >= 4 {
		return recent[0].State == recent[2].State &&
			recent[1].State == recent[3].State &&
			recent[0].State != recent[1].State
	}

	return false
}

// IsDiverging detects if the system is diverging (progress going backward)
func (a *Analyzer) IsDiverging() bool {
	if len(a.history) < 3 {
		return false
	}

	n := len(a.history)
	recent := a.history[n-3:]

	// Check if progress is decreasing
	return recent[0].Progress > recent[1].Progress &&
		recent[1].Progress > recent[2].Progress
}

// IsConverging detects if the system is converging (progress increasing)
func (a *Analyzer) IsConverging() bool {
	if len(a.history) < 3 {
		return false
	}

	n := len(a.history)
	recent := a.history[n-3:]

	// Check if progress is increasing
	return recent[0].Progress < recent[1].Progress &&
		recent[1].Progress < recent[2].Progress
}

// GetMetrics returns stability metrics
func (a *Analyzer) GetMetrics() StabilityMetrics {
	return StabilityMetrics{
		IsOscillating:   a.IsOscillating(),
		IsDiverging:     a.IsDiverging(),
		ConvergenceRate: a.calculateConvergenceRate(),
		OscillationFreq: a.countOscillations(),
	}
}

// calculateConvergenceRate calculates the convergence rate
func (a *Analyzer) calculateConvergenceRate() float64 {
	if len(a.history) < 2 {
		return 0
	}

	n := len(a.history)
	totalProgress := a.history[n-1].Progress - a.history[0].Progress
	totalLoops := float64(a.history[n-1].LoopNumber - a.history[0].LoopNumber)

	if totalLoops == 0 {
		return 0
	}

	return totalProgress / totalLoops
}

// countOscillations counts the number of oscillations in the history
func (a *Analyzer) countOscillations() int {
	count := 0
	for i := 2; i < len(a.history); i++ {
		if a.history[i-2].State == a.history[i].State &&
			a.history[i-1].State != a.history[i].State {
			count++
		}
	}
	return count
}

// GetHistory returns the history
func (a *Analyzer) GetHistory() []LoopSnapshot {
	return a.history
}

// ClearHistory clears the history
func (a *Analyzer) ClearHistory() {
	a.history = make([]LoopSnapshot, 0)
}
