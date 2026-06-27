# Zhulong（烛龙） 详细设计文档

> 版本: v0.3-draft
> 日期: 2026-06-21
> 语言: Go
> 状态: 设计阶段（按优先级整理，已通过可行性审查）

---

## 1. 项目定位

Zhulong（烛龙）是一个**基于 DeepSeek 的通用自主循环 Agent 框架**，核心特性：

- **自主多轮循环**：规划 → 执行 → 反省 → 重新规划，无需每轮人工触发
- **高效上下文管理**：深度优化 DeepSeek prefix-cache，最大化缓存命中率
- **自适应学习**：成功策略复用、认知模型更新、探索/利用平衡
- **生产级可靠性**：检查点恢复、成本控制、人机协作断点、完整可观测性

### 1.1 设计目标

| 目标 | 说明 | 优先级 |
|------|------|--------|
| **高缓存命中率** | **深度优化 DeepSeek prefix-cache，上下文布局只追加不重写** | 🔴 最高 |
| CLI + 桌面端同步 | CLI 和 Windows 桌面端同步开发，共享核心逻辑 | 🔴 最高 |
| 通用性 | 不绑定特定场景（编程/写作/数据分析），通过工具和 prompt 定制 | 🟡 高 |
| 自适应学习 | 成功策略复用、认知模型更新、探索/利用平衡 | 🟡 高 |
| 崩溃可恢复 | 任意时刻中断都能从最近检查点恢复 | 🟡 高 |
| 成本可控 | 多层预算机制，防止无限循环烧 token | 🟡 高 |
| 可观测 | 完整 trace，每步决策可追溯 | 🟢 中 |

### ⚠️ 最高铁律

> **所有架构补充、优化、新功能，绝对不能破坏高缓存命中率这一核心优势。**
>
> 这是 Zhulong 的根基。任何模块如果会破坏 prefix 稳定性，必须重新设计或放弃。
>
> 详见 §4 缓存命中率保障机制。

### 1.2 技术约束

| 约束 | 说明 |
|------|------|
| LLM | **仅支持 DeepSeek**（prefix-cache 优化深度绑定 DeepSeek API） |
| 语言 | Go（核心逻辑 + CLI）|
| 桌面端 | Windows（Go + Wails） |
| UI 设计 | Apple Design 风格（简洁、圆角、毛玻璃、留白） |
| 工具协议 | MCP（Model Context Protocol） |
| 架构 | CLI 和桌面端共享 `pkg/` 核心逻辑 |

### 1.3 致谢

本项目设计思想受 [DeepSeek-Reasonix](https://github.com/esengine/DeepSeek-Reasonix) 启发。所有代码均为独立实现。

### 1.4 理论基础

Zhulong 融合六大系统科学理论（仅保留有实际应用价值的部分）：

| 理论 | 核心应用 | 可行性 |
|------|---------|--------|
| 一般系统论（贝塔朗菲） | 开放系统、备选路径 | ✅ 已实现 |
| 工程控制论（钱学森） | 振荡/发散检测、性能指标 | ✅ 已实现 |
| 信息论（香农） | 信息增益、信息密度 | ✅ 已实现 |
| 耗散结构理论（普利高津） | 停滞检测、探索触发 | ✅ 已实现 |
| 协同学（哈肯） | 序参量识别、役使原理 | ✅ 已实现 |
| 复杂适应系统（霍兰德） | 积木块、认知模型、多样性 | ✅ 已实现 |

---

## 2. 整体架构

```
┌──────────────────────────────────────────────────────────────────┐
│                          Zhulong（烛龙）                          │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐    │
│  │                     Controller                            │    │
│  │                                                           │    │
│  │   ┌────────┐    ┌────────┐    ┌──────────┐    ┌────────┐ │    │
│  │   │Planner │───→│Executor│───→│ Reflector│───→│RePlanner│ │    │
│  │   └───▲────┘    └────────┘    └─────┬────┘    └───┬────┘ │    │
│  │       │                             │             │       │    │
│  │       └─────────────┬───────────────┘             │       │    │
│  │                     ▼                             │       │    │
│  │              ┌─────────────┐                      │       │    │
│  │              │   Goal FSM  │◄─────────────────────┘       │    │
│  │              │   目标状态机  │                              │    │
│  │              └──────┬──────┘                              │    │
│  └─────────────────────┼──────────────────────────────────────┘    │
│                        │                                          │
│  ┌─────────────────────┼──────────────────────────────────────┐    │
│  │                     ▼           Infrastructure             │    │
│  │  ┌────────────┐ ┌────────────┐ ┌────────────┐ ┌──────────┐│    │
│  │  │   Memory   │ │ Compressor │ │   Tools    │ │Checkpoint││    │
│  │  │  三层记忆   │ │  上下文压缩 │ │  工具抽象层 │ │  检查点  ││    │
│  │  └────────────┘ └────────────┘ └────────────┘ └──────────┘│    │
│  │  ┌────────────┐ ┌────────────┐ ┌────────────┐ ┌──────────┐│    │
│  │  │   Budget   │ │   Trace    │ │   Human    │ │ Learning ││    │
│  │  │  成本控制   │ │  可观测性   │ │  人机协作   │ │ 自适应学习││    │
│  │  └────────────┘ └────────────┘ └────────────┘ └──────────┘│    │
│  └────────────────────────────────────────────────────────────┘    │
│                                                                    │
│  ┌────────────────────────────────────────────────────────────┐    │
│  │                    Providers 接口层                         │    │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐     │    │
│  │  │ DeepSeek API │  │  MCP Client  │  │ Custom Tools │     │    │
│  │  │  深度优化     │  │  工具协议     │  │  自定义工具   │     │    │
│  │  └──────────────┘  └──────────────┘  └──────────────┘     │    │
│  └────────────────────────────────────────────────────────────┘    │
└──────────────────────────────────────────────────────────────────┘
```

---

## 3. 核心模块设计（按优先级）

### 3.1 P0：基础模块（必须实现）

#### 3.1.1 Controller — 状态机驱动的循环控制

Controller 是 Zhulong 的大脑，通过有限状态机（FSM）驱动整个循环。

**状态定义**：

```go
// internal/controller/state.go

package controller

type LoopState int

const (
    StateIdle          LoopState = iota // 空闲，等待目标输入
    StatePlanning                        // 规划中：调用 Planner 生成计划
    StateExecuting                       // 执行中：调用 Executor 执行当前步骤
    StateReflecting                      // 反省中：调用 Reflector 评估执行结果
    StateReplanning                      // 重规划中：根据反省结果调整计划
    StateWaitingHuman                    // 等待人工确认/输入
    StateDone                            // 正常完成
    StateError                           // 出错终止
    StateCancelled                       // 用户取消
)
```

**状态转移规则**：

```
                  ┌──────────────────────────────────────┐
                  │                                      │
                  ▼                                      │
  ┌──────┐   ┌──────────┐   ┌──────────┐   ┌──────────┐ │
  │ Idle │──→│ Planning │──→│Executing │──→│Reflecting│ │
  └──────┘   └──────────┘   └──────────┘   └─────┬────┘ │
                  ▲                               │      │
                  │          ┌──────────┐         │      │
                  └──────────│Replanning│◄────────┘      │
                             └──────────┘                │
                                   │                     │
                                   │ (目标完成)           │
                                   ▼                     │
                             ┌──────────┐                │
                             │   Done   │                │
                             └──────────┘                │
                                                         │
  任意状态 ──→ WaitingHuman ──→ (恢复) ───────────────────┘
  任意状态 ──→ Error / Cancelled
```

**转移条件表**：

| 当前状态 | 下一状态 | 触发条件 |
|---------|---------|---------|
| Idle | Planning | 收到目标（Goal）|
| Planning | Executing | 计划生成成功 |
| Planning | Error | LLM 调用失败 / 生成无效计划 |
| Executing | Reflecting | 当前步骤执行完成 |
| Executing | Error | 工具调用失败 / 超时 |
| Reflecting | Done | Reflector 判定目标完成 |
| Reflecting | Replanning | Reflector 判定需要调整计划 |
| Reflecting | Executing | Reflector 判定继续执行下一步 |
| Reflecting | Error | Reflector 判定目标无法完成 |
| Replanning | Executing | 新计划生成成功 |
| Replanning | Error | 重规划失败 |
| 任意 | WaitingHuman | 命中人工断点 / 成本超限 |
| 任意 | Cancelled | 用户主动取消 |

#### 3.1.2 Planner — 规划器

负责将用户目标分解为可执行的步骤序列。

```go
// internal/planner/planner.go

package planner

type Planner interface {
    Plan(ctx context.Context, goal string, memory MemoryReader) (*Plan, error)
    Replan(ctx context.Context, goal string, currentPlan *Plan, memory MemoryReader) (*Plan, error)
}

type Plan struct {
    ID          string
    Steps       []Step
    Rationale   string
    CreatedAt   time.Time
}

type Step struct {
    ID          string
    Description string
    Action      Action
    DependsOn   []string
    Breakpoint  bool
}
```

#### 3.1.3 Executor — 执行器

负责执行计划中的单个步骤。

```go
// internal/executor/executor.go

package executor

type Executor interface {
    Execute(ctx context.Context, step Step, memory MemoryReader) (*StepResult, error)
}

type StepResult struct {
    StepID      string
    Success     bool
    Output      string
    TokensUsed  int
    Duration    time.Duration
    Error       error
}
```

#### 3.1.4 Reflector — 反省器

负责评估执行结果，决定循环的下一步走向。

```go
// internal/reflector/reflector.go

package reflector

type Reflector interface {
    Reflect(ctx context.Context, goal string, plan *Plan, memory MemoryReader) (*Assessment, error)
}

type Assessment struct {
    Decision    Decision
    Reason      string
    Confidence  float64
    Findings    []string
    Suggestions []string
}
```

#### 3.1.5 DeepSeek Provider

深度优化 DeepSeek API 调用，保持 prefix-cache 稳定性。

```go
// internal/provider/deepseek.go

package provider

type DeepSeekProvider struct {
    apiKey  string
    baseURL string
    model   string
    client  *http.Client
}
```

#### 3.1.6 Memory — 三层记忆系统

```
┌─────────────────────────────────────────────────────────┐
│                    Memory System                         │
│                                                         │
│  ┌─────────────────────────────────────────────────┐    │
│  │  Layer 1: Working Memory (当前循环)              │    │
│  │  - 当前计划、步骤结果、评估                       │    │
│  │  - 生命周期: 单次循环 | 完整保留，不压缩          │    │
│  └─────────────────────────────────────────────────┘    │
│                         │                               │
│                     循环结束时                            │
│                         ▼                               │
│  ┌─────────────────────────────────────────────────┐    │
│  │  Layer 2: Session Memory (当前会话)              │    │
│  │  - 循环摘要、关键决策、发现                       │    │
│  │  - 生命周期: 当前会话 | 压缩存储                  │    │
│  └─────────────────────────────────────────────────┘    │
│                         │                               │
│                     会话结束时                            │
│                         ▼                               │
│  ┌─────────────────────────────────────────────────┐    │
│  │  Layer 3: Long-term Memory (持久记忆)            │    │
│  │  - 事实、模式、策略                               │    │
│  │  - 生命周期: 永久 | 持久化存储                    │    │
│  └─────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────┘
```

#### 3.1.7 Compressor — 上下文压缩

保持缓存命中率的核心模块。

```go
// internal/compressor/compressor.go

package compressor

type Compressor interface {
    // Prune prunes stale tool results (Reasonix strategy)
    Prune(messages []Message, currentLoop int) []Message

    // AssembleContext assembles the final context
    AssembleContext(systemPrompt string, skeleton string, sessionSummary string, workingMessages []Message, currentInput string, maxTokens int) []Message
}
```

> ⚠️ 实现说明：`internal/controller/loop.go` 已废弃，循环逻辑由 `pkg/agent.go` 的 `Agent.Run()` 驱动。`controller` 包仅保留 `state.go` 中的 FSM 状态枚举和 Session 管理。

#### 3.1.8 其他 P0 模块

- **Checkpoint**：检查点持久化 + 恢复
- **Budget**：成本控制 + 预算跟踪
- **Trace**：可观测性 + 日志记录
- **Human**：人机协作 + 断点管理
- **Tools**：MCP 工具层 + 内置工具

---

### 3.2 P1：核心增强模块（实现基础功能后优先实现）

#### 3.2.1 振荡/发散检测（工程控制论）

**问题**：Agent 卡在重复失败循环，或进度倒退。

**可行性**：✅ 已实现，只读取历史，不修改上下文。

```go
// internal/stability/analyzer.go

type Analyzer struct {
    history []LoopSnapshot
    window  int
}

// IsOscillating detects A → B → A → B pattern
func (a *Analyzer) IsOscillating() bool { ... }

// IsDiverging detects progress going backward
func (a *Analyzer) IsDiverging() bool { ... }
```

#### 3.2.2 信息增益工具选择（信息论）

**问题**：选择工具时，应该选能提供最多新信息的工具。

**可行性**：✅ 已实现，只影响工具选择，不修改上下文。

```go
// internal/information/gain.go

type InformationGain struct {
    toolHistory map[string][]ToolResult
}

// EstimateGain estimates information gain of calling a tool
func (ig *InformationGain) EstimateGain(toolName string) float64 { ... }
```

#### 3.2.3 停滞检测（耗散结构理论）

**问题**：Agent 卡死时需要自动发现。

**可行性**：✅ 已实现，只读取历史，不修改上下文。

```go
// internal/stagnation/detector.go

type Detector struct {
    windowSize       int
    entropyThreshold float64
    history          []StepInfo
}

// IsStagnating detects if the system is stagnating
func (d *Detector) IsStagnating() bool { ... }

// DiagnoseStagnation diagnoses the type of stagnation
func (d *Detector) DiagnoseStagnation() StagnationType { ... }
```

#### 3.2.4 探索触发（耗散结构理论）

**问题**：停滞时需要增加随机性来突破。

**可行性**：✅ 已实现，只影响 temperature，不修改上下文。

```go
// internal/exploration/trigger.go

type Trigger struct {
    baseTemperature  float64
    maxTemperature   float64
    explorationTools []string
}

// GenerateExploration generates an exploration action
func (t *Trigger) GenerateExploration(stagnationType string) ExplorationAction { ... }
```

#### 3.2.5 序参量识别（协同学）

**问题**：识别真正驱动系统的目标/策略，确保所有行动服务于它。

**可行性**：✅ 已实现，只读取目标，不修改上下文。

```go
// internal/synergetics/order_parameter.go

type OrderParameter struct {
    Goal      string
    Strategy  string
    Priority  float64
    Stability float64
}

// IdentifyOrderParameter identifies the order parameter from a session
func IdentifyOrderParameter(session *Session) *OrderParameter { ... }
```

#### 3.2.6 役使原理（协同学）

**问题**：行动必须服务于目标，拒绝无关行动。

**可行性**：✅ 已实现，只过滤行动，不修改上下文。

```go
// internal/synergetics/slaving.go

type SlavingPrinciple struct {
    orderParameter *OrderParameter
}

// EnforceSlaving enforces the slaving principle on an action
func (sp *SlavingPrinciple) EnforceSlaving(action *Action) (*EnforcedAction, error) { ... }
```

#### 3.2.7 积木块（CAS）

**问题**：成功策略应该被保存和复用。

**可行性**：✅ 已实现，存储在独立存储，不修改上下文。

```go
// internal/learning/building_block.go

type BuildingBlock struct {
    ID          string
    Name        string
    Description string
    Pattern     StrategyPattern
    Context     TaskContext
    SuccessRate float64
    UsageCount  int
}
```

#### 3.2.8 内部模型（CAS）

**问题**：Agent 应该有认知模型，不是无状态执行器。

**可行性**：✅ 已实现，存储在独立存储，不修改上下文。

```go
// internal/learning/internal_model.go

type InternalModel struct {
    WorldModel WorldModel
    TaskModel  TaskModel
    ToolModel  ToolModel
}
```

#### 3.2.9 多样性管理（CAS）

**问题**：避免过度依赖单一策略。

**可行性**：✅ 已实现，只读取统计，不修改上下文。

```go
// internal/learning/diversity.go

type DiversityManager struct {
    toolUsage     map[string]int
    strategyUsage map[string]int
    threshold     float64
}
```

#### 3.2.10 混沌边缘（CAS）

**问题**：在"利用已知"和"探索未知"之间保持平衡。

**可行性**：✅ 已实现，只影响 temperature，不修改上下文。

```go
// internal/learning/edge_of_chaos.go

type EdgeOfChaos struct {
    balance         float64
    baseTemperature float64
    maxTemperature  float64
}
```

#### 3.2.11 信息密度优化（信息论）

**问题**：上下文窗口有容量上限，必须最大化每 token 携带的有用信息。

**可行性**：✅ 已实现，只影响压缩策略，不修改 prefix。

```go
// internal/information/density.go

type DensityCalculator struct {
    tokenCounter TokenCounter
}

// Density calculates the information density of messages
func (dc *DensityCalculator) Density(messages []Message) float64 { ... }
```

---

### 3.3 P2：扩展模块（基础稳定后实现）

#### 3.3.1 渐进式披露

**问题**：大量技能/工具描述会占用大量上下文 token。

**解决方案**：渐进式披露机制，按需加载技能内容。

**可行性**：✅ 只加载元数据，完整内容按需加载，不增加 prefix 长度。

```
渐进式披露三阶段：

Discovery（发现）：
  启动时只加载 name + description
  注入 system prompt 的 Skills 清单
  节省 token，不加载完整内容

Activation（激活）：
  任务匹配时，AI 调用 view_skill() 加载完整 SKILL.md
  按需加载，不浪费上下文

Execution（执行）：
  按指南步骤执行，按需加载 scripts/、references/
```

```go
// internal/skills/manager.go

type SkillManager struct {
    skills      map[string]*Skill
    searchPaths []string
}

// Discover discovers all skills (only loads metadata)
func (sm *SkillManager) Discover() error { ... }

// ViewSkill loads the full content of a skill (Activation phase)
func (sm *SkillManager) ViewSkill(name string) (*Skill, error) { ... }
```

#### 3.3.2 审批引擎

**问题**：某些工具调用可能有风险，需要人工确认。但有些场景用户完全信任 Agent，不想被打断。

**解决方案**：审批引擎 + 三种执行模式。

**可行性**：✅ 只控制工具执行，不修改上下文。与 Human 模块整合。

**三种执行模式**：

| 模式 | 说明 | 适用场景 |
|------|------|---------|
| **ask** | 每次风险操作都询问用户 | 调试、学习、不熟悉的任务 |
| **auto** | 低风险自动执行，高风险才询问 | 日常使用、半信任场景 |
| **yolo** | 全部自动执行，不询问 | 完全信任、自动化任务 |

```go
// internal/approval/engine.go

package approval

// ExecutionMode 执行模式
type ExecutionMode int

const (
    ModeAsk  ExecutionMode = iota // 询问模式：每次风险操作都询问
    ModeAuto                       // 自动模式：低风险自动，高风险询问
    ModeYolo                       // YOLO 模式：全部自动，不询问
)

// ApprovalEngine 审批引擎
type ApprovalEngine struct {
    mode     ExecutionMode
    rules    []Rule
    callback func(toolName string, params map[string]interface{}) bool
}

// CheckPermission checks the permission for a tool
func (ae *ApprovalEngine) CheckPermission(toolName string, params map[string]interface{}) Permission {
    // YOLO 模式：全部允许
    if ae.mode == ModeYolo {
        return PermissionAllow
    }

    // 查找规则
    for _, rule := range ae.rules {
        if matchPattern(rule.ToolPattern, toolName) {
            // Ask 模式：严格按照规则
            if ae.mode == ModeAsk {
                return rule.Permission
            }

            // Auto 模式：deny 保持 deny，ask 变为 allow（除了危险操作）
            if ae.mode == ModeAuto {
                if rule.Permission == PermissionDeny {
                    return PermissionDeny
                }
                return PermissionAllow
            }
        }
    }

    // 默认：ask 模式询问，auto/yolo 模式允许
    if ae.mode == ModeAsk {
        return PermissionAsk
    }
    return PermissionAllow
}
```

**与循环的集成**：

```
Executor 收到工具调用
    │
    ▼
审批引擎检查权限
    │
    ├── allow → 正常执行，循环继续
    │
    ├── ask → Controller 进入 WaitingHuman 状态
    │         │
    │         ├── 用户批准 → 执行，循环继续
    │         └── 用户拒绝 → 步骤失败，进入 Reflector
    │
    └── deny → 步骤直接失败，进入 Reflector
               Reflector 决定：重规划 / 放弃
```

**默认规则**：

| 工具 | 权限 | 说明 |
|------|------|------|
| read_file | allow | 读取文件 |
| write_file | ask | 写入文件（auto 模式自动允许） |
| search_file | allow | 搜索文件 |
| execute_command | ask | 执行命令（auto 模式自动允许） |
| web_search | allow | 搜索网络 |
| rm -rf | deny | 危险操作（所有模式都拒绝） |
| format | deny | 危险操作（所有模式都拒绝） |

#### 3.3.3 环境感知器（一般系统论）

**问题**：文件被外部修改时，Agent 需要感知。

**可行性**：✅ 只监控外部变化，不修改上下文。

```go
// internal/environment/monitor.go

type Monitor struct {
    watchers []Watcher
    changes  chan Change
}
```

#### 3.3.4 备选路径规划（一般系统论）

**问题**：主路径失败时需要 Plan B。

**可行性**：✅ 存储在 Plan 结构中，不修改 prefix。

```go
// internal/planner/alternative.go

type AlternativePath struct {
    ID          string
    Description string
    Steps       []Step
    WhenToUse   string
    Confidence  float64
}
```

---

## 4. 缓存命中率保障机制

### ⚠️ 最高铁律：缓存命中率不可破坏

> **所有架构补充、优化、新功能，绝对不能破坏高缓存命中率这一核心优势。**
>
> 这是 Zhulong 的根基。任何模块如果会破坏 prefix 稳定性，必须重新设计或放弃。

**设计审查清单**（每个新模块必须通过）：

| 检查项 | 要求 |
|--------|------|
| 是否修改 system prompt？ | ❌ 绝不允许 |
| 是否修改 skeleton？ | ❌ 除低频刷新外不允许 |
| 是否重排历史内容？ | ❌ 只追加不重排 |
| 是否在稳定区间插入内容？ | ❌ 只能在动态区间操作 |
| 是否增加 prefix 长度？ | ⚠️ 需要评估影响 |
| 是否影响工具结果裁剪？ | ⚠️ 需要评估影响 |

### 4.1 核心原理

```
上下文布局（铁律）：
┌─────── 固定 Prefix（永不变化）────────────────────────┐
│  [system prompt]                                      │
│  [project skeleton]        ← 低频刷新                 │
└───────────────────────────────────────────────────────┘
┌─────── 稳定区间（只追加，不修改）──────────────────────┐
│  [session summary]                                    │
│  [loop 1 summary]                                     │
│  [loop 2 summary]                                     │
└───────────────────────────────────────────────────────┘
┌─────── 动态区间（裁剪活跃区）─────────────────────────┐
│  [current loop: plan + step results]                  │
│  [current turn input]                                 │
└───────────────────────────────────────────────────────┘
```

### 4.2 四条铁律

| 铁律 | 说明 | 优先级 |
|------|------|--------|
| **缓存命中率不可破坏** | 所有新模块必须通过设计审查清单 | 🔴 最高 |
| **Prefix 只追加不修改** | system prompt + skeleton 永不改写 | 🔴 最高 |
| **历史只压缩不重排** | 旧循环压缩为 summary，追加到稳定区间 | 🔴 最高 |
| **裁剪只在动态区间** | 工具结果裁剪只发生在当前循环 | 🔴 最高 |

### 4.3 模块缓存兼容性分析

| 模块 | 对缓存的影响 | 兼容性 |
|------|-------------|--------|
| **P0 模块** | | |
| Controller | 不修改上下文，只驱动循环 | ✅ 完全兼容 |
| Planner | 生成计划存动态区间 | ✅ 完全兼容 |
| Executor | 执行结果存动态区间 | ✅ 完全兼容 |
| Reflector | 评估结果存动态区间 | ✅ 完全兼容 |
| DeepSeek Provider | 深度优化 prefix-cache | ✅ 核心保障 |
| Memory | 三层记忆独立存储 | ✅ 完全兼容 |
| Compressor | 维护缓存布局 | ✅ 核心保障 |
| **P1 模块** | | |
| 振荡/发散检测 | 只读取历史，不修改上下文 | ✅ 完全兼容 |
| 信息增益工具选择 | 只影响工具选择，不修改上下文 | ✅ 完全兼容 |
| 信息密度优化 | 只影响压缩策略，不修改 prefix | ✅ 完全兼容 |
| 停滞检测 | 只读取历史，不修改上下文 | ✅ 完全兼容 |
| 探索触发 | 只影响 temperature，不修改上下文 | ✅ 完全兼容 |
| 序参量识别 | 只读取目标，不修改上下文 | ✅ 完全兼容 |
| 役使原理 | 只过滤行动，不修改上下文 | ✅ 完全兼容 |
| 积木块 | 存储在独立存储，不修改上下文 | ✅ 完全兼容 |
| 内部模型 | 存储在独立存储，不修改上下文 | ✅ 完全兼容 |
| 多样性管理 | 只读取统计，不修改上下文 | ✅ 完全兼容 |
| 混沌边缘 | 只影响 temperature，不修改上下文 | ✅ 完全兼容 |
| **P2 模块** | | |
| 渐进式披露 | 只加载元数据，完整内容按需加载 | ✅ 完全兼容 |
| 审批引擎 | 只控制工具执行，不修改上下文 | ✅ 完全兼容 |
| 环境感知器 | 只监控外部变化，不修改上下文 | ✅ 完全兼容 |
| 备选路径 | 存储在 Plan 结构中，不修改 prefix | ✅ 完全兼容 |

### 4.4 缓存命中率

```
DeepSeek prefix-cache 优化：
  - system prompt + skeleton 永不变化 → prefix 稳定
  - 旧循环工具结果被裁剪 → 中间部分体积小且稳定
  - 新内容只追加在末尾 → prefix 不会被覆盖
  - 缓存命中率: ≥ 80-90%
```

---

## 5. 配置系统

```yaml
# config/default.yaml

# DeepSeek 配置
deepseek:
  model: "deepseek-chat"
  api_key: "${DEEPSEEK_API_KEY}"
  base_url: "https://api.deepseek.com"
  temperature: 0.7
  max_tokens: 4096

# 循环控制
loop:
  max_loops: 50
  max_wall_time: "30m"
  checkpoint_every: 3
  checkpoint_dir: ".zhulong/checkpoints"

# 成本控制
budget:
  max_tokens: 500000
  max_cost: 10.0
  warn_at: 0.8

# 规划器
planner:
  max_steps: 15
  allow_replan: true
  max_replans: 5

# 执行器
executor:
  tool_timeout: "30s"

# 反省器
reflector:
  confidence_threshold: 0.7
  auto_fail_threshold: 0.3

# 压缩
compressor:
  prune:
    enabled: true
    max_age: 2
    keep_signature: true
  skeleton:
    enabled: true
    focus_mode: "auto"
    density: "adaptive"
    max_tokens: 2000

# 停滞检测
stagnation:
  window_size: 3
  entropy_threshold: 0.2

# 探索
exploration:
  base_temperature: 0.7
  max_temperature: 1.5

# 学习
learning:
  building_blocks:
    enabled: true
    min_similarity: 0.7
  internal_model:
    enabled: true
    update_frequency: "per_loop"
  diversity:
    enabled: true
    threshold: 1.0
  edge_of_chaos:
    enabled: true

# 审批
approval:
  mode: "auto"  # ask | auto | yolo
  rules:
    - tool: "read_file"
      permission: "allow"
    - tool: "write_file"
      permission: "ask"
    - tool: "execute_command"
      permission: "ask"
    - tool: "rm -rf"
      permission: "deny"
    - tool: "format"
      permission: "deny"

# 人机协作
human:
  enabled: true
  default_breakpoints:
    - type: "on_error"
    - type: "on_replan"

# 可观测性
trace:
  enabled: true
  output_dir: ".zhulong/traces"
  format: "markdown"

# 工具
tools:
  mcp_config: "mcp.json"
  builtin:
    - "read_file"
    - "write_file"
    - "search_file"
    - "execute_command"
    - "web_search"

# 项目骨架
project:
  path: "."
  ignore_patterns:
    - ".git"
    - "node_modules"
    - "vendor"
    - "dist"
    - "build"
    - ".zhulong"
```

---

## 6. 项目目录结构

```
zhulong/
├── cmd/
│   └── zhulong/
│       └── main.go                 # CLI 入口
│
├── internal/                       # 内部模块（55 个子包）
│   ├── acp/                        # 自适应上下文协议
│   ├── approval/                   # 审批引擎
│   ├── backup/                     # 备份管理
│   ├── blueprint/                  # 自动化蓝图目录
│   ├── board/                      # 无限画布
│   ├── bot/                        # Bot 渠道系统（7 个平台适配器）
│   │   ├── base.go                 # 适配器基类（公共逻辑）
│   │   ├── types.go                # 统一消息格式
│   │   ├── manager.go              # 适配器管理
│   │   ├── registry.go             # 适配器注册中心
│   │   ├── dedup.go                # 消息去重器
│   │   ├── telegram.go             # Telegram 适配器
│   │   ├── feishu.go               # 飞书适配器
│   │   ├── dingtalk.go             # 钉钉适配器
│   │   ├── discord.go              # Discord 适配器
│   │   ├── slack.go                # Slack 适配器
│   │   ├── wecom.go                # WeCom 适配器
│   │   └── github.go               # GitHub 适配器
│   ├── breaker/                    # 断路器
│   ├── budget/                     # 成本控制
│   ├── cache/                      # 缓存管理
│   ├── checkpoint/                 # 检查点持久化
│   ├── compressor/                 # 上下文压缩（Prune + Skeleton + Incremental）
│   ├── controller/                 # 状态机 + 循环控制
│   ├── cronx/                      # 定时任务调度
│   ├── dashboard/                  # Web Dashboard
│   ├── environment/                # 环境感知器
│   ├── evolution/                  # 自进化系统（审查 + 建议 + 守卫者）
│   ├── executor/                   # 执行器
│   ├── exploration/                # 探索触发器
│   ├── gateway/                    # 消息网关（多平台适配器接口）
│   ├── health/                     # 健康检查
│   ├── hook/                       # 钩子引擎（6 阶段执行点）
│   ├── hub/                        # 消息中心
│   ├── human/                      # 人机协作
│   ├── i18n/                       # 多语言（17 语言支持）
│   ├── information/                # 信息论（信息增益 + 信息密度）
│   ├── learning/                   # 自适应学习（积木块 + 多样性 + 混沌边缘）
│   ├── loop/                       # 自治循环引擎
│   ├── mcp/                        # MCP 客户端（stdio + HTTP + SSE）
│   ├── memento/                    # 记忆管理（事实/偏好/快照）
│   ├── memory/                     # 记忆系统（FileStore）
│   ├── models/                     # 多模型池
│   ├── observe/                    # 可观测性增强
│   ├── planner/                    # 规划器（LLM + 备选路径）
│   ├── plugins/                    # 插件管理
│   ├── profile/                    # 用户画像
│   ├── provider/                   # LLM Provider（DeepSeek）
│   ├── qa/                         # 质量保证
│   ├── quality/                    # 质量扫描
│   ├── reflector/                  # 反省器
│   ├── review/                     # 审查记录
│   ├── scheduler/                  # 任务调度
│   ├── security/                   # 安全引擎（输入/路径/命令检查）
│   ├── skills/                     # 技能管道
│   ├── skillset/                   # 预置技能注册表
│   ├── stability/                  # 稳定性分析（振荡/发散检测）
│   ├── stagnation/                 # 停滞检测
│   ├── state/                      # 状态管理
│   ├── synergetics/                # 协同学（序参量 + 役使原理）
│   ├── terminal/                   # 终端后端（Local/Docker/SSH）
│   ├── tools/                      # 工具层（read/write/search/execute）
│   ├── trace/                      # 可观测性（JSONL 日志）
│   ├── upgrade/                    # 升级检查
│   ├── voice/                      # 语音引擎
│   └── workflow/                   # 工作流模式（Hotfix/Tweak/Full）
│
├── pkg/                            # 对外公共 API（CLI 和桌面端共享）
│   ├── agent.go                    # Agent 构造函数 + Run()
│   ├── options.go                  # 函数式选项
│   └── types.go                    # 公共类型导出
│
├── desktop/                        # Windows 桌面端（Wails + React）
│   ├── main.go
│   ├── app.go                      # Wails 绑定（30+ 个方法）
│   ├── agent.go                    # Agent 执行逻辑
│   ├── agent_test.go               # 单元测试
│   └── frontend/                   # React 前端
│       ├── src/
│       │   ├── App.tsx
│       │   ├── components/         # UI 组件（Settings/Sidebar/TopBar/RightPanel/Transcript/Composer/StatusBar/Canvas/ApprovalModal/HistoryPage）
│       │   ├── stores/             # Zustand 状态管理（canvasStore）
│       │   └── main.tsx
│       ├── package.json
│       └── vite.config.ts
│
├── config/                         # 配置文件
│   └── default.yaml                # 默认配置（完整字段）
│
├── docs/
│   ├── design.md                   # 详细设计文档（中文）
│   └── design.en.md                # 详细设计文档（英文）
│
├── examples/                       # 示例
│
├── go.mod
├── Makefile
└── README.md
```

---

## 7. 开发路线图

### Phase 1: P0 基础模块（CLI + 桌面端同步）✅ 已完成

**CLI 核心模块**：
- [x] 项目初始化（go mod, 目录结构）
- [x] Controller 状态机（核心循环）
- [x] Planner（LLM 规划，JSON 解析）
- [x] Executor（单工具调用）
- [x] Reflector（基础反省）
- [x] DeepSeek Provider
- [x] Memory 三层系统
- [x] Compressor（上下文压缩）
- [x] Checkpoint（检查点）
- [x] Budget（成本控制）
- [x] Trace（可观测性）
- [x] Human（人机协作）
- [x] Tools（MCP 工具层）

**桌面端**：
- [x] Wails + React 项目初始化
- [x] 主窗口框架（Apple Design 风格）
- [x] 目标输入界面
- [x] 实时状态展示

**测试**：
- [x] 单元测试覆盖 8 个核心模块

### Phase 2: P1 核心增强 ✅ 已完成

- [x] 振荡/发散检测
- [x] 信息增益工具选择
- [x] 信息密度优化
- [x] 停滞检测
- [x] 探索触发
- [x] 序参量识别
- [x] 役使原理
- [x] 积木块
- [x] 内部模型
- [x] 多样性管理
- [x] 混沌边缘

### Phase 3: P2 扩展模块 ✅ 已完成

- [x] 渐进式披露
- [x] 审批引擎
- [x] 环境感知器
- [x] 备选路径规划

### Phase 4: 打磨 + 文档 ✅ 已完成

- [x] 缓存命中率基准测试
- [x] 使用示例
- [x] API 文档
- [x] README + 贡献指南
- [ ] 桌面端打包（Windows 安装包 — 需要 NSIS/installer 工具链，非代码功能）

### Phase 5: Bot 渠道 + 设置系统 ✅ 已完成

- [x] Bot 渠道框架（参考 Hermes 适配器模式）
- [x] Telegram 适配器（Bot API 轮询）
- [x] 飞书/Lark 适配器（App Token + Webhook）
- [x] 钉钉适配器（Access Token + Webhook）
- [x] Discord 适配器（Bot API 轮询）
- [x] Slack 适配器（Bot Token + Webhook）
- [x] WeCom 适配器（Access Token）
- [x] GitHub 适配器（Webhook）
- [x] 统一消息格式（MessageEvent）
- [x] 适配器注册中心（Registry）
- [x] 消息去重器（Deduplicator）
- [x] 设置页面（11 个 Tab）
- [x] 记忆管理（前端 + 后端 API）
- [x] MCP 客户端连接（stdio + HTTP）
- [x] Config 加载（30+ 配置项）
- [x] Bot Webhook HTTP 路由
- [x] Config API（SetConfigField/GetConfigField）
- [x] 单元测试（6 个测试用例）

---

## 9. Bot 渠道架构设计

### 9.1 设计目标

为烛龙 Agent 提供多平台消息集成能力，支持：
- **多平台接入**：Telegram、飞书、钉钉、Discord、Slack、WeCom、GitHub
- **统一消息格式**：所有平台消息转换为统一的 `MessageEvent`
- **适配器模式**：每个平台实现独立的 `Adapter` 接口
- **动态注册**：支持运行时添加新平台

### 9.2 架构图

```
┌──────────────────────────────────────────────────────────────────┐
│                      Bot 渠道系统                                 │
│                                                                  │
│  ┌────────────────────────────────────────────────────────────┐  │
│  │                    Manager                                  │  │
│  │  - 适配器管理（Add/Remove/Connect/Disconnect）              │  │
│  │  - 消息路由（handleBotMessage → Session）                   │  │
│  │  - 消息去重（Deduplicator）                                 │  │
│  └────────────────────────────────────────────────────────────┘  │
│                            │                                      │
│  ┌─────────────────────────┼──────────────────────────────────┐  │
│  │                         ▼                                    │  │
│  │  ┌──────────────────────────────────────────────────────┐  │  │
│  │  │              Platform Adapters                        │  │  │
│  │  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐│  │  │
│  │  │  │ Telegram │ │  飞书    │ │  钉钉    │ │ Discord  ││  │  │
│  │  │  │ (API)    │ │ (Token)  │ │ (Token)  │ │ (API)    ││  │  │
│  │  │  └──────────┘ └──────────┘ └──────────┘ └──────────┘│  │  │
│  │  │  ┌──────────┐ ┌──────────┐ ┌──────────┐             │  │  │
│  │  │  │  Slack   │ │  WeCom   │ │  GitHub  │             │  │  │
│  │  │  │ (Token)  │ │ (Token)  │ │ (Webhook)│             │  │  │
│  │  │  └──────────┘ └──────────┘ └──────────┘             │  │  │
│  │  └──────────────────────────────────────────────────────┘  │  │
│  └────────────────────────────────────────────────────────────┘  │
│                            │                                      │
│  ┌─────────────────────────┼──────────────────────────────────┐  │
│  │                         ▼                                    │  │
│  │  ┌──────────────────────────────────────────────────────┐  │  │
│  │  │              Message Format                          │  │  │
│  │  │  MessageEvent {                                      │  │  │
│  │  │    Text: string,                                     │  │  │
│  │  │    MessageType: text|image|video|...                 │  │  │
│  │  │    Source: SessionSource {                           │  │  │
│  │  │      Platform, ChatID, UserID, ...                   │  │  │
│  │  │    },                                                │  │  │
│  │  │    RichText: *RichTextContent,                       │  │  │
│  │  │    InteractiveCard: *InteractiveCard,                 │  │  │
│  │  │  }                                                   │  │  │
│  │  └──────────────────────────────────────────────────────┘  │  │
│  └────────────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────────┘
```

### 9.3 核心接口

```go
// Adapter 是所有平台适配器必须实现的接口
type Adapter interface {
    Platform() Platform
    Name() string
    Connect() error
    Disconnect() error
    IsConnected() bool
    Send(chatID string, text string) (*SendResult, error)
    SendImage(chatID string, imageURL string, caption string) (*SendResult, error)
    SendRichText(chatID string, richText *RichTextContent) (*SendResult, error)
    SendInteractiveCard(chatID string, card *InteractiveCard) (*SendResult, error)
    GetChatInfo(chatID string) (*ChatInfo, error)
    SetMessageHandler(handler MessageHandler)
}
```

### 9.4 平台实现状态

| 平台 | 传输方式 | 消息接收 | 消息发送 | 富文本 | 交互卡片 |
|------|---------|---------|---------|--------|----------|
| Telegram | Bot API 轮询 | ✅ | ✅ | ✅ | ✅ |
| 飞书/Lark | App Token + Webhook | ⚠️ | ✅ | ✅ | ✅ |
| 钉钉 | Access Token + Webhook | ⚠️ | ✅ | ✅ | ✅ |
| Discord | Bot API 轮询 | ✅ | ✅ | ✅ | ✅ |
| Slack | Bot Token + Webhook | ⚠️ | ✅ | ✅ | ✅ |
| WeCom | Access Token | ⚠️ | ✅ | ✅ | ✅ |
| GitHub | Webhook | ⚠️ | ✅ | ✅ | ✅ |

---

## 10. MCP 客户端架构设计

### 10.1 设计目标

为烛龙 Agent 提供 MCP（Model Context Protocol）客户端能力：
- **双传输协议**：stdio（本地进程）+ HTTP/SSE（远程服务）
- **JSON-RPC 通信**：标准 MCP 协议实现
- **工具发现**：自动发现服务器提供的工具
- **工具调用**：远程调用 MCP 服务器的工具

### 10.2 架构图

```
┌──────────────────────────────────────────────────────────────────┐
│                      MCP 客户端                                   │
│                                                                  │
│  ┌────────────────────────────────────────────────────────────┐  │
│  │                    Client                                   │  │
│  │  - 连接管理（Connect/Disconnect）                           │  │
│  │  - 工具发现（DiscoverTools）                                │  │
│  │  - 工具调用（CallTool）                                     │  │
│  │  - 通知发送（SendNotification）                             │  │
│  └────────────────────────────────────────────────────────────┘  │
│                            │                                      │
│  ┌─────────────────────────┼──────────────────────────────────┐  │
│  │                         ▼                                    │  │
│  │  ┌──────────────────────────────────────────────────────┐  │  │
│  │  │              Transport Layer                         │  │  │
│  │  │  ┌────────────────────┐  ┌───────────────────────┐  │  │  │
│  │  │  │    stdio           │  │    HTTP/SSE            │  │  │  │
│  │  │  │ (JSON-RPC over     │  │ (JSON-RPC over        │  │  │  │
│  │  │  │  stdin/stdout)     │  │  HTTP POST)            │  │  │  │
│  │  │  └────────────────────┘  └───────────────────────┘  │  │  │
│  │  └──────────────────────────────────────────────────────┘  │  │
│  └────────────────────────────────────────────────────────────┘  │
│                            │                                      │
│  ┌─────────────────────────┼──────────────────────────────────┐  │
│  │                         ▼                                    │  │
│  │  ┌──────────────────────────────────────────────────────┐  │  │
│  │  │              Protocol Layer                          │  │  │
│  │  │  - initialize → capabilities                        │  │  │
│  │  │  - tools/list → tool definitions                    │  │  │
│  │  │  - tools/call → tool results                        │  │  │
│  │  │  - notifications/initialized                        │  │  │
│  │  └──────────────────────────────────────────────────────┘  │  │
│  └────────────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────────┘
```

### 10.3 核心接口

```go
// Client 是 MCP 客户端
type Client struct {
    config     ServerConfig
    connected  bool
    tools      []Tool
    resources  []Resource
}

// 连接管理
func (c *Client) Connect() error
func (c *Client) Close() error

// 工具管理
func (c *Client) ListTools() []Tool
func (c *Client) GetTool(name string) *Tool
func (c *Client) CallTool(ctx context.Context, name string, args map[string]interface{}) (*ToolResult, error)
```

### 10.4 传输协议支持

| 协议 | 状态 | 说明 |
|------|------|------|
| **stdio** | ✅ 完整实现 | JSON-RPC over stdin/stdout |
| **HTTP** | ✅ 完整实现 | JSON-RPC over HTTP POST |
| **SSE** | ✅ 完整实现 | Server-Sent Events + 自动重连 |

---

## 11. Config 系统架构设计

### 11.1 设计目标

为烛龙 Agent 提供完整的配置管理系统：
- **统一配置**：所有模块配置集中管理
- **动态更新**：支持运行时修改配置
- **持久化**：配置保存到 JSON 文件
- **默认值**：从 config/default.yaml 同步默认值

### 11.2 架构图

```
┌──────────────────────────────────────────────────────────────────┐
│                      Config 系统                                  │
│                                                                  │
│  ┌────────────────────────────────────────────────────────────┐  │
│  │                    AppConfig                                │  │
│  │  - 30+ 配置字段                                             │  │
│  │  - JSON 序列化/反序列化                                      │  │
│  │  - 默认值初始化                                              │  │
│  └────────────────────────────────────────────────────────────┘  │
│                            │                                      │
│  ┌─────────────────────────┼──────────────────────────────────┐  │
│  │                         ▼                                    │  │
│  │  ┌──────────────────────────────────────────────────────┐  │  │
│  │  │              API Layer                               │  │  │
│  │  │  - SetConfigField(field, value)                      │  │  │
│  │  │  - GetConfigField(field)                             │  │  │
│  │  │  - GetConfig()                                       │  │  │
│  │  │  - saveConfig()                                      │  │  │
│  │  │  - loadConfig()                                      │  │  │
│  │  └──────────────────────────────────────────────────────┘  │  │
│  └────────────────────────────────────────────────────────────┘  │
│                            │                                      │
│  ┌─────────────────────────┼──────────────────────────────────┐  │
│  │                         ▼                                    │  │
│  │  ┌──────────────────────────────────────────────────────┐  │  │
│  │  │              Storage Layer                           │  │  │
│  │  │  - ~/.zhulong/config.json                            │  │  │
│  │  │  - config/default.yaml (默认值)                       │  │  │
│  │  └──────────────────────────────────────────────────────┘  │  │
│  └────────────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────────┘
```

### 11.3 配置字段

| 分类 | 字段 | 类型 | 默认值 |
|------|------|------|--------|
| **DeepSeek** | deepseekModel | string | deepseek-v4-flash |
| | deepseekBaseUrl | string | https://api.deepseek.com |
| | temperature | float64 | 0.7 |
| | maxTokens | int | 4096 |
| **循环控制** | maxLoops | int | 50 |
| | maxWallTime | string | 30m |
| | checkpointEvery | int | 3 |
| **成本控制** | budgetMaxTokens | int | 500000 |
| | budgetMaxCost | float64 | 10.0 |
| | budgetWarnAt | float64 | 0.8 |
| **规划器** | plannerMaxSteps | int | 15 |
| | plannerAllowReplan | bool | true |
| | plannerMaxReplans | int | 5 |
| **压缩** | compressorPruneEnabled | bool | true |
| | compressorPruneMaxAge | int | 2 |
| | compressorMaxTokens | int | 2000 |
| **停滞检测** | stagnationWindowSize | int | 3 |
| | stagnationEntropyThreshold | float64 | 0.2 |
| **探索** | explorationBaseTemp | float64 | 0.7 |
| | explorationMaxTemp | float64 | 1.5 |
| **多样性** | diversityThreshold | float64 | 1.0 |
| **可观测性** | traceEnabled | bool | true |
| | traceFormat | string | jsonl |
| | traceVerbose | bool | false |
| **Shell** | shell | string | auto |
| **沙箱** | sandboxBash | string | enforce |
| | sandboxNetwork | bool | true |
| | allowWrite | []string | [] |
| **代理** | proxyMode | string | auto |
| | proxyUrl | string | "" |
| | noProxy | string | "" |
| **权限** | permMode | string | ask |
| | permAllow | []string | [] |
| | permAsk | []string | [] |
| | permDeny | []string | [] |

---

## 12. 桌面端架构设计

### 12.1 设计目标

为烛龙 Agent 提供完整的桌面端应用：
- **跨平台**：Windows（Wails + React）
- **实时同步**：前端与后端实时通信
- **完整功能**：所有 CLI 功能在桌面端可用
- **用户体验**：Apple Design 风格

### 12.2 架构图

```
┌──────────────────────────────────────────────────────────────────┐
│                      桌面端架构                                    │
│                                                                  │
│  ┌────────────────────────────────────────────────────────────┐  │
│  │                    Frontend (React)                         │  │
│  │  - 11 个设置 Tab                                            │  │
│  │  - 无限画布                                                  │  │
│  │  - 实时状态展示                                              │  │
│  │  - 消息面板                                                  │  │
│  └────────────────────────────────────────────────────────────┘  │
│                            │                                      │
│  ┌─────────────────────────┼──────────────────────────────────┐  │
│  │                         ▼                                    │  │
│  │  ┌──────────────────────────────────────────────────────┐  │  │
│  │  │              Wails Bindings                          │  │  │
│  │  │  - 45+ 个绑定方法                                     │  │  │
│  │  │  - 实时事件推送                                       │  │  │
│  │  │  - 错误处理                                           │  │  │
│  │  └──────────────────────────────────────────────────────┘  │  │
│  └────────────────────────────────────────────────────────────┘  │
│                            │                                      │
│  ┌─────────────────────────┼──────────────────────────────────┐  │
│  │                         ▼                                    │  │
│  │  ┌──────────────────────────────────────────────────────┐  │  │
│  │  │              Backend (Go)                             │  │  │
│  │  │  - Agent 核心逻辑                                     │  │  │
│  │  │  - Bot 渠道系统                                       │  │  │
│  │  │  - MCP 客户端                                         │  │  │
│  │  │  - Config 系统                                        │  │  │
│  │  │  - 55 个模块实例                                     │  │  │
│  │  └──────────────────────────────────────────────────────┘  │  │
│  └────────────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────────┘
```

### 12.3 Wails 绑定方法

| 分类 | 方法 | 说明 |
|------|------|------|
| **会话管理** | NewSession | 创建新会话 |
| | GetSession | 获取会话 |
| | DeleteSession | 删除会话 |
| | RenameSession | 重命名会话 |
| | SetActiveSession | 设置活跃会话 |
| **Agent 执行** | SendMessage | 发送消息触发 Agent |
| | Stop | 停止当前运行 |
| | Reset | 重置会话 |
| | RespondApproval | 响应审批请求 |
| **模型设置** | SetModel | 设置模型 |
| | SetExecutionMode | 设置执行模式 |
| | SetTemperature | 设置温度 |
| | SetActiveAgent | 设置活跃 Agent |
| **记忆管理** | ListMemory | 列出记忆 |
| | Remember | 记住事实 |
| | Forget | 忘记事实 |
| | RestoreMemory | 恢复记忆 |
| | DeleteMemory | 删除记忆 |
| | SaveDoc | 保存指令文件 |
| | DeleteDoc | 删除指令文件 |
| **MCP 连接** | MCPConnectServer | 连接 MCP 服务器 |
| | MCPDisconnectServer | 断开 MCP 服务器 |
| | MCPListTools | 列出工具 |
| | MCPCallTool | 调用工具 |
| | MCPIsConnected | 检查连接状态 |
| **仪表盘** | StartDashboard | 启动仪表盘 |
| | StopDashboard | 停止仪表盘 |
| | SetDashboardPort | 设置仪表盘端口 |
| **备份** | TriggerBackup | 触发备份 |
| | SetBackupMode | 设置备份模式 |
| **Bot 渠道** | BotConnect | 连接 Bot |
| | BotDisconnect | 断开 Bot |
| | BotSend | 发送消息 |
| | BotIsConnected | 检查连接状态 |
| | BotListAdapters | 列出适配器 |
| | BotRemoveAdapter | 移除适配器 |
| **Config** | SetConfigField | 设置配置字段 |
| | GetConfigField | 获取配置字段 |
| | GetConfig | 获取完整配置 |
| **文件操作** | OpenInExplorer | 在资源管理器中打开 |
| | GetGlobalPath | 获取全局路径 |
| | GetProjectPath | 获取项目路径 |

### 8.1 设计目标

为烛龙 Agent 的 Plan → Execute → Reflect 循环提供可视化展示，让用户能够直观地看到：
- 任务规划的步骤和依赖关系
- 每个步骤的执行状态和结果
- 工具调用的详细信息
- 反省和重规划的过程

**内容创作场景**：
- 代码开发：需求分析→设计→编码→测试
- 数据分析：数据收集→清洗→分析→可视化
- 内容创作：素材收集→大纲→撰写→优化
- 任务规划：目标→子任务→执行→验证
- 学习研究：资料收集→阅读→总结→应用

### 8.2 技术选型

| 组件 | 技术 | 说明 |
|------|------|------|
| **画布引擎** | React Flow | 成熟的无限画布库，MIT协议，可商用 |
| **状态管理** | Zustand | 轻量级状态管理，与React Flow兼容 |
| **UI组件** | 自研 | 保持Apple Design风格一致性 |



### 8.3 节点类型设计

#### 8.3.1 已实现的节点类型

**Agent循环节点**：

| 节点类型 | 组件文件 | 说明 |
|---------|----------|------|
| Plan | `nodes/PlanNode.tsx` | 计划节点，显示目标和步骤数 |
| Step | `nodes/StepNode.tsx` | 步骤节点，显示工具和状态 |

**多媒体节点**：

| 节点类型 | 组件文件 | 说明 |
|---------|----------|------|
| Text | `nodes/TextNode.tsx` | 文本节点，显示内容 |
| Image | `nodes/ImageNode.tsx` | 图片节点，显示图片和提示词 |
| Video | `nodes/VideoNode.tsx` | 视频节点，显示视频和时长 |
| Config | `nodes/ConfigNode.tsx` | 配置节点，显示生成参数 |

#### 8.3.2 节点数据结构

```typescript
// 基础节点数据
interface BaseNodeData {
  id: string;
  type: CanvasNodeType;
  title: string;
  status: NodeStatus;
  metadata?: Record<string, any>;
}

// Plan节点数据
interface PlanNodeData extends BaseNodeData {
  type: CanvasNodeType.Plan;
  goal: string;
  steps: number;
}

// Step节点数据
interface StepNodeData extends BaseNodeData {
  type: CanvasNodeType.Step;
  stepId: string;
  description: string;
  tool: string;
  result?: string;
  tokensUsed?: number;
  duration?: string;
}

// Text节点数据
interface TextNodeData extends BaseNodeData {
  type: CanvasNodeType.Text;
  content: string;
  prompt?: string;
  wordCount?: number;
}

// Image节点数据
interface ImageNodeData extends BaseNodeData {
  type: CanvasNodeType.Image;
  imageUrl?: string;
  prompt?: string;
  width?: number;
  height?: number;
  model?: string;
  aspect?: string;
  referenceIds?: string[];
}

// Video节点数据
interface VideoNodeData extends BaseNodeData {
  type: CanvasNodeType.Video;
  videoUrl?: string;
  prompt?: string;
  duration?: number;
  orientation?: 'landscape' | 'portrait';
  firstFrame?: string;
  lastFrame?: string;
  referenceIds?: string[];
}

// Config节点数据
interface ConfigNodeData extends BaseNodeData {
  type: CanvasNodeType.Config;
  generationMode: 'text' | 'image' | 'video' | 'audio';
  model: string;
  params: Record<string, any>;
  prompt?: string;
  referenceIds?: string[];
}
```

### 8.4 布局算法

#### 8.4.1 自动布局

```
┌─────────────────────────────────────────────────────────┐
│                      Plan Node                          │
│                    (用户目标)                            │
└─────────────────────────────────────────────────────────┘
                          │
        ┌─────────────────┼─────────────────┐
        ▼                 ▼                 ▼
┌───────────────┐ ┌───────────────┐ ┌───────────────┐
│   Step 1      │ │   Step 2      │ │   Step 3      │
│   (pending)   │ │   (running)   │ │   (pending)   │
└───────────────┘ └───────────────┘ └───────────────┘
        │                 │                 │
        ▼                 ▼                 ▼
┌───────────────┐ ┌───────────────┐ ┌───────────────┐
│   Tool 1.1    │ │   Tool 2.1    │ │   Tool 3.1    │
│   (success)   │ │   (calling)   │ │   (pending)   │
└───────────────┘ └───────────────┘ └───────────────┘
                          │
                          ▼
                ┌───────────────┐
                │   Result      │
                │   (最终结果)  │
                └───────────────┘
```

#### 8.4.2 布局参数

```typescript
const layoutConfig = {
  nodeWidth: 200,           // 节点宽度
  nodeHeight: 100,          // 节点高度
  horizontalSpacing: 50,    // 水平间距
  verticalSpacing: 80,      // 垂直间距
  padding: 50,              // 画布内边距
};
```

### 8.5 状态同步机制

#### 8.5.1 后端事件

```go
// internal/trace/events.go
type CanvasEvent struct {
    Type      string      `json:"type"`      // node_added, node_updated, edge_added
    NodeID    string      `json:"nodeId"`
    Data      interface{} `json:"data"`
    Timestamp time.Time   `json:"timestamp"`
}
```

#### 8.5.2 前端订阅

```typescript
// 使用Wails事件系统
wails.EventsOn('canvas:event', (event: CanvasEvent) => {
  switch (event.type) {
    case 'node_added':
      addNode(event.data);
      break;
    case 'node_updated':
      updateNode(event.nodeId, event.data);
      break;
    case 'edge_added':
      addEdge(event.data);
      break;
  }
});
```

### 8.6 交互设计

#### 8.6.1 基础交互

| 交互 | 说明 |
|------|------|
| **拖拽** | 拖拽画布移动视图 |
| **缩放** | 滚轮缩放画布 |
| **选择** | 点击选中节点 |
| **多选** | 框选多个节点 |

#### 8.6.2 节点交互

| 交互 | 说明 |
|------|------|
| **点击展开** | 点击节点展开显示详细信息 |
| **双击编辑** | 双击节点编辑描述（仅pending状态） |
| **右键菜单** | 右键节点显示操作菜单 |
| **拖拽调整** | 拖拽pending状态节点调整顺序 |

#### 8.6.3 视图控制

| 控制 | 说明 |
|------|------|
| **适应画布** | 自动缩放以显示所有节点 |
| **聚焦节点** | 缩放到选中节点 |
| **重置视图** | 恢复到初始视图 |
| **全屏模式** | 进入全屏画布模式 |

### 8.7 组件架构（实际实现）

```
desktop/frontend/src/
├── components/
│   ├── Canvas/                        # 画布组件目录
│   │   ├── Canvas.tsx                 # 主画布组件
│   │   ├── CanvasAssistant.tsx        # 画布助手
│   │   ├── CanvasToolbar.tsx          # 画布工具栏
│   │   ├── NodePanel.tsx             # 节点面板
│   │   ├── AssetPanel.tsx            # 资产管理面板
│   │   ├── VersionPanel.tsx          # 版本管理面板
│   │   ├── ExportPanel.tsx           # 导出面板
│   │   ├── CollaborationPanel.tsx    # 协作面板
│   │   ├── PerformancePanel.tsx      # 性能面板
│   │   ├── OfflinePanel.tsx          # 离线面板
│   │   ├── AIPanel.tsx              # AI增强面板
│   │   ├── PluginPanel.tsx          # 插件面板
│   │   └── nodes/                   # 自定义节点
│   │       ├── PlanNode.tsx         # Plan节点
│   │       ├── StepNode.tsx         # Step节点
│   │       ├── TextNode.tsx         # Text节点
│   │       ├── ImageNode.tsx        # Image节点
│   │       ├── VideoNode.tsx        # Video节点
│   │       └── ConfigNode.tsx       # Config节点
│   ├── Sidebar/                     # 侧边栏
│   ├── Transcript/                  # 对话区域
│   └── RightPanel/                  # 右侧面板
├── stores/
│   └── canvasStore.ts               # 画布状态管理（Zustand）
├── types/
│   └── canvas.ts                    # 画布类型定义
└── styles/
    ├── canvas.css                   # 画布样式
    └── global.css                   # 全局样式
```

### 8.8 实现阶段（已完成 ✅）

#### Phase 1: 基础画布 ✅

- [x] 集成React Flow
- [x] 创建基础画布组件（Canvas.tsx）
- [x] 定义节点类型（6种自定义节点）
- [x] 基础布局和样式
- [x] 状态管理（Zustand）
- [x] 撤销/重做功能
- [x] Chat/Canvas视图切换

#### Phase 2: 画布助手与资产管理 ✅

- [x] 画布助手（CanvasAssistant.tsx）
  - 上下文对话
  - 选中节点引用
  - 多轮对话支持
- [x] 资产管理面板（AssetPanel.tsx）
  - 资产列表、类型过滤、搜索
  - 资产创建和删除
- [x] 节点面板（NodePanel.tsx）
  - 节点类型列表
  - 拖拽创建节点
- [x] 画布工具栏（CanvasToolbar.tsx）
  - 面板切换按钮
  - 撤销/重做按钮
  - 缩放控制按钮

#### Phase 3: 版本管理与导出 ✅

- [x] 版本管理面板（VersionPanel.tsx）
  - 版本创建、恢复、删除、列表
- [x] 导出与分享面板（ExportPanel.tsx）
  - JSON/PNG/SVG导出
  - JSON导入
  - 分享链接生成
- [x] 协作功能面板（CollaborationPanel.tsx）
  - 协作者邀请、角色管理
  - 在线状态显示

#### Phase 4: 性能优化与AI增强 ✅

- [x] 性能优化面板（PerformancePanel.tsx）
  - 性能等级、统计、建议、监控
- [x] 离线支持面板（OfflinePanel.tsx）
  - 网络状态、本地存储、同步控制
- [x] AI增强面板（AIPanel.tsx）
  - 自动布局、节点建议、工作流优化
- [x] 插件面板（PluginPanel.tsx）
  - 插件列表、启用/禁用、安装/卸载

### 8.9 与现有UI的集成

```
┌─────────────────────────────────────────────────────────┐
│                    烛龙桌面端                            │
│                                                         │
│  ┌─────────┐  ┌─────────────────────────┐  ┌─────────┐│
│  │         │  │      视图切换            │  │         ││
│  │ Sidebar │  │  ┌─────┐ ┌─────┐        │  │  Right  ││
│  │         │  │  │Chat │ │Canvas│        │  │  Panel  ││
│  │         │  │  └─────┘ └─────┘        │  │         ││
│  │         │  │                         │  │         ││
│  │         │  │  ┌─────────────────┐    │  │         ││
│  │         │  │  │                 │    │  │         ││
│  │         │  │  │   Canvas View   │    │  │         ││
│  │         │  │  │                 │    │  │         ││
│  │         │  │  │                 │    │  │         ││
│  │         │  │  └─────────────────┘    │  │         ││
│  │         │  │                         │  │         ││
│  └─────────┘  └─────────────────────────┘  └─────────┘│
│                                                         │
└─────────────────────────────────────────────────────────┘
```

### 8.10 MCP协议集成设计

#### 8.10.1 设计理念

烛龙画布将通过MCP协议与Agent内核通信，实现：

1. **解耦设计** - 画布作为独立的MCP客户端，Agent作为MCP服务器
2. **标准化通信** - 使用MCP协议规范，便于扩展和维护
3. **双向交互** - 画布可以调用Agent，Agent也可以推送状态到画布

#### 8.10.2 架构设计

```
┌─────────────────────────────────────────────────────────┐
│                    画布前端 (React Flow)                 │
│  ┌─────────────────────────────────────────────────┐   │
│  │  MCP Client                                      │   │
│  │  - 发送画布操作指令                              │   │
│  │  - 接收Agent状态推送                             │   │
│  │  - 管理节点和连线状态                            │   │
│  └─────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────┘
                          │
                          │ MCP协议
                          ▼
┌─────────────────────────────────────────────────────────┐
│                    Agent内核 (Go)                        │
│  ┌─────────────────────────────────────────────────┐   │
│  │  MCP Server                                      │   │
│  │  - 暴露Agent状态查询接口                         │   │
│  │  - 接收画布操作指令                              │   │
│  │  - 推送实时状态更新                              │   │
│  └─────────────────────────────────────────────────┘   │
│  ┌─────────────────────────────────────────────────┐   │
│  │  Agent核心                                       │   │
│  │  - Planner / Executor / Reflector                │   │
│  │  - Memory / Checkpoint / Trace                   │   │
│  └─────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────┘
```

#### 8.10.3 MCP工具定义

```go
// internal/mcp/canvas_tools.go

// 画布相关的MCP工具
var CanvasTools = []Tool{
    {
        Name:        "canvas_get_plan",
        Description: "获取当前任务的计划结构",
        InputSchema: map[string]interface{}{},
    },
    {
        Name:        "canvas_get_step_status",
        Description: "获取指定步骤的执行状态",
        InputSchema: map[string]interface{}{
            "step_id": map[string]string{"type": "string"},
        },
    },
    {
        Name:        "canvas_update_node",
        Description: "更新画布节点状态",
        InputSchema: map[string]interface{}{
            "node_id": map[string]string{"type": "string"},
            "status":  map[string]string{"type": "string"},
            "data":    map[string]string{"type": "object"},
        },
    },
    {
        Name:        "canvas_add_node",
        Description: "添加新的画布节点",
        InputSchema: map[string]interface{}{
            "type": map[string]string{"type": "string"},
            "data": map[string]string{"type": "object"},
            "position": map[string]string{"type": "object"},
        },
    },
    {
        Name:        "canvas_add_edge",
        Description: "添加节点连线",
        InputSchema: map[string]interface{}{
            "source": map[string]string{"type": "string"},
            "target": map[string]string{"type": "string"},
        },
    },
}
```

#### 8.10.4 状态推送机制

```go
// internal/mcp/canvas_events.go

type CanvasEvent struct {
    Type      string      `json:"type"`      // node_added, node_updated, edge_added, plan_updated
    NodeID    string      `json:"nodeId,omitempty"`
    Data      interface{} `json:"data"`
    Timestamp time.Time   `json:"timestamp"`
}

// 事件推送接口
type CanvasEventBroadcaster interface {
    Broadcast(event CanvasEvent)
    Subscribe(handler func(CanvasEvent))
}
```

#### 8.10.5 前端MCP客户端

```typescript
// desktop/frontend/src/services/canvasMcp.ts

interface McpTool {
  name: string;
  description: string;
  inputSchema: Record<string, any>;
}

class CanvasMcpClient {
  private tools: Map<string, McpTool> = new Map();
  private eventHandlers: Map<string, Function[]> = new Map();

  // 连接到Agent MCP服务器
  async connect(agentId: string): Promise<void> {
    // 通过Wails调用Go后端
    const tools = await window.go.main.App.GetCanvasTools(agentId);
    tools.forEach(tool => this.tools.set(tool.name, tool));
  }

  // 调用MCP工具
  async callTool(name: string, args: Record<string, any>): Promise<any> {
    return await window.go.main.App.CallCanvasTool(name, args);
  }

  // 订阅事件
  on(event: string, handler: Function): void {
    if (!this.eventHandlers.has(event)) {
      this.eventHandlers.set(event, []);
    }
    this.eventHandlers.get(event)!.push(handler);
  }

  // 处理来自Agent的事件
  handleEvent(event: CanvasEvent): void {
    const handlers = this.eventHandlers.get(event.type) || [];
    handlers.forEach(handler => handler(event));
  }
}
```

#### 8.10.6 与Chat视图的集成

```
┌─────────────────────────────────────────────────────────┐
│                    烛龙桌面端                            │
│                                                         │
│  ┌─────────┐  ┌─────────────────────────┐  ┌─────────┐│
│  │         │  │      视图切换            │  │         ││
│  │ Sidebar │  │  ┌─────┐ ┌─────┐        │  │  Right  ││
│  │         │  │  │Chat │ │Canvas│        │  │  Panel  ││
│  │         │  │  └─────┘ └─────┘        │  │         ││
│  │         │  │                         │  │         ││
│  │         │  │  ┌─────────────────┐    │  │         ││
│  │         │  │  │                 │    │  │         ││
│  │         │  │  │  Chat / Canvas  │    │  │         ││
│  │         │  │  │  共享Agent状态  │    │  │         ││
│  │         │  │  │                 │    │  │         ││
│  │         │  │  └─────────────────┘    │  │         ││
│  │         │  │                         │  │         ││
│  └─────────┘  └─────────────────────────┘  └─────────┘│
│                                                         │
└─────────────────────────────────────────────────────────┘

Chat视图和Canvas视图共享同一个Agent实例：
- Chat视图：文本对话形式展示Agent执行过程
- Canvas视图：可视化节点形式展示Agent执行过程
- 两个视图实时同步，切换视图不会丢失状态
```

### 8.11 多媒体节点类型（已实现 ✅）

#### 8.11.1 节点类型定义

节点设计，烛龙画布支持以下多媒体节点类型：

```typescript
// 节点类型枚举（已实现）
enum CanvasNodeType {
  // Agent循环节点
  Plan = 'plan',           // 计划节点 ✅
  Step = 'step',           // 步骤节点 ✅
  Tool = 'tool',           // 工具节点（设计中）
  Result = 'result',       // 结果节点（设计中）

  // 多媒体节点
  Text = 'text',           // 文本节点 ✅
  Image = 'image',         // 图片节点 ✅
  Video = 'video',         // 视频节点 ✅
  Audio = 'audio',         // 音频节点（设计中）
  Storyboard = 'storyboard', // 分镜节点（设计中）
  Config = 'config',       // 配置节点 ✅

  // 资产节点
  Asset = 'asset',         // 资产节点（设计中）
  Reference = 'reference', // 参考节点（设计中）
}
```

#### 8.11.2 多媒体节点特性

| 节点类型 | 特性 | 参考项目 |
|---------|------|----------|
| **Text** | 文本生成、提示词输入、多轮对话 | — |
| **Image** | 图片生成、图生图、参考图编辑 | — |
| **Video** | 视频生成、首帧/尾帧控制、时长设置 | — |
| **Audio** | TTS语音生成、语音选择、语速控制 | — |
| **Storyboard** | 分镜编辑、场景描述、镜头设置 | — |
| **Config** | 生成配置、模型选择、参数设置 | — |

#### 8.11.3 节点数据结构

```typescript
// 基础节点数据
interface BaseNodeData {
  id: string;
  type: CanvasNodeType;
  title: string;
  status: 'idle' | 'loading' | 'success' | 'error';
  position: { x: number; y: number };
  size: { width: number; height: number };
  metadata?: Record<string, any>;
}

// 文本节点数据
interface TextNodeData extends BaseNodeData {
  type: CanvasNodeType.Text;
  content: string;           // 文本内容
  prompt?: string;           // 提示词
  wordCount?: number;        // 字数
}

// 图片节点数据
interface ImageNodeData extends BaseNodeData {
  type: CanvasNodeType.Image;
  imageUrl?: string;         // 图片URL
  prompt?: string;           // 生成提示词
  width?: number;            // 图片宽度
  height?: number;           // 图片高度
  model?: string;            // 使用的模型
  aspect?: string;           // 宽高比
  referenceIds?: string[];   // 参考图片ID列表
}

// 视频节点数据
interface VideoNodeData extends BaseNodeData {
  type: CanvasNodeType.Video;
  videoUrl?: string;         // 视频URL
  prompt?: string;           // 生成提示词
  duration?: number;         // 时长（秒）
  orientation?: 'landscape' | 'portrait'; // 方向
  firstFrame?: string;       // 首帧图片
  lastFrame?: string;        // 尾帧图片
  referenceIds?: string[];   // 参考素材ID列表
}

// 音频节点数据
interface AudioNodeData extends BaseNodeData {
  type: CanvasNodeType.Audio;
  audioUrl?: string;         // 音频URL
  text?: string;             // TTS文本
  voice?: string;            // 语音选择
  speed?: number;            // 语速
  format?: string;           // 音频格式
}

// 分镜节点数据
interface StoryboardNodeData extends BaseNodeData {
  type: CanvasNodeType.Storyboard;
  scenes: StoryboardScene[]; // 场景列表
  script?: string;           // 剧本内容
}

interface StoryboardScene {
  id: string;
  description: string;       // 场景描述
  imageUrl?: string;         // 场景图片
  duration?: number;         // 场景时长
  cameraAngle?: string;      // 镜头角度
  dialogue?: string;         // 对白
}

// 配置节点数据
interface ConfigNodeData extends BaseNodeData {
  type: CanvasNodeType.Config;
  generationMode: 'text' | 'image' | 'video' | 'audio'; // 生成模式
  model: string;             // 模型选择
  params: Record<string, any>; // 生成参数
  prompt?: string;           // 组装后的提示词
  referenceIds?: string[];   // 参考节点ID列表
}
```

#### 8.11.4 缓存命中率保障

**铁律**：多媒体节点不影响Agent循环的缓存命中率

```
┌─────────────────────────────────────────────────────────┐
│                    缓存命中率保障                        │
│                                                         │
│  1. Agent循环节点（Plan/Step/Tool/Result）              │
│     - prefix只追加不修改                                │
│     - 历史只压缩不重排                                  │
│     - 裁剪只在动态区间                                  │
│                                                         │
│  2. 多媒体节点（Text/Image/Video/Audio）                │
│     - 独立于Agent循环的缓存体系                         │
│     - 不影响prefix稳定性                                │
│     - 资产数据单独存储                                  │
│                                                         │
│  3. 配置节点（Config）                                  │
│     - 生成参数不进入Agent上下文                         │
│     - 只在生成时使用                                    │
│     - 不影响缓存命中率                                  │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

### 8.12 连续工作流模式

#### 8.12.1 三段式生成流程

三段式设计，烛龙画布支持以下工作流模式：

```
[文本节点(提示词)] --连接--> [Config节点(生成配置)] --生成--> [结果节点(图片/视频/音频)]
[参考节点] ---连接------/
```

**工作流示例**：

```
┌─────────────────────────────────────────────────────────┐
│                    图片生成工作流                        │
│                                                         │
│   ┌─────────────┐      ┌─────────────┐                 │
│   │ Text Node   │      │ Config Node │                 │
│   │ "提示词"    │─────→│ "生成配置"  │                 │
│   └─────────────┘      └──────┬──────┘                 │
│                                │                        │
│   ┌─────────────┐             │                        │
│   │ Reference   │─────────────┘                        │
│   │ Node        │                                       │
│   │ "参考图片"  │                                       │
│   └─────────────┘                                       │
│                                │                        │
│                                ▼                        │
│                       ┌─────────────┐                   │
│                       │ Image Node  │                   │
│                       │ "生成结果"  │                   │
│                       └─────────────┘                   │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

#### 8.12.2 @引用机制

 @引用设计，支持在提示词中引用上游节点内容：

```typescript
// @引用语法
const prompt = "根据 @[node:text-1] 的描述，生成一张 @[node:image-2] 风格的图片";

// 运行时解析
function resolveReferences(prompt: string, nodes: Map<string, Node>): string {
  let resolved = prompt;
  for (const match of prompt.matchAll(/@\[node:([^\]]+)\]/g)) {
    const nodeId = match[1];
    const node = nodes.get(nodeId);
    if (node) {
      // 替换为实际内容
      resolved = resolved.replace(match[0], node.data.content || node.data.prompt || '');
    }
  }
  return resolved;
}
```

#### 8.12.3 连线追踪

```typescript
// 获取节点的所有上游节点
function getUpstreamNodes(nodeId: string, edges: Edge[]): string[] {
  return edges
    .filter(edge => edge.target === nodeId)
    .map(edge => edge.source);
}

// 获取节点的所有下游节点
function getDownstreamNodes(nodeId: string, edges: Edge[]): string[] {
  return edges
    .filter(edge => edge.source === nodeId)
    .map(edge => edge.target);
}

// 构建生成上下文
function buildGenerationContext(nodeId: string, nodes: Map<string, Node>, edges: Edge[]): GenerationContext {
  const upstreamNodes = getUpstreamNodes(nodeId, edges);
  const references = upstreamNodes
    .map(id => nodes.get(id))
    .filter(node => node && isResourceNode(node));

  return {
    prompt: resolveReferences(nodes.get(nodeId).data.prompt, nodes),
    references: references.map(node => ({
      id: node.id,
      type: node.type,
      content: node.data.content || node.data.imageUrl,
    })),
  };
}
```

#### 8.12.4 缓存命中率保障

**铁律**：连续工作流不影响Agent循环的缓存命中率

```
┌─────────────────────────────────────────────────────────┐
│                    工作流与缓存分离                      │
│                                                         │
│  Agent循环（缓存敏感）：                                │
│  - system prompt + skeleton 永不改写                    │
│  - 旧循环压缩为summary追加                              │
│  - 工具结果裁剪只在当前循环                             │
│                                                         │
│  多媒体工作流（缓存无关）：                             │
│  - 独立的生成上下文                                     │
│  - 不进入Agent循环的prefix                              │
│  - 资产数据单独存储和管理                               │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

### 8.13 资产管理系统

#### 8.13.1 资产类型定义

资产管理设计：

```typescript
// 资产类型
enum AssetType {
  // 内容资产
  Text = 'text',             // 文本资产
  Image = 'image',           // 图片资产
  Video = 'video',           // 视频资产
  Audio = 'audio',           // 音频资产

  // 结构资产
  Script = 'script',         // 剧本资产
  Storyboard = 'storyboard', // 分镜资产
  Outline = 'outline',       // 大纲资产

  // 角色资产
  Character = 'character',   // 角色资产
  Scene = 'scene',           // 场景资产
  Prop = 'prop',             // 道具资产

  // 参考资产
  Reference = 'reference',   // 参考资产
  Template = 'template',     // 模板资产
}

// 资产数据结构
interface Asset {
  id: string;
  type: AssetType;
  name: string;
  description?: string;
  content?: string;          // 文本内容
  url?: string;              // 媒体URL
  thumbnailUrl?: string;     // 缩略图URL
  metadata?: Record<string, any>;
  projectId: string;         // 所属项目ID
  createdAt: string;
  updatedAt: string;

  // 衍生资产
  parentId?: string;         // 父资产ID
  deriveType?: 'variant' | 'version' | 'branch'; // 衍生类型
}
```

#### 8.13.2 项目化资产沉淀

项目化资产管理：

```typescript
// 项目数据结构
interface Project {
  id: string;
  name: string;
  description?: string;
  assets: Asset[];           // 项目资产列表
  nodes: CanvasNode[];       // 画布节点列表
  edges: Edge[];             // 画布连线列表
  createdAt: string;
  updatedAt: string;
}

// 资产查询
function listAssets(projectId: string, type?: AssetType): Asset[] {
  return project.assets
    .filter(asset => !type || asset.type === type)
    .sort((a, b) => new Date(b.updatedAt).getTime() - new Date(a.updatedAt).getTime());
}

// 资产创建
function createAsset(projectId: string, input: CreateAssetInput): Asset {
  const asset: Asset = {
    id: generateId(),
    type: input.type,
    name: input.name,
    description: input.description,
    content: input.content,
    url: input.url,
    metadata: input.metadata,
    projectId,
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
  };
  project.assets.push(asset);
  return asset;
}
```

#### 8.13.3 衍生资产系统

衍生资产设计：

```typescript
// 衍生资产
interface DeriveAsset extends Asset {
  parentId: string;          // 父资产ID
  deriveType: 'variant' | 'version' | 'branch';
  prompt?: string;           // 生成提示词
  state: 'pending' | 'generating' | 'completed' | 'failed';
}

// 创建衍生资产
function createDeriveAsset(parentId: string, input: CreateDeriveAssetInput): DeriveAsset {
  const parent = getAsset(parentId);
  if (!parent) throw new Error('Parent asset not found');

  const derive: DeriveAsset = {
    id: generateId(),
    type: parent.type,
    name: input.name || `${parent.name} - 变体`,
    description: input.description,
    parentId,
    deriveType: input.deriveType || 'variant',
    prompt: input.prompt,
    state: 'pending',
    projectId: parent.projectId,
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
  };

  project.assets.push(derive);
  return derive;
}

// 生成衍生资产
async function generateDeriveAsset(deriveId: string): Promise<void> {
  const derive = getAsset(deriveId) as DeriveAsset;
  if (!derive) throw new Error('Derive asset not found');

  derive.state = 'generating';
  try {
    // 调用AI生成
    const result = await generateAsset(derive.type, derive.prompt);
    derive.url = result.url;
    derive.content = result.content;
    derive.state = 'completed';
  } catch (error) {
    derive.state = 'failed';
    derive.metadata = { ...derive.metadata, error: error.message };
  }
  derive.updatedAt = new Date().toISOString();
}
```

#### 8.13.4 资产引用机制

```typescript
// 节点引用资产
interface AssetReference {
  nodeId: string;            // 节点ID
  assetId: string;           // 资产ID
  referenceType: 'input' | 'output' | 'reference'; // 引用类型
}

// 获取节点引用的资产
function getNodeAssets(nodeId: string): Asset[] {
  const references = assetReferences.filter(ref => ref.nodeId === nodeId);
  return references.map(ref => getAsset(ref.assetId)).filter(Boolean);
}

// 获取引用资产的节点
function getAssetNodes(assetId: string): CanvasNode[] {
  const references = assetReferences.filter(ref => ref.assetId === assetId);
  return references.map(ref => getNode(ref.nodeId)).filter(Boolean);
}
```

#### 8.13.5 缓存命中率保障

**铁律**：资产管理系统不影响Agent循环的缓存命中率

```
┌─────────────────────────────────────────────────────────┐
│                    资产与缓存分离                        │
│                                                         │
│  Agent循环（缓存敏感）：                                │
│  - 资产数据不进入Agent上下文                            │
│  - 只在需要时通过工具调用获取                           │
│  - 工具结果裁剪只在当前循环                             │
│                                                         │
│  资产管理（缓存无关）：                                 │
│  - 独立的存储体系                                       │
│  - 支持大文件存储                                       │
│  - 支持版本管理                                         │
│  - 支持衍生资产                                         │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

### 8.14 画布助手

#### 8.14.1 核心功能

画布助手设计：

| 功能 | 说明 | 参考项目 |
|------|------|----------|
| **上下文对话** | 围绕选中节点进行对话 | — |
| **选中节点引用** | 选中节点自动作为上下文 | — |
| **多轮对话** | 支持多轮对话历史 | — |
| **画布快照** | 每次请求带上画布JSON快照 | — |
| **@资源引用** | 对话中@引用画布资源 | — |

#### 8.14.2 上下文构建

```typescript
// 构建助手上下文
interface AssistantContext {
  selectedNodes: CanvasNode[];    // 选中的节点
  canvasSnapshot: CanvasSnapshot; // 画布快照
  chatHistory: ChatMessage[];     // 对话历史
  userMessage: string;            // 用户消息
}

// 构建上下文消息
function buildAssistantMessages(context: AssistantContext): Message[] {
  const messages: Message[] = [];

  // 系统提示词
  messages.push({
    role: 'system',
    content: `你是烛龙画布助手。当前画布包含 ${context.canvasSnapshot.nodes.length} 个节点。
你可以帮助用户分析画布内容、生成新节点、修改现有节点等。
使用 @引用画布上的资源节点。`,
  });

  // 对话历史（最近8条）
  const recentHistory = context.chatHistory.slice(-8);
  messages.push(...recentHistory);

  // 用户消息（包含选中节点和画布快照）
  messages.push({
    role: 'user',
    content: buildUserMessage(context),
  });

  return messages;
}

// 构建用户消息
function buildUserMessage(context: AssistantContext): string {
  let content = '';

  // 选中节点的文本内容
  if (context.selectedNodes.length > 0) {
    content += '选中节点：\n';
    for (const node of context.selectedNodes) {
      if (node.type === 'text') {
        content += `- ${node.title}: ${node.data.content}\n`;
      } else if (node.type === 'image') {
        content += `- ${node.title}: [图片]\n`;
      }
    }
    content += '\n';
  }

  // 画布快照（压缩版）
  content += `当前画布：${JSON.stringify(compressSnapshot(context.canvasSnapshot))}\n\n`;

  // 用户需求
  content += `用户需求：${context.userMessage}`;

  return content;
}
```

#### 8.14.3 @资源引用

```typescript
// @引用解析
function parseResourceReferences(message: string): ResourceReference[] {
  const references: ResourceReference[] = [];
  const regex = /@\[([^\]]+)\]/g;
  let match;

  while ((match = regex.exec(message)) !== null) {
    const ref = match[1];
    if (ref.startsWith('node:')) {
      const nodeId = ref.substring(5);
      references.push({ type: 'node', id: nodeId });
    } else if (ref.startsWith('asset:')) {
      const assetId = ref.substring(6);
      references.push({ type: 'asset', id: assetId });
    }
  }

  return references;
}

// 获取引用的资源内容
function getReferencedContent(references: ResourceReference[]): ReferencedContent[] {
  return references.map(ref => {
    if (ref.type === 'node') {
      const node = getNode(ref.id);
      return {
        id: ref.id,
        type: node.type,
        title: node.title,
        content: node.data.content || node.data.prompt,
        imageUrl: node.data.imageUrl,
      };
    } else if (ref.type === 'asset') {
      const asset = getAsset(ref.id);
      return {
        id: ref.id,
        type: asset.type,
        title: asset.name,
        content: asset.content,
        imageUrl: asset.url,
      };
    }
    return null;
  }).filter(Boolean);
}
```

#### 8.14.4 画布快照

```typescript
// 画布快照
interface CanvasSnapshot {
  nodes: NodeSnapshot[];
  edges: EdgeSnapshot[];
  viewport: Viewport;
}

interface NodeSnapshot {
  id: string;
  type: string;
  title: string;
  status: string;
  content?: string;          // 文本内容
  imageUrl?: string;         // 图片URL
  position: { x: number; y: number };
}

interface EdgeSnapshot {
  id: string;
  source: string;
  target: string;
}

// 压缩快照（减少token消耗）
function compressSnapshot(snapshot: CanvasSnapshot): object {
  return {
    nodes: snapshot.nodes.map(node => ({
      id: node.id,
      type: node.type,
      title: node.title,
      status: node.status,
      content: node.content?.substring(0, 100), // 截断长文本
    })),
    edges: snapshot.edges.map(edge => ({
      source: edge.source,
      target: edge.target,
    })),
  };
}
```

#### 8.14.5 缓存命中率保障

**铁律**：画布助手不影响Agent循环的缓存命中率

```
┌─────────────────────────────────────────────────────────┐
│                    助手与缓存分离                        │
│                                                         │
│  Agent循环（缓存敏感）：                                │
│  - 助手对话不进入Agent上下文                            │
│  - 助手生成的内容不修改prefix                           │
│  - 助手操作独立于Agent循环                              │
│                                                         │
│  画布助手（缓存无关）：                                 │
│  - 独立的对话上下文                                     │
│  - 独立的工具调用                                       │
│  - 独立的状态管理                                       │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

### 8.15 Ops操作抽象

#### 8.15.1 操作类型定义

 Ops 抽象设计：

```typescript
// 画布操作类型
type CanvasOp =
  // 节点操作
  | { type: 'add_node'; nodeType: string; position: Position; metadata?: Record<string, any> }
  | { type: 'update_node'; id: string; patch?: Partial<NodeData>; metadata?: Record<string, any> }
  | { type: 'delete_node'; id?: string; ids?: string[]; nodeType?: string }
  | { type: 'move_node'; id: string; position: Position }
  | { type: 'resize_node'; id: string; size: Size }

  // 连线操作
  | { type: 'connect_nodes'; fromNodeId: string; toNodeId: string }
  | { type: 'disconnect_nodes'; fromNodeId: string; toNodeId: string }
  | { type: 'delete_connections'; id?: string; ids?: string[]; all?: boolean }

  // 选择操作
  | { type: 'select_nodes'; ids: string[] }
  | { type: 'deselect_all' }

  // 视图操作
  | { type: 'set_viewport'; viewport: Viewport }
  | { type: 'fit_view' }
  | { type: 'zoom_in' }
  | { type: 'zoom_out' }

  // 分组操作
  | { type: 'group_nodes'; ids: string[]; groupId?: string }
  | { type: 'ungroup_nodes'; groupId: string }

  // 生成操作
  | { type: 'run_generation'; nodeId: string; mode?: string; prompt?: string }
  | { type: 'cancel_generation'; nodeId: string }

  // 资产操作
  | { type: 'create_asset'; assetType: string; input: CreateAssetInput }
  | { type: 'update_asset'; id: string; patch: Partial<Asset> }
  | { type: 'delete_asset'; id: string }

  // 批量操作
  | { type: 'batch_ops'; ops: CanvasOp[] };
```

#### 8.15.2 操作执行引擎

```typescript
// 操作执行结果
interface OpResult {
  success: boolean;
  snapshot: CanvasSnapshot;
  error?: string;
}

// 执行操作
function applyCanvasOp(state: CanvasState, op: CanvasOp): OpResult {
  switch (op.type) {
    case 'add_node':
      return addNode(state, op);
    case 'update_node':
      return updateNode(state, op);
    case 'delete_node':
      return deleteNode(state, op);
    case 'connect_nodes':
      return connectNodes(state, op);
    case 'disconnect_nodes':
      return disconnectNodes(state, op);
    // ... 其他操作
    default:
      return { success: false, snapshot: state.snapshot, error: `Unknown op type: ${op.type}` };
  }
}

// 批量执行操作
function applyCanvasOps(state: CanvasState, ops: CanvasOp[]): OpResult {
  let currentState = state;
  for (const op of ops) {
    const result = applyCanvasOp(currentState, op);
    if (!result.success) {
      return result;
    }
    currentState = { ...currentState, snapshot: result.snapshot };
  }
  return { success: true, snapshot: currentState.snapshot };
}
```

#### 8.15.3 撤销/重做支持

```typescript
// 操作历史
interface OperationHistory {
  past: CanvasSnapshot[];    // 历史快照
  present: CanvasSnapshot;   // 当前快照
  future: CanvasSnapshot[];  // 未来快照（用于重做）
}

// 执行操作并记录历史
function executeWithHistory(state: OperationHistory, ops: CanvasOp[]): OperationHistory {
  const result = applyCanvasOps({ snapshot: state.present }, ops);
  if (!result.success) {
    return state;
  }

  return {
    past: [...state.past, state.present],
    present: result.snapshot,
    future: [], // 执行新操作后清空未来历史
  };
}

// 撤销
function undo(state: OperationHistory): OperationHistory {
  if (state.past.length === 0) {
    return state;
  }

  const previous = state.past[state.past.length - 1];
  return {
    past: state.past.slice(0, -1),
    present: previous,
    future: [state.present, ...state.future],
  };
}

// 重做
function redo(state: OperationHistory): OperationHistory {
  if (state.future.length === 0) {
    return state;
  }

  const next = state.future[0];
  return {
    past: [...state.past, state.present],
    present: next,
    future: state.future.slice(1),
  };
}
```

#### 8.15.4 Agent确认机制

```typescript
// 需要确认的操作类型
const REQUIRES_CONFIRMATION = new Set([
  'delete_node',
  'delete_connections',
  'delete_asset',
  'batch_ops',
]);

// 检查操作是否需要确认
function requiresConfirmation(op: CanvasOp): boolean {
  return REQUIRES_CONFIRMATION.has(op.type);
}

// 执行操作（带确认）
async function executeWithConfirmation(
  state: CanvasState,
  ops: CanvasOp[],
  confirm: (ops: CanvasOp[]) => Promise<boolean>
): Promise<OpResult> {
  // 检查是否需要确认
  const needsConfirmation = ops.some(op => requiresConfirmation(op));

  if (needsConfirmation) {
    const approved = await confirm(ops);
    if (!approved) {
      return { success: false, snapshot: state.snapshot, error: 'User denied' };
    }
  }

  return applyCanvasOps(state, ops);
}
```

#### 8.15.5 缓存命中率保障

**铁律**：Ops操作不影响Agent循环的缓存命中率

```
┌─────────────────────────────────────────────────────────┐
│                    Ops与缓存分离                         │
│                                                         │
│  Agent循环（缓存敏感）：                                │
│  - Ops操作不进入Agent上下文                             │
│  - Ops操作不修改prefix                                  │
│  - Ops操作独立于Agent循环                               │
│                                                         │
│  Ops操作（缓存无关）：                                  │
│  - 独立的操作历史                                       │
│  - 独立的撤销/重做                                      │
│  - 独立的确认机制                                       │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

### 8.16 协作功能

#### 8.16.1 多Agent协作

多Agent协作设计：

```typescript
// Agent协作管理
interface AgentCollaboration {
  agents: AgentInfo[];       // 参与协作的Agent列表
  tasks: Task[];             // 任务列表
  messages: AgentMessage[];  // Agent间消息
  workspace: Workspace;      // 共享工作空间
}

// Agent信息
interface AgentInfo {
  id: string;
  name: string;
  role: 'coordinator' | 'worker' | 'reviewer'; // 角色
  status: 'idle' | 'working' | 'waiting' | 'completed';
  capabilities: string[];    // 能力列表
}

// Agent间消息
interface AgentMessage {
  id: string;
  from: string;              // 发送者Agent ID
  to: string;                // 接收者Agent ID
  type: 'task' | 'question' | 'answer' | 'feedback';
  content: string;
  timestamp: string;
}

// 任务分配
interface Task {
  id: string;
  description: string;
  assignedTo: string;        // 分配的Agent ID
  status: 'pending' | 'in_progress' | 'completed' | 'failed';
  dependencies: string[];    // 依赖的任务ID列表
  result?: any;
}
```

#### 8.16.2 团队协作

```typescript
// 协作会话
interface CollaborationSession {
  id: string;
  name: string;
  participants: Participant[]; // 参与者列表
  canvas: CanvasSnapshot;      // 共享画布
  cursors: Cursor[];           // 参与者光标
  selections: Selection[];     // 参与者选择
  createdAt: string;
}

// 参与者
interface Participant {
  id: string;
  name: string;
  role: 'owner' | 'editor' | 'viewer';
  color: string;               // 光标颜色
  isOnline: boolean;
}

// 实时同步
interface SyncMessage {
  type: 'cursor_move' | 'selection_change' | 'op' | 'chat';
  userId: string;
  data: any;
  timestamp: string;
}
```

#### 8.16.3 缓存命中率保障

**铁律**：协作功能不影响Agent循环的缓存命中率

```
┌─────────────────────────────────────────────────────────┐
│                    协作与缓存分离                        │
│                                                         │
│  Agent循环（缓存敏感）：                                │
│  - 协作消息不进入Agent上下文                            │
│  - 协作操作不修改prefix                                 │
│  - 协作状态独立于Agent循环                              │
│                                                         │
│  协作功能（缓存无关）：                                 │
│  - 独立的同步机制                                       │
│  - 独立的权限管理                                       │
│  - 独立的消息系统                                       │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

### 8.17 导出与分享

#### 8.17.1 导出功能

```typescript
// 导出格式
enum ExportFormat {
  PNG = 'png',               // 图片格式
  SVG = 'svg',               // 矢量格式
  JSON = 'json',             // JSON格式
  PDF = 'pdf',               // PDF格式
}

// 导出选项
interface ExportOptions {
  format: ExportFormat;
  includeMetadata: boolean;  // 是否包含元数据
  quality: number;           // 图片质量（0-100）
  scale: number;             // 缩放比例
  background: boolean;       // 是否包含背景
}

// 导出画布为图片
async function exportCanvasAsImage(options: ExportOptions): Promise<Blob> {
  const canvas = document.createElement('canvas');
  const ctx = canvas.getContext('2d');

  // 设置画布尺寸
  canvas.width = options.scale * canvasWidth;
  canvas.height = options.scale * canvasHeight;

  // 绘制背景
  if (options.background) {
    ctx.fillStyle = '#ffffff';
    ctx.fillRect(0, 0, canvas.width, canvas.height);
  }

  // 绘制节点和连线
  for (const node of nodes) {
    drawNode(ctx, node, options.scale);
  }
  for (const edge of edges) {
    drawEdge(ctx, edge, options.scale);
  }

  // 导出为Blob
  return new Promise(resolve => {
    canvas.toBlob(resolve, `image/${options.format}`, options.quality / 100);
  });
}

// 导出工作流为JSON
function exportWorkflowAsJSON(): string {
  const workflow = {
    version: '1.0',
    name: project.name,
    nodes: nodes.map(node => ({
      id: node.id,
      type: node.type,
      data: node.data,
      position: node.position,
    })),
    edges: edges.map(edge => ({
      id: edge.id,
      source: edge.source,
      target: edge.target,
    })),
    metadata: {
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
    },
  };

  return JSON.stringify(workflow, null, 2);
}
```

#### 8.17.2 导入功能

```typescript
// 导入工作流
function importWorkflowFromJSON(json: string): ImportResult {
  try {
    const workflow = JSON.parse(json);

    // 验证格式
    if (!workflow.version || !workflow.nodes || !workflow.edges) {
      return { success: false, error: 'Invalid workflow format' };
    }

    // 导入节点
    const importedNodes = workflow.nodes.map(node => ({
      ...node,
      id: generateId(), // 生成新ID避免冲突
    }));

    // 导入连线
    const nodeIdMap = new Map(workflow.nodes.map((node, i) => [node.id, importedNodes[i].id]));
    const importedEdges = workflow.edges.map(edge => ({
      ...edge,
      id: generateId(),
      source: nodeIdMap.get(edge.source) || edge.source,
      target: nodeIdMap.get(edge.target) || edge.target,
    }));

    return {
      success: true,
      nodes: importedNodes,
      edges: importedEdges,
    };
  } catch (error) {
    return { success: false, error: error.message };
  }
}
```

#### 8.17.3 分享功能

```typescript
// 分享链接
interface ShareLink {
  id: string;
  url: string;
  permissions: 'view' | 'edit';
  expiresAt?: string;
  password?: string;
}

// 创建分享链接
function createShareLink(options: ShareLinkOptions): ShareLink {
  const link: ShareLink = {
    id: generateId(),
    url: `${baseUrl}/canvas/shared/${id}`,
    permissions: options.permissions || 'view',
    expiresAt: options.expiresAt,
    password: options.password,
  };

  // 保存到数据库
  saveShareLink(link);

  return link;
}

// 访问分享链接
function accessShareLink(linkId: string, password?: string): AccessResult {
  const link = getShareLink(linkId);
  if (!link) {
    return { success: false, error: 'Link not found' };
  }

  // 检查是否过期
  if (link.expiresAt && new Date(link.expiresAt) < new Date()) {
    return { success: false, error: 'Link expired' };
  }

  // 检查密码
  if (link.password && link.password !== password) {
    return { success: false, error: 'Invalid password' };
  }

  // 返回画布数据
  return {
    success: true,
    canvas: getCanvasData(),
    permissions: link.permissions,
  };
}
```

### 8.18 版本管理

#### 8.18.1 版本历史

```typescript
// 画布版本
interface CanvasVersion {
  id: string;
  name: string;
  description?: string;
  snapshot: CanvasSnapshot;
  createdAt: string;
  createdBy: string;
}

// 版本历史管理
interface VersionHistory {
  versions: CanvasVersion[];
  currentVersionId: string;
}

// 创建版本
function createVersion(name: string, description?: string): CanvasVersion {
  const version: CanvasVersion = {
    id: generateId(),
    name,
    description,
    snapshot: deepClone(currentSnapshot),
    createdAt: new Date().toISOString(),
    createdBy: currentUser.id,
  };

  versionHistory.versions.push(version);
  versionHistory.currentVersionId = version.id;

  return version;
}

// 恢复版本
function restoreVersion(versionId: string): CanvasSnapshot {
  const version = versionHistory.versions.find(v => v.id === versionId);
  if (!version) {
    throw new Error('Version not found');
  }

  // 保存当前状态为新版本
  createVersion(`Auto-save before restore`, `Restored from version ${version.name}`);

  // 恢复快照
  currentSnapshot = deepClone(version.snapshot);
  versionHistory.currentVersionId = versionId;

  return currentSnapshot;
}
```

#### 8.18.2 快照对比

```typescript
// 快照差异
interface SnapshotDiff {
  added: NodeDiff[];         // 新增的节点
  removed: NodeDiff[];       // 删除的节点
  modified: NodeDiff[];      // 修改的节点
  edgesAdded: EdgeDiff[];    // 新增的连线
  edgesRemoved: EdgeDiff[];  // 删除的连线
}

interface NodeDiff {
  id: string;
  type: string;
  title: string;
  before?: Partial<NodeData>;
  after?: Partial<NodeData>;
}

interface EdgeDiff {
  id: string;
  source: string;
  target: string;
}

// 对比两个快照
function diffSnapshots(before: CanvasSnapshot, after: CanvasSnapshot): SnapshotDiff {
  const diff: SnapshotDiff = {
    added: [],
    removed: [],
    modified: [],
    edgesAdded: [],
    edgesRemoved: [],
  };

  // 找出新增和修改的节点
  for (const afterNode of after.nodes) {
    const beforeNode = before.nodes.find(n => n.id === afterNode.id);
    if (!beforeNode) {
      diff.added.push({ id: afterNode.id, type: afterNode.type, title: afterNode.title });
    } else if (JSON.stringify(beforeNode.data) !== JSON.stringify(afterNode.data)) {
      diff.modified.push({
        id: afterNode.id,
        type: afterNode.type,
        title: afterNode.title,
        before: beforeNode.data,
        after: afterNode.data,
      });
    }
  }

  // 找出删除的节点
  for (const beforeNode of before.nodes) {
    const afterNode = after.nodes.find(n => n.id === beforeNode.id);
    if (!afterNode) {
      diff.removed.push({ id: beforeNode.id, type: beforeNode.type, title: beforeNode.title });
    }
  }

  // 找出新增和删除的连线
  for (const afterEdge of after.edges) {
    const beforeEdge = before.edges.find(e => e.id === afterEdge.id);
    if (!beforeEdge) {
      diff.edgesAdded.push({ id: afterEdge.id, source: afterEdge.source, target: afterEdge.target });
    }
  }
  for (const beforeEdge of before.edges) {
    const afterEdge = after.edges.find(e => e.id === beforeEdge.id);
    if (!afterEdge) {
      diff.edgesRemoved.push({ id: beforeEdge.id, source: beforeEdge.source, target: beforeEdge.target });
    }
  }

  return diff;
}
```

### 8.19 性能优化

#### 8.19.1 虚拟化渲染

```typescript
// 虚拟化配置
interface VirtualizationConfig {
  enabled: boolean;
  overscan: number;          // 额外渲染的节点数量
  threshold: number;         // 启用虚拟化的节点数量阈值
}

// 获取可见节点
function getVisibleNodes(
  nodes: CanvasNode[],
  viewport: Viewport,
  config: VirtualizationConfig
): CanvasNode[] {
  if (!config.enabled || nodes.length < config.threshold) {
    return nodes;
  }

  // 计算可见区域
  const visibleArea = {
    x: viewport.x - config.overscan * viewport.zoom,
    y: viewport.y - config.overscan * viewport.zoom,
    width: viewport.width + 2 * config.overscan * viewport.zoom,
    height: viewport.height + 2 * config.overscan * viewport.zoom,
  };

  // 过滤可见节点
  return nodes.filter(node => {
    const nodeArea = {
      x: node.position.x,
      y: node.position.y,
      width: node.size.width,
      height: node.size.height,
    };

    return rectsOverlap(visibleArea, nodeArea);
  });
}
```

#### 8.19.2 批量操作优化

```typescript
// 批量操作
function batchOperations(ops: CanvasOp[]): CanvasOp[] {
  // 合并相同类型的操作
  const merged = new Map<string, CanvasOp[]>();

  for (const op of ops) {
    const key = op.type;
    if (!merged.has(key)) {
      merged.set(key, []);
    }
    merged.get(key).push(op);
  }

  // 合并后的操作
  const batched: CanvasOp[] = [];

  for (const [type, typeOps] of merged) {
    if (type === 'update_node') {
      // 合并节点更新
      const nodeUpdates = new Map<string, any>();
      for (const op of typeOps) {
        const updateOp = op as { type: 'update_node'; id: string; patch?: any };
        const existing = nodeUpdates.get(updateOp.id) || {};
        nodeUpdates.set(updateOp.id, { ...existing, ...updateOp.patch });
      }

      for (const [id, patch] of nodeUpdates) {
        batched.push({ type: 'update_node', id, patch });
      }
    } else {
      // 其他操作直接添加
      batched.push(...typeOps);
    }
  }

  return batched;
}
```

#### 8.19.3 懒加载

```typescript
// 懒加载配置
interface LazyLoadConfig {
  enabled: boolean;
  threshold: number;         // 启用懒加载的节点数量阈值
  preloadDistance: number;   // 预加载距离
}

// 懒加载节点内容
async function loadNodeContent(nodeId: string): Promise<NodeContent> {
  const node = getNode(nodeId);
  if (!node) {
    throw new Error('Node not found');
  }

  // 检查是否已加载
  if (node.data.content) {
    return node.data.content;
  }

  // 加载内容
  const content = await fetchNodeContent(nodeId);

  // 更新节点
  updateNode(nodeId, { data: { ...node.data, content } });

  return content;
}

// 预加载附近节点
async function preloadNearbyNodes(nodeId: string, distance: number): Promise<void> {
  const node = getNode(nodeId);
  if (!node) {
    return;
  }

  // 获取附近的节点
  const nearbyNodes = getNodesInRadius(node.position, distance);

  // 预加载内容
  await Promise.all(
    nearbyNodes.map(n => loadNodeContent(n.id).catch(() => {}))
  );
}
```

### 8.20 离线支持

#### 8.20.1 本地存储

```typescript
// 本地存储配置
interface LocalStorageConfig {
  enabled: boolean;
  maxSize: number;           // 最大存储大小（字段）
  autoSave: boolean;         // 自动保存
  autoSaveInterval: number;  // 自动保存间隔（毫秒）
}

// 本地存储管理
class LocalStorageManager {
  private config: LocalStorageConfig;
  private db: IDBDatabase;

  constructor(config: LocalStorageConfig) {
    this.config = config;
  }

  // 初始化数据库
  async init(): Promise<void> {
    return new Promise((resolve, reject) => {
      const request = indexedDB.open('zhulong-canvas', 1);

      request.onerror = () => reject(request.error);
      request.onsuccess = () => {
        this.db = request.result;
        resolve();
      };

      request.onupgradeneeded = (event) => {
        const db = (event.target as IDBOpenDBRequest).result;

        // 创建对象存储
        if (!db.objectStoreNames.contains('canvases')) {
          db.createObjectStore('canvases', { keyPath: 'id' });
        }
        if (!db.objectStoreNames.contains('assets')) {
          db.createObjectStore('assets', { keyPath: 'id' });
        }
        if (!db.objectStoreNames.contains('versions')) {
          db.createObjectStore('versions', { keyPath: 'id' });
        }
      };
    });
  }

  // 保存画布
  async saveCanvas(canvas: CanvasData): Promise<void> {
    const transaction = this.db.transaction(['canvases'], 'readwrite');
    const store = transaction.objectStore('canvases');
    await store.put(canvas);
  }

  // 加载画布
  async loadCanvas(id: string): Promise<CanvasData | null> {
    const transaction = this.db.transaction(['canvases'], 'readonly');
    const store = transaction.objectStore('canvases');
    return new Promise((resolve, reject) => {
      const request = store.get(id);
      request.onerror = () => reject(request.error);
      request.onsuccess = () => resolve(request.result || null);
    });
  }

  // 保存资产
  async saveAsset(asset: Asset): Promise<void> {
    const transaction = this.db.transaction(['assets'], 'readwrite');
    const store = transaction.objectStore('assets');
    await store.put(asset);
  }

  // 加载资产
  async loadAsset(id: string): Promise<Asset | null> {
    const transaction = this.db.transaction(['assets'], 'readonly');
    const store = transaction.objectStore('assets');
    return new Promise((resolve, reject) => {
      const request = store.get(id);
      request.onerror = () => reject(request.error);
      request.onsuccess = () => resolve(request.result || null);
    });
  }
}
```

#### 8.20.2 离线工作

```typescript
// 离线状态
enum OfflineStatus {
  Online = 'online',
  Offline = 'offline',
  Syncing = 'syncing',
}

// 离线管理
class OfflineManager {
  private status: OfflineStatus = OfflineStatus.Online;
  private pendingOps: CanvasOp[] = [];

  // 检查网络状态
  checkNetworkStatus(): OfflineStatus {
    return navigator.onLine ? OfflineStatus.Online : OfflineStatus.Offline;
  }

  // 添加待同步操作
  addPendingOp(op: CanvasOp): void {
    this.pendingOps.push(op);
  }

  // 同步待同步操作
  async syncPendingOps(): Promise<void> {
    if (this.pendingOps.length === 0) {
      return;
    }

    this.status = OfflineStatus.Syncing;

    try {
      // 批量发送操作
      await sendOpsToServer(this.pendingOps);

      // 清空待同步操作
      this.pendingOps = [];

      this.status = OfflineStatus.Online;
    } catch (error) {
      console.error('Sync failed:', error);
      this.status = OfflineStatus.Offline;
    }
  }
}
```

#### 8.20.3 数据同步

```typescript
// 同步配置
interface SyncConfig {
  enabled: boolean;
  provider: 'webdav' | 'cloud' | 'custom';
  interval: number;          // 同步间隔（毫秒）
  conflictResolution: 'local' | 'remote' | 'merge';
}

// 同步管理
class SyncManager {
  private config: SyncConfig;
  private lastSyncTime: Date | null = null;

  constructor(config: SyncConfig) {
    this.config = config;
  }

  // 同步数据
  async sync(): Promise<SyncResult> {
    if (!this.config.enabled) {
      return { success: true, message: 'Sync disabled' };
    }

    try {
      // 获取本地数据
      const localData = await getLocalData();

      // 获取远程数据
      const remoteData = await getRemoteData();

      // 合并数据
      const mergedData = await mergeData(localData, remoteData, this.config.conflictResolution);

      // 保存合并后的数据
      await saveLocalData(mergedData);
      await saveRemoteData(mergedData);

      this.lastSyncTime = new Date();

      return { success: true, message: 'Sync completed' };
    } catch (error) {
      return { success: false, message: error.message };
    }
  }
}
```

### 8.21 AI增强

#### 8.21.1 AI自动布局

```typescript
// AI布局建议
interface LayoutSuggestion {
  nodes: { id: string; position: Position }[];
  confidence: number;
  reason: string;
}

// 获取AI布局建议
async function getAILayoutSuggestion(nodes: CanvasNode[]): Promise<LayoutSuggestion> {
  // 构建提示词
  const prompt = `请为以下节点推荐布局：
${nodes.map(n => `- ${n.title} (${n.type})`).join('\n')}

请返回JSON格式的布局建议，包含每个节点的推荐位置。`;

  // 调用AI
  const response = await callAI(prompt);

  // 解析响应
  const suggestion = JSON.parse(response);

  return {
    nodes: suggestion.nodes,
    confidence: suggestion.confidence,
    reason: suggestion.reason,
  };
}

// 应用AI布局
async function applyAILayout(): Promise<void> {
  const suggestion = await getAILayoutSuggestion(nodes);

  // 应用布局
  const ops: CanvasOp[] = suggestion.nodes.map(node => ({
    type: 'move_node' as const,
    id: node.id,
    position: node.position,
  }));

  await applyCanvasOps(state, ops);
}
```

#### 8.21.2 AI推荐节点

```typescript
// 节点推荐
interface NodeRecommendation {
  type: string;
  title: string;
  description: string;
  confidence: number;
  position: Position;
}

// 获取节点推荐
async function getNodeRecommendations(context: string): Promise<NodeRecommendation[]> {
  // 构建提示词
  const prompt = `基于以下上下文，推荐可能需要的节点：
上下文：${context}

请返回JSON格式的节点推荐列表。`;

  // 调用AI
  const response = await callAI(prompt);

  // 解析响应
  const recommendations = JSON.parse(response);

  return recommendations;
}

// 应用节点推荐
async function applyNodeRecommendation(recommendation: NodeRecommendation): Promise<void> {
  const op: CanvasOp = {
    type: 'add_node',
    nodeType: recommendation.type,
    position: recommendation.position,
    metadata: {
      title: recommendation.title,
      description: recommendation.description,
    },
  };

  await applyCanvasOp(state, op);
}
```

#### 8.21.3 AI优化工作流

```typescript
// 工作流优化建议
interface WorkflowOptimization {
  suggestions: OptimizationSuggestion[];
  estimatedImprovement: number; // 预期改进百分比
}

interface OptimizationSuggestion {
  type: 'merge' | 'split' | 'reorder' | 'remove';
  nodeIds: string[];
  reason: string;
  impact: 'low' | 'medium' | 'high';
}

// 获取工作流优化建议
async function getWorkflowOptimization(): Promise<WorkflowOptimization> {
  // 构建提示词
  const prompt = `请分析以下工作流并提供优化建议：
节点：${nodes.map(n => `${n.title} (${n.type})`).join(', ')}
连线：${edges.map(e => `${e.source} -> ${e.target}`).join(', ')}

请返回JSON格式的优化建议。`;

  // 调用AI
  const response = await callAI(prompt);

  // 解析响应
  const optimization = JSON.parse(response);

  return optimization;
}
```

### 8.22 节点扩展

#### 8.22.1 自定义节点类型

```typescript
// 节点类型定义
interface NodeTypeDefinition {
  type: string;
  name: string;
  description: string;
  icon: string;
  defaultSize: Size;
  inputs: NodePort[];
  outputs: NodePort[];
  component: React.ComponentType<NodeProps>;
}

// 节点端口
interface NodePort {
  id: string;
  name: string;
  type: 'text' | 'image' | 'video' | 'audio' | 'any';
  required: boolean;
}

// 节点注册
class NodeRegistry {
  private types = new Map<string, NodeTypeDefinition>();

  // 注册节点类型
  register(definition: NodeTypeDefinition): void {
    this.types.set(definition.type, definition);
  }

  // 获取节点类型
  getType(type: string): NodeTypeDefinition | undefined {
    return this.types.get(type);
  }

  // 获取所有节点类型
  getAllTypes(): NodeTypeDefinition[] {
    return Array.from(this.types.values());
  }
}
```

#### 8.22.2 插件系统

```typescript
// 插件接口
interface CanvasPlugin {
  id: string;
  name: string;
  version: string;
  description: string;

  // 生命周期
  onInstall?: () => Promise<void>;
  onUninstall?: () => Promise<void>;
  onActivate?: () => Promise<void>;
  onDeactivate?: () => Promise<void>;

  // 扩展点
  nodeTypes?: NodeTypeDefinition[];
  tools?: ToolDefinition[];
  panels?: PanelDefinition[];
}

// 插件管理
class PluginManager {
  private plugins = new Map<string, CanvasPlugin>();

  // 安装插件
  async install(plugin: CanvasPlugin): Promise<void> {
    // 验证插件
    if (!this.validatePlugin(plugin)) {
      throw new Error('Invalid plugin');
    }

    // 注册节点类型
    if (plugin.nodeTypes) {
      for (const nodeType of plugin.nodeTypes) {
        nodeRegistry.register(nodeType);
      }
    }

    // 调用生命周期钩子
    if (plugin.onInstall) {
      await plugin.onInstall();
    }

    this.plugins.set(plugin.id, plugin);
  }

  // 卸载插件
  async uninstall(pluginId: string): Promise<void> {
    const plugin = this.plugins.get(pluginId);
    if (!plugin) {
      return;
    }

    // 调用生命周期钩子
    if (plugin.onUninstall) {
      await plugin.onUninstall();
    }

    this.plugins.delete(pluginId);
  }
}
```

### 8.23 设计参考

| 项目 | 参考内容 | 烛龙实现方式 |
|------|----------|--------------|
| **React Flow** | API设计、扩展机制、性能优化 | 按官方文档自行实现 |

---

## 9. 风险与缓解

| 风险 | 影响 | 缓解策略 |
|------|------|---------|
| LLM 输出 JSON 格式不稳定 | 计划解析失败 | 多次重试 + 容错解析 + 降级到纯文本模式 |
| 自主循环死循环 | 无限消耗 token | 最大循环次数 + 成本上限 + 超时机制 + 停滞检测 |
| DeepSeek API 变更 | prefix-cache 优化失效 | 密切关注 API 更新，及时适配 |
| P1 模块过多 | 开发周期拉长 | 优先实现核心 P0，P1 按需迭代 |
| 积木块/内部模型效果不明显 | 学习能力不如预期 | 先实现基础版本，根据实际效果迭代优化 |

---

## 致谢

本项目设计思想受 [DeepSeek-Reasonix](https://github.com/esengine/DeepSeek-Reasonix) 启发，其 MCP 工具协议规范、prefix-cache 稳定性设计思路为 Zhulong 的缓存命中率铁律奠定了基础。所有代码均为独立实现。 |

---

## 致谢

本项目设计思想受 [DeepSeek-Reasonix](https://github.com/esengine/DeepSeek-Reasonix) 启发，其 MCP 工具协议规范、prefix-cache 稳定性设计思路为 Zhulong 的缓存命中率铁律奠定了基础。所有代码均为独立实现。

---
## 附录 E: 实际实现状态总结

> **更新日期**：2026-06-22
> **实现状态**：P0-P4 全部完成

### E.1 后端模块实现状态

| 模块 | 状态 | 测试 | 说明 |
|------|------|------|------|
| Controller | ✅ | ✅ | 状态机控制器 |
| Planner | ✅ | ✅ | LLM规划器 |
| Executor | ✅ | ✅ | 工具执行器 |
| Reflector | ✅ | ✅ | 反省器 |
| Memory | ✅ | ✅ | 三层记忆系统 |
| Compressor | ✅ | ✅ | 上下文压缩 |
| Checkpoint | ✅ | ✅ | 检查点 |
| Budget | ✅ | ✅ | 成本控制 |
| Trace | ✅ | ✅ | 可观测性 |
| Human | ✅ | ✅ | 人机协作 |
| Tools | ✅ | ✅ | 工具层 |
| Provider | ✅ | ✅ | DeepSeek Provider |
| Stability | ✅ | ✅ | 稳定性分析 |
| Information | ✅ | ✅ | 信息论 |
| Stagnation | ✅ | ✅ | 停滞检测 |
| Exploration | ✅ | ✅ | 探索触发 |
| Synergetics | ✅ | ✅ | 协同学 |
| Learning | ✅ | ✅ | 自适应学习 |
| Skills | ✅ | ✅ | 技能管理 |
| Approval | ✅ | ✅ | 审批引擎 |
| Environment | ✅ | ✅ | 环境感知 |
| **breaker** | ✅ | ✅ | 断路器 |
| **workflow** | ✅ | ✅ | 工作流模式 (full/hotfix/tweak) |
| **quality** | ✅ | ✅ | 8 维 GC 扫描器 |
| **evolution** | ✅ | ✅ | 自进化 (reviewer/curator/suggester) |
| **hook** | ✅ | ✅ | 3 层自动化钩子 (pre/post tool/plan/reflect) |
| **health** | ✅ | ✅ | 运行时健康检查 (P4 独立服务) |
| **state** | ✅ | ✅ | 状态持久化 (P4 独立服务) |
| **upgrade** | ✅ | ✅ | 智能升级 (P4 独立服务) |
| **cache** | ✅ | ✅ | PrefixCache 高级缓存 (P4 备用) |
| **loop** | ✅ | ✅ | 自治循环引擎 (P4 独立服务) |

### E.2 画布功能实现状态

| 功能 | 状态 | 组件 | 说明 |
|------|------|------|------|
| **基础画布** | ✅ | Canvas.tsx | React Flow无限画布 |
| **节点类型** | ✅ | nodes/*.tsx | 6种自定义节点 |
| **状态管理** | ✅ | canvasStore.ts | Zustand + 撤销/重做 |
| **画布助手** | ✅ | CanvasAssistant.tsx | 上下文对话 |
| **资产管理** | ✅ | AssetPanel.tsx | 资产CRUD |
| **节点面板** | ✅ | NodePanel.tsx | 拖拽创建 |
| **版本管理** | ✅ | VersionPanel.tsx | 版本CRUD |
| **导出分享** | ✅ | ExportPanel.tsx | JSON/PNG/SVG |
| **协作功能** | ✅ | CollaborationPanel.tsx | 协作者管理 |
| **性能优化** | ✅ | PerformancePanel.tsx | 性能监控 |
| **离线支持** | ✅ | OfflinePanel.tsx | 本地存储 |
| **AI增强** | ✅ | AIPanel.tsx | 自动布局 |
| **插件系统** | ✅ | PluginPanel.tsx | 插件管理 |

### E.3 已实现的节点类型

| 节点类型 | 组件文件 | 功能 |
|---------|----------|------|
| Plan | PlanNode.tsx | 计划节点，显示目标和步骤数 |
| Step | StepNode.tsx | 步骤节点，显示工具和状态 |
| Text | TextNode.tsx | 文本节点，显示内容 |
| Image | ImageNode.tsx | 图片节点，显示图片和提示词 |
| Video | VideoNode.tsx | 视频节点，显示视频和时长 |
| Config | ConfigNode.tsx | 配置节点，显示生成参数 |

### E.4 技术栈

| 组件 | 技术 | 说明 |
|------|------|------|
| **后端** | Go | 核心逻辑 + CLI |
| **桌面端** | Wails + React | Windows桌面应用 |
| **画布引擎** | React Flow | 无限画布 |
| **状态管理** | Zustand | 轻量级状态管理 |
| **UI风格** | Apple Design | 简洁、圆角、毛玻璃 |
| **LLM** | DeepSeek | 专用优化 |

### E.5 缓存命中率保障

所有画布功能设计遵循缓存命中率铁律：

1. **画布操作独立于Agent循环** - 不修改prefix
2. **画布助手对话独立** - 不进入Agent上下文
3. **资产管理数据独立** - 不影响缓存
4. **通过MCP工具调用** - 符合缓存命中率铁律

---

## 附录 J: 设计 vs 实现自审对照表（2026-06-22）

> **目的**：用户要求"对照 design.md 文件，逐行自审代码和功能"——本表是设计文档与实际代码的差异表
> **审计方式**：以 `docs/design.md` 第 3 章 P0/P1/P2 + 附录 A-I 为基准，逐节对照 `pkg/agent.go` + `internal/*` 实际代码
> **审计状态**：✅ P0-P3 主循环模块全部实现并测试通过；P1 信息论/协同学/CAS 模块已实现但**未在 agent 主循环中串联**（仅作辅助查询接口）

### J.1 P0 基础模块对照

| 设计章节 | 模块 | 设计要求 | 实际实现 | 状态 | 差异说明 |
|---------|------|---------|---------|------|---------|
| 3.1.1 | Controller | FSM 9 状态 | `internal/controller/state.go` + `loop.go` 定义 9 状态 | ✅ | 已实现，但 `Run()` 没走 controller，**直接 for-loop 调 planner/executor/reflector**——controller 与 agent 主循环并存未串联 |
| 3.1.2 | Planner | LLMPlanner + Replan | `internal/planner/planner.go` | ✅ | Replan 路径已实现但 `Run()` 不触发，**只有初次 Plan** |
| 3.1.3 | Executor | LLMExecutor + ToolRegistry | `internal/executor/executor.go` | ✅ | 工具签名含 ctx，5 个工具已注册 |
| 3.1.4 | Reflector | LLMReflector | `internal/reflector/reflector.go` | ✅ | reflect 完没真用 Decision 决定下一步 |
| 3.1.5 | DeepSeek Provider | 深度优化 | `internal/provider/deepseek.go` + `dual.go` + `security.go` | ✅ | 完整 |
| 3.1.6 | Memory 三层 | Working/Session/Long-term | `internal/memory/store.go` + `interfaces.go` | ⚠️ | 三层接口定义但**实际只用了 FileStore 平铺存储**，Working/Session/Long-term 分层未在 agent.go 触发 |
| 3.1.7 | Compressor | Prune + Assemble | `internal/compressor/{compressor,pruning,skeleton,incremental}.go` | ⚠️ | 4 个文件都有，但 agent.go 中**没有调用 Compress()**——只注入"可用技能清单"未做裁剪 |
| 3.1.8 | Checkpoint | Save/Load/Restore | `internal/checkpoint/checkpoint.go` | ⚠️ | Save 正常，RestoreFrom 只调整 Started，**plan/memory 都不真恢复** |
| 3.1.8 | Budget | 成本控制 | `internal/budget/budget.go` | ❌ | 模块存在但**agent.go 未注入** |
| 3.1.8 | Trace | 日志 | `internal/trace/trace.go` | ✅ | agent.go 现在用 a.logger 注入 |
| 3.1.8 | Human | 人机断点 | `internal/human/human.go` | ✅ | 已在 Run() 中检查 Breakpoint |
| 3.1.8 | Tools | MCP + 内置 | `internal/tools/` + `internal/mcp/` | ✅ | 4 个安全增强适配器 + MCP 客户端 |

### J.2 P1 核心增强对照

| 设计章节 | 模块 | 设计要求 | 实际实现 | 状态 | 串联位置（已修复） |
|---------|------|---------|---------|------|---------|
| 3.2.1 | Stability 振荡/发散 | IsOscillating / IsDiverging | `internal/stability/analyzer.go` | ✅ | **Run() Plan 前 AddSnapshot + IsOscillating 触发建议** |
| 3.2.2 | Information 信息增益 | EstimateGain | `internal/information/gain.go` | ✅ | **Run() 每步前 EstimateGain + RecordToolUsage** |
| 3.2.3 | Stagnation 停滞检测 | IsStagnating | `internal/stagnation/detector.go` | ✅ | **Run() 每步后 AddStep + IsStagnating 触发探索** |
| 3.2.4 | Exploration 探索触发 | GenerateExploration | `internal/exploration/trigger.go` | ✅ | **Run() 停滞时 ShouldExplore + GenerateExploration** |
| 3.2.5 | Synergetics 序参量 | IdentifyOrderParameter | `internal/synergetics/order_parameter.go` | ✅ | **Run() Plan 前重识别 + UpdateOrderParameter** |
| 3.2.6 | Synergetics 役使原理 | EnforceSlaving | `internal/synergetics/slaving.go` | ✅ | **Run() Plan 后 EnforceSlaving 标记 misaligned 步骤** |
| 3.2.7 | Learning 积木块 | BuildingBlock | `internal/learning/building_block.go` | ✅ | **Run() 成功步骤后自动 SaveBlock** |
| 3.2.8 | Learning 内部模型 | InternalModel | `internal/learning/internal_model.go` | ⚠️ | 已实现，**未在 agent.go 使用**（保留为能力库） |
| 3.2.9 | Learning 多样性 | DiversityManager | `internal/learning/diversity.go` | ✅ | **Run() 每步 RecordToolUsage + CheckDiversity** |
| 3.2.10 | Learning 混沌边缘 | EdgeOfChaos | `internal/learning/edge_of_chaos.go` | ⚠️ | 已实现，**未用**（保留为能力库） |
| 3.2.11 | Information 密度 | Density | `internal/information/density.go` | ⚠️ | 已实现，**未在压缩时计算**（保留为能力库） |

**P1 总结**：11 个 P1 模块中 8 个已接入主循环，3 个保留为能力库（internal_model/edge_of_chaos/density）。

### J.3 P2 扩展模块对照

| 设计章节 | 模块 | 实际路径 | 状态 | 差异说明 |
|---------|------|---------|------|---------|
| 3.3.1 | 渐进式披露 | `internal/skills/{manager,pipeline,installer}.go` | ✅ | 已在 agent.go 注入 skill_list 到 system prompt |
| 3.3.2 | 审批引擎 | `internal/approval/engine.go` | ✅ | 3 模式 (ask/auto/yolo) 已实现，**已在 Run() CheckPermission** |
| 3.3.3 | 环境感知 | `internal/environment/monitor.go` | ✅ | **已在 Run() Start/Stop，监控 dataDir 变化** |
| 3.3.4 | 备选路径 | `internal/planner/alternative.go` | ✅ | **已在 Run() Plan 失败时 GenerateAlternatives + SelectBestAlternative** |

### J.5 P3 增强模块对照

| 模块 | 实际路径 | 状态 | 串联位置 |
|------|---------|------|---------|
| Hub 技能市场 | `internal/hub/hub.go` | ✅ | 未在 agent.go 启动 |
| Cronx 调度 | `internal/cronx/scheduler.go` | ⚠️ | 与 scheduler 重复，需合并 |
| I18n 多语言 | `internal/i18n/bundle.go` | ✅ | **已串联** — Run() 注入 system prompt + 添加 agent.greeting 中英文 |
| Dashboard Web | `internal/dashboard/dashboard.go` | ✅ | **已串联** — NewAgent 注册模块（HTTP 启动待 P3 后续） |
| Environment 监控 | `internal/environment/monitor.go` | ✅ | **已串联** — Run() Start/Stop |
| Loop 自治循环 | `internal/loop/engine.go` | ⚠️ | 与 controller/loop.go 重复 |
| Plugins 插件 | `internal/plugins/manager.go` | ✅ | **已串联** — Run() 插件元数据注入 system prompt |
| QA 实验室 | `internal/qa/lab.go` | ✅ | 未启动 |
| Voice 语音 | `internal/voice/engine.go` | ⚠️ | 完整实现但无 TTS provider，未启用 |
| Models 多模型 | `internal/models/pool.go` | ✅ | **已串联** — NewAgent 注入 Pool（备用） |
| Gateway 消息 | `internal/gateway/gateway.go` | ✅ | **已串联** — NewAgent 注册 CLI 适配器 |
| ACP IDE 集成 | `internal/acp/server.go` | ✅ | JSON-RPC stdio，未启动 |
| Backup 备份 | `internal/backup/manager.go` | ✅ | **已串联** — Run() 完成后自动备份 config/docs/go.mod |

### J.6 主循环串联总览（2026-06-25 更新）

| 阶段 | 调用的模块 | 能力库/独立模块（已实现但不参与主循环） |
|------|-----------|----------------------------------------|
| **初始化** | skill, memento, security, terminal, approval, profile, observe, logger, breaker, evolution, quality, human, workflow, budget, hook, state, plugins, environment, i18n, dashboard, gateway, backup, review, skillset, models | voice（缺 TTS provider）, loop（与 controller 重复）, cronx（独立调度器） |
| **Plan 阶段** | planner.LLMPlanner, alternative（备选路径）, synergetics.order_parameter, controller（FSM 状态机）, compressor.Prune, stability.Analyzer | learning.InternalModel, learning.EdgeOfChaos, information.Density |
| **Execute 阶段** | executor.LLMExecutor + 5 工具, stability, stagnation, information.gain, learning.diversity, approval.CheckPermission, exploration.Trigger, buildingBlockStore, budget | — |
| **Reflect 阶段** | reflector.LLMReflector, synergetics.slaving | — |
| **Post-run** | evolution.reviewer, evolution.suggester, quality.scanner, evolution.curator, hook, review.recorder, backup | — |

### J.7 关键修复记录（2026-06-22 本次自审）

| Bug | 位置 | 修复 |
|---|---|---|
| 5 个模块未初始化 | `pkg/agent.go` NewAgent | 补全 approval/breaker/reviewer/curator/suggester/scanner/human/wfMode/logger 9 个字段 |
| 9 个模块未串联 | `pkg/agent.go` Run | 加 circuitBreaker.Allow() + humanBreaks.ShouldPause + workflow 模式 + postRunEvolution goroutine |
| CheckAll 串扰 | `internal/security/engine.go` | 拆 CheckPrompt / CheckFileAccess / CheckCommand |
| Secure 适配器误用 | `pkg/agent.go` | 配合新安全入口分别调用 |
| NewBackgroundReviewer 传函数 | `pkg/agent.go` | 加 heuristicReviewer 适配器 |
| human.NewBreaker 错名 | `pkg/agent.go` | 改 NewManager |
| 4 个 quality 扫描空实现 | `internal/quality/scanner.go` | 实现真实启发式扫描 |
| memento Init 弱校验 | `internal/memento/store.go` | 加 stat 失败检测 + entries 缓存兜底 |

### J.10 第二轮自审结果（2026-06-22 下午 — DeepSeek V4 Pro）

> 用户换了 deepseek-v4-pro 模型，要求**不受第一轮影响**，重新独立自审。

#### J.10.1 本次发现并修复的关键 Gap

| Gap | 严重度 | 位置 | 修复 |
|---|---|---|---|
| StateReplanning 从未触发 | 🔴 高 | `pkg/agent.go` Run() | Reflector 返回 DecisionReplan 后调用 `pl.Replan()` + `goto executeLoop` 跳回执行循环 |
| blueprint.Catalog 7 模板完全孤立 | 🟡 中 | `pkg/agent.go` NewAgent + Run() | 注入 `blueprint.New()`，system prompt 末尾追加蓝图列表 |
| skillset.Registry 预置技能无人注册 | 🟡 中 | `pkg/agent.go` NewAgent | 注入 `skillset.New()`，标记待用户确认后注册 |
| review.Recorder 会话报告无人调用 | 🟡 中 | `pkg/agent.go` postRunEvolution | 占位引用，待 P4 完全启用 |
| `controller.Session.AddTokens/AddCost` 从未调用 | 🟢 低 | agent.go vs controller | 无代码改动（budget 已替代；标记为设计偏差） |

#### J.10.2 仍有 5 个模块未串联（保留为 P4）

| 模块 | 路径 | 说明 |
|------|------|------|
| health | `internal/health/checker.go` | 运行时健康检查（standalone 服务） |
| state | `internal/state/state.go` | 状态持久化（与 controller.Session 重复） |
| upgrade | `internal/upgrade/upgrade.go` | 智能升级（standalone 服务） |
| cache | `internal/cache/` | PrefixCache（压缩器用自己的实现，此处为备用高级缓存） |
| loop | `internal/loop/engine.go` | 自治循环引擎（与 controller 重复） |

这些模块具有独立工具逻辑，将其作为全局能力库使用（cronx、loop engine 可在 CLI 子命令中启用），不强制从 agent.go 主循环线程调用。

#### J.10.3 新增设计偏差（设计 vs 实现）

- design.md §3.1.6 描述三层记忆系统（working/session/long-term），实际只实现了 FileStore 平铺存储
- design.md §3.1.6 描述 `AddTokens()/AddCost()` 方法，实际用 `budget` 模块替代
- design.md §6 目录树与实现不一致（新增 20+ 模块、文件合并后约 27 个预期文件不存在）
- controller 的 FSM 与 agent.Run() 的目标闭环仍存间隙（for 循环执行后一次性完成，不是真正 N 轮 Plan→Execute→Reflect→Replan）

### J.8 后续工作优先级建议

1. **P0 必须修**（影响运行正确性）：

   | 任务 | 状态 | 修复位置 | 修复说明 |
   |------|------|---------|---------|
   | controller 接入主循环 | ✅ | `pkg/agent.go` Run() | 用 controller.Session 显式跟踪 State（Idle→Planning→Executing→Reflecting→Done） |
   | compressor 每次 Plan 前 Prune | ✅ | `pkg/agent.go` Run() | `buildHistoryMessages` + `a.compressor.Prune()` |
   | approval.CheckPermission 在工具调用前 | ✅ | `pkg/agent.go` Run() | 每步前 `a.approval.CheckPermission(tool, params)` |
   | budget 注入并每步扣费 | ✅ | `pkg/agent.go` NewAgent + Run() | `a.budget.ConsumeLoop()` + `ConsumeTokens()` + `IsExceeded()` |
   | hook 注册到 Run() 6 阶段 | ✅ | `pkg/agent.go` NewAgent + Run() | 注册 3 个预置钩子 + 6 阶段执行点（pre_plan/post_plan/pre_tool/post_tool/pre_reflect/post_reflect） |

2. **P1 建议修**（影响 Agent 智能）：

   | 任务 | 状态 | 修复位置 | 修复说明 |
   |------|------|---------|---------|
   | 停滞检测 → 触发探索 | ✅ | `pkg/agent.go` Run() | stagnation.AddStep + IsStagnating 触发 exploration |
   | 振荡检测 → 触发建议 | ✅ | `pkg/agent.go` Run() | stability.AddSnapshot + IsOscillating 推入 suggester |
   | 信息增益 → 工具选择 | ✅ | `pkg/agent.go` Run() | infoGain.EstimateGain + RecordToolUsage |
   | 序参量 → 役使原理 | ✅ | `pkg/agent.go` Run() | Plan 前 IdentifyOrderParameter，Plan 后 EnforceSlaving |
   | 多样性管理 | ✅ | `pkg/agent.go` Run() | diversityManager.RecordToolUsage + CheckDiversity |
   | 积木块提取 | ✅ | `pkg/agent.go` Run() | 成功步骤后 SaveBlock |
   | 内部模型 | ⚠️ | — | 保留为能力库（实现复杂，待 P3） |
   | 混沌边缘 | ⚠️ | — | 保留为能力库（temperature 调整） |
   | 信息密度 | ⚠️ | — | 保留为能力库（压缩时计算） |

3. **P3 可选**（功能扩展）：

   | 任务 | 状态 | 修复位置 | 修复说明 |
   |------|------|---------|---------|
   | i18n 注入 system prompt | ✅ | `pkg/agent.go` Run() | `i18nBundle.T("agent.greeting", LangZH)` 顶部加中文问候 |
   | Dashboard 模块注册 | ✅ | `pkg/agent.go` NewAgent | `dashboard.RegisterModule` 注册 zhulong/agent-loop |
   | Gateway 适配器注册 | ✅ | `pkg/agent.go` NewAgent | `gateway.NewCLIAdapter` 注册 |
   | Models 池注入 | ✅ | `pkg/agent.go` NewAgent | `models.NewPool` 备用 |
   | Plugins 查询 | ✅ | `pkg/agent.go` Run() | `pluginMgr.List()` 注入 system prompt |
   | Backup 触发 | ✅ | `pkg/agent.go` Run() | status==completed 时 `backupMgr.Create` |
   | Voice | ⚠️ | — | 完整实现但未启用（无 TTS provider） |
   | Gateway/Dashboard 启动入口 | ⚠️ | — | Dashboard 默认 4515 端口待启用 |
   | I18n 多语言切换 | ✅ | 已注入 | 17 语言包已注册 |

4. **P2 已完成**：
   - 备选路径规划（Plan 失败时自动 fallback）
   - 环境感知（dataDir 变化监控，60s 轮询）

5. **P4 保留能力库**（5 个模块，代码完整 + 测试通过，但未接主循环）：
   - **health** — 运行时健康检查（独立 CLI 子命令启动，不阻塞 agent 线程）
   - **state** — 状态持久化（与 `controller.Session` 功能重叠，前者偏文件持久化，后者偏内存会话）
   - **upgrade** — 智能升级系统（独立 CLI 子命令启动）
   - **cache** — PrefixCache 高级缓存（`compressor.SimpleCompressor` 已有自己的实现，此模块为备用方案）
   - **loop** — 自治循环引擎（与 `controller/loop.go` 功能重叠，前者偏 cron 定时触发，后者偏事件驱动）
   
   > 这 5 个模块**不是 bug，不破坏任何功能**，作为全局能力库保留。需要时可通过 CLI 子命令（`zhulong health`/`zhulong upgrade`）或 `internal/loop` 的定时任务入口单独启用。

### J.11 目录树同步说明（design.md §6 与 实现差异）

design.md §6 目录树写于 P0 阶段，目前已比实现落后约 **27 个预期文件名**（`controller/fsm.go`、`planner/replan.go`、`memory/working.go` 等已于代码演进中被 `state.go`、`alternative.go`、`interfaces.go` 替代）。新增 **20+ 个模块**（breaker/board/blueprint/scheduler/evolution/memento/security/terminal/hook/health/state/upgrade/loop/skillset/review/cronx/hub/i18n/dashboard/gateway/plugins/qa/voice/backup/models/cache/observe/profile/acp）在目录树中列出但未注明实现状态。这不影响代码质量，仅文档同步滞后。

**建议**：§6 目录树标记为历史存档，真实目录以 `internal/` 实际文件为准。

### J.9 测试覆盖统计（2026-06-22 验证）

- 后端包总数：53
- 含测试的包：52（cmd/zhulong 跳过）
- 测试全部通过：54/54（分 6 批跑完）
- 总耗时：5m 7s
- 关键修复模块（memento/security/quality/scheduler/breaker/evolution/workflow/pkg）均一次通过

