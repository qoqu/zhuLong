package stagnation

import (
	"testing"
	"time"
)

func TestStagnationType_String(t *testing.T) {
	tests := []struct {
		st       StagnationType
		expected string
	}{
		{StagnationNone, "none"},
		{StagnationInfoStarved, "info_starved"},
		{StagnationLooping, "looping"},
		{StagnationBlocked, "blocked"},
		{StagnationType(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.st.String(); got != tt.expected {
				t.Errorf("StagnationType.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestNewDetector(t *testing.T) {
	detector := NewDetector(3, 0.2)

	if detector == nil {
		t.Error("NewDetector() should not return nil")
	}
	if detector.windowSize != 3 {
		t.Errorf("Detector windowSize = %v, want 3", detector.windowSize)
	}
	if detector.entropyThreshold != 0.2 {
		t.Errorf("Detector entropyThreshold = %v, want 0.2", detector.entropyThreshold)
	}
}

func TestNewDetector_Defaults(t *testing.T) {
	detector := NewDetector(0, 0)

	if detector.windowSize != 3 {
		t.Errorf("Default windowSize = %v, want 3", detector.windowSize)
	}
	if detector.entropyThreshold != 0.2 {
		t.Errorf("Default entropyThreshold = %v, want 0.2", detector.entropyThreshold)
	}
}

func TestDetector_AddStep(t *testing.T) {
	detector := NewDetector(3, 0.2)

	step := StepInfo{
		StepNumber:    1,
		NewInfoScore:  0.5,
		ToolUsed:      "read_file",
		ProgressDelta: 0.1,
		Timestamp:     time.Now(),
	}

	detector.AddStep(step)

	if len(detector.GetHistory()) != 1 {
		t.Errorf("History length = %v, want 1", len(detector.GetHistory()))
	}
}

func TestDetector_IsStagnating(t *testing.T) {
	detector := NewDetector(3, 0.2)

	// Not enough history
	if detector.IsStagnating() {
		t.Error("Should not be stagnating with empty history")
	}

	// Add steps with high information gain
	detector.AddStep(StepInfo{StepNumber: 1, NewInfoScore: 0.8, ToolUsed: "read_file"})
	detector.AddStep(StepInfo{StepNumber: 2, NewInfoScore: 0.6, ToolUsed: "write_file"})
	detector.AddStep(StepInfo{StepNumber: 3, NewInfoScore: 0.7, ToolUsed: "search_file"})

	if detector.IsStagnating() {
		t.Error("Should not be stagnating with high information gain")
	}

	// Add steps with low information gain
	detector2 := NewDetector(3, 0.2)
	detector2.AddStep(StepInfo{StepNumber: 1, NewInfoScore: 0.1, ToolUsed: "read_file"})
	detector2.AddStep(StepInfo{StepNumber: 2, NewInfoScore: 0.05, ToolUsed: "read_file"})
	detector2.AddStep(StepInfo{StepNumber: 3, NewInfoScore: 0.02, ToolUsed: "read_file"})

	if !detector2.IsStagnating() {
		t.Error("Should be stagnating with low information gain")
	}
}

func TestDetector_DiagnoseStagnation(t *testing.T) {
	detector := NewDetector(3, 0.2)

	// Not stagnating
	if detector.DiagnoseStagnation() != StagnationNone {
		t.Error("Should be StagnationNone when not stagnating")
	}

	// Info starved
	detector.AddStep(StepInfo{StepNumber: 1, NewInfoScore: 0.1, ToolUsed: "read_file", ProgressDelta: 0.0})
	detector.AddStep(StepInfo{StepNumber: 2, NewInfoScore: 0.05, ToolUsed: "write_file", ProgressDelta: 0.0})
	detector.AddStep(StepInfo{StepNumber: 3, NewInfoScore: 0.02, ToolUsed: "search_file", ProgressDelta: 0.0})

	if detector.DiagnoseStagnation() != StagnationInfoStarved {
		t.Errorf("DiagnoseStagnation() = %v, want StagnationInfoStarved", detector.DiagnoseStagnation())
	}

	// Looping
	detector2 := NewDetector(3, 0.2)
	detector2.AddStep(StepInfo{StepNumber: 1, NewInfoScore: 0.1, ToolUsed: "read_file", ProgressDelta: 0.0})
	detector2.AddStep(StepInfo{StepNumber: 2, NewInfoScore: 0.05, ToolUsed: "read_file", ProgressDelta: 0.0})
	detector2.AddStep(StepInfo{StepNumber: 3, NewInfoScore: 0.02, ToolUsed: "read_file", ProgressDelta: 0.0})

	if detector2.DiagnoseStagnation() != StagnationLooping {
		t.Errorf("DiagnoseStagnation() = %v, want StagnationLooping", detector2.DiagnoseStagnation())
	}

	// Blocked
	detector3 := NewDetector(3, 0.2)
	detector3.AddStep(StepInfo{StepNumber: 1, NewInfoScore: 0.1, ToolUsed: "read_file", ProgressDelta: -0.1})
	detector3.AddStep(StepInfo{StepNumber: 2, NewInfoScore: 0.05, ToolUsed: "write_file", ProgressDelta: -0.2})
	detector3.AddStep(StepInfo{StepNumber: 3, NewInfoScore: 0.02, ToolUsed: "search_file", ProgressDelta: -0.1})

	if detector3.DiagnoseStagnation() != StagnationBlocked {
		t.Errorf("DiagnoseStagnation() = %v, want StagnationBlocked", detector3.DiagnoseStagnation())
	}
}

func TestDetector_ClearHistory(t *testing.T) {
	detector := NewDetector(3, 0.2)

	detector.AddStep(StepInfo{StepNumber: 1, NewInfoScore: 0.5, ToolUsed: "read_file"})
	detector.AddStep(StepInfo{StepNumber: 2, NewInfoScore: 0.6, ToolUsed: "write_file"})

	detector.ClearHistory()

	if len(detector.GetHistory()) != 0 {
		t.Errorf("History length after clear = %v, want 0", len(detector.GetHistory()))
	}
}
