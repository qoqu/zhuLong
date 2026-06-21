package checkpoint

import (
	"fmt"
	"testing"
	"time"
)

func TestNewFileStore(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStore(dir)
	if store == nil {
		t.Fatal("NewFileStore returned nil")
	}
	if store.dir != dir {
		t.Errorf("expected dir %s, got %s", dir, store.dir)
	}
}

func TestFileStoreSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStore(dir)

	cp := &Checkpoint{
		ID:          "test-checkpoint",
		SessionID:   "test-session",
		State:       "executing",
		Goal:        "test goal",
		CurrentStep: 3,
		LoopCount:   5,
		TokensUsed:  1000,
		Cost:        0.5,
		CreatedAt:   time.Now(),
	}

	// Save checkpoint
	if err := store.Save(cp); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Load by ID
	loaded, err := store.LoadByID("test-checkpoint")
	if err != nil {
		t.Fatalf("LoadByID failed: %v", err)
	}

	if loaded.ID != cp.ID {
		t.Errorf("expected ID %s, got %s", cp.ID, loaded.ID)
	}
	if loaded.SessionID != cp.SessionID {
		t.Errorf("expected SessionID %s, got %s", cp.SessionID, loaded.SessionID)
	}
	if loaded.State != cp.State {
		t.Errorf("expected State %s, got %s", cp.State, loaded.State)
	}
	if loaded.Goal != cp.Goal {
		t.Errorf("expected Goal %s, got %s", cp.Goal, loaded.Goal)
	}
	if loaded.CurrentStep != cp.CurrentStep {
		t.Errorf("expected CurrentStep %d, got %d", cp.CurrentStep, loaded.CurrentStep)
	}
	if loaded.LoopCount != cp.LoopCount {
		t.Errorf("expected LoopCount %d, got %d", cp.LoopCount, loaded.LoopCount)
	}
	if loaded.TokensUsed != cp.TokensUsed {
		t.Errorf("expected TokensUsed %d, got %d", cp.TokensUsed, loaded.TokensUsed)
	}
}

func TestFileStoreLoadLatest(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStore(dir)

	// Save multiple checkpoints
	for i := 0; i < 3; i++ {
		cp := &Checkpoint{
			ID:        fmt.Sprintf("checkpoint-%d", i),
			SessionID: "test-session",
			State:     "executing",
			Goal:      "test goal",
			CreatedAt: time.Now().Add(time.Duration(i) * time.Minute),
		}
		if err := store.Save(cp); err != nil {
			t.Fatalf("Save failed: %v", err)
		}
	}

	// Load latest
	latest, err := store.LoadLatest()
	if err != nil {
		t.Fatalf("LoadLatest failed: %v", err)
	}

	if latest.ID != "checkpoint-2" {
		t.Errorf("expected ID checkpoint-2, got %s", latest.ID)
	}
}

func TestFileStoreList(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStore(dir)

	// Save multiple checkpoints
	for i := 0; i < 3; i++ {
		cp := &Checkpoint{
			ID:        fmt.Sprintf("checkpoint-%d", i),
			SessionID: "test-session",
			State:     "executing",
			Goal:      "test goal",
			CreatedAt: time.Now(),
		}
		if err := store.Save(cp); err != nil {
			t.Fatalf("Save failed: %v", err)
		}
	}

	// List checkpoints
	checkpoints, err := store.List("test-session")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(checkpoints) != 3 {
		t.Errorf("expected 3 checkpoints, got %d", len(checkpoints))
	}
}

func TestFileStoreDelete(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStore(dir)

	cp := &Checkpoint{
		ID:        "test-checkpoint",
		SessionID: "test-session",
		State:     "executing",
		Goal:      "test goal",
		CreatedAt: time.Now(),
	}

	// Save checkpoint
	if err := store.Save(cp); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Delete checkpoint
	if err := store.Delete("test-checkpoint"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Try to load deleted checkpoint
	_, err := store.LoadByID("test-checkpoint")
	if err == nil {
		t.Error("expected error loading deleted checkpoint")
	}
}

func TestCheckpointManager(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStore(dir)
	manager := NewCheckpointManager(store, 3)

	// Test ShouldCheckpoint
	for i := 0; i < 10; i++ {
		expected := (i+1)%3 == 0
		actual := manager.ShouldCheckpoint()
		if actual != expected {
			t.Errorf("iteration %d: expected ShouldCheckpoint=%v, got %v", i, expected, actual)
		}
	}

	// Test SaveCheckpoint
	if err := manager.SaveCheckpoint("test-session", "executing", "test goal", nil, 5, 10, 1000, 0.5); err != nil {
		t.Fatalf("SaveCheckpoint failed: %v", err)
	}

	// Load and verify
	latest, err := manager.LoadLatestCheckpoint()
	if err != nil {
		t.Fatalf("LoadLatestCheckpoint failed: %v", err)
	}

	if latest.SessionID != "test-session" {
		t.Errorf("expected SessionID test-session, got %s", latest.SessionID)
	}
	if latest.State != "executing" {
		t.Errorf("expected State executing, got %s", latest.State)
	}
}

func TestFileStoreLoadByIDNotFound(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStore(dir)

	_, err := store.LoadByID("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent checkpoint")
	}
}

func TestFileStoreDeleteNotFound(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStore(dir)

	err := store.Delete("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent checkpoint")
	}
}
