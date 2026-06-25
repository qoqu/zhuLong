// Package bridge provides the desktop bridge between the Go backend
// and the Wails-based React frontend.
//
// All public methods on App are exposed to the frontend as JS functions
// via the Wails runtime (see wails.json + Wails build pipeline).
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/qoqu/zhuLong/internal/backup"
	"github.com/qoqu/zhuLong/internal/bot"
	"github.com/qoqu/zhuLong/internal/dashboard"
	"github.com/qoqu/zhuLong/internal/environment"
	"github.com/qoqu/zhuLong/internal/mcp"
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
	Bots         map[string]bot.AdapterConfig `json:"bots"`

	// DeepSeek
	DeepSeekModel   string  `json:"deepseekModel"`
	DeepSeekBaseURL string  `json:"deepseekBaseUrl"`
	Temperature     float64 `json:"temperature"`
	MaxTokens       int     `json:"maxTokens"`

	// 循环控制
	MaxLoops    int    `json:"maxLoops"`
	MaxWallTime string `json:"maxWallTime"` // duration string like "30m"
	CheckpointEvery int `json:"checkpointEvery"`

	// 成本控制
	BudgetMaxTokens int     `json:"budgetMaxTokens"`
	BudgetMaxCost   float64 `json:"budgetMaxCost"`
	BudgetWarnAt    float64 `json:"budgetWarnAt"`

	// 规划器
	PlannerMaxSteps int  `json:"plannerMaxSteps"`
	PlannerAllowReplan bool `json:"plannerAllowReplan"`
	PlannerMaxReplans int  `json:"plannerMaxReplans"`

	// 执行器
	ToolTimeout string `json:"toolTimeout"`

	// 压缩
	CompressorPruneEnabled bool `json:"compressorPruneEnabled"`
	CompressorPruneMaxAge  int  `json:"compressorPruneMaxAge"`
	CompressorMaxTokens    int  `json:"compressorMaxTokens"`

	// 停滞检测
	StagnationWindowSize     int     `json:"stagnationWindowSize"`
	StagnationEntropyThresh  float64 `json:"stagnationEntropyThreshold"`

	// 探索
	ExplorationBaseTemp  float64 `json:"explorationBaseTemp"`
	ExplorationMaxTemp   float64 `json:"explorationMaxTemp"`

	// 学习
	DiversityThreshold float64 `json:"diversityThreshold"`

	// 可观测性
	TraceEnabled  bool   `json:"traceEnabled"`
	TraceFormat   string `json:"traceFormat"`
	TraceVerbose  bool   `json:"traceVerbose"`

	// Shell
	Shell string `json:"shell"` // auto | bash | powershell

	// 沙箱
	SandboxBash    string   `json:"sandboxBash"` // enforce | off
	SandboxNetwork bool     `json:"sandboxNetwork"`
	AllowWrite     []string `json:"allowWrite"`

	// 网络
	ProxyMode string `json:"proxyMode"` // auto | custom | off
	ProxyURL  string `json:"proxyUrl"`
	NoProxy   string `json:"noProxy"`

	// 权限
	PermMode string   `json:"permMode"` // ask | allow | deny
	PermAllow []string `json:"permAllow"`
	PermAsk   []string `json:"permAsk"`
	PermDeny  []string `json:"permDeny"`

	// Hooks
	Hooks map[string]interface{} `json:"hooks"`
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
	globals   []GlobalInfo
	sessions  map[string]*SessionState
	activeID  string
	activeAge string

	// 应用配置（第四步：前端可改）
	config AppConfig

	// P3 模块实例
	dashboardSrv *dashboard.Dashboard
	pluginMgr    *plugins.Manager
	backupMgr    *backup.Manager
	mcpMgr       *mcp.Manager
	botMgr       *bot.Manager

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

// GlobalInfo describes a workspace (top-level node in the sidebar tree).
type GlobalInfo struct {
	ID       string        `json:"id"`
	Name     string        `json:"name"`
	Path     string        `json:"path,omitempty"`
	Projects []ProjectInfo `json:"projects"`
}

// ProjectInfo describes a project/workspace under a Global.
type ProjectInfo struct {
	ID       string        `json:"id"`
	Name     string        `json:"name"`
	Sessions []SessionInfo `json:"sessions"`
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
			MonitorPath:            ".",
			BackupMode:             "on-completion",
			BackupOnFail:           false,
			DashboardPort:          7788,
			PluginsPath:            filepath.Join(os.TempDir(), "zhulong-plugins"),
			Bots:                   make(map[string]bot.AdapterConfig),
			DeepSeekModel:          "deepseek-v4-flash",
			DeepSeekBaseURL:        "https://api.deepseek.com",
			Temperature:            0.7,
			MaxTokens:              4096,
			MaxLoops:               50,
			MaxWallTime:            "30m",
			CheckpointEvery:        3,
			BudgetMaxTokens:        500000,
			BudgetMaxCost:          10.0,
			BudgetWarnAt:           0.8,
			PlannerMaxSteps:        15,
			PlannerAllowReplan:     true,
			PlannerMaxReplans:      5,
			ToolTimeout:            "30s",
			CompressorPruneEnabled: true,
			CompressorPruneMaxAge:  2,
			CompressorMaxTokens:    2000,
			StagnationWindowSize:   3,
			StagnationEntropyThresh: 0.2,
			ExplorationBaseTemp:    0.7,
			ExplorationMaxTemp:     1.5,
			DiversityThreshold:     1.0,
			TraceEnabled:           true,
			TraceFormat:            "jsonl",
			TraceVerbose:           false,
			Shell:                  "auto",
			SandboxBash:            "enforce",
			SandboxNetwork:         true,
			ProxyMode:              "auto",
			PermMode:               "ask",
			Hooks:                  make(map[string]interface{}),
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
	a.globals = []GlobalInfo{
		{
			ID:   "global-1",
			Name: "Default Workspace",
			Projects: []ProjectInfo{
				{
					ID:       "project-1",
					Name:     "My Project",
					Sessions: []SessionInfo{},
				},
			},
		},
	}
	a.activeID = ""
	a.activeAge = "auto"
}

// startup is called by Wails at app start.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	// Load persisted state; fall back to seedDefaults if missing.
	if !a.loadState() {
		a.seedDefaults()
	}
	a.loadConfig()
	a.loadActiveIDs()
	a.markWailsReady()
	a.initEnvironmentMonitor()
	// Initialize MCP manager
	a.mcpMgr = mcp.NewManager()
	// Initialize Bot manager
	a.botMgr = bot.NewManager()
	a.botMgr.SetMessageHandler(a.handleBotMessage)
	// Load persisted bot connections
	if a.config.Bots != nil {
		for name, cfg := range a.config.Bots {
			if cfg.Enabled {
				_ = a.botMgr.AddAdapter(cfg)
				_ = a.botMgr.Connect(name)
			}
		}
	}
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

// ListGlobals returns the three-tier tree: Globals → Projects → Sessions.
func (a *App) ListGlobals() []GlobalInfo { return a.globals }

// CreateGlobal creates a new workspace (top-level Global).
func (a *App) CreateGlobal(name string) (*GlobalInfo, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("global name cannot be empty")
	}

	global := GlobalInfo{
		ID:       fmt.Sprintf("g%d", time.Now().UnixMilli()),
		Name:     name,
		Projects: []ProjectInfo{},
	}
	a.globals = append(a.globals, global)
	a.emitGlobals()
	return &a.globals[len(a.globals)-1], nil
}

// CreateProject creates a new project/workspace under the given Global.
func (a *App) CreateProject(globalID, name string) (*ProjectInfo, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("project name cannot be empty")
	}

	for i := range a.globals {
		if a.globals[i].ID == globalID {
			project := ProjectInfo{
				ID:       fmt.Sprintf("p%d", time.Now().UnixMilli()),
				Name:     name,
				Sessions: []SessionInfo{},
			}
			a.globals[i].Projects = append(a.globals[i].Projects, project)
			a.emitGlobals()
			return &a.globals[i].Projects[len(a.globals[i].Projects)-1], nil
		}
	}
	return nil, fmt.Errorf("global %q not found", globalID)
}

// SetActiveAgent switches the active agent (persists across sessions).
func (a *App) SetActiveAgent(id string) {
	a.mu.Lock()
	a.activeAge = id
	a.mu.Unlock()
	go a.saveActiveIDs()
}

// SetActiveSession switches the active session in the sidebar.
func (a *App) SetActiveSession(id string) *SessionState {
	a.mu.Lock()
	a.activeID = id
	// 1) in-memory?
	if s, ok := a.sessions[id]; ok {
		a.mu.Unlock()
		a.saveActiveIDs()
		return s
	}
	// 2) on-disk?
	a.mu.Unlock()
	s := a.loadSession(id)
	if s != nil {
		a.saveActiveIDs()
		return s
	}
	// 3) lazy-materialise from globals tree
	a.mu.Lock()
	for _, g := range a.globals {
		for _, p := range g.Projects {
			for _, si := range p.Sessions {
				if si.ID == id {
					s2 := &SessionState{
						Info:    si,
						Status:  "idle",
						Mode:    "auto",
						Model:   a.modelFor(si.AgentID),
						Created: time.Now(),
						Updated: time.Now(),
					}
					a.sessions[id] = s2
					a.mu.Unlock()
					a.saveActiveIDs()
					return s2
				}
			}
		}
	}
	a.mu.Unlock()
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

// NewSession creates a fresh empty session in the given project and returns its state.
func (a *App) NewSession(projectID string) *SessionState {
	a.mu.Lock()
	defer a.mu.Unlock()
	id := fmt.Sprintf("s%d", time.Now().UnixMilli())
	info := SessionInfo{
		ID: id, Title: "新会话", AgentID: a.activeAge, ProjectID: projectID,
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
	// Inject into the target Project
	for i := range a.globals {
		for j := range a.globals[i].Projects {
			if a.globals[i].Projects[j].ID == projectID {
				a.globals[i].Projects[j].Sessions = append([]SessionInfo{info}, a.globals[i].Projects[j].Sessions...)
				break
			}
		}
	}
	a.activeID = id
	a.emitGlobals()
	runtime.EventsEmit(a.ctx, "session:created", st)
	return st
}

// DeleteSession removes a session.
func (a *App) DeleteSession(id string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.sessions, id)
	a.deleteSessionFile(id)
	// Remove from globals tree
	for i := range a.globals {
		for j := range a.globals[i].Projects {
			filtered := a.globals[i].Projects[j].Sessions[:0]
			for _, s := range a.globals[i].Projects[j].Sessions {
				if s.ID != id {
					filtered = append(filtered, s)
				}
			}
			a.globals[i].Projects[j].Sessions = filtered
		}
	}
	if a.activeID == id {
		// Switch to the first available session
		for _, g := range a.globals {
			for _, p := range g.Projects {
				if len(p.Sessions) > 0 {
					a.activeID = p.Sessions[0].ID
					goto done
				}
			}
		}
	done:
	}
	a.emitGlobals()
	a.saveActiveIDs()
	runtime.EventsEmit(a.ctx, "session:deleted", id)
}

// RenameSession updates the title of a session.
func (a *App) RenameSession(id, title string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	// Update in globals tree
	for i := range a.globals {
		for j := range a.globals[i].Projects {
			for k := range a.globals[i].Projects[j].Sessions {
				if a.globals[i].Projects[j].Sessions[k].ID == id {
					a.globals[i].Projects[j].Sessions[k].Title = title
					a.globals[i].Projects[j].Sessions[k].Preview = title
				}
			}
		}
	}
	// Update in session state
	if s, ok := a.sessions[id]; ok {
		s.Info.Title = title
		s.Info.Preview = title
	}
	a.emitGlobals()
}

// GetSession returns the full state of a session.
// If not in memory, loads from disk first.
func (a *App) GetSession(id string) *SessionState {
	// 1) in-memory?
	a.mu.Lock()
	s, ok := a.sessions[id]
	a.mu.Unlock()
	if ok {
		return s
	}
	// 2) on-disk?
	return a.loadSession(id)
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
	a.emitGlobals()
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

// shutdown is called by Wails when the app is about to exit.
// 确保所有会话状态都落盘。
func (a *App) shutdown(ctx context.Context) {
	a.mu.Lock()
	sessions := make([]*SessionState, 0, len(a.sessions))
	for _, s := range a.sessions {
		sessions = append(sessions, s)
	}
	a.mu.Unlock()
	for _, s := range sessions {
		a.saveSession(s)
	}
}

// ════════════════════════════════════════
// Memory — 记忆管理 API
// ════════════════════════════════════════

// MemoryFactDTO is the serializable form of a saved memory fact.
type MemoryFactDTO struct {
	Name        string    `json:"name"`
	Title       string    `json:"title,omitempty"`
	Description string    `json:"description"`
	Type        string    `json:"type"` // user | feedback | project | reference
	Body        string    `json:"body"`
	CreatedAt   time.Time `json:"createdAt"`
}

// MemoryDocDTO is an instruction file.
type MemoryDocDTO struct {
	Path      string    `json:"path"`
	Scope     string    `json:"scope"` // user | project | local
	Body      string    `json:"body"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// MemoryArchiveDTO is an archived (inactive) memory.
type MemoryArchiveDTO struct {
	MemoryFactDTO
	ArchivedAt time.Time `json:"archivedAt"`
}

// MemoryView is the full state returned to the frontend.
type MemoryView struct {
	Facts     []MemoryFactDTO     `json:"facts"`
	Archives  []MemoryArchiveDTO  `json:"archives"`
	Docs      []MemoryDocDTO      `json:"docs"`
	StoreDir  string              `json:"storeDir"`
	Available bool                `json:"available"`
}

func (a *App) memoryDir() string {
	return filepath.Join(zhulongDir(), "memory")
}

func (a *App) memoryPath(filename string) string {
	return filepath.Join(a.memoryDir(), filename)
}

// ListMemory returns the full memory view.
func (a *App) ListMemory() *MemoryView {
	dir := a.memoryDir()
	os.MkdirAll(dir, 0755)
	view := &MemoryView{
		StoreDir:  dir,
		Available: true,
	}
	a.readJSONFile(a.memoryPath("facts.json"), &view.Facts)
	a.readJSONFile(a.memoryPath("archives.json"), &view.Archives)
	a.readJSONFile(a.memoryPath("docs.json"), &view.Docs)
	return view
}

// GetMemoryStoreDir returns the memory storage directory path.
func (a *App) GetMemoryStoreDir() string { return a.memoryDir() }

// Remember adds or updates a memory fact.
func (a *App) Remember(name, title, description, memType, body string) *MemoryFactDTO {
	if name == "" {
		name = fmt.Sprintf("mem-%d", time.Now().UnixMilli())
	}
	fact := MemoryFactDTO{
		Name:        name,
		Title:       title,
		Description: description,
		Type:        memType,
		Body:        body,
		CreatedAt:   time.Now(),
	}
	view := a.ListMemory()
	found := false
	for i, f := range view.Facts {
		if f.Name == name {
			view.Facts[i] = fact
			found = true
			break
		}
	}
	if !found {
		view.Facts = append([]MemoryFactDTO{fact}, view.Facts...)
	}
	a.writeJSONFile(a.memoryPath("facts.json"), view.Facts)
	return &fact
}

// Forget archives (removes from active) a memory by name.
func (a *App) Forget(name string) error {
	view := a.ListMemory()
	var target *MemoryFactDTO
	for i, f := range view.Facts {
		if f.Name == name {
			target = &f
			view.Facts = append(view.Facts[:i], view.Facts[i+1:]...)
			break
		}
	}
	if target == nil {
		return fmt.Errorf("memory %q not found", name)
	}
	arch := MemoryArchiveDTO{
		MemoryFactDTO: *target,
		ArchivedAt:    time.Now(),
	}
	view.Archives = append([]MemoryArchiveDTO{arch}, view.Archives...)
	a.writeJSONFile(a.memoryPath("facts.json"), view.Facts)
	a.writeJSONFile(a.memoryPath("archives.json"), view.Archives)
	return nil
}

// RestoreMemory brings an archived memory back to active.
func (a *App) RestoreMemory(name string) error {
	view := a.ListMemory()
	var target *MemoryArchiveDTO
	for i, ar := range view.Archives {
		if ar.Name == name {
			target = &ar
			view.Archives = append(view.Archives[:i], view.Archives[i+1:]...)
			break
		}
	}
	if target == nil {
		return fmt.Errorf("archive %q not found", name)
	}
	view.Facts = append([]MemoryFactDTO{target.MemoryFactDTO}, view.Facts...)
	a.writeJSONFile(a.memoryPath("facts.json"), view.Facts)
	a.writeJSONFile(a.memoryPath("archives.json"), view.Archives)
	return nil
}

// DeleteMemory permanently removes a memory.
func (a *App) DeleteMemory(name string) error {
	view := a.ListMemory()
	filtered := make([]MemoryFactDTO, 0, len(view.Facts))
	for _, f := range view.Facts {
		if f.Name != name {
			filtered = append(filtered, f)
		}
	}
	if len(filtered) == len(view.Facts) {
		return fmt.Errorf("memory %q not found", name)
	}
	a.writeJSONFile(a.memoryPath("facts.json"), filtered)
	return nil
}

// SaveDoc creates or updates an instruction document.
func (a *App) SaveDoc(path, scope, body string) *MemoryDocDTO {
	doc := MemoryDocDTO{
		Path:      path,
		Scope:     scope,
		Body:      body,
		UpdatedAt: time.Now(),
	}
	view := a.ListMemory()
	found := false
	for i, d := range view.Docs {
		if d.Path == path {
			view.Docs[i] = doc
			found = true
			break
		}
	}
	if !found {
		view.Docs = append(view.Docs, doc)
	}
	a.writeJSONFile(a.memoryPath("docs.json"), view.Docs)
	return &doc
}

// DeleteDoc removes an instruction document.
func (a *App) DeleteDoc(path string) error {
	view := a.ListMemory()
	filtered := make([]MemoryDocDTO, 0, len(view.Docs))
	for _, d := range view.Docs {
		if d.Path != path {
			filtered = append(filtered, d)
		}
	}
	if len(filtered) == len(view.Docs) {
		return fmt.Errorf("doc %q not found", path)
	}
	a.writeJSONFile(a.memoryPath("docs.json"), filtered)
	return nil
}

// readJSONFile reads JSON into dst; ignores errors silently (best-effort).
func (a *App) readJSONFile(path string, dst interface{}) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	_ = json.Unmarshal(data, dst)
}

// writeJSONFile writes src as pretty-JSON to path.
func (a *App) writeJSONFile(path string, src interface{}) {
	os.MkdirAll(filepath.Dir(path), 0755)
	data, _ := json.MarshalIndent(src, "", "  ")
	_ = os.WriteFile(path, data, 0644)
}

// ════════════════════════════════════════
// MCP — 前端绑定方法
// ════════════════════════════════════════

// MCPConnectServer connects to an MCP server by config
func (a *App) MCPConnectServer(name, transport, command string, args []string, url string) error {
	if a.mcpMgr == nil {
		a.mcpMgr = mcp.NewManager()
	}
	config := mcp.ServerConfig{
		Name:    name,
		Enabled: true,
		Transport: mcp.TransportConfig{
			Type:    transport,
			Command: command,
			Args:    args,
			URL:     url,
		},
	}
	if err := a.mcpMgr.AddServer(config); err != nil {
		// Server already exists, try to reconnect
		return a.mcpMgr.Connect(name)
	}
	return a.mcpMgr.Connect(name)
}

// MCPDisconnectServer disconnects from an MCP server
func (a *App) MCPDisconnectServer(name string) error {
	if a.mcpMgr == nil {
		return fmt.Errorf("MCP manager not initialized")
	}
	return a.mcpMgr.Disconnect(name)
}

// MCPListTools returns all tools from connected servers
func (a *App) MCPListTools() map[string][]mcp.Tool {
	if a.mcpMgr == nil {
		return nil
	}
	return a.mcpMgr.ListAllTools()
}

// MCPCallTool calls a tool on the appropriate server
func (a *App) MCPCallTool(name string, args map[string]interface{}) (*mcp.ToolResult, error) {
	if a.mcpMgr == nil {
		return nil, fmt.Errorf("MCP manager not initialized")
	}
	return a.mcpMgr.CallTool(context.Background(), name, args)
}

// MCPIsConnected checks if a server is connected
func (a *App) MCPIsConnected(name string) bool {
	if a.mcpMgr == nil {
		return false
	}
	return a.mcpMgr.IsServerConnected(name)
}

// ════════════════════════════════════════
// Bot — 前端绑定方法
// ════════════════════════════════════════

// BotConnect connects a bot adapter
func (a *App) BotConnect(name, platform, token, appID, appSecret, webhookURL string) error {
	if a.botMgr == nil {
		a.botMgr = bot.NewManager()
		a.botMgr.SetMessageHandler(a.handleBotMessage)
	}
	config := bot.AdapterConfig{
		Platform:   bot.Platform(platform),
		Name:       name,
		Enabled:    true,
		Token:      token,
		AppID:      appID,
		AppSecret:  appSecret,
		WebhookURL: webhookURL,
	}

	// Persist to config
	a.mu.Lock()
	if a.config.Bots == nil {
		a.config.Bots = make(map[string]bot.AdapterConfig)
	}
	a.config.Bots[name] = config
	a.saveConfig()
	a.mu.Unlock()

	if err := a.botMgr.AddAdapter(config); err != nil {
		// Adapter already exists, try to reconnect
		return a.botMgr.Connect(name)
	}

	// 如果有 webhook 配置，启动 HTTP server 监听
	if webhookURL != "" {
		go a.startWebhookServer(name, config)
	}

	return a.botMgr.Connect(name)
}

// startWebhookServer starts an HTTP server to receive webhook events
func (a *App) startWebhookServer(name string, config bot.AdapterConfig) {
	mux := http.NewServeMux()

	// 注册 webhook 路由
	path := fmt.Sprintf("/webhook/%s", name)
	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		// 根据平台类型处理
		switch config.Platform {
		case bot.PlatformFeishu:
			if adapter, err := a.botMgr.GetAdapter(name); err == nil {
				if feishu, ok := adapter.(*bot.FeishuAdapter); ok {
					if err := feishu.HandleWebhook(body); err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
				}
			}
		case bot.PlatformDingTalk:
			if adapter, err := a.botMgr.GetAdapter(name); err == nil {
				if dingtalk, ok := adapter.(*bot.DingTalkAdapter); ok {
					if err := dingtalk.HandleWebhook(body); err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
				}
			}
		case bot.PlatformWebhook: // GitHub
			signature := r.Header.Get("X-Hub-Signature-256")
			if adapter, err := a.botMgr.GetAdapter(name); err == nil {
				if github, ok := adapter.(*bot.GitHubAdapter); ok {
					if err := github.HandleWebhook(body, signature); err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
				}
			}
		default:
			http.Error(w, "Unsupported platform", http.StatusNotImplemented)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	// 启动 HTTP server
	addr := fmt.Sprintf(":%d", 8080+hashString(name)%1000)
	log.Printf("[webhook] Starting %s webhook server on %s", name, addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Printf("[webhook] Failed to start %s webhook server: %v", name, err)
	}
}

// hashString returns a simple hash of a string
func hashString(s string) int {
	h := 0
	for _, c := range s {
		h = h*31 + int(c)
	}
	if h < 0 {
		h = -h
	}
	return h
}

// BotDisconnect disconnects a bot adapter
func (a *App) BotDisconnect(name string) error {
	if a.botMgr == nil {
		return fmt.Errorf("bot manager not initialized")
	}
	return a.botMgr.Disconnect(name)
}

// BotSend sends a message through a bot adapter
func (a *App) BotSend(adapterName, chatID, text string) (*bot.SendResult, error) {
	if a.botMgr == nil {
		return nil, fmt.Errorf("bot manager not initialized")
	}
	adapter, err := a.botMgr.GetAdapter(adapterName)
	if err != nil {
		return nil, err
	}
	return adapter.Send(chatID, text)
}

// BotIsConnected checks if a bot adapter is connected
func (a *App) BotIsConnected(name string) bool {
	if a.botMgr == nil {
		return false
	}
	return a.botMgr.IsConnected(name)
}

// BotListAdapters returns all bot adapters
func (a *App) BotListAdapters() []bot.AdapterConfig {
	if a.botMgr == nil {
		return nil
	}
	return a.botMgr.ListAdapters()
}

// BotRemoveAdapter removes a bot adapter
func (a *App) BotRemoveAdapter(name string) error {
	if a.botMgr == nil {
		return fmt.Errorf("bot manager not initialized")
	}

	// Remove from config
	a.mu.Lock()
	if a.config.Bots != nil {
		delete(a.config.Bots, name)
		a.saveConfig()
	}
	a.mu.Unlock()

	return a.botMgr.RemoveAdapter(name)
}

// handleBotMessage handles incoming bot messages and routes them to sessions
func (a *App) handleBotMessage(event *bot.MessageEvent) {
	// Create a session ID based on chat ID
	sessionID := fmt.Sprintf("bot-%s-%s", event.Source.Platform, event.Source.ChatID)

	a.mu.Lock()
	session, exists := a.sessions[sessionID]
	if !exists {
		// Create a new session for this bot chat
		info := SessionInfo{
			ID: sessionID,
			Title: fmt.Sprintf("Bot - %s", event.Source.ChatName),
			AgentID: "auto",
			ProjectID: "default",
		}
		session = &SessionState{
			Info: info,
			Messages: []MessageDTO{},
			Logs: []LogDTO{},
			Plan: []PlanStepDTO{},
		}
		a.sessions[sessionID] = session
	}

	// Add user message to session
	msg := MessageDTO{
		ID: event.MessageID,
		Role: "user",
		Content: event.Text,
		Time: time.Now(),
	}
	session.Messages = append(session.Messages, msg)
	a.mu.Unlock()

	// Emit update to frontend
	a.emitSession(session)

	// Start agent run in background
	go func() {
		a.RunAgent(a.ctx, session)
	}()
}

// SetConfigField updates a single config field and persists
func (a *App) SetConfigField(field string, value interface{}) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	switch field {
	case "deepseekModel":
		a.config.DeepSeekModel = value.(string)
	case "deepseekBaseUrl":
		a.config.DeepSeekBaseURL = value.(string)
	case "temperature":
		a.config.Temperature = value.(float64)
	case "maxTokens":
		a.config.MaxTokens = value.(int)
	case "maxLoops":
		a.config.MaxLoops = value.(int)
	case "maxWallTime":
		a.config.MaxWallTime = value.(string)
	case "budgetMaxTokens":
		a.config.BudgetMaxTokens = value.(int)
	case "budgetMaxCost":
		a.config.BudgetMaxCost = value.(float64)
	case "budgetWarnAt":
		a.config.BudgetWarnAt = value.(float64)
	case "plannerMaxSteps":
		a.config.PlannerMaxSteps = value.(int)
	case "toolTimeout":
		a.config.ToolTimeout = value.(string)
	case "stagnationWindowSize":
		a.config.StagnationWindowSize = value.(int)
	case "explorationBaseTemp":
		a.config.ExplorationBaseTemp = value.(float64)
	case "explorationMaxTemp":
		a.config.ExplorationMaxTemp = value.(float64)
	case "diversityThreshold":
		a.config.DiversityThreshold = value.(float64)
	case "traceEnabled":
		a.config.TraceEnabled = value.(bool)
	case "shell":
		a.config.Shell = value.(string)
	case "proxyMode":
		a.config.ProxyMode = value.(string)
	case "proxyUrl":
		a.config.ProxyURL = value.(string)
	case "permMode":
		a.config.PermMode = value.(string)
	case "modelProviders":
		// 模型供应商配置（由前端管理，保存到 JSON 文件）
		a.saveModelProviders(value)
	default:
		return fmt.Errorf("unknown config field: %s", field)
	}

	a.saveConfig()
	return nil
}

// saveModelProviders saves model providers to a JSON file
func (a *App) saveModelProviders(providers interface{}) {
	data, err := json.Marshal(providers)
	if err != nil {
		return
	}
	configDir := filepath.Join(os.Getenv("APPDATA"), "zhulong")
	os.MkdirAll(configDir, 0755)
	providersPath := filepath.Join(configDir, "providers.json")
	os.WriteFile(providersPath, data, 0644)
}

// GetConfigField returns a single config field value
func (a *App) GetConfigField(field string) interface{} {
	a.mu.Lock()
	defer a.mu.Unlock()

	switch field {
	case "deepseekModel":
		return a.config.DeepSeekModel
	case "deepseekBaseUrl":
		return a.config.DeepSeekBaseURL
	case "temperature":
		return a.config.Temperature
	case "maxTokens":
		return a.config.MaxTokens
	case "maxLoops":
		return a.config.MaxLoops
	case "maxWallTime":
		return a.config.MaxWallTime
	case "budgetMaxTokens":
		return a.config.BudgetMaxTokens
	case "budgetMaxCost":
		return a.config.BudgetMaxCost
	case "budgetWarnAt":
		return a.config.BudgetWarnAt
	case "plannerMaxSteps":
		return a.config.PlannerMaxSteps
	case "toolTimeout":
		return a.config.ToolTimeout
	case "stagnationWindowSize":
		return a.config.StagnationWindowSize
	case "explorationBaseTemp":
		return a.config.ExplorationBaseTemp
	case "explorationMaxTemp":
		return a.config.ExplorationMaxTemp
	case "diversityThreshold":
		return a.config.DiversityThreshold
	case "traceEnabled":
		return a.config.TraceEnabled
	case "shell":
		return a.config.Shell
	case "proxyMode":
		return a.config.ProxyMode
	case "proxyUrl":
		return a.config.ProxyURL
	case "permMode":
		return a.config.PermMode
	default:
		return nil
	}
}

// === Helpers ===

// CheckEnvVar checks if an environment variable is set and non-empty
func (a *App) CheckEnvVar(name string) bool {
	return os.Getenv(name) != ""
}

// GetEnvVar returns an environment variable value (masked for security)
func (a *App) GetEnvVar(name string) string {
	val := os.Getenv(name)
	if val == "" {
		return ""
	}
	// 返回掩码值，不暴露完整密钥
	if len(val) > 8 {
		return val[:4] + "****" + val[len(val)-4:]
	}
	return "****"
}

// OpenInExplorer opens a folder in the system file explorer
func (a *App) OpenInExplorer(path string) {
	runtime.BrowserOpenURL(a.ctx, path)
}

// GetGlobalPath returns the filesystem path for a global workspace
func (a *App) GetGlobalPath(globalID string) string {
	for _, g := range a.globals {
		if g.ID == globalID {
			if g.Path != "" {
				return g.Path
			}
			return filepath.Join(zhulongDir(), "workspaces", g.ID)
		}
	}
	return ""
}

// GetProjectPath returns the filesystem path for a project
func (a *App) GetProjectPath(globalID, projectID string) string {
	for _, g := range a.globals {
		if g.ID == globalID {
			for _, p := range g.Projects {
				if p.ID == projectID {
					return filepath.Join(zhulongDir(), "workspaces", g.ID, p.ID)
				}
			}
		}
	}
	return ""
}

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
		Models:    &ModuleItemDTO{Status: "idle", Details: map[string]any{"poolSize": 0, "availableModels": []string{"deepseek-v4-flash", "deepseek-v4-pro"}}},
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
	// 同步持久化，直接保存 s 而非重新从 a.sessions[id] 读取
	if s != nil {
		a.saveSession(s)
	}
}

func (a *App) emitGlobals() {
	if a.ctx == nil {
		return
	}
	if !hasWailsEvents(a.ctx) {
		return
	}
	runtime.EventsEmit(a.ctx, "globals:update", a.globals)
	a.saveState()
}

// ── Persistence ──────────────────────────────────────────────────────────

// zhulongDir returns ~/.zhulong (creates it if missing).
func zhulongDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".zhulong"
	}
	dir := filepath.Join(home, ".zhulong")
	os.MkdirAll(dir, 0755)
	return dir
}

// saveState writes a.globals to ~/.zhulong/state.json
func (a *App) saveState() {
	dir := zhulongDir()
	data, err := json.MarshalIndent(a.globals, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(filepath.Join(dir, "state.json"), data, 0644)
}

// loadState reads ~/.zhulong/state.json into a.globals.
// Returns true if loaded successfully and non-empty.
func (a *App) loadState() bool {
	dir := zhulongDir()
	data, err := os.ReadFile(filepath.Join(dir, "state.json"))
	if err != nil {
		return false
	}
	var gs []GlobalInfo
	if err := json.Unmarshal(data, &gs); err != nil {
		return false
	}
	if len(gs) > 0 {
		a.globals = gs
		return true
	}
	return false
}

// loadConfig loads the application config from ~/.zhulong/config.json
func (a *App) loadConfig() {
	dir := zhulongDir()
	data, err := os.ReadFile(filepath.Join(dir, "config.json"))
	if err != nil {
		return
	}
	_ = json.Unmarshal(data, &a.config)
}

// saveConfig saves the application config to ~/.zhulong/config.json
func (a *App) saveConfig() {
	dir := zhulongDir()
	data, _ := json.MarshalIndent(a.config, "", "  ")
	_ = os.WriteFile(filepath.Join(dir, "config.json"), data, 0644)
}

// parseDuration parses a duration string like "30m" or "1h" into time.Duration
func parseDuration(s string, fallback time.Duration) time.Duration {
	if s == "" {
		return fallback
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return fallback
	}
	return d
}

// saveSession writes the given session state to ~/.zhulong/sessions/<id>.json.
// Accepts *SessionState directly to avoid reading a.sessions[id] under lock,
// which would race with RunAgent's unsynchronized s.Messages appends.
func (a *App) saveSession(s *SessionState) {
	if s == nil || s.Info.ID == "" {
		return
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return
	}
	dir := filepath.Join(zhulongDir(), "sessions")
	os.MkdirAll(dir, 0755)
	_ = os.WriteFile(filepath.Join(dir, s.Info.ID+".json"), data, 0644)
}

// loadSession reads ~/.zhulong/sessions/<id>.json into a.sessions[id].
// Returns the loaded state (or nil).
func (a *App) loadSession(id string) *SessionState {
	dir := filepath.Join(zhulongDir(), "sessions")
	data, err := os.ReadFile(filepath.Join(dir, id+".json"))
	if err != nil {
		return nil
	}
	var s SessionState
	if err := json.Unmarshal(data, &s); err != nil {
		return nil
	}
	a.mu.Lock()
	a.sessions[id] = &s
	a.mu.Unlock()
	return &s
}

// deleteSessionFile removes ~/.zhulong/sessions/<id>.json (called on deletion).
func (a *App) deleteSessionFile(id string) {
	dir := filepath.Join(zhulongDir(), "sessions")
	os.Remove(filepath.Join(dir, id+".json"))
}

// ── Active IDs persistence ──────────────────────────────

// loadActiveIDs restores activeID / activeAgent from ~/.zhulong/active.json
func (a *App) loadActiveIDs() {
	dir := zhulongDir()
	data, err := os.ReadFile(filepath.Join(dir, "active.json"))
	if err != nil {
		return
	}
	var m struct {
		ActiveID    string `json:"activeID"`
		ActiveAgent string `json:"activeAgent"`
	}
	_ = json.Unmarshal(data, &m)
	if m.ActiveID != "" {
		a.activeID = m.ActiveID
	}
	if m.ActiveAgent != "" {
		a.activeAge = m.ActiveAgent
	}
}

func (a *App) saveActiveIDs() {
	dir := zhulongDir()
	data, _ := json.MarshalIndent(map[string]string{
		"activeID":   a.activeID,
		"activeAgent": a.activeAge,
	}, "", "  ")
	_ = os.WriteFile(filepath.Join(dir, "active.json"), data, 0644)
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
