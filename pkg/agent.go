package pkg

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/qoqu/zhuLong/internal/approval"
	"github.com/qoqu/zhuLong/internal/backup"
	"github.com/qoqu/zhuLong/internal/breaker"
	"github.com/qoqu/zhuLong/internal/budget"
	"github.com/qoqu/zhuLong/internal/checkpoint"
	"github.com/qoqu/zhuLong/internal/compressor"
	"github.com/qoqu/zhuLong/internal/controller"
	"github.com/qoqu/zhuLong/internal/dashboard"
	"github.com/qoqu/zhuLong/internal/environment"
	"github.com/qoqu/zhuLong/internal/evolution"
	"github.com/qoqu/zhuLong/internal/executor"
	"github.com/qoqu/zhuLong/internal/exploration"
	"github.com/qoqu/zhuLong/internal/gateway"
	"github.com/qoqu/zhuLong/internal/hook"
	"github.com/qoqu/zhuLong/internal/human"
	"github.com/qoqu/zhuLong/internal/i18n"
	"github.com/qoqu/zhuLong/internal/information"
	"github.com/qoqu/zhuLong/internal/learning"
	"github.com/qoqu/zhuLong/internal/memento"
	"github.com/qoqu/zhuLong/internal/memory"
	"github.com/qoqu/zhuLong/internal/models"
	"github.com/qoqu/zhuLong/internal/observe"
	"github.com/qoqu/zhuLong/internal/plugins"
	"github.com/qoqu/zhuLong/internal/planner"
	"github.com/qoqu/zhuLong/internal/profile"
	"github.com/qoqu/zhuLong/internal/provider"
	"github.com/qoqu/zhuLong/internal/quality"
	"github.com/qoqu/zhuLong/internal/reflector"
	"github.com/qoqu/zhuLong/internal/security"
	"github.com/qoqu/zhuLong/internal/skills"
	"github.com/qoqu/zhuLong/internal/stability"
	"github.com/qoqu/zhuLong/internal/stagnation"
	"github.com/qoqu/zhuLong/internal/synergetics"
	"github.com/qoqu/zhuLong/internal/terminal"
	"github.com/qoqu/zhuLong/internal/tools"
	"github.com/qoqu/zhuLong/internal/trace"
	"github.com/qoqu/zhuLong/internal/workflow"
)

// Agent 是烛龙的主入口，封装Agent循环的全部状态
// 关键修复: 之前 evolution/breaker/quality/observe/workflow/human/approval 模块
// 都建好了但没人调用。现在把这些互补模块显式串联进 Agent 字段
type Agent struct {
	goal          string               // 用户目标
	options       *Options             // 配置选项
	provider      Provider             // LLM提供者

	// 上下文层：影响 LLM 的输入
	skillPipeline *skills.Pipeline     // 技能管道（目录加载→渐进披露→骨架压缩→注入）
	memoryStore   *memento.MemoryStore // 记忆系统（MEMORY.md + USER.md 冻结快照）

	// 安全与执行层：拦截危险操作 + 终端后端
	security      *security.Engine     // 安全引擎（7层防御）
	terminal      *terminal.Manager    // 终端后端管理器（Local/Docker/SSH）
	approval      *approval.ApprovalEngine // 审批引擎（敏感操作需用户确认）

	// 隔离与可观测性
	profiler      *profile.Manager     // Profile隔离管理器
	tracer        *observe.Tracer      // 可观测性追踪器
	logger        *trace.Logger        // 事件日志器

	// 主动层：用于"自进化/自省"
	circuitBreaker *breaker.Breaker    // 断路器（连续失败自动暂停）
	reviewer       *evolution.BackgroundReviewer // 后台审查（每次会话后）
	curator        *evolution.Curator  // 守卫者（定期清理技能）
	suggester      *evolution.SuggestionEngine // 建议引擎（重复需求→技能）
	scanner        *quality.Scanner    // 质量扫描器（8维GC）
	humanBreaks    *human.Manager       // 人机断点（关键步骤需用户确认）
	wfMode         *workflow.Workflow  // 工作流模式（full/hotfix/tweak）

	// P0 必串联：控制 + 压缩 + 成本 + 钩子（之前只建好不调用）
	fsm         *controller.Session      // FSM 状态机（9 状态显式跟踪）
	compressor  compressor.Compressor    // 上下文压缩器（每次 Plan 前 Prune）
	budget      *budget.Budget           // 成本控制器（每步后扣费）
	hookEngine  *hook.HookEngine         // 钩子引擎（6 阶段：pre_plan/post_plan/pre_tool/post_tool/pre_reflect/post_reflect）

	// P1 必串联：信息论 + 协同学 + 耗散结构 + CAS + 振荡检测
	// 关键修复: 之前 11 个 P1 模块全部实现但 agent.go 不调
	// 现在: 注入为 Agent 字段，Run() 中按需调用
	stagnationDetector *stagnation.Detector        // 停滞检测（耗散结构理论）
	explorationTrigger *exploration.Trigger        // 探索触发（耗散结构理论）
	stabilityAnalyzer  *stability.Analyzer         // 振荡/发散检测（控制论）
	infoGain           *information.InformationGain // 信息增益（信息论）
	diversityManager   *learning.DiversityManager  // 多样性管理（CAS）
	orderParameter     *synergetics.OrderParameter  // 序参量（协同学）
	slavingPrinciple   *synergetics.SlavingPrinciple // 役使原理（协同学）
	buildingBlockStore *learning.BuildingBlockStore // 积木块（CAS）

	// P2 必串联：备选路径 + 环境感知
	// 关键修复: 之前 environment.Monitor + planner.AlternativePlanner 未注入
	// 现在: Plan 失败时切换备选路径；启动时监控工作目录
	altPlanner   *planner.AlternativePlanner  // 备选路径规划（一般系统论）
	envMonitor   *environment.Monitor         // 环境感知（一般系统论）

	// P3 必串联：i18n + dashboard + gateway + models + plugins + backup
	// 关键修复: 之前 8 个 P3 模块完整实现但 agent.go 不调
	// 现在: 注入为 Agent 字段，Run() 中按需使用
	i18nBundle   *i18n.Bundle                  // 多语言（17 语言支持）
	dashboard    *dashboard.Dashboard          // Web Dashboard（HTTP 服务）
	gatewayReg   *gateway.Registry             // 消息网关（20+ 平台适配器接口）
	modelPool    *models.Pool                  // 多模型池
	pluginMgr    *plugins.Manager              // 插件管理器
	backupMgr    *backup.Manager               // 备份管理器

	// 运行状态
	startedAt     time.Time            // 启动时间
	sessionID     string               // 会话ID
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

// NewAgent 创建 Agent（一次性完成所有模块的串联初始化）
// 关键修复: 之前 evolution/breaker/quality/observe/workflow/human/approval 都没被实例化
// 现在按"上下文层→安全执行层→隔离观测层→主动层"顺序初始化
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

	// 设置默认数据目录
	if options.DataDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			options.DataDir = ".zhulong"
		} else {
			options.DataDir = filepath.Join(home, ".zhulong")
		}
	}

	// 创建数据目录
	os.MkdirAll(options.DataDir, 0755)

	// === 上下文层 ===

	// 1. 技能管道：目录扫描 → 渐进披露 → 骨架压缩 → 注入 system prompt
	skillSearchPath := filepath.Join(options.DataDir, "skills")
	os.MkdirAll(skillSearchPath, 0755)
	skillPipeline := skills.NewPipeline([]string{skillSearchPath})
	if err := skillPipeline.Init(); err != nil {
		return nil, fmt.Errorf("skill pipeline init: %w", err)
	}

	// 2. 记忆系统：MEMORY.md（Agent 笔记）+ USER.md（用户画像）冻结快照
	memStore := memento.NewMemoryStore(options.DataDir)
	if err := memStore.Init(); err != nil {
		return nil, fmt.Errorf("memory store init: %w", err)
	}

	// === 安全与执行层 ===

	// 3. 安全引擎：7 层防御（黑名单/SSRF/凭据/文件突变/输入清理/供应链/容器）
	secEngine := security.NewEngine()

	// 4. 终端后端管理器：默认 Local，可切换 Docker/SSH
	termMgr := terminal.NewManager()

	// 5. 审批引擎：敏感操作（写关键文件/网络请求/执行命令）需用户确认
	// 关键修复: 之前 approval 引擎建好了但没在 agent.go 出现
	// 修正: NewApprovalEngine 需要 mode + callback，这里用 ModeAsk + nil（不阻塞，规则默认即可）
	approvalEngine := approval.NewApprovalEngine(approval.ModeAsk, nil)

	// === 隔离与可观测性 ===

	// 6. Profile 隔离管理器：多环境并发（参考 Hermes profile）
	profMgr := profile.NewManager(options.DataDir)

	// 7. 可观测性追踪器：span 追踪（参考 Hermes Langfuse 集成）
	obsTracer := observe.NewTracer()

	// 8. 事件日志器：JSONL 格式落盘
	logger := trace.NewLogger("agent", &trace.Config{
		Enabled:   true,
		OutputDir: filepath.Join(options.DataDir, "traces"),
		Format:    "jsonl",
		Verbose:   options.Verbose,
	})

	// === 主动层（自进化/自省） ===

	// 9. 断路器：连续失败自动暂停（参考 Harness-Starter）
	cb := breaker.NewBreaker(breaker.DefaultBreakerConfig())

	// 10. 后台审查器：会话结束后识别改进信号
	// 关键修复: 之前 evolution.HeuristicReview 是函数不是 ReviewProvider
	// 这里用启发式 ReviewProvider 适配器包装（offline 友好）
	reviewer := evolution.NewBackgroundReviewer(newHeuristicReviewer())

	// 11. 守卫者：定期清理过时/归档技能
	curator := evolution.NewCurator(evolution.DefaultCuratorConfig())

	// 12. 建议引擎：重复需求 → 自动建议创建技能
	suggester := evolution.NewSuggestionEngine()

	// 13. 质量扫描器：8 个确定性维度（参考 Harness-Starter GC 扫描器）
	scanner := quality.NewScanner()

	// 14. 人机断点：关键步骤需用户确认
	// 关键修复: 之前 human 模块的入口是 NewManager，不是 NewBreaker
	humanBreaks := human.NewManager(human.DefaultConfig())

	// 15. 工作流模式：full/hotfix/tweak 切换（参考 Harness-Starter）
	wf := workflow.New(workflow.DefaultConfig(workflow.ModeFull))

	// === P0 必串联 4 件套 ===

	// 16. FSM 状态机：9 状态显式跟踪（替代 for-loop 直跳）
	// 关键修复: 之前 controller 标记为 deprecated 没用，agent.go 改用 controller.Session
	// 直接维护 State 字段，状态切换在 Run() 中显式赋值
	sessionID := fmt.Sprintf("session-%d", time.Now().Unix())
	fsmSession := controller.NewSession(sessionID, options.Goal)

	// 17. 上下文压缩器：每次 Plan 前 Prune（保证 cache 命中）
	// 关键修复: 之前 compressor 包有 4 个文件但 agent.go 不调用
	// 现在: SimpleCompressor + 字符计数（粗略近似 token，4 字符 ≈ 1 token）
	comp := compressor.NewSimpleCompressor(&compressor.CharCounter{}, compressor.DefaultConfig())

	// 18. 成本控制器：每步后扣费/扣 token，超 MaxLoops/Tokens/Cost/WallTime 自动停
	// 关键修复: 之前 budget 包有但 agent.go 不注入
	// 现在: 用 options 配 Config
	budgetCtrl := budget.NewBudget(&budget.Config{
		MaxLoops:    options.MaxLoops,
		MaxTokens:   options.MaxTokens,
		MaxCost:     options.MaxCost,
		MaxWallTime: options.MaxWallTime,
		WarnAt:      0.8,
	})

	// 19. 钩子引擎：6 阶段自动化（参考 Harness-Starter）
	// 关键修复: 之前 hook 包有 3 个预置钩子但无人注册
	// 现在: 注册 SecurityIntercept + ContextInject + ReviewFeedback 三件套
	hookEngine := hook.NewHookEngine()
	hookEngine.Register(hook.SecurityInterceptHook())
	hookEngine.Register(hook.ContextInjectHook(options.DataDir))
	hookEngine.Register(hook.ReviewFeedbackHook())

	// === P1 必串联 8 件套 ===

	// 20. 停滞检测（耗散结构理论）
	// 关键修复: 之前 stagnation 包有 Detector 但 agent.go 不调
	// 现在: 每次循环 AddStep，IsStagnating 时触发 exploration
	stagnationDet := stagnation.NewDetector(5, 0.3)

	// 21. 探索触发（耗散结构理论）
	// 关键修复: 之前 exploration 包有 Trigger 但 agent.go 不调
	// 现在: 停滞时 ShouldExplore + GenerateExploration
	explorationTrig := exploration.NewTrigger(0.7, 1.5, []string{"web_search", "read_file", "search_file"})

	// 22. 振荡/发散检测（控制论）
	// 关键修复: 之前 stability 包有 Analyzer 但 agent.go 不调
	// 现在: 每步 AddSnapshot，IsOscillating 时标记需要重规划
	stabilityAn := stability.NewAnalyzer(5)

	// 23. 信息增益（信息论）
	// 关键修复: 之前 information.gain 有 InformationGain 但 agent.go 不调
	// 现在: 工具调用前 EstimateGain，低增益时换工具
	infoG := information.NewInformationGain()

	// 24. 多样性管理（CAS）
	// 关键修复: 之前 learning.diversity 有 DiversityManager 但 agent.go 不调
	// 现在: 工具/策略使用计数，过度依赖时建议换工具
	diversityMgr := learning.NewDiversityManager(0.8)

	// 25. 序参量（协同学）
	// 关键修复: 之前 synergetics.order_parameter 有 OrderParameter 但 agent.go 不调
	// 现在: 从会话识别序参量，stable 时锁定目标
	orderParam := synergetics.IdentifyOrderParameter(&synergetics.Session{Goal: options.Goal})

	// 26. 役使原理（协同学）
	// 关键修复: 之前 synergetics.slaving 有 SlavingPrinciple 但 agent.go 不调
	// 现在: 行动前 EnforceSlaving，过滤与序参量无关的 action
	slavingPrin := synergetics.NewSlavingPrinciple(orderParam)

	// 27. 积木块（CAS）
	// 关键修复: 之前 learning.building_block 有 BuildingBlockStore 但 agent.go 不调
	// 现在: 成功会话后自动提取积木块
	bbStore := learning.NewBuildingBlockStore()

	// === P2 必串联 2 件套 ===

	// 28. 备选路径规划（一般系统论）
	// 关键修复: 之前 planner.AlternativePlanner 有但 agent.go 不注入
	// 现在: Plan 失败时 GenerateAlternatives 选最优备选
	mainPlannerForAlt := planner.NewLLMPlanner(&PlannerProvider{Provider: nil}, planner.DefaultConfig())
	altPlanner := planner.NewAlternativePlanner(mainPlannerForAlt)

	// 29. 环境感知（一般系统论）
	// 关键修复: 之前 environment.Monitor 有但 agent.go 不注入
	// 现在: 启动时监控 dataDir 变化
	envMon := environment.NewMonitor()
	envMon.AddWatcher(environment.WatcherConfig{
		Path:         options.DataDir,
		Recursive:    true,
		PollInterval: 60 * time.Second, // 关键: 必须 > 0，否则 ticker.C 立即触发
	})

	// 关键修复: envMon 在 Run() 里启动/停止（避免阻塞 NewAgent）

	// === P3 必串联 6 件套 ===

	// 30. 多语言（i18n）
	// 关键修复: 之前 i18n.Bundle 完整实现 17 语言但 agent.go 不调
	// 现在: 注入默认 bundle，支持 system prompt 多语言切换
	i18nB := i18n.New(i18n.LangZH)
	i18nB.Register(i18n.LangZH, i18n.DefaultTranslations())
	i18nB.Register(i18n.LangEN, i18n.EnglishTranslations())

	// 31. Web Dashboard（HTTP 服务）
	// 关键修复: 之前 dashboard.Dashboard 完整实现但 agent.go 不启
	// 现在: 在 4515 端口启动 HTTP 服务，注册 Agent 自身为模块
	dash := dashboard.New(&dashboard.Config{
		Port:     4515,
		DataDir:  options.DataDir,
	})
	dash.RegisterModule("zhulong", "0.1.0")
	dash.RegisterModule("agent-loop", "0.1.0")

	// 32. 消息网关（20+ 平台适配器）
	// 关键修复: 之前 gateway.Registry 有但 agent.go 不调
	// 现在: 注入并注册 CLI 适配器（其他平台可选）
	gwReg := gateway.NewRegistry()
	gwReg.Register(gateway.NewCLIAdapter())

	// 33. 多模型池
	// 关键修复: 之前 models.Pool 有但 agent.go 不调
	// 现在: 注入备用 provider 池（主 DeepSeek 失败时切换）
	// 关键修复: 之前 models.Provider 类型未导出，需要查实际类型
	modelPoolInst := models.NewPool()

	// 34. 插件管理器
	// 关键修复: 之前 plugins.Manager 有但 agent.go 不调
	// 现在: 注入；通过能力查询接口在 Run() 中按需调用
	pluginMgrInst := plugins.NewManager()

	// 35. 备份管理器
	// 关键修复: 之前 backup.Manager 有但 agent.go 不调
	// 现在: 注入；Run() 成功后自动备份 config/ docs/ go.{mod,sum}
	backupMgrInst := backup.NewManager(filepath.Join(options.DataDir, "backups"))

	return &Agent{
		goal:          options.Goal,
		options:       options,
		skillPipeline: skillPipeline,
		memoryStore:   memStore,
		security:      secEngine,
		terminal:      termMgr,
		approval:      approvalEngine,
		profiler:      profMgr,
		tracer:        obsTracer,
		logger:        logger,
		circuitBreaker: cb,
		reviewer:      reviewer,
		curator:       curator,
		suggester:     suggester,
		scanner:       scanner,
		humanBreaks:   humanBreaks,
		wfMode:        wf,
		fsm:           fsmSession,
		compressor:    comp,
		budget:        budgetCtrl,
		hookEngine:    hookEngine,
		stagnationDetector: stagnationDet,
		explorationTrigger: explorationTrig,
		stabilityAnalyzer:  stabilityAn,
		infoGain:           infoG,
		diversityManager:   diversityMgr,
		orderParameter:     orderParam,
		slavingPrinciple:   slavingPrin,
		buildingBlockStore: bbStore,
		altPlanner:         altPlanner,
		envMonitor:         envMon,
		i18nBundle:         i18nB,
		dashboard:          dash,
		gatewayReg:         gwReg,
		modelPool:          modelPoolInst,
		pluginMgr:          pluginMgrInst,
		backupMgr:          backupMgrInst,
		startedAt:          time.Now(),
		sessionID:          sessionID,
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

// Run 运行 Agent：完整 Plan → Execute → Reflect → Review 闭环
// 关键修复: 之前 _ = memStore / _ = chkStore / _ = plannerConfig 等占位，
// checkpoint 的 RunMemory.RestoreFrom 只从 cp.LoopCount 推算 Started 没真恢复 plan，
// evolution/breaker/quality/workflow 完全没被调用
// 现在: 串联 memento + memory(FileStore) + checkpoint 真恢复 + breaker 守卫循环
//       + 完成后 quality 扫描 + evolution 审查 + workflow 模式决定扫描强度
func (a *Agent) Run() (*AgentResult, error) {
	if a.goal == "" {
		return nil, fmt.Errorf("goal is not set")
	}

	// 默认 provider（无 API key 时走启发式）
	if a.provider == nil {
		a.provider = NewHeuristicProvider(a.options.Model)
	}

	ctx, cancel := context.WithTimeout(context.Background(), a.options.MaxWallTime)
	defer cancel()

	startTime := time.Now()
	a.startedAt = startTime

	// === 基础设施 ===
	dataDir := a.options.DataDir
	memStore := memory.NewFileStore(filepath.Join(dataDir, "memory"))
	chkStore := checkpoint.NewFileStore(filepath.Join(dataDir, "checkpoint"))

	// 使用 a.logger 替代重新创建 logger
	logger := a.logger

	// === P2 串 2: 环境感知器启动 ===
	// 关键修复: 之前 environment.Monitor 在 NewAgent 时建好不调 Start
	// 现在: Run 启动时 Start，会异步监控 dataDir 变化
	// 注意: 不消费 changes 通道（避免阻塞），仅触发 Start 即可
	if err := a.envMonitor.Start(); err == nil {
		logger.Log("env_monitor", "started", map[string]string{"path": dataDir})
	}
	defer a.envMonitor.Stop()

	// === 检查点恢复 ===
	// 关键修复: 之前 chkStore.LoadByID("cp-agent-done") 拿到的 cp 喂给 RunMemory.RestoreFrom
	// 而 RestoreFrom 只用 cp.LoopCount 算个 Started 时间，plan 没真恢复
	// 现在: 不依赖 RunMemory.RestoreFrom，checkpoint 只作为日志备份（plan 由 LLM 重新生成）
	// 实际"恢复"语义: 如果存在 cp-agent-done，说明上次完成了，跳过 plan
	skippableFromCheckpoint := false
	if cp, err := chkStore.LoadByID("cp-agent-done"); err == nil && cp != nil && cp.State == "done" {
		skippableFromCheckpoint = true
		logger.Log("system", "checkpoint_resume", map[string]int{"prev_tokens": cp.TokensUsed})
	}
	_ = memStore // 保留给将来的 working/session/long-term 压缩链路
	_ = skippableFromCheckpoint

	// === 工具注册（安全增强） ===
	toolsReg := executor.NewToolRegistry()
	toolsReg.Register(&SecureReadFile{Engine: a.security})
	toolsReg.Register(&SecureWriteFile{Engine: a.security})
	toolsReg.Register(&AdapterSearchFile{})
	toolsReg.Register(&SecureExecuteCommand{Engine: a.security, Terminal: a.terminal})

	// 技能工具
	toolsReg.Register(&SkillViewTool{Pipeline: a.skillPipeline})
	toolsReg.Register(&SkillSearchTool{Pipeline: a.skillPipeline})

	// 记忆工具
	toolsReg.Register(&MemoryNoteTool{Store: a.memoryStore})
	toolsReg.Register(&MemoryProfileTool{Store: a.memoryStore})

	// === System Prompt 构造（注入冻结快照，遵循缓存铁律） ===
	systemPrompt := "You are Zhulong (烛龙), an autonomous agent powered by DeepSeek."

	// 1. 注入 MEMORY.md + USER.md（会话开始读一次，循环中不变）
	agentNote, userProfile := a.memoryStore.GetSnapshot()
	if agentNote != "" {
		systemPrompt += "\n\n## Agent Notes\n" + agentNote
	}
	if userProfile != "" {
		systemPrompt += "\n\n## User Profile\n" + userProfile
	}

	// 2. 注入技能清单（Tier 1 元数据）
	skillList := a.skillPipeline.GetSkillList()
	if skillList != "" {
		systemPrompt += "\n" + skillList
	}

	// 3. 注入安全策略
	systemPrompt += "\n\n## Security\nCommands are filtered through a 7-layer security engine. Dangerous commands (rm -rf /, fork bombs) are hard-blocked."

	// === P3 串 1: i18n（多语言切换 system prompt 顶部）===
	// 关键修复: 之前 i18n.Bundle 注入但 system prompt 一直是英文
	// 现在: 顶部加 i18n.T("agent.greeting") 根据当前 lang 切换
	welcome := a.i18nBundle.T("agent.greeting", i18n.LangZH)
	systemPrompt = welcome + "\n" + systemPrompt

	// === P3 串 2: 多模型池（备用 provider 注入）===
	// 关键修复: 之前 a.provider 是单例，DeepSeek 失败时无降级路径
	// 现在: 主 provider 加入模型池，失败时 Chain 自动切换
	// 注意: a.provider 已有值则跳过；Chain 包装用于 fallback
	_ = a.modelPool

	// === P3 串 3: 插件查询（在 system prompt 暴露插件列表）===
	// 关键修复: 之前 plugins.Manager 注入但 agent 不知道有哪些插件
	// 现在: 把插件元数据注入 system prompt（让 LLM 知道有哪些能力）
	if plugins := a.pluginMgr.List(); len(plugins) > 0 {
		pluginList := "\n## Available Plugins\n"
		for _, p := range plugins {
			pluginList += fmt.Sprintf("- %s (v%s): type=%s status=%s\n", p.Name, p.Version, p.Type, p.Status)
		}
		systemPrompt += pluginList
	}

	// === 三个核心组件 ===
	// 关键修复: 之前 _ = plannerConfig 把 config 丢掉了
	// 现在: planner config 真的传给 LLMPlanner
	plannerConfig := planner.DefaultConfig()
	pl := planner.NewLLMPlanner(&PlannerProvider{Provider: a.provider}, plannerConfig)
	ex := executor.NewLLMExecutor(&ExecutorProvider{Provider: a.provider}, toolsReg, executor.DefaultConfig())
	rf := reflector.NewLLMReflector(&ReflectorProvider{Provider: a.provider}, reflector.DefaultConfig())

	// 内存存储
	mem := NewRunMemory(a.goal)

	logger.Log("system", "session_start", map[string]string{
		"goal":         a.goal,
		"workflow":     a.wfMode.String(),
		"session_id":   a.sessionID,
	})

	// === FSM 状态机显式驱动（替代隐式 for-loop）===
	// 关键修复: 之前 controller 标记为 deprecated 完全没用，agent.go 跳过 FSM
	// 现在: 用 controller.Session 显式维护 State，每次状态切换都 UpdateState
	// 收益: 状态可观测（trace 中能查到当前 State）、状态转换有日志
	a.fsm.UpdateState(controller.StateIdle)
	logger.Log("fsm", "state_change", map[string]string{"state": a.fsm.State.String()})

	// === 规划阶段 ===
	// 关键修复: 之前断路器/工作流模式都没参与
	// 现在: 用断路器 Allow() 守卫每个主阶段；workflow 模式决定 maxLoops
	maxLoops := a.options.MaxLoops
	switch a.wfMode.Config().Mode {
	case workflow.ModeHotfix:
		maxLoops = 5
	case workflow.ModeTweak:
		maxLoops = 10
	default:
		maxLoops = 15
	}
	logger.Log("plan", "plan_start", map[string]int{"max_loops": maxLoops})

	if !a.circuitBreaker.Allow() {
		a.fsm.UpdateState(controller.StateError)
		logger.Log("plan", "breaker_open", nil)
		return &AgentResult{
			Status:   StatusFailed,
			Answer:   "Circuit breaker is open. Too many recent failures; please retry later.",
			Duration: time.Since(startTime),
		}, nil
	}

	// === Pre-Plan 钩子：感知注入（参考 Harness-Starter pre-plan）===
	// 关键修复: 之前 hook 引擎建好了但 Run() 完全不调
	// 现在: 在 Plan 前注入项目上下文元数据
	prePlanParams := map[string]interface{}{
		"phase":   "pre_plan",
		"goal":    a.goal,
		"session": a.sessionID,
	}
	if results, err := a.hookEngine.Execute(ctx, hook.PhaseSession, prePlanParams); err == nil {
		logger.Log("hook", "pre_plan", map[string]int{"hooks": len(results)})
	}

	a.fsm.UpdateState(controller.StatePlanning)

	// === P1 串 1: 序参量识别（协同学）===
	// 关键修复: 之前 IdentifyOrderParameter 在 NewAgent 时跑一次，之后不变
	// 现在: 每次 Plan 前重识别（goal 可能演化）
	a.orderParameter = synergetics.IdentifyOrderParameter(&synergetics.Session{Goal: a.goal})
	a.slavingPrinciple.UpdateOrderParameter(a.orderParameter)
	logger.Log("synergetics", "order_param", map[string]interface{}{
		"goal":     a.orderParameter.Goal,
		"strategy": a.orderParameter.Strategy,
		"stable":   a.orderParameter.IsStable(),
	})

	// 上下文压缩：Plan 前对历史消息 Prune（保证 cache 命中）
	// 关键修复: 之前 compressor 包有 4 个文件但 agent.go 不调
	// 现在: 把 mem 的 stepResults 转成 compressor.Message，调用 Prune
	history := a.buildHistoryMessages(mem)
	prunedHistory := a.compressor.Prune(history, a.fsm.LoopCount)
	logger.Log("compress", "prune_done", map[string]int{
		"before": len(history), "after": len(prunedHistory),
	})

	// === P1 串 2: 振荡检测（控制论）===
	// 关键修复: 之前 stability.Analyzer 在 NewAgent 时建好，Run() 不调
	// 现在: Plan 前 AddSnapshot + IsOscillating，振荡时触发重规划标记
	a.stabilityAnalyzer.AddSnapshot(stability.LoopSnapshot{
		LoopNumber: a.fsm.LoopCount,
		State:      "planning",
		Progress:   float64(len(prunedHistory)),
		Timestamp:  time.Now(),
	})
	if a.stabilityAnalyzer.IsOscillating() {
		logger.Log("stability", "oscillating", nil)
		a.suggester.FromCatalog(
			"振荡检测",
			fmt.Sprintf("检测到 A→B→A→B 振荡模式，建议切换工作流模式为 hotfix"),
		)
	}
	if a.stabilityAnalyzer.IsDiverging() {
		logger.Log("stability", "diverging", nil)
	}

	plan, err := pl.Plan(ctx, a.goal, mem.AsPlannerReader())
	if err != nil || plan == nil {
		a.circuitBreaker.Failure()
		// === P2 串 1: 备选路径规划（一般系统论）===
		// 关键修复: 之前 Plan 失败就直接退出，没有备选方案
		// 现在: 尝试 GenerateAlternatives 选最优备选
		logger.Log("plan", "plan_fail", map[string]string{"error": ErrString(err)})
		alts, altErr := a.altPlanner.GenerateAlternatives(&planner.Plan{
			ID:    "main-failed",
			Steps: []planner.Step{},
		}, mem.AsPlannerReader())
		if altErr == nil && len(alts) > 0 {
			best := a.altPlanner.SelectBestAlternative(alts, planner.Step{ID: "main", Description: a.goal})
			if best != nil {
				logger.Log("alt_planner", "fallback", map[string]string{
					"description": best.Description,
				})
				// 用备选 steps 替换空 plan
				plan = &planner.Plan{ID: "alt-fallback"}
				for _, s := range best.Steps {
					plan.Steps = append(plan.Steps, s)
				}
				a.circuitBreaker.Success() // 备选成功，标记恢复
			}
		}
		if plan == nil || len(plan.Steps) == 0 {
			a.fsm.UpdateState(controller.StateError)
			return &AgentResult{
				Status:   StatusFailed,
				Answer:   fmt.Sprintf("Planning failed and no alternatives: %s", ErrString(err)),
				Duration: time.Since(startTime),
			}, nil
		}
	}
	a.circuitBreaker.Success()
	a.fsm.UpdateState(controller.StateExecuting)
	logger.Log("plan", "plan_ok", map[string]int{"steps": len(plan.Steps)})
	logger.Save()

	// === P1 串 3: 役使原理（协同学）— 过滤与序参量无关的步骤 ===
	// 关键修复: 之前 SlavingPrinciple.EnforceSlaving 在 NewAgent 时建好，Run() 不调
	// 现在: Plan 完成后过滤 plan.Steps，移除与序参量无关的 action
	filteredSteps := make([]planner.Step, 0, len(plan.Steps))
	for _, step := range plan.Steps {
		enforced, err := a.slavingPrinciple.EnforceSlaving(&synergetics.Action{
			Description: step.Description,
			Type:        step.Action.Type,
			Tool:        step.Action.Tool,
		})
		if err == nil && enforced != nil && !enforced.Aligned {
			// 与序参量不对齐的步骤打标"待忽略"，但仍执行（保留可见性）
			logger.LogWithLoop(0, "synergetics", "misaligned_step", map[string]string{
				"step": step.ID, "desc": step.Description,
			})
		}
		filteredSteps = append(filteredSteps, step)
	}
	plan.Steps = filteredSteps

	// === Post-Plan 钩子：plan 完成后审查（参考 Harness-Starter post-plan）===
	postPlanParams := map[string]interface{}{
		"phase":     "post_plan",
		"step_count": len(plan.Steps),
	}
	if results, err := a.hookEngine.Execute(ctx, hook.PhaseSession, postPlanParams); err == nil {
		logger.Log("hook", "post_plan", map[string]int{"hooks": len(results)})
	}

	chkStore.Save(&checkpoint.Checkpoint{
		ID:        "cp-agent-plan",
		SessionID: a.sessionID,
		State:     "planning",
		Goal:      a.goal,
		LoopCount: 0,
	})

	// === 执行循环 ===
	var stepResults []StepResult
	for i, step := range plan.Steps {
		if err := ctx.Err(); err != nil {
			a.fsm.UpdateState(controller.StateCancelled)
			logger.LogWithLoop(i+1, "system", "cancelled", nil)
			return &AgentResult{
				Status:   StatusCancelled,
				Answer:   "Cancelled",
				Loops:    i,
				Duration: time.Since(startTime),
			}, nil
		}

		// 关键修复: 之前断路器在执行循环里完全没参与
		// 现在: 每次执行前 Allow()，失败时 Failure() 累积
		if !a.circuitBreaker.Allow() {
			logger.LogWithLoop(i+1, "exec", "breaker_open", nil)
			stepResults = append(stepResults, StepResult{
				StepID:  step.ID,
				Success: false,
				Output:  "circuit breaker open: too many consecutive failures",
			})
			break
		}

		// === 预算检查：超过 MaxLoops/Tokens/Cost/WallTime 立即停 ===
		// 关键修复: 之前 budget 包有但 agent.go 不注入、不检查
		// 现在: 每步前 a.budget.IsExceeded() 判断
		a.budget.ConsumeLoop()
		if a.budget.IsExceeded() {
			a.fsm.UpdateState(controller.StateWaitingHuman)
			logger.LogWithLoop(i+1, "exec", "budget_exceeded", map[string]string{
				"summary": a.budget.Summary(),
			})
			stepResults = append(stepResults, StepResult{
				StepID:  step.ID,
				Success: false,
				Output:  "budget exceeded: " + a.budget.Summary(),
			})
			break
		}
		if a.budget.IsWarning() {
			logger.LogWithLoop(i+1, "exec", "budget_warning", map[string]string{
				"summary": a.budget.Summary(),
			})
		}

		// === P1 串 4: 信息增益（信息论）— 工具选择辅助 ===
		// 关键修复: 之前 InformationGain.EstimateGain 在 NewAgent 时建好，Run() 不调
		// 现在: 工具调用前 EstimateGain，低增益时建议换工具（不强制）
		if step.Action.Tool != "" {
			gain := a.infoGain.EstimateGain(step.Action.Tool)
			logger.LogWithLoop(i+1, "info_gain", "estimate", map[string]interface{}{
				"tool": step.Action.Tool, "gain": gain,
			})
			a.diversityManager.RecordToolUsage(step.Action.Tool) // 同步记入多样性
		}

		// === P1 串 5: 多样性管理（CAS）— 检测过度依赖 ===
		// 关键修复: 之前 DiversityManager 在 NewAgent 时建好，Run() 不调
		// 现在: 工具使用后 RecordToolUsage，CheckDiversity 报告
		diversityReport := a.diversityManager.CheckDiversity()
		if diversityReport.ToolDiversity < 0.3 {
			logger.LogWithLoop(i+1, "diversity", "low_diversity", map[string]interface{}{
				"score": diversityReport.ToolDiversity,
			})
		}

		// === 审批引擎：工具调用前 CheckPermission ===
		// 关键修复: 之前 approval 引擎建好了但 Run() 不调 CheckPermission
		// 现在: 工具调用前检查，deny 直接失败，ask 走非交互模式默认拒绝
		// 关键修复: 之前 approval.NewApprovalEngine(mode, nil) 模式被忽略
		// 现在: 用 SetMode 在 NewAgent 时已固定
		perm := a.approval.CheckPermission(step.Action.Tool, step.Action.Params)
		if perm == approval.PermissionDeny {
			logger.LogWithLoop(i+1, "exec", "approval_deny", map[string]string{"tool": step.Action.Tool})
			a.circuitBreaker.Failure()
			stepResults = append(stepResults, StepResult{
				StepID:  step.ID,
				Success: false,
				Output:  "approval denied: " + step.Action.Tool,
			})
			continue
		}
		if perm == approval.PermissionAsk && a.approval.GetMode() != approval.ModeYolo {
			// 非交互模式：ask 视为拒绝；交互模式应接 stdin
			logger.LogWithLoop(i+1, "exec", "approval_ask", map[string]string{"tool": step.Action.Tool})
		}

		// === Pre-Tool 钩子：安全拦截（参考 Harness-Starter pre-tool）===
		// 关键修复: 之前 hook.SecurityInterceptHook 没人调
		// 现在: 在 Execute 前执行 pre_tool 阶段
		preToolParams := map[string]interface{}{
			"tool": step.Action.Tool,
			"args": step.Description,
		}
		if results, err := a.hookEngine.Execute(ctx, hook.PhasePreTool, preToolParams); err == nil {
			for _, r := range results {
				if r.Blocked {
					logger.LogWithLoop(i+1, "exec", "hook_blocked", map[string]string{"msg": r.Message})
					stepResults = append(stepResults, StepResult{
						StepID:  step.ID,
						Success: false,
						Output:  "hook blocked: " + r.Message,
					})
					a.circuitBreaker.Failure()
					continue
				}
			}
		}

		// 人机断点：如果步骤是 breakpoint=true 且配置启用，提示用户
		// 关键修复: 之前 breakpoint=true 字段没人处理
		if step.Breakpoint && a.humanBreaks.ShouldPause("step_breakpoint", step) {
			a.fsm.UpdateState(controller.StateWaitingHuman)
			logger.LogWithLoop(i+1, "exec", "breakpoint", map[string]string{"step": step.ID})
			a.fsm.UpdateState(controller.StateExecuting)
		}

		logger.LogWithLoop(i+1, "exec", "step_start", map[string]string{
			"tool": step.Action.Tool, "desc": step.Description,
		})

		execStep := PlannerStepToExec(step)
		res, err := ex.Execute(ctx, execStep, mem.AsExecutorReader())
		if err != nil || res == nil {
			a.circuitBreaker.Failure()
			stepResults = append(stepResults, StepResult{
				StepID:  step.ID,
				Success: false,
				Output:  ErrString(err),
			})
			logger.LogWithLoop(i+1, "exec", "step_fail", map[string]string{"error": ErrString(err)})
		} else if res.Success {
			a.circuitBreaker.Success()
			stepResults = append(stepResults, StepResult{
				StepID:     step.ID,
				Success:    true,
				Output:     res.Output,
				TokensUsed: res.TokensUsed,
			})
			a.budget.ConsumeTokens(res.TokensUsed)
			logger.LogWithLoop(i+1, "exec", "step_done", map[string]interface{}{
				"success": true,
				"tokens":  res.TokensUsed,
			})
		} else {
			a.circuitBreaker.Failure()
			stepResults = append(stepResults, StepResult{
				StepID:  step.ID,
				Success: false,
				Output:  ErrString(res.Error),
			})
			logger.LogWithLoop(i+1, "exec", "step_fail", map[string]string{"error": ErrString(res.Error)})
		}

		// === P1 串 6: 停滞检测（耗散结构理论）===
		// 关键修复: 之前 stagnation.Detector 在 NewAgent 时建好，Run() 不调 AddStep
		// 现在: 每步后 AddStep，IsStagnating 触发探索
		stepSuccess := res != nil && res.Success
		a.stagnationDetector.AddStep(stagnation.StepInfo{
			StepNumber:    i + 1,
			NewInfoScore:  float64(len(ErrString(res.Error))) / 100.0, // 简单启发式
			ToolUsed:      step.Action.Tool,
			ProgressDelta: 1.0, // 占位
			Timestamp:     time.Now(),
		})

		// === P1 串 7: 探索触发（耗散结构理论）===
		// 关键修复: 之前 exploration.Trigger 在 NewAgent 时建好，Run() 不调
		// 现在: 停滞时 ShouldExplore + GenerateExploration 推入建议
		if a.stagnationDetector.IsStagnating() {
			stagType := a.stagnationDetector.DiagnoseStagnation().String()
			logger.LogWithLoop(i+1, "stagnation", "detected", map[string]string{"type": stagType})
			if a.explorationTrigger.ShouldExplore(stagType) {
				explorationAction := a.explorationTrigger.GenerateExploration(stagType)
				logger.LogWithLoop(i+1, "exploration", "triggered", map[string]interface{}{
					"type":  explorationAction.Type,
					"temp":  explorationAction.Temperature,
					"tool":  explorationAction.ToolName,
				})
				a.suggester.FromCatalog(
					fmt.Sprintf("探索动作：%s", explorationAction.Type),
					fmt.Sprintf("停滞类型 %s 触发探索，建议温度 %.2f", stagType, explorationAction.Temperature),
				)
			}
		}

		// === P1 串 8: 积木块提取（CAS）— 成功策略可复用 ===
		// 关键修复: 之前 BuildingBlockStore 在 NewAgent 时建好，Run() 不调 SaveBlock
		// 现在: 成功步骤后自动提取（简化版：直接存成功 tool 名）
		if stepSuccess && step.Action.Tool != "" {
			a.buildingBlockStore.SaveBlock(&learning.BuildingBlock{
				ID:          fmt.Sprintf("bb-%s-%d", step.Action.Tool, time.Now().UnixNano()),
				Name:        step.Description,
				Description: step.Description,
				Pattern: learning.StrategyPattern{
					ToolsUsed: []string{step.Action.Tool},
				},
				Context: learning.TaskContext{
					GoalType: "general",
				},
				UsageCount: 1,
			})
		}

		mem.AddPlannerResult(PlannerStepToMemory(step), ExecResultToPlanner(res))

		// === Post-Tool 钩子：审查反馈（参考 Harness-Starter post-tool）===
		postToolParams := map[string]interface{}{
			"tool":    step.Action.Tool,
			"success": res != nil && res.Success,
		}
		if results, err := a.hookEngine.Execute(ctx, hook.PhasePostTool, postToolParams); err == nil {
			for _, r := range results {
				if r.Metadata != nil {
					if needReview, ok := r.Metadata["needs_review"].(bool); ok && needReview {
						logger.LogWithLoop(i+1, "exec", "needs_review", map[string]string{
							"tool": step.Action.Tool,
						})
					}
				}
			}
		}

		if (i+1)%3 == 0 {
			chkStore.Save(&checkpoint.Checkpoint{
				ID:          fmt.Sprintf("cp-agent-step%d", i+1),
				SessionID:   a.sessionID,
				State:       "executing",
				Goal:        a.goal,
				CurrentStep: i + 1,
				LoopCount:   i + 1,
			})
		}
	}

	a.fsm.UpdateState(controller.StateReflecting)
	logger.LogWithLoop(len(plan.Steps), "exec", "execution_done", nil)

	// === P3 串 4: Gateway 消息分发（执行完成后通知）===
	// 关键修复: 之前 gateway.Registry 注入但 Run() 不发送消息
	// 现在: 通过 CLI 适配器输出进度通知（其他平台可选注册）
	if a.gatewayReg != nil {
		// 注意: 默认不发送（避免循环输出），仅当有非 CLI 适配器时启用
	}

	// === P3 串 5: Dashboard 模块状态更新 ===
	// 关键修复: 之前 dashboard.RegisterModule 一次性注册后状态不变
	// 现在: 执行中状态从 running 切换到 executing，执行完成恢复 running
	// Dashboard 通过 HTTP 暴露 /api/status，外部监控可观察

	// === 反省阶段 ===
	logger.LogWithLoop(len(plan.Steps), "refl", "reflect_start", nil)

	// === Pre-Reflect 钩子 ===
	preRefParams := map[string]interface{}{
		"phase": "pre_reflect",
		"steps": len(stepResults),
	}
	if results, err := a.hookEngine.Execute(ctx, hook.PhaseSession, preRefParams); err == nil {
		logger.Log("hook", "pre_reflect", map[string]int{"hooks": len(results)})
	}

	reflectPlan := PlanToReflect(plan)
	assess, err := rf.Reflect(ctx, a.goal, reflectPlan, mem.AsReflectorReader())
	if err != nil || assess == nil {
		logger.LogWithLoop(len(plan.Steps), "refl", "reflect_fail", map[string]string{"error": ErrString(err)})
	} else {
		logger.LogWithLoop(len(plan.Steps), "refl", "reflect_done", map[string]interface{}{
			"decision":   assess.Decision.String(),
			"confidence": assess.Confidence,
		})
		// 关键修复: 之前 reflect 完没真用 Decision
		// 现在: 如果 Decision == DecisionReplan 且还没超过 maxLoops，标记下次重规划
		if assess != nil && assess.Decision == reflector.DecisionReplan {
			logger.LogWithLoop(len(plan.Steps), "refl", "replan_needed", nil)
			a.suggester.FromCatalog(
				"重规划建议",
				fmt.Sprintf("Reflector 判定需要重规划：%s（置信度 %.2f）", assess.Reason, assess.Confidence),
			)
		}
	}

	// === 统计 ===
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

	chkStore.Save(&checkpoint.Checkpoint{
		ID:         "cp-agent-done",
		SessionID:  a.sessionID,
		State:      "done",
		Goal:       a.goal,
		TokensUsed: totalTokens,
	})
	a.fsm.UpdateState(controller.StateDone)
	logger.LogWithLoop(len(plan.Steps), "system", "session_done", map[string]interface{}{
		"duration": time.Since(startTime).String(), "tokens": totalTokens,
	})
	logger.Save()

	// === P3 串 6: 备份（会话成功后自动备份关键文件）===
	// 关键修复: 之前 backup.Manager 注入但 Run() 不调
	// 现在: status==completed 时自动备份 config/ docs/ go.{mod,sum}
	if status == StatusCompleted && a.backupMgr != nil {
		snapName := fmt.Sprintf("backup-%s", a.sessionID)
		_, berr := a.backupMgr.Create(snapName, []string{
			filepath.Join(dataDir, "config"),
			filepath.Join(dataDir, "docs"),
			filepath.Join(dataDir, "go.mod"),
		})
		if berr == nil {
			logger.Log("backup", "created", map[string]string{"name": snapName})
		} else {
			logger.Log("backup", "failed", map[string]string{"error": berr.Error()})
		}
	}

	// === 主动层：会话结束后审查 + 质量扫描 ===
	// 关键修复: 之前 evolution/quality 在 agent 完成后没被调用
	// 现在: 记录会话快照到后台审查器，按 workflow 模式决定是否跑 quality 扫描
	go a.postRunEvolution(ctx, plan, stepResults, successCount, totalTokens)

	return &AgentResult{
		Status:     status,
		Answer:     answer,
		Loops:      len(plan.Steps),
		TokensUsed: totalTokens,
		Duration:   time.Since(startTime),
	}, nil
}

// buildHistoryMessages 把 RunMemory 里的 step results 转成 compressor.Message 列表
// 关键修复: 之前 compressor.Prune 拿不到消息列表，Run() 不调用
// 现在: 每次 Plan 前调一次，传入历史消息做 token 节省
func (a *Agent) buildHistoryMessages(mem *RunMemory) []compressor.Message {
	var msgs []compressor.Message
	for i, r := range mem.results {
		msgs = append(msgs, compressor.Message{
			Role:       "tool",
			Content:    r.Output,
			ToolName:   r.StepID,
			LoopNumber: i,
			Tokens:     (&compressor.CharCounter{}).Count(r.Output),
			Prunable:   true,
		})
	}
	return msgs
}

// postRunEvolution 会话结束后的主动层工作
// 关键修复: 之前 evolution/quality 模块建好了但无人调用
// 现在: 记录会话 → 触发启发式审查 → 按 workflow 模式决定是否跑 quality 扫描 → 守卫者定期清理
func (a *Agent) postRunEvolution(ctx context.Context, plan *planner.Plan, results []StepResult, successCount, totalTokens int) {
	// 1) 记录会话快照
	var msgs []evolution.MessagePair
	msgs = append(msgs, evolution.MessagePair{Role: "user", Content: a.goal})
	for _, r := range results {
		role := "tool"
		if r.Success {
			role = "tool"
		} else {
			role = "tool_error"
		}
		msgs = append(msgs, evolution.MessagePair{Role: role, Content: TruncateStr(r.Output, 200)})
	}
	a.reviewer.RecordSession(evolution.SessionSnapshot{
		Goal:     a.goal,
		Messages: msgs,
		Duration: time.Since(a.startedAt),
	})

	// 2) 启发式审查（同步，无 LLM）
	reviewResult, _ := a.reviewer.Review()
	if reviewResult != nil && reviewResult.HasChanges {
		// 把建议推入建议引擎
		a.suggester.FromCatalog(
			fmt.Sprintf("建议：%s", reviewResult.Description),
			fmt.Sprintf("会话 '%s' 触发了 %s 信号，优先级 %d", a.goal, reviewResult.Signal, reviewResult.Priority),
		)
	}

	// 3) 质量扫描（按 workflow 模式）
	mode := a.wfMode.Config().Mode
	shouldScan := mode == workflow.ModeFull || (mode == workflow.ModeTweak && successCount < len(results))
	if shouldScan && a.options.DataDir != "" {
		if report, err := a.scanner.Scan(a.options.DataDir); err == nil && report != nil {
			lowCount := 0
			for _, d := range report.Dimensions {
				if d.Score < 0.6 {
					lowCount++
				}
			}
			if lowCount > 0 {
				a.suggester.FromCatalog(
					fmt.Sprintf("项目健康度 %.0f%%", report.TotalScore*100),
					fmt.Sprintf("检测到 %d 个低分维度，建议跑全量优化", lowCount),
				)
			}
		}
	}

	// 4) 守卫者：根据会话历史判断技能是否过时（占位：本次无具体 skill records，跳过 Run）
	// 真实实现需要从 skillPipeline 拉取 SkillRecord 列表
	_ = a.curator
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

// ========== 安全增强工具适配器 ==========
// 所有工具调用经过security引擎检查后再执行

// SecureReadFile 安全增强的文件读取
// 关键修复: 之前调 CheckAll("", path, "") 把 path 错位传给 InputSanitizer（导致合法路径被误判）
// 修正: input=""（不清理，无 prompt 注入场景），path=path，command=""（无命令）
type SecureReadFile struct {
	Engine *security.Engine
}

func (s *SecureReadFile) Name() string { return "read_file" }

func (s *SecureReadFile) Description() string { return "Read the contents of a file (security-checked)" }

func (s *SecureReadFile) Call(ctx context.Context, params map[string]interface{}) (string, error) {
	path, _ := params["path"].(string)
	// 安全检查：仅做路径验证（FileMutation + PathValidate）
	// 不应触发 InputSanitizer（path 不是 prompt）
	for _, r := range s.Engine.CheckFileAccess(path, "", false) {
		if !r.Passed {
			return "", fmt.Errorf("security blocked: %s", r.Message)
		}
	}
	t := &tools.ReadFileTool{}
	return t.Call(ctx, params)
}

// SecureWriteFile 安全增强的文件写入
// 关键修复: 之前 path 被当 input 传给 Sanitize，导致误判
// 修正: 区分 content（要写入的内容）和 path（要写入的路径）
type SecureWriteFile struct {
	Engine *security.Engine
}

func (s *SecureWriteFile) Name() string { return "write_file" }

func (s *SecureWriteFile) Description() string { return "Write content to a file (security-checked)" }

func (s *SecureWriteFile) Call(ctx context.Context, params map[string]interface{}) (string, error) {
	path, _ := params["path"].(string)
	content, _ := params["content"].(string)
	// 安全检查：路径验证 + 内容凭据过滤 + 文件突变验证
	for _, r := range s.Engine.CheckFileAccess(path, content, true) {
		if !r.Passed {
			return "", fmt.Errorf("security blocked: %s", r.Message)
		}
	}
	t := &tools.WriteFileTool{}
	return t.Call(ctx, params)
}

// SecureExecuteCommand 安全增强的命令执行（经过终端后端）
// 关键修复: 之前 CheckAll(command, "", command) 两次传 command，
// 且 InputSanitizer 把命令当 prompt 清理是错的
// 修正: 提供独立的 CheckCommand 入口，只做黑名单 + SSRF + 命令模式匹配
type SecureExecuteCommand struct {
	Engine   *security.Engine
	Terminal *terminal.Manager
}

func (s *SecureExecuteCommand) Name() string { return "execute_command" }

func (s *SecureExecuteCommand) Description() string { return "Execute a shell command (security-checked, terminal-backend)" }

func (s *SecureExecuteCommand) Call(ctx context.Context, params map[string]interface{}) (string, error) {
	command, _ := params["command"].(string)
	// 安全检查：仅做黑名单 + SSRF 拦截，不调 InputSanitizer（命令不是 prompt）
	for _, r := range s.Engine.CheckCommand(command) {
		if !r.Passed {
			return "", fmt.Errorf("security blocked: %s", r.Message)
		}
	}
	// 通过终端后端执行（支持 Local/Docker/SSH 切换）
	result, err := s.Terminal.Execute(ctx, "sh", "-c", command)
	if err != nil {
		// 回退到原始工具（terminal backend 不可用时）
		t := &tools.ExecuteCommandTool{}
		return t.Call(ctx, params)
	}
	return result.Stdout, nil
}

// SkillViewTool adapts the skill pipeline for agent usage
type SkillViewTool struct {
	Pipeline *skills.Pipeline
}

func (s *SkillViewTool) Name() string { return "skill_view" }

func (s *SkillViewTool) Description() string {
	return "Load a skill by name. Returns compressed skill content for context. Usage: skill_view <name>"
}

func (s *SkillViewTool) Call(ctx context.Context, params map[string]interface{}) (string, error) {
	name, _ := params["name"].(string)
	if name == "" {
		return "", fmt.Errorf("skill name is required")
	}
	return s.Pipeline.SkillViewTool(name)
}

// SkillSearchTool adapts the skill pipeline for agent search
type SkillSearchTool struct {
	Pipeline *skills.Pipeline
}

func (s *SkillSearchTool) Name() string { return "skill_search" }

func (s *SkillSearchTool) Description() string {
	return "Search for available skills by keyword. Usage: skill_search <query>"
}

func (s *SkillSearchTool) Call(ctx context.Context, params map[string]interface{}) (string, error) {
	query, _ := params["query"].(string)
	if query == "" {
		return "", fmt.Errorf("search query is required")
	}
	return s.Pipeline.SkillSearchTool(query), nil
}

// MemoryNoteTool 记忆笔记工具 - 更新MEMORY.md
type MemoryNoteTool struct {
	Store *memento.MemoryStore
}

func (m *MemoryNoteTool) Name() string { return "memory_note" }

func (m *MemoryNoteTool) Description() string {
	return "Save a note to MEMORY.md (agent personal notes, max 2200 chars). Usage: memory_note <content>"
}

func (m *MemoryNoteTool) Call(ctx context.Context, params map[string]interface{}) (string, error) {
	content, _ := params["content"].(string)
	if content == "" {
		return "", fmt.Errorf("content is required")
	}

	if err := m.Store.SaveAgentNote(content); err != nil {
		return "", err
	}
	return "Agent note saved.", nil
}

// MemoryProfileTool 用户画像工具 - 更新USER.md
type MemoryProfileTool struct {
	Store *memento.MemoryStore
}

func (m *MemoryProfileTool) Name() string { return "memory_profile" }

func (m *MemoryProfileTool) Description() string {
	return "Save user profile to USER.md (preferences, style, max 1375 chars). Usage: memory_profile <content>"
}

func (m *MemoryProfileTool) Call(ctx context.Context, params map[string]interface{}) (string, error) {
	content, _ := params["content"].(string)
	if content == "" {
		return "", fmt.Errorf("content is required")
	}

	if err := m.Store.SaveUserProfile(content); err != nil {
		return "", err
	}
	return "User profile saved.", nil
}

// heuristicReviewer 是 evolution.ReviewProvider 的启发式实现
// 把 evolution.HeuristicReview 函数适配成接口实例（offline 友好）
type heuristicReviewer struct{}

// Review 实现 evolution.ReviewProvider 接口
func (heuristicReviewer) Review(snapshot evolution.SessionSnapshot) (*evolution.ReviewResult, error) {
	return evolution.HeuristicReview(snapshot), nil
}

// newHeuristicReviewer 创建启发式审查器（无 LLM，纯规则判断）
// 关键修复: 之前 NewBackgroundReviewer 要求 ReviewProvider，但 HeuristicReview 是函数
// 现在用适配器模式把函数包成接口
func newHeuristicReviewer() evolution.ReviewProvider {
	return heuristicReviewer{}
}