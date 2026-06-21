package stability

import (
	"testing"
	"time"
)

func TestNewAnalyzer(t *testing.T) {
	analyzer := NewAnalyzer(4)

	if analyzer == nil {
		t.Error("NewAnalyzer() should not return nil")
	}
	if analyzer.window != 4 {
		t.Errorf("Analyzer window = %v, want 4", analyzer.window)
	}
}

func TestNewAnalyzer_DefaultWindow(t *testing.T) {
	analyzer := NewAnalyzer(0)

	if analyzer.window != 4 {
		t.Errorf("Default window = %v, want 4", analyzer.window)
	}
}

func TestAnalyzer_AddSnapshot(t *testing.T) {
	analyzer := NewAnalyzer(4)

	snapshot := LoopSnapshot{
		LoopNumber: 1,
		State:      "executing",
		Progress:   0.5,
		Timestamp:  time.Now(),
	}

	analyzer.AddSnapshot(snapshot)

	if len(analyzer.GetHistory()) != 1 {
		t.Errorf("History length = %v, want 1", len(analyzer.GetHistory()))
	}
}

func TestAnalyzer_IsOscillating(t *testing.T) {
	analyzer := NewAnalyzer(4)

	// Not enough history
	if analyzer.IsOscillating() {
		t.Error("Should not be oscillating with empty history")
	}

	// Add non-oscillating pattern
	analyzer.AddSnapshot(LoopSnapshot{LoopNumber: 1, State: "executing", Progress: 0.1})
	analyzer.AddSnapshot(LoopSnapshot{LoopNumber: 2, State: "executing", Progress: 0.2})
	analyzer.AddSnapshot(LoopSnapshot{LoopNumber: 3, State: "reflecting", Progress: 0.3})
	analyzer.AddSnapshot(LoopSnapshot{LoopNumber: 4, State: "done", Progress: 0.4})

	if analyzer.IsOscillating() {
		t.Error("Should not be oscillating with non-oscillating pattern")
	}

	// Add oscillating pattern: A → B → A → B
	analyzer2 := NewAnalyzer(4)
	analyzer2.AddSnapshot(LoopSnapshot{LoopNumber: 1, State: "executing", Progress: 0.1})
	analyzer2.AddSnapshot(LoopSnapshot{LoopNumber: 2, State: "reflecting", Progress: 0.2})
	analyzer2.AddSnapshot(LoopSnapshot{LoopNumber: 3, State: "executing", Progress: 0.1})
	analyzer2.AddSnapshot(LoopSnapshot{LoopNumber: 4, State: "reflecting", Progress: 0.2})

	if !analyzer2.IsOscillating() {
		t.Error("Should be oscillating with A → B → A → B pattern")
	}
}

func TestAnalyzer_IsDiverging(t *testing.T) {
	analyzer := NewAnalyzer(4)

	// Not enough history
	if analyzer.IsDiverging() {
		t.Error("Should not be diverging with empty history")
	}

	// Add non-diverging pattern
	analyzer.AddSnapshot(LoopSnapshot{LoopNumber: 1, Progress: 0.1})
	analyzer.AddSnapshot(LoopSnapshot{LoopNumber: 2, Progress: 0.2})
	analyzer.AddSnapshot(LoopSnapshot{LoopNumber: 3, Progress: 0.3})

	if analyzer.IsDiverging() {
		t.Error("Should not be diverging with increasing progress")
	}

	// Add diverging pattern: progress decreasing
	analyzer2 := NewAnalyzer(4)
	analyzer2.AddSnapshot(LoopSnapshot{LoopNumber: 1, Progress: 0.3})
	analyzer2.AddSnapshot(LoopSnapshot{LoopNumber: 2, Progress: 0.2})
	analyzer2.AddSnapshot(LoopSnapshot{LoopNumber: 3, Progress: 0.1})

	if !analyzer2.IsDiverging() {
		t.Error("Should be diverging with decreasing progress")
	}
}

func TestAnalyzer_IsConverging(t *testing.T) {
	analyzer := NewAnalyzer(4)

	// Not enough history
	if analyzer.IsConverging() {
		t.Error("Should not be converging with empty history")
	}

	// Add converging pattern: progress increasing
	analyzer.AddSnapshot(LoopSnapshot{LoopNumber: 1, Progress: 0.1})
	analyzer.AddSnapshot(LoopSnapshot{LoopNumber: 2, Progress: 0.2})
	analyzer.AddSnapshot(LoopSnapshot{LoopNumber: 3, Progress: 0.3})

	if !analyzer.IsConverging() {
		t.Error("Should be converging with increasing progress")
	}
}

func TestAnalyzer_GetMetrics(t *testing.T) {
	analyzer := NewAnalyzer(4)

	analyzer.AddSnapshot(LoopSnapshot{LoopNumber: 1, State: "planning", Progress: 0.1})
	analyzer.AddSnapshot(LoopSnapshot{LoopNumber: 2, State: "executing", Progress: 0.2})
	analyzer.AddSnapshot(LoopSnapshot{LoopNumber: 3, State: "reflecting", Progress: 0.3})
	analyzer.AddSnapshot(LoopSnapshot{LoopNumber: 4, State: "done", Progress: 0.4})

	metrics := analyzer.GetMetrics()

	if metrics.IsOscillating {
		t.Error("Metrics.IsOscillating should be false")
	}
	if metrics.IsDiverging {
		t.Error("Metrics.IsDiverging should be false")
	}
}

func TestAnalyzer_ClearHistory(t *testing.T) {
	analyzer := NewAnalyzer(4)

	analyzer.AddSnapshot(LoopSnapshot{LoopNumber: 1, Progress: 0.1})
	analyzer.AddSnapshot(LoopSnapshot{LoopNumber: 2, Progress: 0.2})

	analyzer.ClearHistory()

	if len(analyzer.GetHistory()) != 0 {
		t.Errorf("History length after clear = %v, want 0", len(analyzer.GetHistory()))
	}
}
