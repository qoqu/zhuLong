package main

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/qoqu/zhuLong/internal/planner"
)

func TestHeuristicProvider_Plan(t *testing.T) {
	p := newProvider("test")
	resp, err := p.Chat(context.Background(),
		"You are a task planner. Return JSON.",
		"Goal: 分析代码库并改进")
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if !strings.Contains(resp, "\"steps\"") {
		t.Errorf("plan response missing steps: %s", resp)
	}
	if !strings.Contains(resp, "analyze") && !strings.Contains(resp, "search_file") {
		t.Errorf("expected analyze/search_file in plan: %s", resp)
	}
}

func TestHeuristicProvider_Reflect(t *testing.T) {
	p := newProvider("test")
	resp, err := p.Chat(context.Background(),
		"You are a task reflector. Return JSON.",
		"Goal: x")
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if !strings.Contains(resp, "decision") {
		t.Errorf("reflect response missing decision: %s", resp)
	}
}

func TestHeuristicProvider_Generic(t *testing.T) {
	p := newProvider("test")
	resp, err := p.Chat(context.Background(),
		"system",
		"hello world")
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if !strings.Contains(resp, "hello world") {
		t.Errorf("generic response missing user content: %s", resp)
	}
}

func TestRunMemory_AddAndRead(t *testing.T) {
	m := newRunMemory("test goal")
	if got := m.asPlannerReader().GetSessionSummary(); !strings.Contains(got, "test goal") {
		t.Errorf("summary = %q", got)
	}

	m.AddPlannerResult(plannerStepToMemory(planner.Step{ID: "s1"}), planner.StepResult{StepID: "s1", Success: true, Output: "ok"})
	pres := m.asPlannerReader().GetStepResults()
	if len(pres) != 1 || pres[0].StepID != "s1" {
		t.Errorf("planner results: %+v", pres)
	}
	rres := m.asReflectorReader().GetStepResults()
	if len(rres) != 1 || rres[0].StepID != "s1" {
		t.Errorf("reflector results: %+v", rres)
	}
}

func TestApprovalRisk(t *testing.T) {
	cases := []struct {
		tool string
		want string
	}{
		{"read_file", "low"},
		{"search_file", "low"},
		{"write_file", "high"},
		{"execute_command", "high"},
		{"unknown", "medium"},
	}
	for _, c := range cases {
		if got := approvalRisk(c.tool, nil); got != c.want {
			t.Errorf("approvalRisk(%q) = %q, want %q", c.tool, got, c.want)
		}
	}
}

func TestIsRiskyTool(t *testing.T) {
	if !isRiskyTool("execute_command") {
		t.Error("execute_command should be risky")
	}
	if isRiskyTool("read_file") {
		t.Error("read_file should not be risky")
	}
}

func TestRunAgent_EndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	app := &App{sessions: map[string]*SessionState{}}
	app.activeID = "test"
	app.activeAge = "auto"
	app.ctx = context.Background()

	s := &SessionState{
		Info:   SessionInfo{ID: "test", Title: "test"},
		Status: "idle",
		Goal:   "分析一下代码库",
		Mode:   "yolo",
		Model:  "test",
		Created: time.Now(),
		Updated: time.Now(),
		Stats:   app.zeroStats(),
	}
	app.sessions["test"] = s

	// Heuristic plan returns ~4 steps each with ~250ms delay = ~1s +
	// any tool exec time. Give 30s for `go vet` and friends.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	done := make(chan struct{})
	go func() {
		app.RunAgent(ctx, s)
		close(done)
	}()

	select {
	case <-done:
	case <-ctx.Done():
		t.Fatal("RunAgent timed out")
	}

	if s.Status != "done" && s.Status != "error" && s.Status != "cancelled" {
		t.Errorf("final status = %q", s.Status)
	}
	if len(s.Plan) == 0 {
		t.Error("plan should not be empty")
	}
	if len(s.Messages) == 0 {
		t.Error("messages should not be empty")
	}
	hasAssistant := false
	for _, m := range s.Messages {
		if m.Role == "assistant" {
			hasAssistant = true
		}
	}
	if !hasAssistant {
		t.Error("expected at least one assistant message")
	}
	t.Logf("plan steps: %d, messages: %d, status: %s, tokens: %d",
		len(s.Plan), len(s.Messages), s.Status, s.Stats.SessionTokens)
}

func TestRunAgent_Cancelled(t *testing.T) {
	app := &App{sessions: map[string]*SessionState{}}
	app.activeID = "test"
	app.activeAge = "auto"
	app.ctx = context.Background()

	s := &SessionState{
		Info: SessionInfo{ID: "test", Title: "test"},
		Status: "idle",
		Mode: "yolo",
		Model: "test",
		Stats: app.zeroStats(),
	}
	app.sessions["test"] = s

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	app.RunAgent(ctx, s)
	if s.Status != "done" && s.Status != "error" && s.Status != "cancelled" {
		t.Errorf("final status = %q after cancel", s.Status)
	}
}

func TestConvertPlan(t *testing.T) {
	in := &planner.Plan{
		ID: "p1",
		Steps: []planner.Step{
			{ID: "s1", Description: "step one"},
			{ID: "s2", Description: "step two"},
		},
	}
	out := convertPlan(in)
	if len(out) != 2 {
		t.Fatalf("len = %d", len(out))
	}
	if out[0].ID != "s1" || out[0].Status != "pending" {
		t.Errorf("step 0 = %+v", out[0])
	}
}

func TestCountHelpers(t *testing.T) {
	steps := []PlanStepDTO{
		{Status: "completed"},
		{Status: "completed"},
		{Status: "failed"},
		{Status: "pending"},
	}
	if got := countCompleted(steps); got != 2 {
		t.Errorf("completed = %d", got)
	}
	if got := countFailed(steps); got != 1 {
		t.Errorf("failed = %d", got)
	}
}

func TestAdapterToReflectorPlan(t *testing.T) {
	p := &planner.Plan{ID: "p1"}
	p.Steps = append(p.Steps, planner.Step{ID: "s1", Description: "one"})
	rp := planToReflect(p)
	if rp == nil || rp.ID != "p1" || len(rp.Steps) != 1 {
		t.Errorf("planToReflect: %+v", rp)
	}
	if planToReflect(nil) != nil {
		t.Error("nil plan should return nil")
	}
}
