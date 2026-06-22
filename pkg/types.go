// Package pkg 提供 Agent 主循环及对外接口
//
// ⚠️ 关于 Tool 接口: 本包 types.go 定义的 Tool 接口（无 ctx 参数）
// 仅用于类型占位/向后兼容。实际工具实现必须实现 executor.Tool
// 接口（含 ctx context.Context 参数），由 agent.go 中的 adapter 包装。
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
// 关键修复: 之前这只是个重复占位 interface，实际每个模块有自己的 MemoryReader
// （executor.MemoryReader, planner.MemoryReader, reflector.MemoryReader）
// 这里仅保留通用字段描述，供 agent.go 中的 PlannerMemoryReader/ExecutorMemoryReader/
// ReflectorMemoryReader 适配器实现
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

// Tool 占位接口
// 关键修复: 之前这里的 Tool 接口与 executor.Tool 不一致（少 ctx 参数），
// 任何直接实现 pkg.Tool 的类型都无法注册到 executor.ToolRegistry
// 现在: 标记为 Deprecated，实际实现请用 executor.Tool（含 ctx）
//
// 保留目的: 兼容历史调用方 + 类型占位
type Tool interface {
	Name() string
	Description() string
	Call(params map[string]interface{}) (string, error)
}
