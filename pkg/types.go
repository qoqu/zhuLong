package pkg

import (
	"time"
)

// AgentStatus represents the final status of an agent run
type AgentStatus string

const (
	StatusCompleted AgentStatus = "completed"
	StatusFailed    AgentStatus = "failed"
	StatusCancelled AgentStatus = "cancelled"
)

// AgentResult contains the result of an agent run
type AgentResult struct {
	Status      AgentStatus
	Answer      string
	Loops       int
	TokensUsed  int
	Cost        float64
	Duration    time.Duration
	Trace       []TraceEntry
}

// TraceEntry represents a single trace entry
type TraceEntry struct {
	Timestamp time.Time
	Loop      int
	Phase     string
	Event     string
	Data      interface{}
	Duration  time.Duration
}

// Goal represents the user's goal
type Goal struct {
	Description string
	Constraints []string
}

// Plan represents a plan to achieve the goal
type Plan struct {
	ID          string
	Steps       []Step
	Rationale   string
	CreatedAt   time.Time
}

// Step represents a single step in a plan
type Step struct {
	ID          string
	Description string
	Action      Action
	DependsOn   []string
	Breakpoint  bool
}

// Action represents an action to be executed
type Action struct {
	Type    string                 // "tool_call" | "llm_generate" | "human_input"
	Tool    string                 // Tool name (for tool_call)
	Params  map[string]interface{} // Tool parameters
	Prompt  string                 // LLM prompt (for llm_generate)
}

// StepResult represents the result of executing a step
type StepResult struct {
	StepID      string
	Success     bool
	Output      string
	TokensUsed  int
	Duration    time.Duration
	Error       error
}

// Assessment represents a reflection assessment
type Assessment struct {
	Decision    Decision
	Reason      string
	Confidence  float64
	Findings    []string
	Suggestions []string
}

// Decision represents the reflection decision
type Decision int

const (
	DecisionComplete Decision = iota
	DecisionContinue
	DecisionReplan
	DecisionFail
)

// MemoryReader provides read access to memory
type MemoryReader interface {
	GetWorkingMemory() *WorkingMemory
	GetSessionSummary() string
	GetLongTermMemory() *LongTermMemory
}

// WorkingMemory contains the current loop's memory
type WorkingMemory struct {
	CurrentPlan  *Plan
	StepResults  []StepResult
	Assessments  []Assessment
}

// LongTermMemory contains persistent memory
type LongTermMemory struct {
	Facts    map[string]string
	Patterns map[string]int
}

// Tool represents a tool that can be called
type Tool interface {
	Name() string
	Description() string
	Call(params map[string]interface{}) (string, error)
}
