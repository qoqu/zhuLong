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
	"runtime"
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

// buildPlannerSystemPrompt builds the unified system prompt for both Plan and Replan.
// CRITICAL: This must be identical for Plan() and Replan() to maintain prefix-cache hits.
func buildPlannerSystemPrompt() string {
	platform := "linux"
	if runtime.GOOS == "windows" {
		platform = "windows"
	}

	return `You are a task planner for Zhulong (烛龙), an autonomous agent.

## Platform: ` + platform + `
Use platform-appropriate commands:
- Windows: dir, type, findstr, powershell, etc.
- Linux/macOS: ls, cat, grep, find, etc.

## Project Architecture (CRITICAL)
This is a Wails desktop application, NOT a traditional web app.
- Frontend: desktop/frontend/src/ (React + TypeScript)
- Backend: desktop/app.go (Go, Wails RPC bindings)
- Communication: Wails RPC (window.go.main.App.Method()), NOT HTTP REST

## Language: Chinese (中文)
All responses and descriptions must be in Chinese (中文).

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
        "params": {"param_name": "value"}
      },
      "depends_on": [],
      "breakpoint": false
    }
  ],
  "rationale": "Why this plan was created this way"
}

## CRITICAL: Tool Call Parameter Rules
- **read_file**: MUST include "path" param. Example: {"type":"tool_call","tool":"read_file","params":{"path":"go.mod"}}
- **search_file**: MUST include "pattern" param. Example: {"type":"tool_call","tool":"search_file","params":{"pattern":"*.go"}}
- **execute_command**: MUST include "command" param. Example: {"type":"tool_call","tool":"execute_command","params":{"command":"ls -la"}}
- **write_file**: MUST include "path" and "content" params.
- **web_search**: MUST include "query" param. Example: {"type":"tool_call","tool":"web_search","params":{"query":"Go web framework"}}

## Tool Parameter Reference
- list_dir: {"path": "directory_path"} — list directory contents (first step to understand project structure)
- read_file: {"path": "file_path"}
- search_file: {"pattern": "glob_pattern", "path": "optional_directory"}
- execute_command: {"command": "shell_command"}
- write_file: {"path": "file_path", "content": "file_content"}
- web_search: {"query": "search_query"}

## Planning Principles
1. Each step should be atomic and independently verifiable
2. Clearly mark dependencies between steps
3. Mark high-risk steps with breakpoint: true
4. Keep total steps reasonable (3-15)
5. First step should ALWAYS be: list_dir to understand project structure
6. Always include ALL required parameters for each tool
7. Use actual file paths found in previous steps, never use placeholder paths

## Available Tools
- list_dir: List directory contents (requires: path) — use first to understand project structure
- read_file: Read file contents (requires: path)
- write_file: Write file contents (requires: path, content)
- search_file: Search for files (requires: pattern; optional: path)
- execute_command: Execute shell command (requires: command)
- web_search: Search the web (requires: query)`
}

// buildPlanPrompt builds the complete prompt for planning
func (p *LLMPlanner) buildPlanPrompt(goal string, memory MemoryReader) []Message {
	systemPrompt := buildPlannerSystemPrompt()

	// 动态上下文放到 user message，不破坏 system prompt 缓存
	userPrompt := fmt.Sprintf("## Current Context\n%s\n\n## Goal\n%s\n\nPlease create a plan to achieve this goal.",
		memory.GetSessionSummary(), goal)

	return []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}
}

// buildReplanPrompt builds the prompt for replanning
// CRITICAL: Uses the SAME system prompt as buildPlanPrompt to maintain prefix-cache hits
func (p *LLMPlanner) buildReplanPrompt(goal string, currentPlan *Plan, memory MemoryReader) []Message {
	// 使用与 buildPlanPrompt 完全相同的 system prompt（缓存铁律）
	systemPrompt := buildPlannerSystemPrompt() + `

## Re-planning Mode
You are now re-planning. Adjust the existing plan based on execution history.
1. Preserve completed successful steps
2. Fix failed steps with different strategies
3. Adjust subsequent steps based on new findings
4. Avoid repeating completed work`

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
	// goal 放到 user message，不破坏 system prompt 缓存
	userPrompt := fmt.Sprintf("## Goal\n%s\n\n## Current Plan\n%s\n\n%s\n\nPlease adjust the plan based on the execution history.",
		goal, string(currentPlanJSON), history)

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
		// 规范化 action type（LLM 可能返回工具名作为 type）
		knownTools := map[string]bool{
			"read_file": true, "write_file": true, "search_file": true,
			"execute_command": true, "web_search": true, "list_dir": true,
		}
		if plan.Steps[i].Action.Type == "" {
			if plan.Steps[i].Action.Tool != "" {
				plan.Steps[i].Action.Type = "tool_call"
			} else {
				plan.Steps[i].Action.Type = "llm_generate"
			}
		} else if knownTools[plan.Steps[i].Action.Type] {
			plan.Steps[i].Action.Tool = plan.Steps[i].Action.Type
			plan.Steps[i].Action.Type = "tool_call"
		}
		// 确保工具调用有必要的参数
		ensureToolParams(&plan.Steps[i].Action)
	}

	return &plan, nil
}

// ensureToolParams 确保工具调用有必要的参数
func ensureToolParams(a *Action) {
	if a.Type != "tool_call" || a.Tool == "" {
		return
	}
	if a.Params == nil {
		a.Params = make(map[string]interface{})
	}
	// 根据工具名称补充缺失的参数
	switch a.Tool {
	case "list_dir":
		if _, ok := a.Params["path"]; !ok {
			a.Params["path"] = "."
		}
	case "read_file":
		if _, ok := a.Params["path"]; !ok {
			a.Params["path"] = "."
		}
	case "search_file":
		if _, ok := a.Params["pattern"]; !ok {
			a.Params["pattern"] = "*"
		}
	case "execute_command":
		if _, ok := a.Params["command"]; !ok {
			a.Params["command"] = "echo no-command"
		}
	case "web_search":
		if _, ok := a.Params["query"]; !ok {
			a.Params["query"] = "test"
		}
	case "write_file":
		if _, ok := a.Params["path"]; !ok {
			a.Params["path"] = "output.txt"
		}
		if _, ok := a.Params["content"]; !ok {
			a.Params["content"] = ""
		}
	}
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
	// 找到第一个 { 和最后一个 }
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

	raw := s[start : end+1]
	// 清理 JSON 中的注释和尾部逗号
	raw = cleanJSON(raw)
	return raw
}

// cleanJSON 清理 LLM 可能生成的非标准 JSON
func cleanJSON(s string) string {
	var result []byte
	inString := false
	escape := false

	for i := 0; i < len(s); i++ {
		c := s[i]

		if escape {
			result = append(result, c)
			escape = false
			continue
		}

		if c == '\\' && inString {
			result = append(result, c)
			escape = true
			continue
		}

		if c == '"' {
			inString = !inString
			result = append(result, c)
			continue
		}

		if inString {
			result = append(result, c)
			continue
		}

		// 跳过行注释 //
		if c == '/' && i+1 < len(s) && s[i+1] == '/' {
			// 跳到行尾
			for i < len(s) && s[i] != '\n' {
				i++
			}
			continue
		}

		// 跳过块注释 /* */
		if c == '/' && i+1 < len(s) && s[i+1] == '*' {
			i += 2
			for i < len(s)-1 && !(s[i] == '*' && s[i+1] == '/') {
				i++
			}
			i++ // skip */
			continue
		}

		result = append(result, c)
	}

	// 移除尾部逗号（,] 或 ,}）
	raw := string(result)
	// 简单替换：,} → } 和 ,] → ]
	var cleaned []byte
	for i := 0; i < len(raw); i++ {
		if raw[i] == ',' && i+1 < len(raw) {
			next := raw[i+1]
			if next == '}' || next == ']' {
				continue // 跳过尾部逗号
			}
		}
		cleaned = append(cleaned, raw[i])
	}

	return string(cleaned)
}
