// Package bridge connects desktop UI to the real agent loop.
//
// Wiring:
//
//   SendMessage(goal)
//     └── runLoop(ctx, s)
//          ├── HeuristicPlanner.Plan(goal)         → plan
//          ├── for each step:
//          │     ├── if ask mode + risky → emit approval modal
//          │     ├── HeuristicExecutor.Execute(step) → result
//          │     ├── append to memory
//          │     └── emit session update
//          ├── HeuristicReflector.Reflect(plan, results) → assessment
//          └── emit final answer
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/qoqu/zhuLong/internal/executor"
	"github.com/qoqu/zhuLong/internal/planner"
	"github.com/qoqu/zhuLong/internal/reflector"
	"github.com/qoqu/zhuLong/internal/tools"
)

// HeuristicProvider is a no-LLM provider used as a stand-in while the
// real DeepSeek provider is being wired up. It always returns sensible
// JSON for plan / reflect prompts, so the desktop UI can exercise the
// full plan → execute → reflect → output flow without an API key.
//
// Replace with provider.NewDeepSeekProvider(config) once available.
type HeuristicProvider struct {
	model string
}

func newProvider(model string) *HeuristicProvider {
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
	return fmt.Sprintf("（heuristic）已完成：%s", truncateStr(user, 120)), nil
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

func truncateStr(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

// RunAgent runs a full plan → execute → reflect loop for a goal.
// It pushes session updates via emit at every meaningful state change
// so the UI stays in sync.
func (a *App) RunAgent(ctx context.Context, s *SessionState) {
	defer func() {
		a.mu.Lock()
		if s.Approval != nil {
			s.Approval = nil
		}
		a.mu.Unlock()
	}()

	provider := newProvider(s.Model)
	toolsReg := executor.NewToolRegistry()
	toolsReg.Register(&adapterReadFile{})
	toolsReg.Register(&adapterWriteFile{})
	toolsReg.Register(&adapterSearchFile{})
	toolsReg.Register(&adapterExecuteCommand{})

	pl := planner.NewLLMPlanner(&plannerProvider{provider: provider}, planner.DefaultConfig())
	ex := executor.NewLLMExecutor(&executorProvider{provider: provider}, toolsReg, executor.DefaultConfig())
	rf := reflector.NewLLMReflector(&reflectorProvider{provider: provider}, reflector.DefaultConfig())

	mem := newRunMemory(s.Goal)

	// === Planning ===
	s.Status = "planning"
	a.appendLog(s, "plan", "Planning...", "goal="+truncateStr(s.Goal, 60))
	a.emitSession(s)
	a.wait(ctx, 400*time.Millisecond)

	plan, err := pl.Plan(ctx, s.Goal, mem.asPlannerReader())
	if err != nil || plan == nil {
		a.appendLog(s, "plan", "Plan failed", errString(err))
		s.Status = "error"
		a.emitSession(s)
		return
	}

	s.Plan = convertPlan(plan)
	a.appendLog(s, "plan", fmt.Sprintf("Plan created: %d steps", len(plan.Steps)), plan.Rationale)
	a.emitSession(s)

	// === Execution loop ===
	s.Status = "executing"
	for i, step := range plan.Steps {
		if err := ctx.Err(); err != nil {
			a.appendLog(s, "system", "Cancelled", errString(err))
			s.Status = "cancelled"
			a.emitSession(s)
			return
		}

		// Approval gate
		if step.Breakpoint || isRiskyTool(step.Action.Tool) {
			if s.Mode == "ask" || s.Mode == "auto" {
				if !a.requestApproval(ctx, s, step) {
					a.appendLog(s, "system", "Denied by user", step.Action.Tool)
					s.Plan[i].Status = "failed"
					mem.AddPlannerResult(plannerStepToMemory(step), planner.StepResult{
						StepID:  step.ID,
						Success: false,
						Output:  "user denied",
					})
					a.emitSession(s)
					break
				}
			}
		}

		s.Plan[i].Status = "running"
		a.appendLog(s, "exec", fmt.Sprintf("Step %d/%d %s", i+1, len(plan.Steps), step.Action.Tool), step.Description)
		a.emitSession(s)
		a.wait(ctx, 200*time.Millisecond)

		execStep := plannerStepToExec(step)
		res, err := ex.Execute(ctx, execStep, mem.asExecutorReader())
		if err != nil || res == nil {
			s.Plan[i].Status = "failed"
			a.appendLog(s, "exec", "Step failed", errString(err))
		} else if res.Success {
			s.Plan[i].Status = "completed"
			s.Stats.RequestCount++
			s.Stats.SessionTokens += res.TokensUsed
			if res.TokensUsed == 0 {
				s.Stats.SessionTokens += 800
			}
			s.Stats.TotalUsed = s.Stats.SessionTokens
			s.Stats.UsagePercent = float64(s.Stats.TotalUsed) / float64(s.Stats.TotalLimit) * 100
			if step.Action.Type == "tool_call" {
				s.Stats.MainCount++
				s.Stats.MainCost += 0.0025
				s.Files = append(s.Files, step.Action.Tool)
			} else {
				s.Stats.MainCount++
				s.Stats.MainCost += 0.0035
			}
			a.appendLog(s, "exec", fmt.Sprintf("Step %d/%d done", i+1, len(plan.Steps)), truncateStr(res.Output, 80))
			s.Messages = append(s.Messages, MessageDTO{
				ID:       fmt.Sprintf("t%d", time.Now().UnixNano()),
				Role:     "tool",
				Content:  truncateStr(res.Output, 200),
				Time:     time.Now(),
				ToolName: step.Action.Tool,
			})
		} else {
			s.Plan[i].Status = "failed"
			a.appendLog(s, "exec", "Step failed", errString(res.Error))
		}

		mem.AddPlannerResult(plannerStepToMemory(step), execResultToPlanner(res))
		a.emitSession(s)
		a.wait(ctx, 250*time.Millisecond)
	}

	// === Reflect ===
	s.Status = "reflecting"
	a.appendLog(s, "refl", "Reflecting on progress", "score=?")
	a.emitSession(s)
	a.wait(ctx, 300*time.Millisecond)

	reflectPlan := planToReflect(plan)
	assess, err := rf.Reflect(ctx, s.Goal, reflectPlan, mem.asReflectorReader())
	if err != nil || assess == nil {
		a.appendLog(s, "refl", "Reflection failed", errString(err))
	} else {
		a.appendLog(s, "refl", fmt.Sprintf("Decision: %s (%.0f%%)", assess.Decision, assess.Confidence*100), assess.Reason)
		for _, f := range assess.Findings {
			s.Messages = append(s.Messages, MessageDTO{
				ID:      fmt.Sprintf("f%d", time.Now().UnixNano()),
				Role:    "system",
				Content: "💡 " + f,
				Time:    time.Now(),
			})
		}
	}

	// === Final answer ===
	s.Stats.Elapsed = time.Since(mem.started).Round(time.Millisecond).String()
	final := "任务完成。"
	switch {
	case countFailed(s.Plan) == len(s.Plan) && len(s.Plan) > 0:
		final = "所有步骤均失败，建议检查环境或调整目标。"
	case countFailed(s.Plan) > 0:
		final = fmt.Sprintf("任务部分完成：%d 步成功，%d 步失败。", countCompleted(s.Plan), countFailed(s.Plan))
	}
	final += fmt.Sprintf("（耗时 %s，共消耗 %d tokens）", s.Stats.Elapsed, s.Stats.SessionTokens)
	s.Messages = append(s.Messages, MessageDTO{
		ID:      fmt.Sprintf("m%d", time.Now().UnixNano()),
		Role:    "assistant",
		Content: final,
		Time:    time.Now(),
	})
	s.Status = "done"
	a.emitSession(s)
	a.emitProjects()
}

// requestApproval pushes a modal and blocks until the user responds.
// Returns true if approved, false if denied.
func (a *App) requestApproval(ctx context.Context, s *SessionState, step planner.Step) bool {
	a.mu.Lock()
	s.Approval = &ApprovalRequestDTO{
		ID:        fmt.Sprintf("ap%d", time.Now().UnixNano()),
		Tool:      step.Action.Tool,
		Args:      step.Action.Params,
		Risk:      approvalRisk(step.Action.Tool, step.Action.Params),
		Reason:    approvalReason(step),
		CreatedAt: time.Now(),
	}
	a.mu.Unlock()
	a.emitSession(s)
	a.appendLog(s, "system", "Awaiting approval", step.Action.Tool)

	for {
		a.mu.Lock()
		cleared := s.Approval == nil
		a.mu.Unlock()
		if cleared {
			return true
		}
		if ctx.Err() != nil {
			return false
		}
		select {
		case <-ctx.Done():
			return false
		case <-time.After(120 * time.Millisecond):
		}
	}
}

func approvalRisk(tool string, params map[string]interface{}) string {
	if strings.Contains(tool, "command") || strings.Contains(tool, "write") {
		return "high"
	}
	if strings.Contains(tool, "read") || strings.Contains(tool, "search") {
		return "low"
	}
	return "medium"
}

func approvalReason(step planner.Step) string {
	switch step.Action.Type {
	case "tool_call":
		return fmt.Sprintf("准备调用工具 %q：%s", step.Action.Tool, step.Description)
	case "llm_generate":
		return "准备进行一次 LLM 生成，可能产生 token 消耗。"
	default:
		return step.Description
	}
}

func isRiskyTool(tool string) bool {
	switch tool {
	case "execute_command", "write_file":
		return true
	}
	return false
}

func (a *App) setActiveStep(i int) {
	for j := range a.activeID {
		_ = j
	}
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// === Provider adapters: bridge HeuristicProvider → planner/executor/reflector ===
//
// Each internal module defines its own Message/StepResult types, so we
// need a thin wrapper for each one that translates to the shared
// (system, user) signature exposed by HeuristicProvider.Chat.

type providerCore interface {
	Chat(ctx context.Context, system, user string) (string, error)
}

type plannerProvider struct {
	provider providerCore
}

func (p *plannerProvider) Chat(ctx context.Context, messages []planner.Message) (string, error) {
	system, user := splitMessages(messages)
	return p.provider.Chat(ctx, system, user)
}

type executorProvider struct {
	provider providerCore
}

func (p *executorProvider) Chat(ctx context.Context, messages []executor.Message) (string, error) {
	system, user := splitExecMessages(messages)
	return p.provider.Chat(ctx, system, user)
}

type reflectorProvider struct {
	provider providerCore
}

func (p *reflectorProvider) Chat(ctx context.Context, messages []reflector.Message) (string, error) {
	system, user := splitReflectMessages(messages)
	return p.provider.Chat(ctx, system, user)
}

// splitMessages extracts the system / user content from any of the
// three internal Message shapes. They all share the same JSON tag
// layout, so reflection works without per-package adapters.
func splitMessages(messages []planner.Message) (string, string) {
	var system, user string
	for _, m := range messages {
		if m.Role == "system" {
			system = m.Content
		}
		if m.Role == "user" {
			user = m.Content
		}
	}
	// planner / executor / reflector Message types are identical
	// (Role, Content) so the planner view is sufficient. We mirror
	// into the other two here so the wrappers stay one-liners.
	return system, user
}

func splitExecMessages(messages []executor.Message) (string, string) {
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

func splitReflectMessages(messages []reflector.Message) (string, string) {
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

// runMemory is an in-memory implementation of the read interfaces
// used by planner/executor/reflector. It exposes three adapters that
// each convert internal types to the package-specific StepResult shape.
type runMemory struct {
	goal    string
	results []planner.StepResult
	started time.Time
}

func newRunMemory(goal string) *runMemory {
	return &runMemory{goal: goal, started: time.Now()}
}

func (m *runMemory) AddPlannerResult(step planner.Step, r planner.StepResult) {
	m.results = append(m.results, r)
}

func (m *runMemory) asPlannerReader() *plannerMemoryReader { return &plannerMemoryReader{m: m} }
func (m *runMemory) asExecutorReader() *executorMemoryReader { return &executorMemoryReader{m: m} }
func (m *runMemory) asReflectorReader() *reflectorMemoryReader { return &reflectorMemoryReader{m: m} }

type plannerMemoryReader struct{ m *runMemory }

func (r *plannerMemoryReader) GetSessionSummary() string {
	return fmt.Sprintf("Goal: %s\nExecuted steps so far: %d", r.m.goal, len(r.m.results))
}
func (r *plannerMemoryReader) GetStepResults() []planner.StepResult { return r.m.results }

type executorMemoryReader struct{ m *runMemory }

func (r *executorMemoryReader) GetSessionSummary() string {
	return fmt.Sprintf("Goal: %s\nExecuted steps so far: %d", r.m.goal, len(r.m.results))
}

type reflectorMemoryReader struct{ m *runMemory }

func (r *reflectorMemoryReader) GetSessionSummary() string {
	return fmt.Sprintf("Goal: %s\nExecuted steps so far: %d", r.m.goal, len(r.m.results))
}
func (r *reflectorMemoryReader) GetStepResults() []reflector.StepResult {
	out := make([]reflector.StepResult, len(r.m.results))
	for i, x := range r.m.results {
		out[i] = reflector.StepResult{StepID: x.StepID, Success: x.Success, Output: x.Output}
	}
	return out
}

func convertPlan(p *planner.Plan) []PlanStepDTO {
	out := make([]PlanStepDTO, len(p.Steps))
	for i, s := range p.Steps {
		out[i] = PlanStepDTO{
			ID:          s.ID,
			Description: s.Description,
			Status:      "pending",
		}
	}
	return out
}

func plannerStepToMemory(s planner.Step) planner.Step {
	return s
}

func plannerStepToExec(s planner.Step) executor.Step {
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

func execResultToPlanner(r *executor.StepResult) planner.StepResult {
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

func planToReflect(p *planner.Plan) *reflector.Plan {
	if p == nil {
		return nil
	}
	rp := &reflector.Plan{ID: p.ID}
	for _, s := range p.Steps {
		rp.Steps = append(rp.Steps, reflector.Step{ID: s.ID, Description: s.Description})
	}
	return rp
}

func countCompleted(steps []PlanStepDTO) int {
	n := 0
	for _, s := range steps {
		if s.Status == "completed" {
			n++
		}
	}
	return n
}
func countFailed(steps []PlanStepDTO) int {
	n := 0
	for _, s := range steps {
		if s.Status == "failed" {
			n++
		}
	}
	return n
}

// === Tool adapters bridging internal/tools → executor.Tool ===

type adapterReadFile struct{}

func (a *adapterReadFile) Name() string        { return "read_file" }
func (a *adapterReadFile) Description() string { return "Read the contents of a file" }
func (a *adapterReadFile) Call(ctx context.Context, params map[string]interface{}) (string, error) {
	t := &tools.ReadFileTool{}
	return t.Call(ctx, params)
}

type adapterWriteFile struct{}

func (a *adapterWriteFile) Name() string        { return "write_file" }
func (a *adapterWriteFile) Description() string { return "Write content to a file" }
func (a *adapterWriteFile) Call(ctx context.Context, params map[string]interface{}) (string, error) {
	t := &tools.WriteFileTool{}
	return t.Call(ctx, params)
}

type adapterSearchFile struct{}

func (a *adapterSearchFile) Name() string        { return "search_file" }
func (a *adapterSearchFile) Description() string { return "Search for files matching a pattern" }
func (a *adapterSearchFile) Call(ctx context.Context, params map[string]interface{}) (string, error) {
	t := &tools.SearchFileTool{}
	return t.Call(ctx, params)
}

type adapterExecuteCommand struct{}

func (a *adapterExecuteCommand) Name() string        { return "execute_command" }
func (a *adapterExecuteCommand) Description() string { return "Execute a shell command" }
func (a *adapterExecuteCommand) Call(ctx context.Context, params map[string]interface{}) (string, error) {
	t := &tools.ExecuteCommandTool{}
	return t.Call(ctx, params)
}
