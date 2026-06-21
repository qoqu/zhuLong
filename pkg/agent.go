package pkg

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/qoqu/zhuLong/internal/checkpoint"
	"github.com/qoqu/zhuLong/internal/executor"
	"github.com/qoqu/zhuLong/internal/memory"
	"github.com/qoqu/zhuLong/internal/planner"
	"github.com/qoqu/zhuLong/internal/provider"
	"github.com/qoqu/zhuLong/internal/reflector"
	"github.com/qoqu/zhuLong/internal/tools"
	"github.com/qoqu/zhuLong/internal/trace"
)

// Agent is the main entry point for the Zhulong agent
type Agent struct {
	goal     string
	options  *Options
	provider Provider
}

// Provider is the interface for LLM providers
type Provider interface {
	Chat(ctx context.Context, system, user string) (string, error)
}

// Options contains configuration for the agent
type Options struct {
	Goal        string
	MaxLoops    int
	MaxTokens   int
	MaxCost     float64
	MaxWallTime time.Duration
	Verbose     bool
	Model       string
	DataDir     string
}

// Option is a function that configures the agent
type Option func(*Options)

// WithGoal sets the goal for the agent
func WithGoal(goal string) Option {
	return func(o *Options) {
		o.Goal = goal
	}
}

// WithMaxLoops sets the maximum number of loops
func WithMaxLoops(n int) Option {
	return func(o *Options) {
		o.MaxLoops = n
	}
}

// WithMaxTokens sets the maximum number of tokens
func WithMaxTokens(n int) Option {
	return func(o *Options) {
		o.MaxTokens = n
	}
}

// WithMaxCost sets the maximum cost
func WithMaxCost(cost float64) Option {
	return func(o *Options) {
		o.MaxCost = cost
	}
}

// WithMaxWallTime sets the maximum wall time
func WithMaxWallTime(d time.Duration) Option {
	return func(o *Options) {
		o.MaxWallTime = d
	}
}

// WithVerbose enables verbose output
func WithVerbose(verbose bool) Option {
	return func(o *Options) {
		o.Verbose = verbose
	}
}

// WithModel sets the model for the agent
func WithModel(model string) Option {
	return func(o *Options) {
		o.Model = model
	}
}

// WithDataDir sets the data directory for the agent
func WithDataDir(dir string) Option {
	return func(o *Options) {
		o.DataDir = dir
	}
}

// NewAgent creates a new agent
func NewAgent(opts ...Option) (*Agent, error) {
	options := &Options{
		MaxLoops:    50,
		MaxTokens:   500000,
		MaxCost:     10.0,
		MaxWallTime: 30 * time.Minute,
		Verbose:     false,
		Model:       "deepseek-chat",
		DataDir:     "",
	}

	for _, opt := range opts {
		opt(options)
	}

	// Set default data directory
	if options.DataDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			options.DataDir = ".zhulong"
		} else {
			options.DataDir = filepath.Join(home, ".zhulong")
		}
	}

	// Create data directory if it doesn't exist
	os.MkdirAll(options.DataDir, 0755)

	return &Agent{
		goal:    options.Goal,
		options: options,
	}, nil
}

// SetGoal sets the goal for the agent
func (a *Agent) SetGoal(goal string) {
	a.goal = goal
}

// SetProvider sets the provider for the agent
func (a *Agent) SetProvider(p Provider) {
	a.provider = p
}

// Run runs the agent
func (a *Agent) Run() (*AgentResult, error) {
	if a.goal == "" {
		return nil, fmt.Errorf("goal is not set")
	}

	// Use default provider if none set
	if a.provider == nil {
		a.provider = NewHeuristicProvider(a.options.Model)
	}

	ctx, cancel := context.WithTimeout(context.Background(), a.options.MaxWallTime)
	defer cancel()

	startTime := time.Now()

	// Initialize infrastructure
	dataDir := a.options.DataDir
	memStore := memory.NewFileStore(filepath.Join(dataDir, "memory"))
	chkStore := checkpoint.NewFileStore(filepath.Join(dataDir, "checkpoint"))
	logger := trace.NewLogger("agent", &trace.Config{
		Enabled:   true,
		OutputDir: filepath.Join(dataDir, "traces"),
		Format:    "jsonl",
		Verbose:   a.options.Verbose,
	})
	_ = memStore
	_ = chkStore

	// Initialize tools
	toolsReg := executor.NewToolRegistry()
	toolsReg.Register(&AdapterReadFile{})
	toolsReg.Register(&AdapterWriteFile{})
	toolsReg.Register(&AdapterSearchFile{})
	toolsReg.Register(&AdapterExecuteCommand{})

	// Initialize planner, executor, reflector
	pl := planner.NewLLMPlanner(&PlannerProvider{Provider: a.provider}, planner.DefaultConfig())
	ex := executor.NewLLMExecutor(&ExecutorProvider{Provider: a.provider}, toolsReg, executor.DefaultConfig())
	rf := reflector.NewLLMReflector(&ReflectorProvider{Provider: a.provider}, reflector.DefaultConfig())

	// Initialize memory
	mem := NewRunMemory(a.goal)

	logger.Log("system", "session_start", map[string]string{"goal": a.goal})

	// Try to restore from checkpoint
	if cp, err := chkStore.LoadByID("cp-agent-done"); err == nil && cp != nil {
		mem.RestoreFrom(cp)
		logger.Log("system", "checkpoint_restore", nil)
	}

	// === Planning ===
	logger.Log("plan", "plan_start", nil)

	plan, err := pl.Plan(ctx, a.goal, mem.AsPlannerReader())
	if err != nil || plan == nil {
		logger.Log("plan", "plan_fail", map[string]string{"error": ErrString(err)})
		return &AgentResult{
			Status:   StatusFailed,
			Answer:   fmt.Sprintf("Planning failed: %s", ErrString(err)),
			Duration: time.Since(startTime),
		}, nil
	}

	logger.Log("plan", "plan_ok", map[string]int{"steps": len(plan.Steps)})
	logger.Save()

	// Save checkpoint after planning
	chkStore.Save(&checkpoint.Checkpoint{
		ID:        "cp-agent-plan",
		SessionID: "agent",
		State:     "planning",
		Goal:      a.goal,
		LoopCount: 0,
	})

	// === Execution loop ===
	var stepResults []StepResult
	for i, step := range plan.Steps {
		if err := ctx.Err(); err != nil {
			logger.LogWithLoop(i+1, "system", "cancelled", nil)
			return &AgentResult{
				Status:   StatusCancelled,
				Answer:   "Cancelled",
				Loops:    i,
				Duration: time.Since(startTime),
			}, nil
		}

		logger.LogWithLoop(i+1, "exec", "step_start", map[string]string{
			"tool": step.Action.Tool, "desc": step.Description,
		})

		execStep := PlannerStepToExec(step)
		res, err := ex.Execute(ctx, execStep, mem.AsExecutorReader())
		if err != nil || res == nil {
			stepResults = append(stepResults, StepResult{
				StepID:  step.ID,
				Success: false,
				Output:  ErrString(err),
			})
			logger.LogWithLoop(i+1, "exec", "step_fail", map[string]string{"error": ErrString(err)})
		} else if res.Success {
			stepResults = append(stepResults, StepResult{
				StepID:     step.ID,
				Success:    true,
				Output:     res.Output,
				TokensUsed: res.TokensUsed,
			})
			logger.LogWithLoop(i+1, "exec", "step_done", map[string]interface{}{
				"success": true,
				"tokens":  res.TokensUsed,
			})
		} else {
			stepResults = append(stepResults, StepResult{
				StepID:  step.ID,
				Success: false,
				Output:  ErrString(res.Error),
			})
			logger.LogWithLoop(i+1, "exec", "step_fail", map[string]string{"error": ErrString(res.Error)})
		}

		mem.AddPlannerResult(PlannerStepToMemory(step), ExecResultToPlanner(res))

		// Save checkpoint every 3 steps
		if (i+1)%3 == 0 {
			chkStore.Save(&checkpoint.Checkpoint{
				ID:          fmt.Sprintf("cp-agent-step%d", i+1),
				SessionID:   "agent",
				State:       "executing",
				Goal:        a.goal,
				CurrentStep: i + 1,
				LoopCount:   i + 1,
			})
		}
	}

	logger.LogWithLoop(len(plan.Steps), "exec", "execution_done", nil)

	// === Reflect ===
	logger.LogWithLoop(len(plan.Steps), "refl", "reflect_start", nil)

	reflectPlan := PlanToReflect(plan)
	assess, err := rf.Reflect(ctx, a.goal, reflectPlan, mem.AsReflectorReader())
	if err != nil || assess == nil {
		logger.LogWithLoop(len(plan.Steps), "refl", "reflect_fail", map[string]string{"error": ErrString(err)})
	} else {
		logger.LogWithLoop(len(plan.Steps), "refl", "reflect_done", map[string]interface{}{
			"decision":   assess.Decision.String(),
			"confidence": assess.Confidence,
		})
	}

	// === Final result ===
	totalTokens := 0
	for _, r := range stepResults {
		totalTokens += r.TokensUsed
	}

	successCount := 0
	for _, r := range stepResults {
		if r.Success {
			successCount++
		}
	}

	var status AgentStatus
	var answer string
	if successCount == len(stepResults) {
		status = StatusCompleted
		answer = fmt.Sprintf("任务完成。共执行 %d 步，全部成功。", len(stepResults))
	} else if successCount > 0 {
		status = StatusCompleted
		answer = fmt.Sprintf("任务部分完成：%d 步成功，%d 步失败。", successCount, len(stepResults)-successCount)
	} else {
		status = StatusFailed
		answer = fmt.Sprintf("任务失败：所有 %d 步均失败。", len(stepResults))
	}

	// Save final checkpoint
	chkStore.Save(&checkpoint.Checkpoint{
		ID:         "cp-agent-done",
		SessionID:  "agent",
		State:      "done",
		Goal:       a.goal,
		TokensUsed: totalTokens,
	})
	logger.LogWithLoop(len(plan.Steps), "system", "session_done", map[string]interface{}{
		"duration": time.Since(startTime).String(), "tokens": totalTokens,
	})
	logger.Save()

	return &AgentResult{
		Status:     status,
		Answer:     answer,
		Loops:      len(plan.Steps),
		TokensUsed: totalTokens,
		Duration:   time.Since(startTime),
	}, nil
}

// ErrString returns the error string or empty string if nil
func ErrString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// TruncateStr truncates a string to n characters
func TruncateStr(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

// HeuristicProvider is a no-LLM provider used as a stand-in while the
// real DeepSeek provider is being wired up. It always returns sensible
// JSON for plan / reflect prompts, so the CLI can exercise the
// full plan → execute → reflect → output flow without an API key.
type HeuristicProvider struct {
	model string
}

// NewHeuristicProvider creates a new heuristic provider
func NewHeuristicProvider(model string) *HeuristicProvider {
	return &HeuristicProvider{model: model}
}

func (h *HeuristicProvider) Chat(ctx context.Context, system, user string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	// Planner prompt
	if strings.Contains(system, "task planner") {
		return h.planJSON(user), nil
	}
	// Reflector prompt
	if strings.Contains(system, "task reflector") {
		return h.reflectJSON(user), nil
	}
	// Generic LLM generate → summarise the user prompt
	return fmt.Sprintf("（heuristic）已完成：%s", TruncateStr(user, 120)), nil
}

func (h *HeuristicProvider) planJSON(goal string) string {
	g := strings.ToLower(goal)
	steps := []map[string]interface{}{}

	addTool := func(tool string, params map[string]interface{}, desc string, breakpoint bool) {
		steps = append(steps, map[string]interface{}{
			"id":          fmt.Sprintf("step-%d", len(steps)+1),
			"description": desc,
			"action": map[string]interface{}{
				"type":   "tool_call",
				"tool":   tool,
				"params": params,
			},
			"breakpoint": breakpoint,
		})
	}

	switch {
	case strings.Contains(g, "分析") || strings.Contains(g, "analyze") || strings.Contains(g, "code"):
		addTool("search_file", map[string]interface{}{"pattern": "*.go", "dir": "."}, "扫描项目结构与 Go 源文件", false)
		addTool("read_file", map[string]interface{}{"path": "go.mod"}, "读取 go.mod 了解依赖", false)
		addTool("execute_command", map[string]interface{}{"command": "go vet ./..."}, "运行 go vet 静态分析", false)
		addTool("llm_generate", map[string]interface{}{"prompt": "请基于以上扫描结果输出代码质量总结"}, "生成分析报告", false)
	case strings.Contains(g, "修复") || strings.Contains(g, "fix"):
		addTool("execute_command", map[string]interface{}{"command": "go test ./... 2>&1 | head -50"}, "运行测试，定位失败用例", false)
		addTool("search_file", map[string]interface{}{"pattern": "*_test.go"}, "查找相关测试文件", false)
		addTool("read_file", map[string]interface{}{"path": "main.go"}, "读取可疑源文件", false)
		addTool("write_file", map[string]interface{}{"path": "main.go", "content": "// fix applied"}, "应用修复", true)
	case strings.Contains(g, "测试") || strings.Contains(g, "test"):
		addTool("search_file", map[string]interface{}{"pattern": "*_test.go"}, "查找已有测试", false)
		addTool("read_file", map[string]interface{}{"path": "main.go"}, "阅读主模块", false)
		addTool("write_file", map[string]interface{}{"path": "main_test.go", "content": "package main\n\nimport \"testing\"\n\nfunc TestSample(t *testing.T) { t.Log(\"ok\") }"}, "写入单元测试", true)
		addTool("execute_command", map[string]interface{}{"command": "go test ./... -v"}, "运行测试", false)
	case strings.Contains(g, "搜索") || strings.Contains(g, "search") || strings.Contains(g, "web"):
		addTool("execute_command", map[string]interface{}{"command": "echo 'web search disabled in offline mode'"}, "执行 web 搜索（当前为离线模式）", false)
		addTool("llm_generate", map[string]interface{}{"prompt": "请提供该主题的关键信息"}, "汇总搜索结果", false)
	default:
		addTool("read_file", map[string]interface{}{"path": "README.md"}, "读取 README 了解项目背景", false)
		addTool("search_file", map[string]interface{}{"pattern": "*.go", "dir": "."}, "扫描 Go 源文件", false)
		addTool("execute_command", map[string]interface{}{"command": "go build ./..."}, "编译验证", false)
		addTool("llm_generate", map[string]interface{}{"prompt": "请基于以上信息给出可执行建议"}, "汇总输出", false)
	}

	resp := map[string]interface{}{
		"id":        fmt.Sprintf("plan-%d", time.Now().Unix()),
		"steps":     steps,
		"rationale": "由 HeuristicProvider 自动生成（无 LLM 调用）。",
	}
	b, _ := json.Marshal(resp)
	return string(b)
}

func (h *HeuristicProvider) reflectJSON(goal string) string {
	_ = goal
	resp := map[string]interface{}{
		"decision":   "complete",
		"reason":     "所有规划步骤均已执行，输出已生成。",
		"confidence": 0.78,
		"findings": []string{
			"heuristic 模式：不调用真实 LLM",
			"plan/reflect 闭环已贯通",
		},
		"suggestions": []string{
			"接入真实 DeepSeek provider 后置信度可提升",
		},
	}
	b, _ := json.Marshal(resp)
	return string(b)
}

// DeepSeekProvider wraps the real provider.DeepSeekProvider into our Provider interface
type DeepSeekProvider struct {
	provider *provider.DeepSeekProvider
}

// NewDeepSeekProvider creates a new DeepSeek provider
func NewDeepSeekProvider(apiKey, model string) *DeepSeekProvider {
	dp := provider.NewDeepSeekProvider(&provider.Config{
		APIKey:  apiKey,
		BaseURL: "https://api.deepseek.com",
		Model:   model,
		Timeout: 120 * time.Second,
	})
	return &DeepSeekProvider{provider: dp}
}

func (d *DeepSeekProvider) Chat(ctx context.Context, system, user string) (string, error) {
	resp, err := d.provider.Chat(ctx, []provider.Message{
		{Role: "system", Content: system},
		{Role: "user", Content: user},
	})
	if err != nil {
		return "", err
	}
	return resp, nil
}

// PlannerProvider bridges Provider → planner.LLMProvider
type PlannerProvider struct {
	Provider Provider
}

// Chat implements the planner.LLMProvider interface
func (p *PlannerProvider) Chat(ctx context.Context, messages []planner.Message) (string, error) {
	system, user := SplitMessages(messages)
	return p.Provider.Chat(ctx, system, user)
}

// ExecutorProvider bridges Provider → executor.LLMProvider
type ExecutorProvider struct {
	Provider Provider
}

// Chat implements the executor.LLMProvider interface
func (p *ExecutorProvider) Chat(ctx context.Context, messages []executor.Message) (string, error) {
	system, user := SplitExecMessages(messages)
	return p.Provider.Chat(ctx, system, user)
}

// ReflectorProvider bridges Provider → reflector.LLMProvider
type ReflectorProvider struct {
	Provider Provider
}

// Chat implements the reflector.LLMProvider interface
func (p *ReflectorProvider) Chat(ctx context.Context, messages []reflector.Message) (string, error) {
	system, user := SplitReflectMessages(messages)
	return p.Provider.Chat(ctx, system, user)
}

// SplitMessages extracts the system / user content from planner messages
func SplitMessages(messages []planner.Message) (string, string) {
	var system, user string
	for _, m := range messages {
		if m.Role == "system" {
			system = m.Content
		}
		if m.Role == "user" {
			user = m.Content
		}
	}
	return system, user
}

// SplitExecMessages extracts the system / user content from executor messages
func SplitExecMessages(messages []executor.Message) (string, string) {
	var system, user string
	for _, m := range messages {
		if m.Role == "system" {
			system = m.Content
		}
		if m.Role == "user" {
			user = m.Content
		}
	}
	return system, user
}

// SplitReflectMessages extracts the system / user content from reflector messages
func SplitReflectMessages(messages []reflector.Message) (string, string) {
	var system, user string
	for _, m := range messages {
		if m.Role == "system" {
			system = m.Content
		}
		if m.Role == "user" {
			user = m.Content
		}
	}
	return system, user
}

// RunMemory is an in-memory implementation of the read interfaces
// used by planner/executor/reflector. It exposes three adapters that
// each convert internal types to the package-specific StepResult shape.
type RunMemory struct {
	goal    string
	results []planner.StepResult
	Started time.Time
}

// NewRunMemory creates a new RunMemory
func NewRunMemory(goal string) *RunMemory {
	return &RunMemory{goal: goal, Started: time.Now()}
}

// AddPlannerResult adds a step result to the memory
func (m *RunMemory) AddPlannerResult(step planner.Step, r planner.StepResult) {
	m.results = append(m.results, r)
}

// AsPlannerReader returns a planner-compatible memory reader
func (m *RunMemory) AsPlannerReader() *PlannerMemoryReader { return &PlannerMemoryReader{M: m} }

// AsExecutorReader returns an executor-compatible memory reader
func (m *RunMemory) AsExecutorReader() *ExecutorMemoryReader { return &ExecutorMemoryReader{M: m} }

// AsReflectorReader returns a reflector-compatible memory reader
func (m *RunMemory) AsReflectorReader() *ReflectorMemoryReader {
	return &ReflectorMemoryReader{M: m}
}

// RestoreFrom restores memory state from a checkpoint
func (m *RunMemory) RestoreFrom(cp *checkpoint.Checkpoint) {
	// Restore counters from checkpoint
	m.Started = time.Now().Add(-time.Duration(cp.LoopCount) * 2 * time.Second) // estimate
}

// PlannerMemoryReader provides planner-compatible memory access
type PlannerMemoryReader struct{ M *RunMemory }

// GetSessionSummary returns a summary of the session
func (r *PlannerMemoryReader) GetSessionSummary() string {
	return fmt.Sprintf("Goal: %s\nExecuted steps so far: %d", r.M.goal, len(r.M.results))
}

// GetStepResults returns the step results
func (r *PlannerMemoryReader) GetStepResults() []planner.StepResult { return r.M.results }

// ExecutorMemoryReader provides executor-compatible memory access
type ExecutorMemoryReader struct{ M *RunMemory }

// GetSessionSummary returns a summary of the session
func (r *ExecutorMemoryReader) GetSessionSummary() string {
	return fmt.Sprintf("Goal: %s\nExecuted steps so far: %d", r.M.goal, len(r.M.results))
}

// ReflectorMemoryReader provides reflector-compatible memory access
type ReflectorMemoryReader struct{ M *RunMemory }

// GetSessionSummary returns a summary of the session
func (r *ReflectorMemoryReader) GetSessionSummary() string {
	return fmt.Sprintf("Goal: %s\nExecuted steps so far: %d", r.M.goal, len(r.M.results))
}

// GetStepResults returns the step results in reflector format
func (r *ReflectorMemoryReader) GetStepResults() []reflector.StepResult {
	out := make([]reflector.StepResult, len(r.M.results))
	for i, x := range r.M.results {
		out[i] = reflector.StepResult{StepID: x.StepID, Success: x.Success, Output: x.Output}
	}
	return out
}

// PlannerStepToMemory converts a planner step to itself (identity)
func PlannerStepToMemory(s planner.Step) planner.Step {
	return s
}

// PlannerStepToExec converts a planner step to an executor step
func PlannerStepToExec(s planner.Step) executor.Step {
	return executor.Step{
		ID:          s.ID,
		Description: s.Description,
		Action: executor.Action{
			Type:   s.Action.Type,
			Tool:   s.Action.Tool,
			Params: s.Action.Params,
			Prompt: s.Action.Prompt,
		},
		DependsOn:  s.DependsOn,
		Breakpoint: s.Breakpoint,
	}
}

// ExecResultToPlanner converts an executor result to a planner result
func ExecResultToPlanner(r *executor.StepResult) planner.StepResult {
	if r == nil {
		return planner.StepResult{Success: false, Output: "no result"}
	}
	out := r.Output
	if r.Error != nil {
		out = r.Error.Error()
	}
	return planner.StepResult{
		StepID:  r.StepID,
		Success: r.Success,
		Output:  out,
	}
}

// PlanToReflect converts a planner plan to a reflector plan
func PlanToReflect(p *planner.Plan) *reflector.Plan {
	if p == nil {
		return nil
	}
	rp := &reflector.Plan{ID: p.ID}
	for _, s := range p.Steps {
		rp.Steps = append(rp.Steps, reflector.Step{ID: s.ID, Description: s.Description})
	}
	return rp
}

// Tool adapters bridging internal/tools → executor.Tool

// AdapterReadFile adapts the internal read_file tool
type AdapterReadFile struct{}

// Name returns the tool name
func (a *AdapterReadFile) Name() string { return "read_file" }

// Description returns the tool description
func (a *AdapterReadFile) Description() string { return "Read the contents of a file" }

// Call executes the tool
func (a *AdapterReadFile) Call(ctx context.Context, params map[string]interface{}) (string, error) {
	t := &tools.ReadFileTool{}
	return t.Call(ctx, params)
}

// AdapterWriteFile adapts the internal write_file tool
type AdapterWriteFile struct{}

// Name returns the tool name
func (a *AdapterWriteFile) Name() string { return "write_file" }

// Description returns the tool description
func (a *AdapterWriteFile) Description() string { return "Write content to a file" }

// Call executes the tool
func (a *AdapterWriteFile) Call(ctx context.Context, params map[string]interface{}) (string, error) {
	t := &tools.WriteFileTool{}
	return t.Call(ctx, params)
}

// AdapterSearchFile adapts the internal search_file tool
type AdapterSearchFile struct{}

// Name returns the tool name
func (a *AdapterSearchFile) Name() string { return "search_file" }

// Description returns the tool description
func (a *AdapterSearchFile) Description() string { return "Search for files matching a pattern" }

// Call executes the tool
func (a *AdapterSearchFile) Call(ctx context.Context, params map[string]interface{}) (string, error) {
	t := &tools.SearchFileTool{}
	return t.Call(ctx, params)
}

// AdapterExecuteCommand adapts the internal execute_command tool
type AdapterExecuteCommand struct{}

// Name returns the tool name
func (a *AdapterExecuteCommand) Name() string { return "execute_command" }

// Description returns the tool description
func (a *AdapterExecuteCommand) Description() string { return "Execute a shell command" }

// Call executes the tool
func (a *AdapterExecuteCommand) Call(ctx context.Context, params map[string]interface{}) (string, error) {
	t := &tools.ExecuteCommandTool{}
	return t.Call(ctx, params)
}