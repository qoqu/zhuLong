// Package bridge provides the desktop bridge between the Go backend
// and the Wails-based React frontend.
//
// All public methods on App are exposed to the frontend as JS functions
// via the Wails runtime (see wails.json + Wails build pipeline).
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/qoqu/zhuLong/internal/backup"
	"github.com/qoqu/zhuLong/internal/dashboard"
	"github.com/qoqu/zhuLong/internal/environment"
	"github.com/qoqu/zhuLong/internal/plugins"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// AppConfig 应用配置（第四步：可被前端修改）
type AppConfig struct {
	MonitorPath  string `json:"monitorPath"`
	BackupMode   string `json:"backupMode"`   // immediate | on-completion | off
	BackupOnFail bool   `json:"backupOnFail"`
	DashboardPort int   `json:"dashboardPort"`
	PluginsPath  string `json:"pluginsPath"`
}

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

	// 应用配置（第四步：前端可改）
	config AppConfig

	// P3 模块实例
	dashboardSrv *dashboard.Dashboard
	pluginMgr    *plugins.Manager
	backupMgr    *backup.Manager

	// P2 envMonitor
	envMonitor   *environment.Monitor
	envStop      chan struct{}
	envRunning   bool
	envChangeBuf []ChangeDTO
	envChangeLock sync.Mutex
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
	Info        SessionInfo         `json:"info"`
	Goal        string              `json:"goal"`
	Status      string              `json:"status"` // idle|planning|executing|...
	Mode        string              `json:"mode"`   // ask|auto|yolo
	Temperature string              `json:"temperature"` // auto|0.0|0.3|0.7|1.0
	Model       string              `json:"model"`
	Messages    []MessageDTO        `json:"messages"`
	Logs        []LogDTO            `json:"logs"`
	Plan        []PlanStepDTO       `json:"plan"`
	Stats       RuntimeStatsDTO     `json:"stats"`
	Files       []string            `json:"files"`
	Changes     []ChangeDTO         `json:"changes"`
	Approval    *ApprovalRequestDTO `json:"approval,omitempty"`
	Created     time.Time           `json:"created"`
	Updated     time.Time           `json:"updated"`
	// New fields for enhanced UI
	MemoryState   *MemoryStateDTO   `json:"memoryState,omitempty"`
	LearningState *LearningStateDTO `json:"learningState,omitempty"`
	ModuleState   *ModuleStateDTO   `json:"moduleState,omitempty"`
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
	// 同步 FSM 状态（Controller.Details.fsmState 的镜像，给 StatusBar 显示）
	FsmState string `json:"fsmState,omitempty"`
	// 预算联动字段
	BudgetUsed     int  `json:"budgetUsed,omitempty"`
	BudgetLimit    int  `json:"budgetLimit,omitempty"`
	BudgetWarning  bool `json:"budgetWarning,omitempty"`
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

// MemoryStateDTO is the memory state sent to the frontend.
type MemoryStateDTO struct {
	Episodic   MemoryEpisodicDTO   `json:"episodic"`
	Semantic   MemorySemanticDTO   `json:"semantic"`
	Procedural MemoryProceduralDTO `json:"procedural"`
}

type MemoryEpisodicDTO struct {
	Count       int    `json:"count"`
	TotalTokens int    `json:"totalTokens"`
	LastUpdated string `json:"lastUpdated"`
}

type MemorySemanticDTO struct {
	Count       int      `json:"count"`
	Categories  []string `json:"categories"`
	LastUpdated string   `json:"lastUpdated"`
}

type MemoryProceduralDTO struct {
	Count        int     `json:"count"`
	SuccessRate  float64 `json:"successRate"`
	LastUpdated  string  `json:"lastUpdated"`
}

// LearningStateDTO is the learning state sent to the frontend.
type LearningStateDTO struct {
	CognitiveModel  LearningCognitiveModelDTO `json:"cognitiveModel"`
	Diversity      LearningDiversityDTO      `json:"diversity"`
	ExplorationRate float64               `json:"explorationRate"`
	UtilizationRate float64               `json:"utilizationRate"`
	SuccessPatterns  int                  `json:"successPatterns"`
	LastLearning    string                `json:"lastLearning"`
}

type LearningCognitiveModelDTO struct {
	Updated     bool    `json:"updated"`
	LastUpdate  string  `json:"lastUpdate"`
	Confidence  float64 `json:"confidence"`
}

type LearningDiversityDTO struct {
	Score      float64  `json:"score"`
	Strategies []string `json:"strategies"`
}

// ModuleStateDTO is the module state sent to the frontend.
type ModuleStateDTO struct {
	// P0 modules
	Controller *ModuleItemDTO `json:"controller,omitempty"`
	Planner    *ModuleItemDTO `json:"planner,omitempty"`
	Executor   *ModuleItemDTO `json:"executor,omitempty"`
	Reflector  *ModuleItemDTO `json:"reflector,omitempty"`
	Memory     *ModuleItemDTO `json:"memory,omitempty"`
	Compressor *ModuleItemDTO `json:"compressor,omitempty"`
	Checkpoint *ModuleItemDTO `json:"checkpoint,omitempty"`
	Budget     *ModuleItemDTO `json:"budget,omitempty"`
	Trace      *ModuleItemDTO `json:"trace,omitempty"`
	Human      *ModuleItemDTO `json:"human,omitempty"`
	Tools      *ModuleItemDTO `json:"tools,omitempty"`
	DeepSeek   *ModuleItemDTO `json:"deepseek,omitempty"`
	
	// P1 modules
	Stagnation  *ModuleItemDTO `json:"stagnation,omitempty"`
	Exploration *ModuleItemDTO `json:"exploration,omitempty"`
	Stability   *ModuleItemDTO `json:"stability,omitempty"`
	Information *ModuleItemDTO `json:"information,omitempty"`
	Synergetics *ModuleItemDTO `json:"synergetics,omitempty"`
	Learning    *ModuleItemDTO `json:"learningModule,omitempty"`
	
	// P2 modules
	AltPlanner *ModuleItemDTO `json:"altPlanner,omitempty"`
	EnvMonitor *ModuleItemDTO `json:"envMonitor,omitempty"`
	NoiseHandler *ModuleItemDTO `json:"noiseHandler,omitempty"`
	Redundancy  *ModuleItemDTO `json:"redundancy,omitempty"`
	
	// P3 modules
	I18N      *ModuleItemDTO `json:"i18n,omitempty"`
	Plugins   *ModuleItemDTO `json:"plugins,omitempty"`
	Dashboard *ModuleItemDTO `json:"dashboard,omitempty"`
	Models    *ModuleItemDTO `json:"models,omitempty"`
	Backup    *ModuleItemDTO `json:"backup,omitempty"`
}

type ModuleItemDTO struct {
	Status  string         `json:"status"` // active|idle|error
	Details map[string]any `json:"details,omitempty"`
}

// NewApp creates a new App with sane defaults.
func NewApp() *App {
	a := &App{
		sessions: map[string]*SessionState{},
		config: AppConfig{
			MonitorPath:   ".",
			BackupMode:    "on-completion",
			BackupOnFail:  false,
			DashboardPort: 7788,
			PluginsPath:   filepath.Join(os.TempDir(), "zhulong-plugins"),
		},
	}
	a.seedDefaults()
	a.initPlugins()
	a.initBackup()
	return a
}

func (a *App) initPlugins() {
	a.pluginMgr = plugins.NewManager()
	// 真实加载目录中的插件
	if loaded, _ := a.pluginMgr.LoadFromDir(a.config.PluginsPath); len(loaded) > 0 {
		fmt.Printf("[plugins] loaded: %v\n", loaded)
	}
}

func (a *App) scanPluginsDir(dir string) {
	if _, err := os.Stat(dir); err != nil {
		return // 目录不存在
	}
	// 真实加载 + 同步到活跃 session 的 Plugins 模块
	loaded, _ := a.pluginMgr.LoadFromDir(dir)
	a.mu.Lock()
	if s, ok := a.sessions[a.activeID]; ok && s.ModuleState != nil && s.ModuleState.Plugins != nil {
		s.ModuleState.Plugins.Details["loadedPlugins"] = len(loaded)
		infos := a.pluginMgr.List()
		names := make([]string, 0, len(infos))
		for _, p := range infos {
			names = append(names, p.Name)
		}
		s.ModuleState.Plugins.Details["pluginList"] = names
		if len(loaded) > 0 {
			s.ModuleState.Plugins.Status = "active"
		}
	}
	a.mu.Unlock()
}

func (a *App) initBackup() {
	backupDir := filepath.Join(os.TempDir(), "zhulong-backup")
	os.MkdirAll(backupDir, 0755)
	a.backupMgr = backup.NewManager(backupDir)
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
	a.initEnvironmentMonitor()
}

// initEnvironmentMonitor 启动 P2 模块：envMonitor 监听当前目录变化
// 变化通过 changes:update 事件实时推送到前端 ChangesTab
func (a *App) initEnvironmentMonitor() {
	if a.envRunning {
		return
	}
	a.envMonitor = environment.NewMonitor()
	a.envMonitor.AddWatcher(environment.WatcherConfig{
		Path:         a.config.MonitorPath,
		Recursive:    true,
		PollInterval: 60 * time.Second,
	})
	if err := a.envMonitor.Start(); err != nil {
		return
	}
	a.envStop = make(chan struct{})
	a.envChangeBuf = make([]ChangeDTO, 0)
	a.envRunning = true

	go a.envMonitorPushLoop()
}

func (a *App) envMonitorPushLoop() {
	ch := a.envMonitor.GetChanges()
	for {
		select {
		case <-a.envStop:
			return
		case c, ok := <-ch:
			if !ok {
				return
			}
			a.handleEnvChange(c)
		}
	}
}

func (a *App) handleEnvChange(c environment.Change) {
	dto := ChangeDTO{
		Path: c.Path,
		Kind: c.Type.String(),
		Time: c.Timestamp.Format("15:04:05"),
	}

	// 1) 缓存到活跃 session 的 Changes
	a.mu.Lock()
	if s, ok := a.sessions[a.activeID]; ok {
		s.Changes = append(s.Changes, dto)
		if len(s.Changes) > 200 {
			s.Changes = s.Changes[len(s.Changes)-200:]
		}
	}

	// 2) 更新 envMonitor 模块状态
	envChangesCount := 0
	if s, ok := a.sessions[a.activeID]; ok && s.ModuleState != nil && s.ModuleState.EnvMonitor != nil {
		s.ModuleState.EnvMonitor.Details["polling"] = true
		// 累加计数
		if v, ok := s.ModuleState.EnvMonitor.Details["changesDetected"].(int); ok {
			envChangesCount = v + 1
		} else {
			envChangesCount = 1
		}
		s.ModuleState.EnvMonitor.Details["changesDetected"] = envChangesCount
		s.ModuleState.EnvMonitor.Status = "active"
	}
	a.mu.Unlock()

	// 3) 推送事件给前端
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "changes:update", dto)
		if s, ok := a.sessions[a.activeID]; ok {
			runtime.EventsEmit(a.ctx, "session:update", s)
		}
	}
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
		Info:        info,
		Status:      "idle",
		Mode:        "auto",
		Temperature: "auto",
		Model:       a.modelFor(a.activeAge),
		Messages:    []MessageDTO{},
		Logs:        []LogDTO{},
		Plan:        []PlanStepDTO{},
		Created:     time.Now(),
		Updated:     time.Now(),
		Stats:       a.zeroStats(),
		MemoryState:   a.initMemoryState(),
		LearningState: a.initLearningState(),
		ModuleState:   a.initModuleState(),
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

// SetTemperature updates the sampling temperature for the active session.
// "auto" | "0.0" | "0.3" | "0.7" | "1.0" — 影响 provider 输出多样性
func (a *App) SetTemperature(temp string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if s, ok := a.sessions[a.activeID]; ok {
		s.Temperature = temp
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

// === File tree ===

// TreeNode is a single entry in the workspace file tree sent to the frontend.
type TreeNode struct {
	Name     string     `json:"name"`
	Path     string     `json:"path"`
	IsDir    bool       `json:"isDir"`
	Children []TreeNode `json:"children,omitempty"`
	Size     int64      `json:"size,omitempty"`
}

// ListWorkspaceTree returns the directory tree rooted at the given path,
// pruned to maxDepth levels and skipping common directories that clutter
// the view (node_modules, .git, dist, build, vendor, .idea, .vscode).
func (a *App) ListWorkspaceTree(root string, maxDepth int) TreeNode {
	if maxDepth <= 0 {
		maxDepth = 4
	}
	if root == "" {
		root = "."
	}
	info, err := os.Stat(root)
	if err != nil {
		return TreeNode{Name: filepath.Base(root), Path: root, IsDir: true}
	}
	if !info.IsDir() {
		return TreeNode{Name: info.Name(), Path: root, Size: info.Size()}
	}
	return buildTree(root, "", 0, maxDepth)
}

var skipDirs = map[string]bool{
	".git":       true,
	"node_modules": true,
	"dist":       true,
	"build":      true,
	"vendor":     true,
	".idea":      true,
	".vscode":    true,
	"__pycache__": true,
	".workbuddy": true,
}

func buildTree(abs, rel string, depth, maxDepth int) TreeNode {
	name := filepath.Base(abs)
	if rel == "" {
		name = "."
	}
	node := TreeNode{Name: name, Path: rel, IsDir: true}
	if depth >= maxDepth {
		return node
	}
	entries, err := os.ReadDir(abs)
	if err != nil {
		return node
	}
	// Sort: dirs first, then files
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir() != entries[j].IsDir() {
			return entries[i].IsDir()
		}
		return entries[i].Name() < entries[j].Name()
	})
	for _, e := range entries {
		if skipDirs[e.Name()] {
			continue
		}
		childRel := e.Name()
		if rel != "" {
			childRel = rel + "/" + e.Name()
		}
		childAbs := filepath.Join(abs, e.Name())
		if e.IsDir() {
			node.Children = append(node.Children, buildTree(childAbs, childRel, depth+1, maxDepth))
		} else {
			info, _ := e.Info()
			var size int64
			if info != nil {
				size = info.Size()
			}
			node.Children = append(node.Children, TreeNode{
				Name:  e.Name(),
				Path:  childRel,
				IsDir: false,
				Size:  size,
			})
		}
	}
	return node
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

// === Configuration (第四步) ===

// GetConfig 返回应用配置
func (a *App) GetConfig() AppConfig {
	return a.config
}

// SetMonitorPath 修改 envMonitor 监听路径（重启监控）
func (a *App) SetMonitorPath(path string) error {
	a.mu.Lock()
	a.config.MonitorPath = path
	// 停止旧监控
	if a.envMonitor != nil && a.envRunning {
		a.envMonitor.Stop()
		close(a.envStop)
		a.envRunning = false
	}
	a.mu.Unlock()
	// 启动新监控
	a.initEnvironmentMonitor()
	return nil
}

// SetBackupMode 修改 backup 触发模式
//   immediate: 每步后
//   on-completion: 完成时（仅成功）
//   off: 关闭
func (a *App) SetBackupMode(mode string, onFail bool) {
	a.mu.Lock()
	a.config.BackupMode = mode
	a.config.BackupOnFail = onFail
	a.mu.Unlock()
	a.emitSessionForActive()
}

// SetDashboardPort 修改 Dashboard 端口
func (a *App) SetDashboardPort(port int) error {
	a.mu.Lock()
	a.config.DashboardPort = port
	running := a.dashboardSrv != nil
	a.mu.Unlock()
	if running {
		// 重启 dashboard
		_ = a.StopDashboard()
		return a.StartDashboard()
	}
	return nil
}

// SetPluginsPath 修改 plugins 目录
func (a *App) SetPluginsPath(path string) {
	a.mu.Lock()
	a.config.PluginsPath = path
	a.mu.Unlock()
	a.scanPluginsDir(path)
	a.emitSessionForActive()
}

// StartDashboard 启动 Dashboard HTTP 服务
func (a *App) StartDashboard() error {
	if a.dashboardSrv != nil {
		return nil
	}
	cfg := &dashboard.Config{
		Port:     a.config.DashboardPort,
		DataDir:  filepath.Join(os.TempDir(), "zhulong-dashboard"),
		Username: "admin",
		Password: "",
	}
	d := dashboard.New(cfg)
	// 注册所有已知模块
	d.RegisterModule("controller", "0.4.0")
	d.RegisterModule("planner", "0.4.0")
	d.RegisterModule("executor", "0.4.0")
	d.RegisterModule("reflector", "0.4.0")
	d.RegisterModule("memory", "0.4.0")
	d.RegisterModule("compressor", "0.4.0")
	d.RegisterModule("checkpoint", "0.4.0")
	d.RegisterModule("budget", "0.4.0")
	d.RegisterModule("trace", "0.4.0")
	d.RegisterModule("deepseek", "0.4.0")
	// 注入 session provider
	d.SetSessionProvider(func() interface{} {
		a.mu.Lock()
		defer a.mu.Unlock()
		out := make([]*SessionState, 0, len(a.sessions))
		for _, s := range a.sessions {
			out = append(out, s)
		}
		return out
	})
	if err := d.Start(); err != nil {
		return err
	}
	a.mu.Lock()
	a.dashboardSrv = d
	if s, ok := a.sessions[a.activeID]; ok && s.ModuleState != nil && s.ModuleState.Dashboard != nil {
		s.ModuleState.Dashboard.Status = "active"
		s.ModuleState.Dashboard.Details["enabled"] = true
		s.ModuleState.Dashboard.Details["port"] = a.config.DashboardPort
		s.ModuleState.Dashboard.Details["url"] = fmt.Sprintf("http://localhost:%d", a.config.DashboardPort)
	}
	a.mu.Unlock()
	a.emitSessionForActive()
	return nil
}

// StopDashboard 停止 Dashboard
func (a *App) StopDashboard() error {
	a.mu.Lock()
	d := a.dashboardSrv
	a.dashboardSrv = nil
	if s, ok := a.sessions[a.activeID]; ok && s.ModuleState != nil && s.ModuleState.Dashboard != nil {
		s.ModuleState.Dashboard.Status = "idle"
		s.ModuleState.Dashboard.Details["enabled"] = false
	}
	a.mu.Unlock()
	if d != nil {
		// Dashboard 没有 Stop 方法 - 实际由其 http.Server 关闭
		// 这里只标记状态，不强行关闭
		a.emitSessionForActive()
	}
	return nil
}

// TriggerBackup 手动触发一次备份
func (a *App) TriggerBackup() error {
	if a.backupMgr == nil {
		return fmt.Errorf("backup manager not initialized")
	}
	name := fmt.Sprintf("manual-%s", time.Now().Format("20060102-150405"))
	if _, err := a.backupMgr.Create(name, []string{"./desktop"}); err != nil {
		return err
	}
	a.mu.Lock()
	if s, ok := a.sessions[a.activeID]; ok && s.ModuleState != nil && s.ModuleState.Backup != nil {
		cnt, _ := s.ModuleState.Backup.Details["snapshotCount"].(int)
		s.ModuleState.Backup.Details["snapshotCount"] = cnt + 1
		s.ModuleState.Backup.Details["lastSnapshot"] = time.Now().Format("15:04:05")
		s.ModuleState.Backup.Status = "active"
	}
	a.mu.Unlock()
	a.emitSessionForActive()
	return nil
}

func (a *App) emitSessionForActive() {
	a.mu.Lock()
	s, ok := a.sessions[a.activeID]
	a.mu.Unlock()
	if ok {
		a.emitSession(s)
	}
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

func (a *App) initMemoryState() *MemoryStateDTO {
	return &MemoryStateDTO{
		Episodic:  MemoryEpisodicDTO{Count: 0, TotalTokens: 0, LastUpdated: "从未"},
		Semantic:  MemorySemanticDTO{Count: 0, Categories: []string{}, LastUpdated: "从未"},
		Procedural: MemoryProceduralDTO{Count: 0, SuccessRate: 0.0, LastUpdated: "从未"},
	}
}

func (a *App) initLearningState() *LearningStateDTO {
	return &LearningStateDTO{
		CognitiveModel:  LearningCognitiveModelDTO{Updated: false, LastUpdate: "从未", Confidence: 0.0},
		Diversity:      LearningDiversityDTO{Score: 0.0, Strategies: []string{}},
		ExplorationRate: 0.3,
		UtilizationRate: 0.7,
		SuccessPatterns:  0,
		LastLearning:    "从未",
	}
}

func (a *App) initModuleState() *ModuleStateDTO {
	return &ModuleStateDTO{
		// === P0 核心模块（12 个） ===
		Controller: &ModuleItemDTO{Status: "idle", Details: map[string]any{"fsmState": "Idle", "loop": 0}},
		Planner:    &ModuleItemDTO{Status: "idle", Details: map[string]any{"lastPlan": "未规划"}},
		Executor:   &ModuleItemDTO{Status: "idle", Details: map[string]any{"toolsLoaded": 0, "currentStep": 0}},
		Reflector:  &ModuleItemDTO{Status: "idle", Details: map[string]any{"lastReflection": "从未"}},
		Memory:     &ModuleItemDTO{Status: "idle", Details: map[string]any{"compactionEnabled": true, "itemsCount": 0}},
		Compressor: &ModuleItemDTO{Status: "idle", Details: map[string]any{"compressThreshold": 0.8, "lastPruneAt": "从未"}},
		Checkpoint: &ModuleItemDTO{Status: "idle", Details: map[string]any{"lastCheckpointId": "", "restoredFrom": ""}},
		Budget:     &ModuleItemDTO{Status: "idle", Details: map[string]any{"warningLevel": "ok", "tokensUsed": 0, "tokensLimit": 1000000}},
		Trace:      &ModuleItemDTO{Status: "idle", Details: map[string]any{"traceEnabled": true, "eventsLogged": 0}},
		Human:      &ModuleItemDTO{Status: "idle", Details: map[string]any{"approvalPending": false, "breakpointCount": 0}},
		Tools:      &ModuleItemDTO{Status: "idle", Details: map[string]any{"mcpConnected": false, "toolsLoaded": 0}},
		DeepSeek:   &ModuleItemDTO{Status: "idle", Details: map[string]any{"cacheHitRate": 0.0, "prefixCacheStable": true}},

		// === P1 核心增强（6 个） ===
		Stagnation:  &ModuleItemDTO{Status: "idle", Details: map[string]any{"isStagnating": false, "consecutiveNoProgress": 0, "lastCheckAt": "从未"}},
		Exploration: &ModuleItemDTO{Status: "idle", Details: map[string]any{"explorationRate": 0.3, "lastTriggered": "从未", "triggerCount": 0}},
		Stability:   &ModuleItemDTO{Status: "idle", Details: map[string]any{"isOscillating": false, "divergenceScore": 0.0, "snapshotsCount": 0}},
		Information: &ModuleItemDTO{Status: "idle", Details: map[string]any{"avgGain": 0.0, "toolsTracked": 0, "lastGain": 0.0}},
		Synergetics: &ModuleItemDTO{Status: "idle", Details: map[string]any{"orderParameter": "—", "slavedCount": 0, "misalignedCount": 0}},
		Learning:    &ModuleItemDTO{Status: "idle", Details: map[string]any{"buildingBlocksCount": 0, "diversityScore": 0.0, "patternsSaved": 0}},

		// === P2 扩展模块（4 个） ===
		AltPlanner:   &ModuleItemDTO{Status: "idle", Details: map[string]any{"alternativesGenerated": 0, "lastFallbackAt": "从未"}},
		EnvMonitor:   &ModuleItemDTO{Status: "idle", Details: map[string]any{"polling": false, "pollIntervalSec": 60, "changesDetected": 0}},
		NoiseHandler: &ModuleItemDTO{Status: "idle", Details: map[string]any{"noiseFiltered": 0, "redundancyMerged": 0}},
		Redundancy:   &ModuleItemDTO{Status: "idle", Details: map[string]any{"duplicatesRemoved": 0, "dedupRatio": 0.0}},

		// === P3 增强模块（5 个） ===
		I18N:      &ModuleItemDTO{Status: "idle", Details: map[string]any{"currentLang": "zh", "bundleLoaded": true, "keysCount": 0}},
		Plugins:   &ModuleItemDTO{Status: "idle", Details: map[string]any{"loadedPlugins": 0, "pluginList": []string{}}},
		Dashboard: &ModuleItemDTO{Status: "idle", Details: map[string]any{"enabled": false, "port": 0, "url": ""}},
		Models:    &ModuleItemDTO{Status: "idle", Details: map[string]any{"poolSize": 0, "availableModels": []string{"deepseek-v4-flash"}}},
		Backup:    &ModuleItemDTO{Status: "idle", Details: map[string]any{"autoBackupEnabled": false, "lastSnapshot": "从未", "snapshotCount": 0}},
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
