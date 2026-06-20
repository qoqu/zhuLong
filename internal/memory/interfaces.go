package memory

import (
	"time"
)

// MemoryReader provides read access to memory
type MemoryReader interface {
	GetWorkingMemory() *WorkingMemory
	GetSessionSummary() string
	GetLongTermMemory() *LongTermMemory
	GetStepResults() []StepResult
}

// MemoryWriter provides write access to memory
type MemoryWriter interface {
	AddStepResult(step Step, result StepResult)
	AddAssessment(assessment Assessment)
	CompressWorkingToSession() error
	CompressSessionToLongTerm() error
	UpdateLongTerm(entry LongTermEntry) error
}

// WorkingMemory contains the current loop's memory
type WorkingMemory struct {
	CurrentPlan  *Plan
	StepResults  []StepResult
	Assessments  []Assessment
	TokensUsed   int
}

// SessionMemory contains the session's memory
type SessionMemory struct {
	Goal          string
	LoopSummaries []LoopSummary
	Decisions     []Decision
	Findings      []string
	TokensUsed    int
}

// LongTermMemory contains persistent memory
type LongTermMemory struct {
	Facts    map[string]string
	Patterns map[string]int
	Entries  []LongTermEntry
}

// LongTermEntry represents an entry in long-term memory
type LongTermEntry struct {
	Key       string
	Value     string
	Timestamp time.Time
}

// Plan represents a plan
type Plan struct {
	ID    string
	Steps []Step
}

// Step represents a step
type Step struct {
	ID          string
	Description string
	Action      Action
}

// Action represents an action
type Action struct {
	Type   string
	Tool   string
	Params map[string]interface{}
	Prompt string
}

// StepResult contains the result of a step execution
type StepResult struct {
	StepID     string
	Success    bool
	Output     string
	TokensUsed int
	Duration   time.Duration
}

// Assessment represents a reflection assessment
type Assessment struct {
	Decision   Decision
	Reason     string
	Confidence float64
	Findings   []string
}

// Decision represents the reflection decision
type Decision int

const (
	DecisionComplete Decision = iota
	DecisionContinue
	DecisionReplan
	DecisionFail
)

// LoopSummary contains a summary of a loop
type LoopSummary struct {
	LoopNumber int
	PlanBrief  string
	StepsDone  int
	Assessment string
	TokensUsed int
	Duration   time.Duration
}

// NewWorkingMemory creates a new working memory
func NewWorkingMemory() *WorkingMemory {
	return &WorkingMemory{
		StepResults: make([]StepResult, 0),
		Assessments: make([]Assessment, 0),
	}
}

// NewSessionMemory creates a new session memory
func NewSessionMemory(goal string) *SessionMemory {
	return &SessionMemory{
		Goal:          goal,
		LoopSummaries: make([]LoopSummary, 0),
		Decisions:     make([]Decision, 0),
		Findings:      make([]string, 0),
	}
}

// NewLongTermMemory creates a new long-term memory
func NewLongTermMemory() *LongTermMemory {
	return &LongTermMemory{
		Facts:    make(map[string]string),
		Patterns: make(map[string]int),
		Entries:  make([]LongTermEntry, 0),
	}
}

// AddStepResult adds a step result to working memory
func (wm *WorkingMemory) AddStepResult(step Step, result StepResult) {
	wm.StepResults = append(wm.StepResults, result)
	wm.TokensUsed += result.TokensUsed
}

// AddAssessment adds an assessment to working memory
func (wm *WorkingMemory) AddAssessment(assessment Assessment) {
	wm.Assessments = append(wm.Assessments, assessment)
}

// GetStepResults returns all step results
func (wm *WorkingMemory) GetStepResults() []StepResult {
	return wm.StepResults
}
