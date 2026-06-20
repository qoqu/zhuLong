package checkpoint

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Checkpoint contains the state of a checkpoint
type Checkpoint struct {
	ID          string    `json:"id"`
	SessionID   string    `json:"session_id"`
	State       string    `json:"state"`
	Goal        string    `json:"goal"`
	Plan        *Plan     `json:"plan,omitempty"`
	CurrentStep int       `json:"current_step"`
	LoopCount   int       `json:"loop_count"`
	TokensUsed  int       `json:"tokens_used"`
	Cost        float64   `json:"cost"`
	CreatedAt   time.Time `json:"created_at"`
}

// Plan contains the plan at checkpoint time
type Plan struct {
	ID    string `json:"id"`
	Steps []Step `json:"steps"`
}

// Step represents a step in the plan
type Step struct {
	ID          string `json:"id"`
	Description string `json:"description"`
}

// Store interface for persisting checkpoints
type Store interface {
	Save(cp *Checkpoint) error
	LoadLatest() (*Checkpoint, error)
	LoadByID(id string) (*Checkpoint, error)
	List(sessionID string) ([]Checkpoint, error)
	Delete(id string) error
}

// FileStore implements Store using files
type FileStore struct {
	dir string
}

// NewFileStore creates a new file store
func NewFileStore(dir string) *FileStore {
	return &FileStore{
		dir: dir,
	}
}

// Save saves a checkpoint to a file
func (fs *FileStore) Save(cp *Checkpoint) error {
	path := filepath.Join(fs.dir, cp.SessionID, fmt.Sprintf("%s.json", cp.ID))
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	data, err := json.MarshalIndent(cp, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal checkpoint: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// LoadLatest loads the latest checkpoint
func (fs *FileStore) LoadLatest() (*Checkpoint, error) {
	// Find the latest session directory
	sessions, err := os.ReadDir(fs.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var latestCheckpoint *Checkpoint
	var latestTime time.Time

	for _, session := range sessions {
		if !session.IsDir() {
			continue
		}

		checkpoints, err := fs.List(session.Name())
		if err != nil {
			continue
		}

		for _, cp := range checkpoints {
			if latestCheckpoint == nil || cp.CreatedAt.After(latestTime) {
				latestCheckpoint = &cp
				latestTime = cp.CreatedAt
			}
		}
	}

	return latestCheckpoint, nil
}

// LoadByID loads a checkpoint by ID
func (fs *FileStore) LoadByID(id string) (*Checkpoint, error) {
	// Search in all sessions
	sessions, err := os.ReadDir(fs.dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	for _, session := range sessions {
		if !session.IsDir() {
			continue
		}

		path := filepath.Join(fs.dir, session.Name(), fmt.Sprintf("%s.json", id))
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		var cp Checkpoint
		if err := json.Unmarshal(data, &cp); err != nil {
			continue
		}

		return &cp, nil
	}

	return nil, fmt.Errorf("checkpoint not found: %s", id)
}

// List lists all checkpoints for a session
func (fs *FileStore) List(sessionID string) ([]Checkpoint, error) {
	dir := filepath.Join(fs.dir, sessionID)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var checkpoints []Checkpoint
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		var cp Checkpoint
		if err := json.Unmarshal(data, &cp); err != nil {
			continue
		}

		checkpoints = append(checkpoints, cp)
	}

	return checkpoints, nil
}

// Delete deletes a checkpoint
func (fs *FileStore) Delete(id string) error {
	// Search in all sessions
	sessions, err := os.ReadDir(fs.dir)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	for _, session := range sessions {
		if !session.IsDir() {
			continue
		}

		path := filepath.Join(fs.dir, session.Name(), fmt.Sprintf("%s.json", id))
		if _, err := os.Stat(path); err == nil {
			return os.Remove(path)
		}
	}

	return fmt.Errorf("checkpoint not found: %s", id)
}

// CheckpointManager manages checkpoints
type CheckpointManager struct {
	store         Store
	checkpointEvery int
	counter       int
}

// NewCheckpointManager creates a new checkpoint manager
func NewCheckpointManager(store Store, checkpointEvery int) *CheckpointManager {
	return &CheckpointManager{
		store:           store,
		checkpointEvery: checkpointEvery,
		counter:         0,
	}
}

// ShouldCheckpoint returns true if a checkpoint should be saved
func (cm *CheckpointManager) ShouldCheckpoint() bool {
	cm.counter++
	return cm.counter%cm.checkpointEvery == 0
}

// SaveCheckpoint saves a checkpoint
func (cm *CheckpointManager) SaveCheckpoint(sessionID string, state string, goal string, plan *Plan, currentStep int, loopCount int, tokensUsed int, cost float64) error {
	cp := &Checkpoint{
		ID:          fmt.Sprintf("cp-%d", time.Now().Unix()),
		SessionID:   sessionID,
		State:       state,
		Goal:        goal,
		Plan:        plan,
		CurrentStep: currentStep,
		LoopCount:   loopCount,
		TokensUsed:  tokensUsed,
		Cost:        cost,
		CreatedAt:   time.Now(),
	}

	return cm.store.Save(cp)
}

// LoadLatestCheckpoint loads the latest checkpoint
func (cm *CheckpointManager) LoadLatestCheckpoint() (*Checkpoint, error) {
	return cm.store.LoadLatest()
}
