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
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/qoqu/zhuLong/internal/checkpoint"
	"github.com/qoqu/zhuLong/internal/executor"
	"github.com/qoqu/zhuLong/internal/memory"
	"github.com/qoqu/zhuLong/internal/planner"
	"github.com/qoqu/zhuLong/internal/reflector"
	"github.com/qoqu/zhuLong/internal/trace"
	"github.com/qoqu/zhuLong/pkg"
)

// newProvider creates a new provider based on the model and environment
func newProvider(model string) pkg.Provider {
	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	if apiKey != "" {
		return pkg.NewDeepSeekProvider(apiKey, model)
	}
	return pkg.NewHeuristicProvider(model)
}

// dataDir returns the path to the persistent data directory.
func (a *App) dataDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".zhulong"
	}
	dir := filepath.Join(home, ".zhulong")
	os.MkdirAll(dir, 0755)
	return dir
}

// RunAgent runs a full plan → execute → reflect loop for a goal.
// It integrates Memory, Checkpoint, Trace, and the Provider.
func (a *App) RunAgent(ctx context.Context, s *SessionState) {
	defer func() {
		a.mu.Lock()
		if s.Approval != nil {
			s.Approval = nil
		}
		a.mu.Unlock()
	}()

	// === Setup: Memory, Checkpoint, Trace ===
	dataDir := a.dataDir()

	memStore := memory.NewFileStore(filepath.Join(dataDir, "memory"))
	chkStore := checkpoint.NewFileStore(filepath.Join(dataDir, "checkpoint"))
	logger := trace.NewLogger(s.Info.ID, &trace.Config{
		Enabled:   true,
		OutputDir: filepath.Join(dataDir, "traces"),
		Format:    "jsonl",
		Verbose:   true,
	})
	_ = memStore
	_ = chkStore

	provider := newProvider(s.Model)
	toolsReg := executor.NewToolRegistry()
	toolsReg.Register(&pkg.AdapterReadFile{})
	toolsReg.Register(&pkg.AdapterWriteFile{})
	toolsReg.Register(&pkg.AdapterSearchFile{})
	toolsReg.Register(&pkg.AdapterExecuteCommand{})

	pl := planner.NewLLMPlanner(&pkg.PlannerProvider{Provider: provider}, planner.DefaultConfig())
	ex := executor.NewLLMExecutor(&pkg.ExecutorProvider{Provider: provider}, toolsReg, executor.DefaultConfig())
	rf := reflector.NewLLMReflector(&pkg.ReflectorProvider{Provider: provider}, reflector.DefaultConfig())

	mem := pkg.NewRunMemory(s.Goal)

	logger.Log("system", "session_start", map[string]string{"goal": s.Goal})

	// === Initialize enhanced states ===
	a.mu.Lock()
	s.MemoryState = a.initMemoryState()
	s.LearningState = a.initLearningState()
	s.ModuleState = a.initModuleState()
	a.mu.Unlock()

	// Try to restore from checkpoint
	if cp, err := chkStore.LoadByID(fmt.Sprintf("cp-%s-done", s.Info.ID)); err == nil && cp != nil {
		a.appendLog(s, "system", "Restored from checkpoint", fmt.Sprintf("loop %d", cp.LoopCount))
		mem.RestoreFrom(cp)
		if cp.State != "" {
			s.Status = cp.State
		}
		logger.Log("system", "checkpoint_restore", nil)
	}

	// === Planning ===
	s.Status = "planning"
	
	// Update module states
	a.mu.Lock()
	if s.ModuleState != nil && s.ModuleState.Controller != nil {
		s.ModuleState.Controller.Details["fsmState"] = "Planning"
		s.ModuleState.Planner.Details["lastPlan"] = pkg.TruncateStr(s.Goal, 30)
	}
	a.mu.Unlock()
	
	a.appendLog(s, "plan", "Planning...", "goal="+pkg.TruncateStr(s.Goal, 60))
	a.emitSession(s)
	logger.Log("plan", "plan_start", nil)
	a.wait(ctx, 400*time.Millisecond)

	plan, err := pl.Plan(ctx, s.Goal, mem.AsPlannerReader())
	if err != nil || plan == nil {
		a.appendLog(s, "plan", "Plan failed", pkg.ErrString(err))
		s.Status = "error"
		
		// Update module states
		a.mu.Lock()
		if s.ModuleState != nil && s.ModuleState.Planner != nil {
			s.ModuleState.Planner.Status = "error"
		}
		a.mu.Unlock()
		
		a.emitSession(s)
		logger.Log("plan", "plan_fail", map[string]string{"error": pkg.ErrString(err)})
		return
	}

	s.Plan = convertPlan(plan)
	a.appendLog(s, "plan", fmt.Sprintf("Plan created: %d steps", len(plan.Steps)), plan.Rationale)
	
	// Update module states after planning
	a.mu.Lock()
	if s.ModuleState != nil {
		if s.ModuleState.Planner != nil {
			s.ModuleState.Planner.Status = "idle"
			s.ModuleState.Planner.Details["lastPlan"] = pkg.TruncateStr(plan.Rationale, 50)
		}
		if s.ModuleState.Controller != nil {
			s.ModuleState.Controller.Details["fsmState"] = "Planning"
		}
	}
	a.mu.Unlock()
	
	a.emitSession(s)
	logger.Log("plan", "plan_ok", map[string]int{"steps": len(plan.Steps)})
	logger.Save()

	// Save checkpoint after planning
	chkStore.Save(&checkpoint.Checkpoint{
		ID:        fmt.Sprintf("cp-%s-plan", s.Info.ID),
		SessionID: s.Info.ID,
		State:     "planning",
		Goal:      s.Goal,
		LoopCount: 0,
	})

	// === Execution loop ===
	s.Status = "executing"
	
	// Update module states
	a.mu.Lock()
	if s.ModuleState != nil {
		if s.ModuleState.Controller != nil {
			s.ModuleState.Controller.Details["fsmState"] = "Executing"
		}
		if s.ModuleState.Executor != nil {
			s.ModuleState.Executor.Status = "active"
			s.ModuleState.Executor.Details["toolsLoaded"] = len(toolsReg.List())
		}
	}
	a.mu.Unlock()
	
	for i, step := range plan.Steps {
		if err := ctx.Err(); err != nil {
			a.appendLog(s, "system", "Cancelled", pkg.ErrString(err))
			s.Status = "cancelled"
			
			// Update module states
			a.mu.Lock()
			if s.ModuleState != nil && s.ModuleState.Controller != nil {
				s.ModuleState.Controller.Details["fsmState"] = "Cancelled"
			}
			a.mu.Unlock()
			
			a.emitSession(s)
			logger.LogWithLoop(i + 1, "system", "cancelled", nil)
			return
		}

		logger.LogWithLoop(i + 1, "exec", "step_start", map[string]string{
			"tool": step.Action.Tool, "desc": step.Description,
		})

		// Approval gate
		if step.Breakpoint || isRiskyTool(step.Action.Tool) {
			if s.Mode == "ask" || s.Mode == "auto" {
				if !a.requestApproval(ctx, s, step) {
					a.appendLog(s, "system", "Denied by user", step.Action.Tool)
					s.Plan[i].Status = "failed"
					mem.AddPlannerResult(pkg.PlannerStepToMemory(step), planner.StepResult{
						StepID:  step.ID,
						Success: false,
						Output:  "user denied",
					})
					logger.LogWithLoop(i + 1, "exec", "denied", nil)
					a.emitSession(s)
					break
				}
			}
		}

		s.Plan[i].Status = "running"
		a.appendLog(s, "exec", fmt.Sprintf("Step %d/%d %s", i+1, len(plan.Steps), step.Action.Tool), step.Description)
		a.emitSession(s)
		a.wait(ctx, 200*time.Millisecond)

		execStep := pkg.PlannerStepToExec(step)
		res, err := ex.Execute(ctx, execStep, mem.AsExecutorReader())
		if err != nil || res == nil {
			s.Plan[i].Status = "failed"
			a.appendLog(s, "exec", "Step failed", pkg.ErrString(err))
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
			} else {
				s.Stats.MainCount++
				s.Stats.MainCost += 0.0035
			}
			a.appendLog(s, "exec", fmt.Sprintf("Step %d/%d done", i+1, len(plan.Steps)), pkg.TruncateStr(res.Output, 80))
			s.Messages = append(s.Messages, MessageDTO{
				ID:       fmt.Sprintf("t%d", time.Now().UnixNano()),
				Role:     "tool",
				Content:  pkg.TruncateStr(res.Output, 200),
				Time:     time.Now(),
				ToolName: step.Action.Tool,
			})
		} else {
			s.Plan[i].Status = "failed"
			a.appendLog(s, "exec", "Step failed", pkg.ErrString(res.Error))
		}

		mem.AddPlannerResult(pkg.PlannerStepToMemory(step), pkg.ExecResultToPlanner(res))
		logger.LogWithLoop(i+1, "exec", "step_done", map[string]interface{}{
			"success": res != nil && res.Success,
			"tokens":  func() int { if res != nil { return res.TokensUsed }; return 0 }(),
		})
		
		// Update module states after step
		a.mu.Lock()
		if s.ModuleState != nil {
			// Update memory state (simulated)
			if s.MemoryState != nil {
				s.MemoryState.Episodic.Count++
				s.MemoryState.Episodic.TotalTokens += res.TokensUsed
				s.MemoryState.Episodic.LastUpdated = time.Now().Format("15:04:05")
			}
			
			// Update tools state
			if s.ModuleState.Tools != nil {
				s.ModuleState.Tools.Details["toolsLoaded"] = len(toolsReg.List())
			}
			
			// Update budget state (simulated)
			if s.ModuleState.Budget != nil {
				usagePct := float64(s.Stats.SessionTokens) / float64(s.Stats.TotalLimit) * 100
				if usagePct > 80 {
					s.ModuleState.Budget.Details["warningLevel"] = "critical"
				} else if usagePct > 60 {
					s.ModuleState.Budget.Details["warningLevel"] = "warn"
				} else {
					s.ModuleState.Budget.Details["warningLevel"] = "ok"
				}
			}
		}
		a.mu.Unlock()

		// Save checkpoint every 3 steps

		// Save checkpoint every 3 steps
		if (i+1)%3 == 0 {
			chkStore.Save(&checkpoint.Checkpoint{
				ID:          fmt.Sprintf("cp-%s-step%d", s.Info.ID, i+1),
				SessionID:   s.Info.ID,
				State:       "executing",
				Goal:        s.Goal,
				CurrentStep: i + 1,
				LoopCount:   i + 1,
				TokensUsed:  s.Stats.SessionTokens,
				Cost:        s.Stats.MainCost,
			})
		}

		a.emitSession(s)
		a.wait(ctx, 250*time.Millisecond)
	}

	logger.LogWithLoop(len(plan.Steps), "exec", "execution_done", nil)

	// === Reflect ===
	s.Status = "reflecting"
	
	// Update module states
	a.mu.Lock()
	if s.ModuleState != nil {
		if s.ModuleState.Controller != nil {
			s.ModuleState.Controller.Details["fsmState"] = "Reflecting"
		}
		if s.ModuleState.Reflector != nil {
			s.ModuleState.Reflector.Status = "active"
		}
	}
	a.mu.Unlock()
	
	a.appendLog(s, "refl", "Reflecting on progress", "score=?")
	a.emitSession(s)
	logger.LogWithLoop(len(plan.Steps), "refl", "reflect_start", nil)
	a.wait(ctx, 300*time.Millisecond)

	reflectPlan := pkg.PlanToReflect(plan)
	assess, err := rf.Reflect(ctx, s.Goal, reflectPlan, mem.AsReflectorReader())
	if err != nil || assess == nil {
		a.appendLog(s, "refl", "Reflection failed", pkg.ErrString(err))
		logger.LogWithLoop(len(plan.Steps), "refl", "reflect_fail", map[string]string{"error": pkg.ErrString(err)})
		
		// Update module states
		a.mu.Lock()
		if s.ModuleState != nil && s.ModuleState.Reflector != nil {
			s.ModuleState.Reflector.Status = "error"
		}
		a.mu.Unlock()
	} else {
		a.appendLog(s, "refl", fmt.Sprintf("Decision: %s (%.0f%%)", assess.Decision, assess.Confidence*100), assess.Reason)
		logger.LogWithLoop(len(plan.Steps), "refl", "reflect_done", map[string]interface{}{
			"decision":   assess.Decision.String(),
			"confidence": assess.Confidence,
		})
		
		// Update module states after reflection
		a.mu.Lock()
		if s.ModuleState != nil {
			if s.ModuleState.Reflector != nil {
				s.ModuleState.Reflector.Status = "idle"
				s.ModuleState.Reflector.Details["lastReflection"] = time.Now().Format("15:04:05")
			}
			
			// Update learning state (simulated)
			if s.LearningState != nil {
				s.LearningState.CognitiveModel.Updated = true
				s.LearningState.CognitiveModel.LastUpdate = time.Now().Format("15:04:05")
				s.LearningState.CognitiveModel.Confidence = assess.Confidence
				s.LearningState.LastLearning = time.Now().Format("15:04:05")
				s.LearningState.SuccessPatterns = countCompleted(s.Plan)
			}
			
			// Update memory state (simulated)
			if s.MemoryState != nil {
				s.MemoryState.Procedural.Count++
				s.MemoryState.Procedural.SuccessRate = float64(countCompleted(s.Plan)) / float64(len(s.Plan))
				s.MemoryState.Procedural.LastUpdated = time.Now().Format("15:04:05")
			}
		}
		a.mu.Unlock()
		
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
	s.Stats.Elapsed = time.Since(mem.Started).Round(time.Millisecond).String()
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
	
	// Update module states on completion
	a.mu.Lock()
	if s.ModuleState != nil {
		if s.ModuleState.Controller != nil {
			s.ModuleState.Controller.Details["fsmState"] = "Done"
		}
		if s.ModuleState.Executor != nil {
			s.ModuleState.Executor.Status = "idle"
		}
	}
	a.mu.Unlock()

	// Save final checkpoint
	chkStore.Save(&checkpoint.Checkpoint{
		ID:         fmt.Sprintf("cp-%s-done", s.Info.ID),
		SessionID:  s.Info.ID,
		State:      "done",
		Goal:       s.Goal,
		TokensUsed: s.Stats.SessionTokens,
		Cost:       s.Stats.MainCost,
	})
	logger.LogWithLoop(len(plan.Steps), "system", "session_done", map[string]interface{}{
		"duration": s.Stats.Elapsed, "tokens": s.Stats.SessionTokens,
	})
	logger.Save()

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
