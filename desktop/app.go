// Package bridge provides the desktop bridge between the Go backend
// and the Wails-based React frontend.
//
// All public methods on App are exposed to the frontend as JS functions
// via the Wails runtime (see wails.json + Wails build pipeline).
package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App is the Wails application root. All exported methods are bound to JS.
type App struct {
	ctx    context.Context
	mu     sync.Mutex
	cancel context.CancelFunc

	// Current session
	session *SessionState

	// Cached registry (single source of truth for the UI)
	agents    []AgentInfo
	projects  []ProjectInfo
	sessions  map[string]*SessionState
	activeID  string
	activeAge string
}

// AgentInfo describes an agent identity shown in the top tab bar.
type AgentInfo struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Model string `json:"model"`
	Yolo  bool   `json:"yolo"`
}

// ProjectInfo groups sessions into a left-sidebar tree node.
type ProjectInfo struct {
	ID       string         `json:"id"`
	Name     string         `json:"name"`
	Sessions []SessionInfo  `json:"sessions"`
	Children []ProjectInfo  `json:"children,omitempty"`
}

// SessionInfo is the metadata shown in the sidebar session list.
type SessionInfo struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	AgentID      string `json:"agentId"`
	ProjectID    string `json:"projectId"`
	MessageCount int    `json:"messageCount"`
	ToolCount    int    `json:"toolCount"`
	UpdatedAt    string `json:"updatedAt"`
	Preview      string `json:"preview"`
}

// SessionState is the full runtime state of a session. Emitted to the
// frontend via the OnSessionUpdate event whenever it changes.
type SessionState struct {
	Info     SessionInfo            `json:"info"`
	Goal     string                 `json:"goal"`
	Status   string                 `json:"status"` // idle|planning|executing|...
	Mode     string                 `json:"mode"`   // ask|auto|yolo
	Model    string                 `json:"model"`
	Messages []MessageDTO           `json:"messages"`
	Logs     []LogDTO               `json:"logs"`
	Plan     []PlanStepDTO          `json:"plan"`
	Stats    RuntimeStatsDTO        `json:"stats"`
	Files    []string               `json:"files"`
	Changes  []ChangeDTO            `json:"changes"`
	Approval *ApprovalRequestDTO    `json:"approval,omitempty"`
	Created  time.Time              `json:"created"`
	Updated  time.Time              `json:"updated"`
}

// MessageDTO is a single transcript message.
type MessageDTO struct {
	ID        string    `json:"id"`
	Role      string    `json:"role"` // user|assistant|system|tool
	Content   string    `json:"content"`
	Time      time.Time `json:"time"`
	ToolName  string    `json:"toolName,omitempty"`
	ToolCount int       `json:"toolCount,omitempty"`
}

// LogDTO is a runtime log line.
type LogDTO struct {
	ID     string `json:"id"`
	Time   string `json:"time"`
	Phase  string `json:"phase"`
	Event  string `json:"event"`
	Detail string `json:"detail,omitempty"`
}

// PlanStepDTO is a single plan row.
type PlanStepDTO struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Status      string `json:"status"` // pending|running|completed|failed
}

// RuntimeStatsDTO is the right-panel overview payload.
type RuntimeStatsDTO struct {
	TotalUsed      int     `json:"totalUsed"`
	TotalLimit     int     `json:"totalLimit"`
	UsagePercent   float64 `json:"usagePercent"`
	Prompt         int     `json:"prompt"`
	Completion     int     `json:"completion"`
	Reasoning      int     `json:"reasoning"`
	Other          int     `json:"other"`
	Elapsed        string  `json:"elapsed"`
	RequestCount   int     `json:"requestCount"`
	SessionTokens  int     `json:"sessionTokens"`
	CacheHitRatio  float64 `json:"cacheHitRatio"`
	MainCost       float64 `json:"mainCost"`
	MainCount      int     `json:"mainCount"`
	SubCost        float64 `json:"subCost"`
	SubCount       int     `json:"subCount"`
	Balance        string  `json:"balance"`
	CurrentSession int     `json:"currentSession"`
	SessionCost    string  `json:"sessionCost"`
	Model          string  `json:"model"`
	CacheHit       string  `json:"cacheHit"`
	AvgHit         string  `json:"avgHit"`
	ThisTokens     int     `json:"thisTokens"`
	ThisFee        string  `json:"thisFee"`
	ContextUsed    int     `json:"contextUsed"`
	CompressPct    int     `json:"compressPct"`
	Remaining      string  `json:"remaining"`
}

// ChangeDTO is a file-system change event from the EnvironmentMonitor.
type ChangeDTO struct {
	Path string `json:"path"`
	Kind string `json:"kind"` // created|modified|deleted
	Time string `json:"time"`
}

// ApprovalRequestDTO is rendered as a modal in the UI.
type ApprovalRequestDTO struct {
	ID        string            `json:"id"`
	Tool      string            `json:"tool"`
	Args      map[string]any    `json:"args"`
	Risk      string            `json:"risk"` // low|medium|high
	Reason    string            `json:"reason"`
	CreatedAt time.Time         `json:"createdAt"`
}

// NewApp creates a new App with sane defaults.
func NewApp() *App {
	a := &App{
		sessions: map[string]*SessionState{},
	}
	a.seedDefaults()
	return a
}

func (a *App) seedDefaults() {
	a.agents = []AgentInfo{
		{ID: "auto", Name: "默认 Agent", Model: "deepseek-v4-flash", Yolo: true},
	}
	a.projects = []ProjectInfo{
		{ID: "global", Name: "Global"},
	}
	a.activeID = ""
	a.activeAge = "auto"
}

// startup is called by Wails at app start.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.markWailsReady()
}

// === Sidebar / agents ===

// ListAgents returns all known agents for the top tab bar.
func (a *App) ListAgents() []AgentInfo { return a.agents }

// ListProjects returns the sidebar project/session tree.
func (a *App) ListProjects() []ProjectInfo { return a.projects }

// SetActiveAgent switches the active agent (persists across sessions).
func (a *App) SetActiveAgent(id string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.activeAge = id
}

// SetActiveSession switches the active session in the sidebar.
func (a *App) SetActiveSession(id string) *SessionState {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.activeID = id
	if s, ok := a.sessions[id]; ok {
		return s
	}
	// Lazy materialise from project tree
	for _, p := range a.projects {
		for _, si := range p.Sessions {
			if si.ID == id {
				s := &SessionState{
					Info:    si,
					Status:  "idle",
					Mode:    "auto",
					Model:   a.modelFor(si.AgentID),
					Created: time.Now(),
					Updated: time.Now(),
				}
				a.sessions[id] = s
				return s
			}
		}
	}
	return nil
}

func (a *App) modelFor(agentID string) string {
	for _, ag := range a.agents {
		if ag.ID == agentID {
			return ag.Model
		}
	}
	return "deepseek-v4-flash"
}

// === Composer / chat ===

// NewSession creates a fresh empty session and returns its state.
func (a *App) NewSession() *SessionState {
	a.mu.Lock()
	defer a.mu.Unlock()
	id := fmt.Sprintf("s%d", time.Now().UnixMilli())
	info := SessionInfo{
		ID: id, Title: "新会话", AgentID: a.activeAge, ProjectID: "global",
		MessageCount: 0, ToolCount: 0, UpdatedAt: "刚刚", Preview: "新会话",
	}
	st := &SessionState{
		Info: info, Status: "idle", Mode: "auto", Model: a.modelFor(a.activeAge),
		Messages: []MessageDTO{}, Logs: []LogDTO{}, Plan: []PlanStepDTO{},
		Created: time.Now(), Updated: time.Now(),
		Stats: a.zeroStats(),
	}
	a.sessions[id] = st
	// Inject into Global project
	for i := range a.projects {
		if a.projects[i].ID == "global" {
			a.projects[i].Sessions = append([]SessionInfo{info}, a.projects[i].Sessions...)
			break
		}
	}
	a.activeID = id
	a.emitProjects()
	runtime.EventsEmit(a.ctx, "session:created", st)
	return st
}

// DeleteSession removes a session.
func (a *App) DeleteSession(id string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.sessions, id)
	for i := range a.projects {
		filtered := a.projects[i].Sessions[:0]
		for _, s := range a.projects[i].Sessions {
			if s.ID != id {
				filtered = append(filtered, s)
			}
		}
		a.projects[i].Sessions = filtered
	}
	if a.activeID == id {
		for _, p := range a.projects {
			if len(p.Sessions) > 0 {
				a.activeID = p.Sessions[0].ID
				break
			}
		}
	}
	a.emitProjects()
	runtime.EventsEmit(a.ctx, "session:deleted", id)
}

// RenameSession updates the title of a session.
func (a *App) RenameSession(id, title string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	for i := range a.projects {
		for j := range a.projects[i].Sessions {
			if a.projects[i].Sessions[j].ID == id {
				a.projects[i].Sessions[j].Title = title
				a.projects[i].Sessions[j].Preview = title
				if s, ok := a.sessions[id]; ok {
					s.Info.Title = title
					s.Info.Preview = title
				}
				break
			}
		}
	}
	a.emitProjects()
}

// GetSession returns the full state of a session.
func (a *App) GetSession(id string) *SessionState {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.sessions[id]
}

// SendMessage pushes a user message into a session and starts a simulated
// agent run. In the real product this hands off to Controller.Run.
func (a *App) SendMessage(sessionID, text string) {
	a.mu.Lock()
	s, ok := a.sessions[sessionID]
	if !ok {
		a.mu.Unlock()
		return
	}
	s.Goal = text
	s.Status = "executing"
	s.Updated = time.Now()
	msg := MessageDTO{
		ID: fmt.Sprintf("m%d", time.Now().UnixNano()), Role: "user", Content: text, Time: time.Now(),
	}
	s.Messages = append(s.Messages, msg)
	s.Info.MessageCount = len(s.Messages)
	s.Info.Preview = truncate(text, 40)
	s.Info.UpdatedAt = "刚刚"
	a.mu.Unlock()
	a.emitProjects()
	a.emitSession(s)

	// Cancel any previous run.
	a.mu.Lock()
	if a.cancel != nil {
		a.cancel()
	}
	runCtx, cancel := context.WithCancel(a.ctx)
	a.cancel = cancel
	a.mu.Unlock()

	go a.RunAgent(runCtx, s)
}

func (a *App) wait(ctx context.Context, d time.Duration) {
	select {
	case <-ctx.Done():
	case <-time.After(d):
	}
}

// Stop cancels the current run.
func (a *App) Stop() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.cancel != nil {
		a.cancel()
		a.cancel = nil
	}
	if s, ok := a.sessions[a.activeID]; ok {
		s.Status = "idle"
		a.emitSession(s)
	}
}

// Reset clears the active session back to empty.
func (a *App) Reset() {
	a.mu.Lock()
	defer a.mu.Unlock()
	s, ok := a.sessions[a.activeID]
	if !ok {
		return
	}
	s.Messages = nil
	s.Logs = nil
	s.Plan = nil
	s.Status = "idle"
	s.Goal = ""
	s.Stats = a.zeroStats()
	a.emitSession(s)
}

// === Modes / settings ===

// SetExecutionMode updates the mode for the active session.
func (a *App) SetExecutionMode(mode string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if s, ok := a.sessions[a.activeID]; ok {
		s.Mode = mode
		a.emitSession(s)
	}
}

// SetModel updates the model for the active session.
func (a *App) SetModel(model string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if s, ok := a.sessions[a.activeID]; ok {
		s.Model = model
		s.Stats.Model = model
		a.emitSession(s)
	}
}

// === Approval ===

// RespondApproval answers a pending approval request. approved=true →
// resume, false → abort current step.
func (a *App) RespondApproval(id string, approved bool) {
	a.mu.Lock()
	s, ok := a.sessions[a.activeID]
	if !ok || s.Approval == nil || s.Approval.ID != id {
		a.mu.Unlock()
		return
	}
	if !approved {
		s.Status = "reflecting"
		s.Plan[2].Status = "failed"
		s.Messages = append(s.Messages, MessageDTO{
			ID: fmt.Sprintf("m%d", time.Now().UnixNano()),
			Role: "system",
			Content: "用户拒绝了操作：" + s.Approval.Tool,
			Time: time.Now(),
		})
	} else {
		s.Status = "executing"
	}
	s.Approval = nil
	a.mu.Unlock()
	a.emitSession(s)
}

// === Helpers ===

func (a *App) appendLog(s *SessionState, phase, event, detail string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	s.Logs = append(s.Logs, LogDTO{
		ID:     fmt.Sprintf("l%d", time.Now().UnixNano()),
		Time:   time.Now().Format("15:04:05"),
		Phase:  phase, Event: event, Detail: detail,
	})
	if len(s.Logs) > 200 {
		s.Logs = s.Logs[len(s.Logs)-200:]
	}
}

func (a *App) zeroStats() RuntimeStatsDTO {
	return RuntimeStatsDTO{
		TotalLimit:    1_000_000,
		CacheHitRatio: 0.8,
		Balance:       "¥73.23",
		Model:         a.modelFor(a.activeAge),
		CacheHit:      "未命中",
		AvgHit:        "未命中",
		SessionCost:   "$0.0000",
		Remaining:     "¥73.23",
	}
}

func (a *App) emitSession(s *SessionState) {
	if a.ctx == nil {
		return
	}
	if !hasWailsEvents(a.ctx) {
		return
	}
	runtime.EventsEmit(a.ctx, "session:update", s)
}

func (a *App) emitProjects() {
	if a.ctx == nil {
		return
	}
	if !hasWailsEvents(a.ctx) {
		return
	}
	runtime.EventsEmit(a.ctx, "projects:update", a.projects)
}

// hasWailsEvents checks whether the context carries a wails Events
// implementation. We use a type-assertion on the context value via a
// private sentinel that the wails runtime sets; absent that, we fall
// back to "not wails" so unit tests do not crash on log.Fatal.
func hasWailsEvents(ctx context.Context) bool {
	type wailsCtxKey struct{}
	// The wails runtime stores a *frontend.events on the context. We
	// can't import that package without an import cycle, so we
	// instead use a side-channel: a magic value in a known location
	// is set by our startup hook. For tests, that value is missing.
	v := ctx.Value(wailsReadyKey{})
	if v == nil {
		return false
	}
	if b, ok := v.(bool); ok && b {
		return true
	}
	return false
}

type wailsReadyKey struct{}

func (a *App) markWailsReady() {
	if a.ctx == nil {
		return
	}
	a.ctx = context.WithValue(a.ctx, wailsReadyKey{}, true)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// === Stats ===

// fmtState returns the JSON-encoded stats (used for debugging).
var _ = fmt.Sprintf
