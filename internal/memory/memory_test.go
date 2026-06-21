package memory

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewWorkingMemory(t *testing.T) {
	wm := NewWorkingMemory()

	if wm == nil {
		t.Error("NewWorkingMemory() should not return nil")
	}
	if len(wm.StepResults) != 0 {
		t.Errorf("NewWorkingMemory().StepResults length = %v, want 0", len(wm.StepResults))
	}
	if len(wm.Assessments) != 0 {
		t.Errorf("NewWorkingMemory().Assessments length = %v, want 0", len(wm.Assessments))
	}
	if wm.TokensUsed != 0 {
		t.Errorf("NewWorkingMemory().TokensUsed = %v, want 0", wm.TokensUsed)
	}
}

func TestNewSessionMemory(t *testing.T) {
	goal := "Test goal"
	sm := NewSessionMemory(goal)

	if sm == nil {
		t.Error("NewSessionMemory() should not return nil")
	}
	if sm.Goal != goal {
		t.Errorf("NewSessionMemory().Goal = %v, want %v", sm.Goal, goal)
	}
	if len(sm.LoopSummaries) != 0 {
		t.Errorf("NewSessionMemory().LoopSummaries length = %v, want 0", len(sm.LoopSummaries))
	}
}

func TestNewLongTermMemory(t *testing.T) {
	ltm := NewLongTermMemory()

	if ltm == nil {
		t.Error("NewLongTermMemory() should not return nil")
	}
	if len(ltm.Facts) != 0 {
		t.Errorf("NewLongTermMemory().Facts length = %v, want 0", len(ltm.Facts))
	}
	if len(ltm.Patterns) != 0 {
		t.Errorf("NewLongTermMemory().Patterns length = %v, want 0", len(ltm.Patterns))
	}
}

func TestWorkingMemory_AddStepResult(t *testing.T) {
	wm := NewWorkingMemory()

	step := Step{ID: "step-1", Description: "Test step"}
	result := StepResult{
		StepID:     "step-1",
		Success:    true,
		Output:     "Test output",
		TokensUsed: 100,
	}

	wm.AddStepResult(step, result)

	if len(wm.StepResults) != 1 {
		t.Errorf("StepResults length = %v, want 1", len(wm.StepResults))
	}
	if wm.TokensUsed != 100 {
		t.Errorf("TokensUsed = %v, want 100", wm.TokensUsed)
	}
}

func TestWorkingMemory_AddAssessment(t *testing.T) {
	wm := NewWorkingMemory()

	assessment := Assessment{
		Decision:   DecisionContinue,
		Reason:     "Progress is good",
		Confidence: 0.8,
	}

	wm.AddAssessment(assessment)

	if len(wm.Assessments) != 1 {
		t.Errorf("Assessments length = %v, want 1", len(wm.Assessments))
	}
}

func TestFileStore_SaveLoadWorkingMemory(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStore(dir)

	wm := NewWorkingMemory()
	wm.AddStepResult(Step{ID: "step-1"}, StepResult{StepID: "step-1", Success: true, TokensUsed: 50})

	err := store.SaveWorkingMemory("session-1", wm)
	if err != nil {
		t.Fatalf("SaveWorkingMemory() error = %v", err)
	}

	loaded, err := store.LoadWorkingMemory("session-1")
	if err != nil {
		t.Fatalf("LoadWorkingMemory() error = %v", err)
	}

	if len(loaded.StepResults) != 1 {
		t.Errorf("Loaded StepResults length = %v, want 1", len(loaded.StepResults))
	}
	if loaded.TokensUsed != 50 {
		t.Errorf("Loaded TokensUsed = %v, want 50", loaded.TokensUsed)
	}
}

func TestFileStore_SaveLoadSessionMemory(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStore(dir)

	sm := NewSessionMemory("Test goal")
	sm.LoopSummaries = append(sm.LoopSummaries, LoopSummary{
		LoopNumber: 1,
		StepsDone:  3,
		TokensUsed: 100,
	})

	err := store.SaveSessionMemory("session-1", sm)
	if err != nil {
		t.Fatalf("SaveSessionMemory() error = %v", err)
	}

	loaded, err := store.LoadSessionMemory("session-1")
	if err != nil {
		t.Fatalf("LoadSessionMemory() error = %v", err)
	}

	if loaded.Goal != "Test goal" {
		t.Errorf("Loaded Goal = %v, want 'Test goal'", loaded.Goal)
	}
	if len(loaded.LoopSummaries) != 1 {
		t.Errorf("Loaded LoopSummaries length = %v, want 1", len(loaded.LoopSummaries))
	}
}

func TestFileStore_SaveLoadLongTermMemory(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStore(dir)

	ltm := NewLongTermMemory()
	ltm.Facts["key1"] = "value1"
	ltm.Patterns["pattern1"] = 5

	err := store.SaveLongTermMemory(ltm)
	if err != nil {
		t.Fatalf("SaveLongTermMemory() error = %v", err)
	}

	loaded, err := store.LoadLongTermMemory()
	if err != nil {
		t.Fatalf("LoadLongTermMemory() error = %v", err)
	}

	if loaded.Facts["key1"] != "value1" {
		t.Errorf("Loaded Facts[key1] = %v, want 'value1'", loaded.Facts["key1"])
	}
	if loaded.Patterns["pattern1"] != 5 {
		t.Errorf("Loaded Patterns[pattern1] = %v, want 5", loaded.Patterns["pattern1"])
	}
}

func TestFileStore_LoadNonExistent(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStore(dir)

	// Load non-existent working memory
	wm, err := store.LoadWorkingMemory("nonexistent")
	if err != nil {
		t.Fatalf("LoadWorkingMemory() error = %v", err)
	}
	if wm == nil {
		t.Error("LoadWorkingMemory() should return empty memory, not nil")
	}

	// Load non-existent session memory
	sm, err := store.LoadSessionMemory("nonexistent")
	if err != nil {
		t.Fatalf("LoadSessionMemory() error = %v", err)
	}
	if sm == nil {
		t.Error("LoadSessionMemory() should return empty memory, not nil")
	}

	// Load non-existent long-term memory
	ltm, err := store.LoadLongTermMemory()
	if err != nil {
		t.Fatalf("LoadLongTermMemory() error = %v", err)
	}
	if ltm == nil {
		t.Error("LoadLongTermMemory() should return empty memory, not nil")
	}
}

func TestMemoryManager(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStore(dir)

	goal := "Test goal"
	mm := NewMemoryManager(store, goal)

	// Test GetWorkingMemory
	wm := mm.GetWorkingMemory()
	if wm == nil {
		t.Error("GetWorkingMemory() should not return nil")
	}

	// Test GetSessionSummary
	summary := mm.GetSessionSummary()
	if summary == "" {
		t.Error("GetSessionSummary() should not be empty")
	}

	// Test AddStepResult
	step := Step{ID: "step-1"}
	result := StepResult{StepID: "step-1", Success: true, TokensUsed: 100}
	mm.AddStepResult(step, result)

	if len(mm.GetStepResults()) != 1 {
		t.Errorf("GetStepResults() length = %v, want 1", len(mm.GetStepResults()))
	}

	// Test AddAssessment
	assessment := Assessment{
		Decision:   DecisionContinue,
		Reason:     "Good progress",
		Confidence: 0.8,
		Findings:   []string{"Finding 1"},
	}
	mm.AddAssessment(assessment)

	// Test CompressWorkingToSession
	err := mm.CompressWorkingToSession()
	if err != nil {
		t.Fatalf("CompressWorkingToSession() error = %v", err)
	}

	// Working memory should be reset
	if len(mm.GetWorkingMemory().StepResults) != 0 {
		t.Error("Working memory should be reset after compression")
	}

	// Test Save and Load
	err = mm.Save("session-1")
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	mm2 := NewMemoryManager(store, goal)
	err = mm2.Load("session-1")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
}

func TestFileStore_CreateDirectory(t *testing.T) {
	dir := t.TempDir()
	nestedDir := filepath.Join(dir, "nested", "path")
	store := NewFileStore(nestedDir)

	wm := NewWorkingMemory()
	err := store.SaveWorkingMemory("session-1", wm)
	if err != nil {
		t.Fatalf("SaveWorkingMemory() should create directories, error = %v", err)
	}

	// Verify directory was created
	if _, err := os.Stat(nestedDir); os.IsNotExist(err) {
		t.Error("Directory should have been created")
	}
}
