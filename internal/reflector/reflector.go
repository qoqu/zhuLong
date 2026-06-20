package reflector

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// Reflector interface for reflecting on execution results
type Reflector interface {
	// Reflect reflects on the current state and returns a decision
	Reflect(ctx context.Context, goal string, plan *Plan, memory MemoryReader) (*Assessment, error)
}

// MemoryReader provides read access to memory
type MemoryReader interface {
	GetStepResults() []StepResult
	GetSessionSummary() string
}

// StepResult contains the result of a step execution
type StepResult struct {
	StepID  string
	Success bool
	Output  string
}

// Plan represents the current plan
type Plan struct {
	ID    string
	Steps []Step
}

// Step represents a step in the plan
type Step struct {
	ID          string
	Description string
}

// Decision represents the reflection decision
type Decision int

const (
	DecisionComplete Decision = iota
	DecisionContinue
	DecisionReplan
	DecisionFail
)

// String returns the string representation of the decision
func (d Decision) String() string {
	switch d {
	case DecisionComplete:
		return "complete"
	case DecisionContinue:
		return "continue"
	case DecisionReplan:
		return "replan"
	case DecisionFail:
		return "fail"
	default:
		return "unknown"
	}
}

// Assessment represents a reflection assessment
type Assessment struct {
	Decision    Decision  `json:"decision"`
	Reason      string    `json:"reason"`
	Confidence  float64   `json:"confidence"`
	Findings    []string  `json:"findings,omitempty"`
	Suggestions []string  `json:"suggestions,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
}

// LLMProvider interface for calling LLM
type LLMProvider interface {
	Chat(ctx context.Context, messages []Message) (string, error)
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// LLMReflector implements Reflector using LLM
type LLMReflector struct {
	provider LLMProvider
	config   *Config
}

// Config contains reflector configuration
type Config struct {
	Model              string
	Temperature        float64
	ConfidenceThreshold float64
	AutoFailThreshold   float64
}

// DefaultConfig returns default reflector configuration
func DefaultConfig() *Config {
	return &Config{
		Model:              "deepseek-chat",
		Temperature:        0.7,
		ConfidenceThreshold: 0.7,
		AutoFailThreshold:   0.3,
	}
}

// NewLLMReflector creates a new LLM reflector
func NewLLMReflector(provider LLMProvider, config *Config) *LLMReflector {
	if config == nil {
		config = DefaultConfig()
	}

	return &LLMReflector{
		provider: provider,
		config:   config,
	}
}

// Reflect reflects on the current state and returns a decision
func (r *LLMReflector) Reflect(ctx context.Context, goal string, plan *Plan, memory MemoryReader) (*Assessment, error) {
	// Build prompt
	messages := r.buildPrompt(goal, plan, memory)

	// Call LLM
	response, err := r.provider.Chat(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	// Parse response
	assessment, err := r.parseAssessment(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse assessment: %w", err)
	}

	// Set timestamp
	assessment.Timestamp = time.Now()

	return assessment, nil
}

// buildPrompt builds the prompt for reflection
func (r *LLMReflector) buildPrompt(goal string, plan *Plan, memory MemoryReader) []Message {
	systemPrompt := `You are a task reflector. Evaluate the current execution progress and decide the next action.

## Output Format
Return a JSON object with the following structure:
{
  "decision": "complete|continue|replan|fail",
  "reason": "Why this decision was made",
  "confidence": 0.0-1.0,
  "findings": ["Finding 1", "Finding 2"],
  "suggestions": ["Suggestion 1", "Suggestion 2"]
}

## Decision Rules
- complete: Goal is fully achieved, can output final result
- continue: Progress is normal, continue executing next step
- replan: Discovered major issues with the original plan, need to adjust
- fail: Cannot complete the goal (insufficient tools, missing information, unreasonable goal)

## Evaluation Dimensions
1. Goal completion: How far is the current progress from the goal?
2. Plan effectiveness: Is the original plan strategy correct?
3. Resource consumption: Is it within budget?
4. Risk assessment: Are there risks in continuing execution?`

	// Build execution history
	history := "## Execution History\n\n"
	for _, result := range memory.GetStepResults() {
		status := "✓"
		if !result.Success {
			status = "✗"
		}
		history += fmt.Sprintf("- %s %s: %s\n", status, result.StepID, result.Output)
	}

	// Build plan description
	planDesc := "## Current Plan\n\n"
	for _, step := range plan.Steps {
		planDesc += fmt.Sprintf("- %s: %s\n", step.ID, step.Description)
	}

	userPrompt := fmt.Sprintf("## Goal\n%s\n\n%s\n\n%s\n\nPlease evaluate the progress and decide the next action.",
		goal, planDesc, history)

	return []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}
}

// parseAssessment parses the LLM response into an Assessment
func (r *LLMReflector) parseAssessment(response string) (*Assessment, error) {
	// Extract JSON from response
	jsonStr := extractJSON(response)
	if jsonStr == "" {
		return nil, fmt.Errorf("no JSON found in response")
	}

	// Parse JSON
	var raw struct {
		Decision    string   `json:"decision"`
		Reason      string   `json:"reason"`
		Confidence  float64  `json:"confidence"`
		Findings    []string `json:"findings"`
		Suggestions []string `json:"suggestions"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &raw); err != nil {
		return nil, fmt.Errorf("failed to unmarshal assessment: %w", err)
	}

	// Convert decision
	var decision Decision
	switch raw.Decision {
	case "complete":
		decision = DecisionComplete
	case "continue":
		decision = DecisionContinue
	case "replan":
		decision = DecisionReplan
	case "fail":
		decision = DecisionFail
	default:
		return nil, fmt.Errorf("unknown decision: %s", raw.Decision)
	}

	return &Assessment{
		Decision:    decision,
		Reason:      raw.Reason,
		Confidence:  raw.Confidence,
		Findings:    raw.Findings,
		Suggestions: raw.Suggestions,
	}, nil
}

// extractJSON extracts JSON from a string
func extractJSON(s string) string {
	// Find the first { and last }
	start := -1
	end := -1

	for i, c := range s {
		if c == '{' && start == -1 {
			start = i
		}
		if c == '}' {
			end = i
		}
	}

	if start == -1 || end == -1 || start >= end {
		return ""
	}

	return s[start : end+1]
}
