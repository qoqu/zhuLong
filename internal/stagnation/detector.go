package stagnation

import (
	"time"
)

// StepInfo contains information about a step
type StepInfo struct {
	StepNumber    int
	NewInfoScore  float64
	ToolUsed      string
	ProgressDelta float64
	Timestamp     time.Time
}

// StagnationType represents the type of stagnation
type StagnationType int

const (
	StagnationNone         StagnationType = iota
	StagnationInfoStarved                 // No new information
	StagnationLooping                     // Repeating same actions
	StagnationBlocked                     // Progress blocked
)

// String returns the string representation
func (st StagnationType) String() string {
	switch st {
	case StagnationNone:
		return "none"
	case StagnationInfoStarved:
		return "info_starved"
	case StagnationLooping:
		return "looping"
	case StagnationBlocked:
		return "blocked"
	default:
		return "unknown"
	}
}

// Detector detects stagnation
type Detector struct {
	windowSize       int
	entropyThreshold float64
	history          []StepInfo
}

// NewDetector creates a new stagnation detector
func NewDetector(windowSize int, entropyThreshold float64) *Detector {
	if windowSize <= 0 {
		windowSize = 3
	}
	if entropyThreshold <= 0 {
		entropyThreshold = 0.2
	}

	return &Detector{
		windowSize:       windowSize,
		entropyThreshold: entropyThreshold,
		history:          make([]StepInfo, 0),
	}
}

// AddStep adds a step to the history
func (d *Detector) AddStep(step StepInfo) {
	d.history = append(d.history, step)
}

// IsStagnating detects if the system is stagnating
func (d *Detector) IsStagnating() bool {
	if len(d.history) < d.windowSize {
		return false
	}

	recent := d.history[len(d.history)-d.windowSize:]

	// Check if all recent steps have low information gain
	for _, step := range recent {
		if step.NewInfoScore >= d.entropyThreshold {
			return false // Still getting new information
		}
	}

	return true // No new information in window
}

// DiagnoseStagnation diagnoses the type of stagnation
func (d *Detector) DiagnoseStagnation() StagnationType {
	if !d.IsStagnating() {
		return StagnationNone
	}

	recent := d.history[len(d.history)-d.windowSize:]

	// Check for looping: same tool used repeatedly (more than half the window)
	toolCounts := make(map[string]int)
	for _, step := range recent {
		toolCounts[step.ToolUsed]++
	}
	halfWindow := d.windowSize / 2
	if halfWindow < 2 {
		halfWindow = 2
	}
	for _, count := range toolCounts {
		if count >= halfWindow {
			return StagnationLooping
		}
	}

	// Check for blocked: progress consistently negative
	negativeCount := 0
	for _, step := range recent {
		if step.ProgressDelta < 0 {
			negativeCount++
		}
	}
	if negativeCount >= d.windowSize/2 {
		return StagnationBlocked
	}

	// Default: info starved
	return StagnationInfoStarved
}

// GetHistory returns the history
func (d *Detector) GetHistory() []StepInfo {
	return d.history
}

// ClearHistory clears the history
func (d *Detector) ClearHistory() {
	d.history = make([]StepInfo, 0)
}
