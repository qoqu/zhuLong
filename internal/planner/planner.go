// Package planner 提供基于 LLM 的任务规划器
//
// 职责：
//   1. Plan(goal, memory) —— 把用户目标拆成可执行步骤
//   2. Replan(goal, currentPlan, memory) —— 根据执行历史调整计划
//
// 设计要点：
//   - 通过 MemoryReader 只读获取上下文，保证不破坏 prefix-cache（缓存铁律 #1）
//   - 输出严格的 JSON（id/steps/rationale/depends_on/breakpoint）
//   - 步骤数上限 15，单步原子可验证，breakpoint=true 标记需用户确认的危险步骤
//   - validatePlan 防止循环依赖和步骤数超限
package planner

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// Planner interface for generating plans
type Planner interface {
	// Plan generates a plan for the given goal
	Plan(ctx context.Context, goal string, memory MemoryReader) (*Plan, error)

	// Replan generates a new plan based on reflection
	Replan(ctx context.Context, goal string, currentPlan *Plan, memory MemoryReader) (*Plan, error)
}

// MemoryReader provides read access to memory
type MemoryReader interface {
	GetSessionSummary() string
	GetStepResults() []StepResult
}

// StepResult contains the result of a step execution
type StepResult struct {
	StepID  string
	Success bool
	Output  string
}

// Plan represents a plan to achieve the goal
type Plan struct {
	ID          string    `json:"id"`
	Steps       []Step    `json:"steps"`
	Rationale   string    `json:"rationale"`
	CreatedAt   time.Time `json:"created_at"`
}

// Step represents a single step in the plan
type Step struct {
	ID          string   `json:"id"`
	Description string   `json:"description"`
	Action      Action   `json:"action"`
	DependsOn   []string `json:"depends_on,omitempty"`
	Breakpoint  bool     `json:"breakpoint,omitempty"`
}

// Action represents an action to be executed
type Action struct {
	Type   string                 `json:"type"`
	Tool   string                 `json:"tool,omitempty"`
	Params map[string]interface{} `json:"params,omitempty"`
	Prompt string                 `json:"prompt,omitempty"`
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

// LLMPlanner implements Planner using LLM
type LLMPlanner struct {
	provider LLMProvider
	config   *Config
}

// Config contains planner configuration
type Config struct {
	Model       string
	MaxSteps    int
	Temperature float64
}

// DefaultConfig returns default planner configuration
func DefaultConfig() *Config {
	return &Config{
		Model:       "deepseek-chat",
		MaxSteps:    15,
		Temperature: 0.7,
	}
}

// NewLLMPlanner creates a new LLM planner
func NewLLMPlanner(provider LLMProvider, config *Config) *LLMPlanner {
	if config == nil {
		config = DefaultConfig()
	}

	return &LLMPlanner{
		provider: provider,
		config:   config,
	}
}

// Plan generates a plan for the given goal
func (p *LLMPlanner) Plan(ctx context.Context, goal string, memory MemoryReader) (*Plan, error) {
	// Build prompt
	messages := p.buildPlanPrompt(goal, memory)

	// Call LLM
	response, err := p.provider.Chat(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	// Parse response
	plan, err := p.parsePlan(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse plan: %w", err)
	}

	// Validate plan
	if err := p.validatePlan(plan); err != nil {
		return nil, fmt.Errorf("invalid plan: %w", err)
	}

	return plan, nil
}

// Replan generates a new plan based on reflection
func (p *LLMPlanner) Replan(ctx context.Context, goal string, currentPlan *Plan, memory MemoryReader) (*Plan, error) {
	// Build replan prompt
	messages := p.buildReplanPrompt(goal, currentPlan, memory)

	// Call LLM
	response, err := p.provider.Chat(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	// Parse response
	plan, err := p.parsePlan(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse plan: %w", err)
	}

	// Validate plan
	if err := p.validatePlan(plan); err != nil {
		return nil, fmt.Errorf("invalid plan: %w", err)
	}

	return plan, nil
}

// buildPlanPrompt builds the prompt for planning
func (p *LLMPlanner) buildPlanPrompt(goal string, memory MemoryReader) []Message {
	systemPrompt := `You are a task planner. Your job is to break down a user's goal into clear, executable steps.

## Output Format
Return a JSON object with the following structure:
{
  "id": "plan-1",
  "steps": [
    {
      "id": "step-1",
      "description": "Step description",
      "action": {
        "type": "tool_call",
        "tool": "tool_name",
        "params": {}
      },
      "depends_on": [],
      "breakpoint": false
    }
  ],
  "rationale": "Why this plan was created this way"
}

## Planning Principles
1. Each step should be atomic and independently verifiable
2. Clearly mark dependencies between steps
3. Mark high-risk steps with breakpoint: true
4. Keep total steps reasonable (3-15)
5. First step should usually be "gather information/understand context"

## Available Tools
- read_file: Read file contents
- write_file: Write file contents
- search_file: Search for files
- execute_command: Execute shell command
- web_search: Search the web

## Current Context
` + memory.GetSessionSummary()

	userPrompt := fmt.Sprintf("Goal: %s\n\nPlease create a plan to achieve this goal.", goal)

	return []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}
}

// buildReplanPrompt builds the prompt for replanning
func (p *LLMPlanner) buildReplanPrompt(goal string, currentPlan *Plan, memory MemoryReader) []Message {
	systemPrompt := `You are a task re-planner. Based on execution history and reflection, adjust the current plan.

## Output Format
Return a JSON object with the same structure as the original plan.

## Re-planning Principles
1. Preserve completed successful steps
2. Fix failed steps with different strategies
3. Adjust subsequent steps based on new findings
4. Avoid repeating completed work

## Current Goal
` + goal

	// Build execution history
	history := "## Execution History\n\n"
	for _, result := range memory.GetStepResults() {
		status := "✓"
		if !result.Success {
			status = "✗"
		}
		history += fmt.Sprintf("- %s %s: %s\n", status, result.StepID, result.Output)
	}

	currentPlanJSON, _ := json.MarshalIndent(currentPlan, "", "  ")
	userPrompt := fmt.Sprintf("## Current Plan\n%s\n\n%s\n\nPlease adjust the plan based on the execution history.", string(currentPlanJSON), history)

	return []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}
}

// parsePlan parses the LLM response into a Plan
func (p *LLMPlanner) parsePlan(response string) (*Plan, error) {
	// Try to extract JSON from response
	jsonStr := extractJSON(response)
	if jsonStr == "" {
		return nil, fmt.Errorf("no JSON found in response")
	}

	var plan Plan
	if err := json.Unmarshal([]byte(jsonStr), &plan); err != nil {
		return nil, fmt.Errorf("failed to unmarshal plan: %w", err)
	}

	// Set defaults
	if plan.ID == "" {
		plan.ID = fmt.Sprintf("plan-%d", time.Now().Unix())
	}
	plan.CreatedAt = time.Now()

	// Set step IDs if empty
	for i := range plan.Steps {
		if plan.Steps[i].ID == "" {
			plan.Steps[i].ID = fmt.Sprintf("step-%d", i+1)
		}
	}

	return &plan, nil
}

// validatePlan validates the plan
func (p *LLMPlanner) validatePlan(plan *Plan) error {
	if len(plan.Steps) == 0 {
		return fmt.Errorf("plan has no steps")
	}

	if len(plan.Steps) > p.config.MaxSteps {
		return fmt.Errorf("plan has too many steps: %d (max: %d)", len(plan.Steps), p.config.MaxSteps)
	}

	// Check for circular dependencies
	stepIDs := make(map[string]bool)
	for _, step := range plan.Steps {
		stepIDs[step.ID] = true
	}

	for _, step := range plan.Steps {
		for _, dep := range step.DependsOn {
			if !stepIDs[dep] {
				return fmt.Errorf("step %s depends on non-existent step %s", step.ID, dep)
			}
		}
	}

	return nil
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
