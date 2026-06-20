package memory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Store interface for persisting memory
type Store interface {
	SaveWorkingMemory(sessionID string, wm *WorkingMemory) error
	LoadWorkingMemory(sessionID string) (*WorkingMemory, error)
	SaveSessionMemory(sessionID string, sm *SessionMemory) error
	LoadSessionMemory(sessionID string) (*SessionMemory, error)
	SaveLongTermMemory(ltm *LongTermMemory) error
	LoadLongTermMemory() (*LongTermMemory, error)
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

// SaveWorkingMemory saves working memory to a file
func (fs *FileStore) SaveWorkingMemory(sessionID string, wm *WorkingMemory) error {
	path := filepath.Join(fs.dir, sessionID, "working_memory.json")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	data, err := json.MarshalIndent(wm, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal working memory: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// LoadWorkingMemory loads working memory from a file
func (fs *FileStore) LoadWorkingMemory(sessionID string) (*WorkingMemory, error) {
	path := filepath.Join(fs.dir, sessionID, "working_memory.json")

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return NewWorkingMemory(), nil
		}
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var wm WorkingMemory
	if err := json.Unmarshal(data, &wm); err != nil {
		return nil, fmt.Errorf("failed to unmarshal working memory: %w", err)
	}

	return &wm, nil
}

// SaveSessionMemory saves session memory to a file
func (fs *FileStore) SaveSessionMemory(sessionID string, sm *SessionMemory) error {
	path := filepath.Join(fs.dir, sessionID, "session_memory.json")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	data, err := json.MarshalIndent(sm, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal session memory: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// LoadSessionMemory loads session memory from a file
func (fs *FileStore) LoadSessionMemory(sessionID string) (*SessionMemory, error) {
	path := filepath.Join(fs.dir, sessionID, "session_memory.json")

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return NewSessionMemory(""), nil
		}
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var sm SessionMemory
	if err := json.Unmarshal(data, &sm); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session memory: %w", err)
	}

	return &sm, nil
}

// SaveLongTermMemory saves long-term memory to a file
func (fs *FileStore) SaveLongTermMemory(ltm *LongTermMemory) error {
	path := filepath.Join(fs.dir, "long_term_memory.json")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	data, err := json.MarshalIndent(ltm, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal long-term memory: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// LoadLongTermMemory loads long-term memory from a file
func (fs *FileStore) LoadLongTermMemory() (*LongTermMemory, error) {
	path := filepath.Join(fs.dir, "long_term_memory.json")

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return NewLongTermMemory(), nil
		}
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var ltm LongTermMemory
	if err := json.Unmarshal(data, &ltm); err != nil {
		return nil, fmt.Errorf("failed to unmarshal long-term memory: %w", err)
	}

	return &ltm, nil
}

// MemoryManager manages all three layers of memory
type MemoryManager struct {
	store         Store
	workingMemory *WorkingMemory
	sessionMemory *SessionMemory
	longTermMemory *LongTermMemory
}

// NewMemoryManager creates a new memory manager
func NewMemoryManager(store Store, goal string) *MemoryManager {
	return &MemoryManager{
		store:          store,
		workingMemory:  NewWorkingMemory(),
		sessionMemory:  NewSessionMemory(goal),
		longTermMemory: NewLongTermMemory(),
	}
}

// GetWorkingMemory returns the working memory
func (mm *MemoryManager) GetWorkingMemory() *WorkingMemory {
	return mm.workingMemory
}

// GetSessionSummary returns a summary of the session
func (mm *MemoryManager) GetSessionSummary() string {
	summary := fmt.Sprintf("Goal: %s\n", mm.sessionMemory.Goal)
	summary += fmt.Sprintf("Loops completed: %d\n", len(mm.sessionMemory.LoopSummaries))

	if len(mm.sessionMemory.Findings) > 0 {
		summary += "\nKey findings:\n"
		for _, finding := range mm.sessionMemory.Findings {
			summary += fmt.Sprintf("- %s\n", finding)
		}
	}

	return summary
}

// GetLongTermMemory returns the long-term memory
func (mm *MemoryManager) GetLongTermMemory() *LongTermMemory {
	return mm.longTermMemory
}

// GetStepResults returns all step results
func (mm *MemoryManager) GetStepResults() []StepResult {
	return mm.workingMemory.GetStepResults()
}

// AddStepResult adds a step result to working memory
func (mm *MemoryManager) AddStepResult(step Step, result StepResult) {
	mm.workingMemory.AddStepResult(step, result)
}

// AddAssessment adds an assessment to working memory
func (mm *MemoryManager) AddAssessment(assessment Assessment) {
	mm.workingMemory.AddAssessment(assessment)
}

// CompressWorkingToSession compresses working memory to session memory
func (mm *MemoryManager) CompressWorkingToSession() error {
	// Create a loop summary
	summary := LoopSummary{
		LoopNumber: len(mm.sessionMemory.LoopSummaries) + 1,
		StepsDone:  len(mm.workingMemory.StepResults),
		TokensUsed: mm.workingMemory.TokensUsed,
		Duration:   time.Duration(0), // TODO: calculate duration
	}

	// Get the last assessment if available
	if len(mm.workingMemory.Assessments) > 0 {
		lastAssessment := mm.workingMemory.Assessments[len(mm.workingMemory.Assessments)-1]
		summary.Assessment = lastAssessment.Reason

		// Add findings to session memory
		mm.sessionMemory.Findings = append(mm.sessionMemory.Findings, lastAssessment.Findings...)
	}

	mm.sessionMemory.LoopSummaries = append(mm.sessionMemory.LoopSummaries, summary)
	mm.sessionMemory.TokensUsed += mm.workingMemory.TokensUsed

	// Reset working memory
	mm.workingMemory = NewWorkingMemory()

	return nil
}

// CompressSessionToLongTerm compresses session memory to long-term memory
func (mm *MemoryManager) CompressSessionToLongTerm() error {
	// Create a long-term entry
	entry := LongTermEntry{
		Key:       fmt.Sprintf("session-%d", time.Now().Unix()),
		Value:     mm.GetSessionSummary(),
		Timestamp: time.Now(),
	}

	mm.longTermMemory.Entries = append(mm.longTermMemory.Entries, entry)

	return nil
}

// UpdateLongTerm updates long-term memory
func (mm *MemoryManager) UpdateLongTerm(entry LongTermEntry) error {
	mm.longTermMemory.Entries = append(mm.longTermMemory.Entries, entry)
	return nil
}

// Save saves all memory to the store
func (mm *MemoryManager) Save(sessionID string) error {
	if err := mm.store.SaveWorkingMemory(sessionID, mm.workingMemory); err != nil {
		return fmt.Errorf("failed to save working memory: %w", err)
	}

	if err := mm.store.SaveSessionMemory(sessionID, mm.sessionMemory); err != nil {
		return fmt.Errorf("failed to save session memory: %w", err)
	}

	if err := mm.store.SaveLongTermMemory(mm.longTermMemory); err != nil {
		return fmt.Errorf("failed to save long-term memory: %w", err)
	}

	return nil
}

// Load loads all memory from the store
func (mm *MemoryManager) Load(sessionID string) error {
	wm, err := mm.store.LoadWorkingMemory(sessionID)
	if err != nil {
		return fmt.Errorf("failed to load working memory: %w", err)
	}
	mm.workingMemory = wm

	sm, err := mm.store.LoadSessionMemory(sessionID)
	if err != nil {
		return fmt.Errorf("failed to load session memory: %w", err)
	}
	mm.sessionMemory = sm

	ltm, err := mm.store.LoadLongTermMemory()
	if err != nil {
		return fmt.Errorf("failed to load long-term memory: %w", err)
	}
	mm.longTermMemory = ltm

	return nil
}
