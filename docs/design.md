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

### 1.3 参考项目

> **原则：学习设计思路，从零实现。不 fork、不复制、不引入外部 License 依赖。**

| 项目 | 学习内容 |
|------|---------|
| [DeepSeek-Reasonix](https://github.com/esengine/DeepSeek-Reasonix) | MCP 工具协议规范、确定性工具结果裁剪策略、prefix-cache 稳定性设计思路 |
| [Ailoom-Context](https://github.com/EvanLyu-oss/Ailoom-Context) | 骨架压缩结构设计（skeleton + restore 分离）、焦点模式语义、增量压缩思路 |
| [NB-Agent](https://github.com/ydf0509/nb_agent) | 渐进式披露机制、审批引擎设计 |

所有参考项目均选择独立实现，确保零外部依赖。

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

#### 3.3.1 渐进式披露（借鉴 NB-Agent）

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

#### 3.3.2 审批引擎（借鉴 NB-Agent + Reasonix）

**问题**：某些工具调用可能有风险，需要人工确认。但有些场景用户完全信任 Agent，不想被打断。

**解决方案**：审批引擎 + 三种执行模式。

**可行性**：✅ 只控制工具执行，不修改上下文。与 Human 模块整合。

**三种执行模式**（参考 Reasonix）：

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
├── internal/
│   ├── controller/                 # P0: 状态机 + 循环控制
│   │   ├── fsm.go
│   │   ├── loop.go
│   │   └── session.go
│   │
│   ├── planner/                    # P0: 规划器
│   │   ├── planner.go
│   │   ├── prompts.go
│   │   ├── parser.go
│   │   └── replan.go
│   │
│   ├── executor/                   # P0: 执行器
│   │   ├── executor.go
│   │   ├── scheduler.go
│   │   └── retry.go
│   │
│   ├── reflector/                  # P0: 反省器
│   │   ├── reflector.go
│   │   ├── prompts.go
│   │   └── assessment.go
│   │
│   ├── stability/                  # P1: 稳定性分析器
│   │   └── analyzer.go
│   │
│   ├── information/                # P1: 信息论模块
│   │   ├── gain.go
│   │   └── density.go
│   │
│   ├── stagnation/                 # P1: 停滞检测器
│   │   └── detector.go
│   │
│   ├── exploration/                # P1: 探索触发器
│   │   └── trigger.go
│   │
│   ├── synergetics/                # P1: 协同学模块
│   │   ├── order_parameter.go
│   │   └── slaving.go
│   │
│   ├── learning/                   # P1: 自适应学习模块
│   │   ├── building_block.go
│   │   ├── internal_model.go
│   │   ├── diversity.go
│   │   └── edge_of_chaos.go
│   │
│   ├── skills/                     # P2: 渐进式披露
│   │   └── manager.go
│   │
│   ├── approval/                   # P2: 审批引擎
│   │   └── engine.go
│   │
│   ├── environment/                # P2: 环境感知器
│   │   └── monitor.go
│   │
│   ├── memory/                     # P0: 三层记忆系统
│   │   ├── interfaces.go
│   │   ├── working.go
│   │   ├── session.go
│   │   ├── longterm.go
│   │   └── store.go
│   │
│   ├── compressor/                 # P0: 上下文压缩
│   │   ├── compressor.go
│   │   ├── prune.go
│   │   ├── skeleton.go
│   │   ├── assembler.go
│   │   └── tokenizer.go
│   │
│   ├── checkpoint/                 # P0: 检查点持久化
│   │   ├── checkpoint.go
│   │   ├── store.go
│   │   └── restore.go
│   │
│   ├── budget/                     # P0: 成本控制
│   │   ├── budget.go
│   │   └── pricing.go
│   │
│   ├── trace/                      # P0: 可观测性
│   │   ├── logger.go
│   │   ├── formatter.go
│   │   └── export.go
│   │
│   ├── human/                      # P0: 人机协作
│   │   ├── breakpoint.go
│   │   ├── terminal.go
│   │   └── input.go
│   │
│   ├── tools/                      # P0: 工具抽象层
│   │   ├── interface.go
│   │   ├── mcp.go
│   │   ├── builtin.go
│   │   └── custom.go
│   │
│   └── provider/                   # P0: DeepSeek Provider
│       ├── interface.go
│       ├── deepseek.go
│       └── cache.go
│
├── pkg/                            # 对外公共 API（CLI 和桌面端共享）
│   ├── agent.go                    # Agent 构造函数 + Run()
│   ├── options.go                  # 函数式选项
│   └── types.go                    # 公共类型导出
│
├── desktop/                        # Windows 桌面端（Wails + React）
│   ├── main.go
│   ├── app.go
│   ├── wails.json
│   └── frontend/
│       ├── src/
│       │   ├── App.tsx
│       │   ├── components/
│       │   ├── styles/
│       │   └── main.tsx
│       ├── package.json
│       └── vite.config.ts
│
├── config/
│   ├── default.yaml
│   └── schema.json
│
├── examples/
│   ├── coding/main.go
│   ├── writing/main.go
│   └── analysis/main.go
│
├── testing/
│   ├── unit/
│   ├── integration/
│   └── benchmark/
│
├── docs/
│   ├── design.md
│   ├── api.md
│   └── architecture.md
│
├── go.mod
├── go.sum
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
- [ ] 桌面端打包（Windows 安装包）

---

## 8. 无限画布可视化设计

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

> **设计策略**：学习 TapCanvas 和 Toonflow 的设计思路，从零实现。不 fork、不复制、不引入外部 License 依赖。

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

参考 infinite-canvas 的本地Agent集成方式，烛龙画布将通过MCP协议与Agent内核通信，实现：

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

参考 TapCanvas 和 infinite-canvas 的节点设计，烛龙画布支持以下多媒体节点类型：

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
| **Text** | 文本生成、提示词输入、多轮对话 | TapCanvas、infinite-canvas |
| **Image** | 图片生成、图生图、参考图编辑 | TapCanvas、infinite-canvas |
| **Video** | 视频生成、首帧/尾帧控制、时长设置 | TapCanvas、infinite-canvas |
| **Audio** | TTS语音生成、语音选择、语速控制 | infinite-canvas |
| **Storyboard** | 分镜编辑、场景描述、镜头设置 | TapCanvas、Toonflow |
| **Config** | 生成配置、模型选择、参数设置 | infinite-canvas |

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

参考 infinite-canvas 的三段式设计，烛龙画布支持以下工作流模式：

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

参考 infinite-canvas 的 @引用设计，支持在提示词中引用上游节点内容：

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

参考 TapCanvas 和 Toonflow 的资产管理设计：

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

参考 TapCanvas 的项目化资产管理：

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

参考 Toonflow 的衍生资产设计：

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

参考 infinite-canvas 的画布助手设计：

| 功能 | 说明 | 参考项目 |
|------|------|----------|
| **上下文对话** | 围绕选中节点进行对话 | infinite-canvas |
| **选中节点引用** | 选中节点自动作为上下文 | infinite-canvas |
| **多轮对话** | 支持多轮对话历史 | infinite-canvas |
| **画布快照** | 每次请求带上画布JSON快照 | infinite-canvas |
| **@资源引用** | 对话中@引用画布资源 | infinite-canvas |

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

参考 infinite-canvas 的 Ops 抽象设计：

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

参考 TapCanvas 的多Agent协作设计：

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
| **TapCanvas** | React Flow集成、节点设计、布局算法、资产管理、DAG工作流 | 学习设计思路，自行实现 |
| **Toonflow** | 三层Agent架构、状态同步、可视化调试、衍生资产、章节事件图谱 | 学习架构思路，自行实现 |
| **infinite-canvas** | MCP协议集成、画布助手、上下文对话、三段式生成、Ops抽象、@引用机制 | 学习设计思路，自行实现 |
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

## 附录 A: 与 Reasonix 的学习参考清单

> **策略：学习设计思路，从零实现。不 fork、不复制、不引入外部 License 依赖。**

| 模块 | Reasonix 路径 | 学习内容 | Zhulong 实现方式 |
|------|--------------|---------|-----------------|
| MCP 工具协议 | `internal/mcp/` | MCP 协议规范和消息格式 | 按 MCP 规范自行实现 |
| 工具结果裁剪 | `internal/session/pruning/` | 裁剪策略：保留调用签名，裁剪可重新获取的输出 | 自行实现 |
| Prefix-cache 配置 | `internal/llm/config.go` | prefix 稳定性设计思路 | 自行设计 |
| DeepSeek API 调用 | `internal/llm/deepseek.go` | API 调用格式和错误处理 | 按 DeepSeek API 文档自行实现 |

## 附录 B: 与 Ailoom-Context 的学习参考清单

| 模块 | Ailoom 路径 | 学习内容 | Zhulong 实现方式 |
|------|------------|---------|-----------------|
| 骨架生成策略 | `ailoom_core/compress.py` | 骨架结构设计思路 | 自行设计骨架格式 |
| 焦点模式 | `ailoom_core/focus_modes/` | 各焦点模式的语义定义 | 自行定义焦点模式 |
| 增量压缩 | `ailoom_core/incremental.py` | 增量 diff 策略思路 | 自行实现增量更新 |
| 项目扫描 | `ailoom_core/scan.py` | 目录扫描和过滤策略 | Go 标准库实现 |

## 附录 C: 与 NB-Agent 的学习参考清单

| 模块 | NB-Agent 路径 | 学习内容 | Zhulong 实现方式 |
|------|--------------|---------|-----------------|
| 渐进式披露 | `nb_agent/skills/` | Discovery → Activation → Execution 三阶段 | 自行实现 |
| 审批引擎 | `nb_agent/approval/` | 三级权限（deny > ask > allow） | 自行实现 |
| 上下文裁剪 | `nb_agent/core/context.py` | 根据模型 context_limit 裁剪历史 | 自行实现 |

## 附录 D: 与无限画布项目的学习参考清单

> **策略：学习设计思路，从零实现。不 fork、不复制、不引入外部 License 依赖。**
> **注意：hero8152/Infinite-Canvas 禁止商用，仅学习设计思路，不使用代码。**

### 深度研究总结

经过对三个项目的深度代码研究，总结以下关键学习点：

| 项目 | 学习内容 | Zhulong 实现方式 |
|------|----------|------------------|
| **TapCanvas** | React Flow集成、多媒体节点类型（text/image/video/storyboard）、资产管理系统（项目化资产沉淀）、DAG工作流、多Agent协作 | 学习设计思路，自行实现 |
| **Toonflow** | 三层Agent架构（决策/执行/监督）、衍生资产系统、章节事件图谱、状态同步、可视化调试 | 学习架构思路，自行实现 |
| **basketikun/infinite-canvas** | MCP协议集成、画布助手（上下文对话）、三段式生成流程（提示词→配置→结果）、@引用机制、Ops操作抽象、本地Agent集成 | 学习设计思路，自行实现 |
| **hero8152/Infinite-Canvas** | 多模型支持、扩展功能设计（仅学习思路） | 仅学习思路，不使用代码 |

### 关键学习点

#### 1. 多媒体节点类型（参考 TapCanvas、infinite-canvas）

```typescript
// TapCanvas 节点类型
type TaskNodeKind = 'text' | 'video' | 'image' | 'imageEdit' | 'storyboard'

// infinite-canvas 节点类型
enum CanvasNodeType {
    Image = "image",
    Text = "text",
    Config = "config",
    Video = "video",
    Audio = "audio",
}
```

**学习要点**：
- 每种节点类型有独立的数据结构和特性
- 节点支持多种功能特性（prompt、image、video、storyboard等）
- 节点通过句柄（handle）进行连线

#### 2. 连续工作流模式（参考 infinite-canvas）

```
[文本节点(提示词)] --连接--> [Config节点(生成配置)] --生成--> [结果节点(图片/视频/音频)]
[参考节点] ---连接------/
```

**学习要点**：
- 三段式生成流程：提示词→配置→结果
- `@[node:nodeId]` 语法引用上游节点内容
- 连线追踪上游节点构建生成上下文

#### 3. 资产管理系统（参考 TapCanvas、Toonflow）

**TapCanvas 资产类型**：
- generation（AI生成资产）
- novelDoc（小说文档）
- scriptDoc（剧本脚本）
- storyboardScript（分镜脚本）

**Toonflow 衍生资产系统**：
- role（角色资产）
- tool（道具资产）
- scene（场景资产）
- clip（视频片段）

**学习要点**：
- 项目化资产沉淀，资产与项目关联
- 衍生资产系统，每个资产可以有多个变体
- 资产状态管理：未生成、生成中、已完成、生成失败

#### 4. 画布助手（参考 infinite-canvas）

```typescript
// 上下文构建
async function buildToolAgentMessages(snapshot, history, userMessage) {
    return [
        { role: "system", content: ONLINE_AGENT_PROMPT },
        ...history.slice(-8),  // 最近8条历史
        {
            role: "user",
            content: [
                // 选中节点的文本内容
                ...refs.filter(item => item.text).map(item => ({
                    type: "text",
                    text: `选中节点 ${item.title}：${item.text}`
                })),
                // 当前画布JSON快照
                { type: "text", text: `当前画布：${JSON.stringify(snapshot)}` },
                // 选中节点的图片
                ...refs.filter(item => item.dataUrl).map(item => ({
                    type: "image_url",
                    image_url: { url: item.dataUrl }
                })),
            ],
        },
    ];
}
```

**学习要点**：
- 每次请求带上当前画布完整JSON快照
- 选中节点作为参考上下文自动附加
- 支持@引用画布上的资源节点
- 支持多轮对话，最近8条历史作为上下文

#### 5. Ops操作抽象（参考 infinite-canvas）

```typescript
type CanvasAgentOp =
    | { type: "add_node"; nodeType?; position?; metadata?; }
    | { type: "update_node"; id; patch?; metadata? }
    | { type: "delete_node"; id?; ids?; nodeType? }
    | { type: "connect_nodes"; fromNodeId; toNodeId }
    | { type: "run_generation"; nodeId; mode?; prompt? }
    | { type: "set_viewport"; viewport }
    | { type: "select_nodes"; ids };
```

**学习要点**：
- 所有操作统一为Ops接口
- 用户手动操作、Agent调用、MCP工具调用都通过同一套Ops执行
- 支持撤销/重做（历史记录）
- 支持Agent确认（写操作需用户批准）
- 支持快照对比

#### 6. MCP协议集成（参考 infinite-canvas）

```
浏览器网页 --SSE/HTTP--> Canvas Agent (本地) --stdio--> Codex/Claude Code
```

**学习要点**：
- 画布作为MCP客户端，Agent作为MCP服务器
- MCP工具调用通过HTTP转发到浏览器执行
- 支持工具确认机制（写操作需用户批准）
- 支持SSE事件流实时推送

#### 7. 多Agent协作（参考 TapCanvas）

```typescript
interface CollabAgentManager {
    spawn(options: SpawnOptions): { agentId: string; submissionId: string }
    close(id: string): string
    enqueue(id: string, prompt: string): { submissionId: string }
    sendMailboxMessage(input: MailboxMessageInput): PersistedMailboxMessage
    requestProtocol(input: ProtocolRequestInput): PersistedProtocolRequest
}
```

**学习要点**：
- 团队管理：支持创建子Agent
- 任务队列：任务提交和状态追踪
- 消息传递：邮箱机制和协议请求
- 工作空间协作：文件移交和工作空间导入

#### 8. 章节事件图谱（参考 Toonflow）

```typescript
// 获取章节事件
get_novel_events: tool({
    description: "获取章节事件",
    execute: async ({ chapterIndexes }) => {
        const data = await u.db("o_novel")
            .where("projectId", resTool.data.projectId)
            .whereIn("chapterIndex", chapterIndexes)
            .select("id", "chapterIndex as index", "event");
        return data.map(i => `第${i.index}章:\n${i.event}`).join("\n\n");
    },
}),
```

**学习要点**：
- 自动提取章节事件并结构化存储
- 剧本改编时按事件图谱精准调用上下文
- 解决长文本改编的信息丢失问题
- Agent作为MCP服务器暴露工具和状态
- 标准化通信，便于扩展

#### 2. 画布助手（参考 infinite-canvas）

- 上下文对话：围绕选中节点进行对话
- 结果回流：生成结果直接插入画布
- 视觉上下文驱动的AI创作

#### 3. 三层Agent架构（参考 Toonflow）

```
决策层 (Planner) → 执行层 (Executor) → 监督层 (Reflector)
```

- 明确的职责划分
- 模块化设计
- 便于扩展和维护

#### 4. 状态同步（参考 TapCanvas、Toonflow）

- 后端事件驱动
- 前端实时订阅
- 双向同步机制

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

## 附录 F: 与 Harness-Starter 的学习参考清单

> **策略：学习设计思路，从零实现。不 fork、不复制、不引入外部 License 依赖。**

### F.1 项目概述

Harness-Starter 是一个为 Claude Code 设计的**工程化模板系统**，将开发规范、检查流程和自动化任务固化为可复用的模板。其核心理念是**"系统驱动AI，而非人驱动AI"**。

### F.2 核心参考特性

| 特性 | Harness-Starter 实现 | 烛龙实现方式 | 优先级 |
|------|---------------------|-------------|--------|
| **三层自动化体系** | 安全拦截→感知注入→审查反馈 | 扩展现有approval/memory/reflector | P1 |
| **Circuit Breaker** | 连续3次无改善自动暂停 | 新增stability/breaker.go | P0 |
| **执行/验证分离** | 独立verify-goal技能 | 扩展reflector模块 | P0 |
| **GC扫描器** | 8个确定性维度质量扫描 | 新增quality/scanner.go | P1 |
| **工作流模式** | full/hotfix/tweak模式切换 | 新增workflow/modes.go | P1 |
| **上下文自动注入** | 自动注入Git状态、技术栈 | 扩展memory系统 | P2 |
| **工具执行后审查** | 自动格式化和代码审查 | 扩展executor模块 | P2 |

### F.3 Circuit Breaker 设计

参考 Harness-Starter 的 Circuit Breaker 机制，防止无效循环：

```go
// BreakerConfig 配置断路器
type BreakerConfig struct {
    MaxConsecutiveFailures int           // 最大连续失败次数（默认3）
    CooldownPeriod         time.Duration // 冷却期（默认5分钟）
    CheckInterval          time.Duration // 检查间隔（默认1分钟）
}

// Breaker 断路器
type Breaker struct {
    config           *BreakerConfig
    consecutiveFails int
    lastFailTime     time.Time
    state            BreakerState // closed/open/half-open
}

// Check 检查是否应该继续
func (b *Breaker) Check() bool {
    if b.state == BreakerOpen {
        if time.Since(b.lastFailTime) > b.config.CooldownPeriod {
            b.state = BreakerHalfOpen
            return true // 允许一次尝试
        }
        return false // 仍在冷却期
    }
    return true // closed状态，正常执行
}

// RecordResult 记录执行结果
func (b *Breaker) RecordResult(success bool) {
    if success {
        b.consecutiveFails = 0
        b.state = BreakerClosed
    } else {
        b.consecutiveFails++
        b.lastFailTime = time.Now()
        if b.consecutiveFails >= b.config.MaxConsecutiveFailures {
            b.state = BreakerOpen
        }
    }
}
```

### F.4 执行/验证分离设计

参考 Harness-Starter 的执行/验证分离机制：

```go
// Validator 独立验证器
type Validator struct {
    provider Provider
    config   *ValidatorConfig
}

// Validate 验证执行结果
func (v *Validator) Validate(ctx context.Context, goal string, result *ExecutionResult) (*ValidationResult, error) {
    // 构建验证提示词
    prompt := buildValidationPrompt(goal, result)

    // 调用LLM进行验证（独立于执行器）
    response, err := v.provider.Chat(ctx, validationSystemPrompt, prompt)
    if err != nil {
        return nil, err
    }

    // 解析验证结果
    return parseValidationResult(response)
}

// ValidationResult 验证结果
type ValidationResult struct {
    Passed     bool     // 是否通过
    Score      float64  // 评分（0-1）
    Issues     []string // 发现的问题
    Suggestions []string // 改进建议
}
```

### F.5 GC扫描器设计

参考 Harness-Starter 的8个确定性维度：

```go
// ScanDimension 扫描维度
type ScanDimension struct {
    Name        string
    Description string
    Scanner     func(projectPath string) (*ScanResult, error)
}

// 默认扫描维度
var DefaultDimensions = []ScanDimension{
    {Name: "documentation", Scanner: scanDocumentation},  // 文档完整性
    {Name: "git_status", Scanner: scanGitStatus},         // Git状态
    {Name: "todo_density", Scanner: scanTodoDensity},     // TODO密度
    {Name: "test_coverage", Scanner: scanTestCoverage},   // 测试覆盖
    {Name: "code_quality", Scanner: scanCodeQuality},     // 代码质量
    {Name: "dependency", Scanner: scanDependency},        // 依赖健康
    {Name: "security", Scanner: scanSecurity},            // 安全检查
    {Name: "performance", Scanner: scanPerformance},      // 性能检查
}

// ScanResult 扫描结果
type ScanResult struct {
    Dimension string
    Score     float64  // 0-1
    Issues    []string
    Details   map[string]interface{}
}
```

### F.6 工作流模式设计

参考 Harness-Starter 的工作流模式：

```go
// WorkflowMode 工作流模式
type WorkflowMode string

const (
    ModeFull    WorkflowMode = "full"    // 完整检查
    ModeHotfix  WorkflowMode = "hotfix"  // 紧急修复
    ModeTweak   WorkflowMode = "tweak"   // 微调
)

// WorkflowStage 工作流阶段
type WorkflowStage string

const (
    StageDesign WorkflowStage = "design" // 设计阶段
    StageFix    WorkflowStage = "fix"    // 修复阶段
    StageTest   WorkflowStage = "test"   // 测试阶段
)

// ModeConfig 模式配置
type ModeConfig struct {
    Mode           WorkflowMode
    Stage          WorkflowStage
    ApprovalLevel  ApprovalLevel  // 审批级别
    CheckIntensity float64        // 检查强度（0-1）
    AutoFix        bool           // 是否自动修复
}
```

### F.7 实现阶段

| Phase | 任务 | 时间 |
|-------|------|------|
| **Phase 1** | Circuit Breaker + 执行/验证分离 | 2-3天 |
| **Phase 2** | GC扫描器 + 工作流模式 | 3-4天 |
| **Phase 3** | 上下文自动注入 + 工具执行后审查 | 2-3天 |

### F.8 设计原则

1. **不抄袭代码** - 学习设计思路，从零实现
2. **缓存命中率铁律** - 所有新功能不影响缓存优化
3. **渐进式实现** - 分阶段实现，逐步增强
4. **可配置性** - 所有功能可配置、可禁用

---

## 附录 G: 画布功能差距修复与项目对比

### G.1 差距分析

基于对四个参考项目的深度代码研究，发现烛龙画布存在以下差距：

| 优先级 | 功能 | 来源 | 说明 |
|--------|------|------|------|
| **P0** | @引用机制 | infinite-canvas | 对话中引用节点/资产内容 |
| **P0** | 画布快照集成 | infinite-canvas | 助手对话带画布完整上下文 |
| **P1** | 章节事件图谱 | Toonflow | 从小说提取章节事件并结构化存储 |
| **P1** | 多Agent工作空间协作 | TapCanvas | 工作空间移交、Agent间消息传递 |
| **P2** | 五层内容生产架构 | Toonflow | 导入→解析→角色→剧本→分镜→视频 |

### G.2 @引用机制（P0 - 已实现）

参考 infinite-canvas 的 @[node:nodeId] 语法，实现了：

```typescript
// 解析@引用
function parseReferences(text: string, nodes: CanvasNode[], assets: Asset[]): ReferenceMatch[] {
  const regex = /@\[(node|asset):([^\]]+)\]/g;
  // 匹配 @[node:xxx] 或 @[asset:xxx]
}

// 构建引用上下文
function buildReferenceContext(references, nodes, assets): ResourceReference[] {
  // 从引用中提取节点/资产的标题、内容、图片
}
```

**核心功能**：
- @[node:id] 引用节点内容
- @[asset:id] 引用资产内容
- 输入@时弹出提及菜单，支持过滤
- 消息渲染时@引用高亮显示

### G.3 画布快照集成（P0 - 已实现）

参考 infinite-canvas 的上下文构建方式，在每次助手对话请求时附带完整画布状态：

```typescript
// 构建完整画布快照
function buildFullSnapshot() {
  return {
    nodes: nodes.map(n => ({
      id, type, title, status,
      content, prompt, imageUrl
    })),
    edges: edges.map(e => ({ source, target })),
    selectedNodeIds,
    assetCount,
  };
}
```

**核心功能**：
- 每次请求带完整画布JSON快照
- 选中节点自动作为参考上下文
- 支持多轮对话历史

### G.4 章节事件图谱（P1 - 已实现）

参考 Toonflow 的章节事件图谱设计，实现了结构化的事件管理：

```typescript
interface ChapterEvent {
  id: string;
  chapterId: string;
  description: string;
  characters: string[];
  locations: string[];
  time: string;
  importance: 'low' | 'medium' | 'high';
  dependencies: string[];
}

interface ChapterGraph {
  chapters: Chapter[];
  events: ChapterEvent[];
  relationships: EventRelationship[];
  characters: CharacterInfo[];
  locations: LocationInfo[];
}
```

**核心组件**：`ChapterGraphPanel` - 章节/事件/角色/场景四标签管理面板

### G.5 多Agent工作空间协作（P1 - 已实现）

参考 TapCanvas 的多Agent协作机制：

```typescript
interface AgentWorkspace {
  id: string;
  name: string;
  agents: AgentInfo[];
  messages: AgentMessage[];
  handoffs: WorkspaceHandoff[];
  sharedAssets: string[];
}

interface AgentMessage {
  fromAgentId: string;
  toAgentId: string;
  type: 'request' | 'response' | 'notification' | 'broadcast';
  payload: any;
}
```

**核心组件**：`CollaborationPanel` - 工作空间/Agent管理/消息通信三标签面板

### G.6 五层内容生产架构（P2 - 已实现）

参考 Toonflow 的五层架构设计：

```typescript
enum ProductionLayer {
  Import = 'import',       // 小说导入
  Parse = 'parse',         // 内容解析
  Character = 'character', // 角色生成
  Script = 'script',       // 剧本生成
  Storyboard = 'storyboard', // 分镜生成
  Video = 'video',         // 视频生成
}

interface ProductionPipeline {
  layers: ProductionLayer[];
  currentLayer: ProductionLayer;
  status: ProductionStatus;
  progress: Record<ProductionLayer, number>;
  config: ProductionConfig;
}
```

**核心组件**：`ProductionPanel` - 管道列表/逐层进度/模拟执行

### G.7 项目功能对比

| 功能维度 | 烛龙 | TapCanvas | Toonflow | infinite-canvas |
|---------|------|-----------|----------|-----------------|
| 无限画布 | ✅ React Flow | ✅ React Flow | ✅ Custom | ✅ Custom |
| 节点类型 | ✅ 6种 | ✅ 5种 | ✅ 4种 | ✅ 6种 |
| Ops抽象 | ✅ 全量 | ❌ 无 | ❌ 无 | ✅ 全量 |
| 撤销/重做 | ✅ 快照 | ❌ 无 | ❌ 无 | ✅ Ops历史 |
| 画布助手 | ✅ 带@引用 | ⚠️ 部分 | ❌ 无 | ✅ 完整 |
| @引用机制 | ✅ 双类型 | ❌ 无 | ❌ 无 | ✅ 单类型 |
| 资产管理 | ✅ 完整+衍生 | ✅ 项目化 | ✅ 衍生系统 | ✅ 本地存储 |
| 章节图谱 | ✅ 完整 | ❌ 无 | ✅ 事件驱动 | ❌ 无 |
| 多Agent协作 | ✅ 工作空间 | ✅ 消息+移交 | ✅ 三层架构 | ❌ 无 |
| 生产管道 | ✅ 5层 | ⚠️ DAG | ✅ 5层完整 | ⚠️ 3阶段 |
| 版本管理 | ✅ 快照 | ❌ 无 | ❌ 无 | ✅ Ops历史 |
| 导出/导入 | ✅ JSON/PNG/SVG | ✅ JSON | ⚠️ 部分 | ✅ 完整 |
| MCP集成 | ✅ Go实现 | ❌ 无 | ❌ 无 | ✅ TS实现 |
| 性能监控 | ✅ 面板 | ❌ 无 | ❌ 无 | ⚠️ 基础 |
| AI增强 | ✅ 布局/节点/内容 | ⚠️ 仅生成 | ✅ 全管道 | ✅ 助手+生成 |

**Zhulong 独有优势**：
1. **MCP工具协议** (Go实现) - 四个参考项目均无原生MCP
2. **原子Ops抽象** - 统一操作接口，便于扩展
3. **@[node/asset]双类型引用** - 支持节点和资产双目标
4. **快照撤销/重做** - 包含完整操作历史记录
5. **性能监控面板** - 实时监控性能指标

**覆盖度**：15/15 完全实现 ✅

---

## 附录 H: 与 Hermes-Agent 的学习参考清单

> **策略：学习设计思路，从零实现。不 fork、不复制、不引入外部 License 依赖。**

### H.1 项目概述

Hermes-Agent (NousResearch) 是一个 Python 实现的通用 Agent 框架/平台，12,000+ 次提交。核心理念是 **"与你共同成长的智能体"** ，具有自进化系统、Kanban看板、技能动态加载、自动化蓝图等特性。

### H.2 核心架构差异：主动 vs 被动

| 维度 | Hermes (主动) | 烛龙 (被动) |
|------|-------------|------------|
| **驱动方式** | 系统驱动AI，后台进程自动运行 | 用户驱动AI，等待指令 |
| **自进化** | 每次对话后fork副本审查 + 守卫者定期扫描 | 无 |
| **调度** | 看板tick每60秒轮询，自动spawn Worker | 无后台调度 |
| **自动心跳** | 工具调用间隔自动写DB保活 | 无 |
| **建议系统** | 检测重复需求，主动建议创建自动化 | 无 |

**关键洞察**：烛龙变主动不需要重写架构，只需在现有框架上增加独立的主动层，与Agent循环解耦。

### H.3 主动层架构设计

```
主动层（新增，独立于Agent循环）
├── 调度器 (Scheduler)        — 看板tick / 守卫者扫描 / 自动化cron
├── 事件总线 (EventBus)       — 任务就绪/完成/阻塞事件
├── 看板引擎 (BoardEngine)    — 任务分解/委派/状态流转
└── 建议引擎 (SuggestionEngine) — 检测重复需求/建议创建技能

Agent循环（现有，缓存敏感区）
└── Plan → Execute → Reflect  — prefix永不改写
```

**铁律遵循**：
- 主动层的事件/调度数据**不进入Agent的prefix**
- 调度器通过MCP工具调用触发Agent工作
- 工具结果只进入当前循环的动态区间
- 完全符合缓存命中率铁律

### H.4 看板系统融合

#### H.4.1 看板数据模型

参考 Hermes 的看板系统，烛龙的画布节点扩展：

```go
// BoardNodeStatus 看板节点状态
type BoardNodeStatus string

const (
    StatusTriage  BoardNodeStatus = "triage"   // 粗略想法
    StatusTodo    BoardNodeStatus = "todo"     // 已规划
    StatusReady   BoardNodeStatus = "ready"    // 依赖已完成，可执行
    StatusRunning BoardNodeStatus = "running"  // 正在执行（已claim）
    StatusBlocked BoardNodeStatus = "blocked"  // 阻塞，等待解阻
    StatusReview  BoardNodeStatus = "review"   // 待审查
    StatusDone    BoardNodeStatus = "done"     // 完成
    StatusArchived BoardNodeStatus = "archived" // 归档
)

// BoardTask 看板任务
type BoardTask struct {
    ID               string
    Title            string
    Body             string
    Assignee         string           // Worker profile
    Status           BoardNodeStatus
    Priority         int
    DependsOn        []string         // 依赖的任务ID
    ClaimLock        string           // 当前持有者
    ClaimExpires     int64            // 声明到期时间
    ConsecutiveFails int              // 连续失败计数器
    MaxRetries       int              // 断路器阈值
    WorkerPID        int              // Worker进程ID
    Result           string
    Artifacts        []string         // 产出文件
}
```

#### H.4.2 看板→画布映射

| Hermes看板 | 烛龙画布 | 融合方式 |
|-----------|---------|---------|
| 任务卡片 (Task) | 画布节点 (Node) | 增加Board类型节点 |
| 依赖边 (task_links) | 画布连线 (Edge) | 复用现有连线系统 |
| 状态流转 | 节点状态 (NodeStatus) | 扩展为8种看板状态 |
| Worker委派 | Agent委派 | 调度器spawn子进程 |
| 评论线程 (comments) | 画布助手 | 现有助手系统 |
| 审计日志 (events) | Trace系统 | 复用 |
| 附件 (attachments) | 资产系统 | 复用 |

#### H.4.3 状态流转

```
triage ──(specify)──> todo ──(parents done)──> ready ──(claim)──> running
                                                      ↑               │
                                                      │         ┌─────┼──────┐
                                                      │         │     │      │
                                                      │    [complete] [block] [crash/timeout]
                                                      │         │     │      │
                                                      │         ▼     ▼      ▼
                                                      │       done  blocked  ready (retry)
                                                      │               │
                                                      └──(unblock)───┘
                                                      review ──(claim_review)──> running...
                                                      archived (terminal)
```

### H.5 技能动态加载系统

#### H.5.1 目录结构

参考 Hermes 的 agentskills.io 兼容格式：

```
~/.zhulong/skills/
├── my-skill/
│   ├── SKILL.md           # 主指令文件（必需）
│   ├── references/        # 参考文档、API文档
│   ├── templates/         # 输出模板
│   ├── scripts/           # 可执行脚本
│   └── assets/            # 补充资源文件
└── category/
    └── another-skill/
        └── SKILL.md
```

#### H.5.2 SKILL.md 格式

```yaml
---
name: skill-name              # 必需，最长64字符
description: Brief description # 必需，最长1024字符
version: 1.0.0                # 可选
platforms: [windows]          # 可选，OS限制
environments: [desktop, cli]  # 可选，运行环境
prerequisites:
  env_vars: [API_KEY]
  commands: [git, go]
metadata:
  tags: [coding, analysis]
  related_skills: [code-review]
---
```

#### H.5.3 加载流程

```go
// 技能加载器
type Loader struct {
    searchPaths []string  // 搜索路径（本地目录优先）
}

// Scan 扫描目录加载所有技能
func (l *Loader) Scan() ([]Skill, error) {
    // 遍历 ~/.zhulong/skills/*/SKILL.md
    // 解析YAML前置元数据
    // 验证安全性（路径遍历防护）
    // 注册到Registry
}

// Watch 监听文件变更
func (l *Loader) Watch(callback func(Skill)) {
    // fsnotify 监听技能目录
    // 新增/修改/删除时自动回调
}
```

#### H.5.4 分级暴露

参考 Hermes 的 Tier 1-3 设计：
- **Tier 1**: 仅名称和描述（最小token消耗）
- **Tier 2**: 完整 SKILL.md 内容
- **Tier 3**: 完整内容 + 支持文件（references/templates/scripts/assets）

#### H.5.5 自进化（参考Hermes background_review + curator）

```go
// BackgroundReviewer 后台审查器
type BackgroundReviewer struct {
    provider Provider
}

// Review 审查会话快照，判断是否需要创建/更新技能
func (r *BackgroundReviewer) Review(snapshot SessionSnapshot) (*SkillChange, error) {
    // fork副本Agent
    // 回放会话快照
    // 执行自我审查
    // 工具权限严格限制（只允许memory和skill_manage）
}

// Curator 守卫者
type Curator struct {
    interval time.Duration // 默认7天
}

// Maintain 维护技能库
func (c *Curator) Maintain() error {
    // 标记过时技能（30天未使用）
    // 归档旧技能（90天未使用）
    // 合并相似技能（可选）
}
```

### H.6 自动化工作流模板

#### H.6.1 蓝图系统

参考 Hermes 的 14 个内置蓝图模板：

```go
// BlueprintSlot 蓝图插槽
type BlueprintSlot struct {
    Name     string   `yaml:"name"`
    Type     string   `yaml:"type"`     // time/enum/text/weekdays
    Label    string   `yaml:"label"`
    Default  string   `yaml:"default"`
    Options  []string `yaml:"options,omitempty"`
    Required bool     `yaml:"required"`
    Desc     string   `yaml:"desc"`
}

// Blueprint 自动化蓝图
type Blueprint struct {
    Key              string          `yaml:"key"`
    Title            string          `yaml:"title"`
    Description      string          `yaml:"description"`
    Category         string          `yaml:"category"`  // daily/weekly/email/general
    ScheduleTemplate string          `yaml:"schedule"`  // 带{slot}的cron表达式
    PromptTemplate   string          `yaml:"prompt"`    // 带{slot}的种子指令
    Slots            []BlueprintSlot `yaml:"slots"`
    Skills           []string        `yaml:"skills,omitempty"` // 运行前加载的技能
}
```

#### H.6.2 内置蓝图

| 蓝图 | 分类 | 说明 |
|------|------|------|
| `morning-brief` | daily | 每日晨间简报 |
| `weekly-review` | weekly | 每周项目回顾 |
| `news-digest` | general | 主题新闻摘要 |
| `code-health` | daily | 代码健康扫描 |
| `dependency-check` | weekly | 依赖更新检查 |
| `test-runner` | general | 定时运行测试 |
| `backup` | daily | 自动备份 |
| `report-gen` | general | 定时生成报告 |

#### H.6.3 Cron调度

```go
// Schedule 调度计划
type Schedule struct {
    Expression string // cron表达式 / 间隔 / 一次性时间
    Type       string // cron / interval / once
}

// ParseSchedule 解析调度计划
func ParseSchedule(s string) (*Schedule, error) {
    // "30m" / "2h" / "1d" → 一次性间隔
    // "every 30m" / "every 2h" → 重复间隔
    // "0 9 * * *" → 标准cron
    // "2026-06-22T14:00" → 一次性指定时间
}
```

### H.7 实现阶段

| Phase | 任务 | 时间 | 说明 |
|-------|------|------|------|
| **Phase 1** | 看板→画布融合 | 2-3天 | 扩展节点状态、添加看板节点、状态流转引擎 |
| **Phase 1** | 主动层调度器 | 2-3天 | 事件总线、调度器tick、Worker spawn |
| **Phase 2** | 技能目录加载 | 1-2天 | SKILL.md解析、目录扫描、动态注册、Watch |
| **Phase 2** | 自动化蓝图模板 | 2-3天 | 内置模板、参数化插槽、cron解析 |
| **Phase 3** | 自进化系统 | 3-5天 | 后台审查器、守卫者、建议引擎 |

### H.8 设计原则

1. **不抄袭代码** - 学习设计思路，从零实现
2. **缓存命中率铁律** - 主动层独立于Agent循环，不影响缓存
3. **渐进式实现** - 分阶段实现，逐步增强
4. **可配置性** - 所有功能可配置、可禁用
5. **用户同意优先** - 自进化和建议系统需要用户确认

### H.9 最终差距分析

基于对 Hermes-Agent 项目的全面深度代码研究（12,493次提交），烛龙已完成核心功能对标，但存在以下架构级差距：

| 优先级 | 差距 | 说明 | 烛龙状态 |
|--------|------|------|---------|
| **P0** | 消息网关 | 20+平台适配器（Telegram/钉钉/飞书/Slack等），统一的GatewayRunner管理 | ❌ 缺失 |
| **P0** | 双文件记忆系统 | MEMORY.md（Agent笔记）+ USER.md（用户画像）+ 8种外部记忆提供者 | ❌ 缺失 |
| **P0** | 终端执行后端 | Docker容器隔离/SSH远程/Singularity HPC/Modal云端，仅本地执行不够 | ❌ 缺失 |
| **P1** | 7层安全模型 | 容器隔离/SSRF保护/输入清理/上下文文件扫描/MCP凭据过滤/文件突变验证/供应链审计 | ⚠️ 基础 |
| **P1** | Electron桌面应用 | 流式聊天+并排预览+文件浏览器+语音交互+自动更新 | ❌ 缺失 |
| **P1** | Web Dashboard | 管理配置/API密钥/会话/Profile，支持远程部署 | ❌ 缺失 |
| **P1** | Skills Hub生态 | 10种技能来源市场（official/github/claude-marketplace等）+安全扫描+信任等级 | ❌ 缺失 |
| **P1** | 国际化(i18n) | 17种语言YAML翻译文件 + React useI18n hook | ⚠️ 部分 |
| **P2** | Cron高级特性 | 无Agent模式（零Token消耗）/wakeAgent门控/任务链/广播投递/SILENT静默 | ⚠️ 基础 |
| **P2** | 18+模型提供者 | Nous Portal(300+)/OpenRouter(200+)/国产全线/凭据池轮换/回退链 | ⚠️ 基础 |
| **P2** | 可观测性 | Langfuse集成/trace&span/insights分析/doctor诊断/dump调试 | ⚠️ 基础 |
| **P2** | 插件系统 | 18个插件目录（browser/cron/memory/model-providers等），3种发现源 | ❌ 缺失 |
| **P2** | ACP协议 | 基于stdio/JSON-RPC的编辑器原生集成（VS Code/Zed/JetBrains） | ❌ 缺失 |
| **P3** | 批量轨迹生成 | ShareGPT格式轨迹，用于训练下一代工具调用模型 | ❌ 缺失 |
| **P3** | 备份与恢复 | hermes backup/import/checkpoints，原子写入 | ❌ 缺失 |
| **P3** | Profile隔离 | 独立HERMES_HOME，可并发运行，导入/导出/别名 | ❌ 缺失 |
| **P3** | 语音交互 | 语音备忘录转录(STT)+文字转语音(TTS)+CLI语音模式 | ❌ 缺失 |

### H.10 烛龙领先于 Hermes 的特性

| 特性 | 烛龙 | Hermes |
|------|------|--------|
| **自动化建议引擎** | ✅ SuggestionEngine（Phase 3） | ❌ 无独立引擎 |
| **骨架压缩** | ✅ 4预设+5焦点+3密度 | ❌ 无 |
| **缓存命中率铁律** | ✅ 系统级约束，prefix永不改写 | ❌ 无 |
| **Ops操作抽象** | ✅ 原子操作接口，撤销/重做 | ❌ 无 |
| **@引用机制** | ✅ 画布内@[node/asset]资源引用 | ❌ 无 |
| **无限画布可视化** | ✅ React Flow + 12种面板 | ⚠️ 仅Kanban看板 |

### H.11 推荐开发路线

```
Phase 4: 消息网关（P0）      — 20+平台适配器 + GatewayRunner
Phase 5: 记忆系统（P0）      — MEMORY.md + USER.md + 外部提供者
Phase 6: 终端后端（P0）      — Docker/SSH/云端隔离执行
Phase 7: 安全增强（P1）      — 7层安全模型
Phase 8: 桌面完善（P1）      — Electron应用 + Web Dashboard
Phase 9: Skills Hub（P1）    — 技能市场生态
```

---

## 附录 I: 与 OpenClaw 的学习参考清单

> **策略：学习设计思路，从零实现。不 fork、不复制、不引入外部 License 依赖。**

### I.1 项目概述

OpenClaw 是一个 TypeScript 实现的企业级个人AI助手平台（61,419次提交），采用 monorepo + pnpm workspace 架构。其格言是 "Your own personal AI assistant. Any OS. Any Platform."

### I.2 核心技术栈

| 组件 | 技术 | 说明 |
|------|------|------|
| 核心语言 | TypeScript | 全栈统一 |
| 包管理 | pnpm workspace | monorepo |
| 构建 | tsdown | 现代 TypeScript 打包 |
| Linter | oxlint/oxfmt | Rust 实现，超高速 |
| 测试 | Vitest | - |
| 部署 | Docker / Fly.io / Render | 多平台 |

### I.3 核心架构对比

| 维度 | OpenClaw | 烛龙 | 差异 |
|------|---------|------|------|
| **Agent 运行时** | `packages/agent-core` 独立包 | `pkg/agent.go` | ✅ 思路一致 |
| **技能系统** | `skills/` 动态加载 | `internal/skills/` 目录+管道 | ✅ 已实现 |
| **会话压缩** | session compaction | `internal/compressor/` 骨架压缩 | ✅ 已实现 |
| **LLM 提供者** | gateway + model catalog | `internal/provider/` | ✅ 已实现 |
| **CLAUDE.md** | AGENTS.md 符号链接 | 类似 Harness-Starter | ✅ 已参考 |
| **插件 SDK** | `packages/plugin-sdk` | 无 | ❌ 缺少 |
| **QA 体系** | `qa/` 集成测试点 | 仅单元测试 | ⚠️ 可增强 |

### I.4 最值得借鉴的三大特性

#### 1. QA Lab（质量保障实验室）

OpenClaw 拥有专业的质量保障体系，烛龙缺少集成测试和端到端测试：

```
qa/
├── tests/           集成测试用例
├── lab.mjs          QA Lab 运行器
└── http-api/        HTTP API 测试集
```

**设计思路**：
- 独立的 QA 目录，不是散落在各 package 中
- 专用的 Lab 运行器，非标准 test runner
- 包含 HTTP API 级别的集成测试
- 验证 Agent 的实际行为而非代码单元

**烛龙实现方式**：
```go
// internal/qa/lab.go
type QALab struct {
    tests []QATest
    runner *QARunner
}

// QATest 集成测试
type QATest struct {
    Name string
    // 输入 → 执行 → 验证
    Input    string
    Setup    func() error
    Execute  func(ctx) (*Result, error)
    Verify   func(*Result) error
    Teardown func() error
}
```

#### 2. .agents/ 自托管开发代理

OpenClaw 用 AI 开发 AI，通过自托管的开发代理自动完成代码审查、测试等任务：

```
.agents/
├── autoreview/      自动代码审查代理
├── autotest/        自动测试代理
└── config.yaml      代理配置
```

**设计思路**：
- 项目元目录 `.agents/` 与 `.claude/` 同级
- 每个代理有独立的配置和技能
- 代理可以调用项目自身的能力（用烛龙开发烛龙）
- 代理的输出直接作为 PR 评论/测试报告

**烛龙实现方式**：
```go
// .agents/ 目录结构
.agents/
├── reviewer/          // 自动审查代理
│   ├── CLAUDE.md      // 代理行为规则
│   └── config.yaml    // 代理配置
├── tester/            // 自动测试代理
│   └── config.yaml
└── manager.go         // 代理管理器
```

#### 3. ClawScore 技能评分系统

社区驱动的技能质量评判机制：

```
type ClawScore struct {
    SkillName string
    Score      float64  // 0-5
    Reviews    int      // 评价数
    Version    string
    Tags       []string
}
```

**设计思路**：
- 用户评价驱动
- 评分影响技能排序推荐
- 版本关联
- 防刷机制

### I.5 工程实践借鉴

| 实践 | OpenClaw | 烛龙现状 | 借鉴价值 |
|------|---------|---------|---------|
| 提交规范 | feat/fix/chore/test/refactor/dosc + 详细描述 | 有基本规范 | ✅ 一致 |
| QA Lab | 独立qa/目录+专用运行器 | 仅单元测试 | ⭐ 值得引入 |
| .agents/ | 自托管开发代理 | 无 | ⭐ 值得引入 |
| AGENTS.md | AI辅助开发的上下文文件 | 类似Harness-Starter | ✅ 已参考 |
| pre-commit hooks | .pre-commit-config.yaml | 无 | ⚠️ 可引入 |
| CodeQL分析 | GitHub安全分析 | 无 | ⚠️ 可引入 |

### I.6 设计原则

1. **不抄袭代码** - 学习设计思路，从零实现
2. **缓存命中率铁律** - 所有新功能不影响缓存优化
3. **分阶段实现** - QA Lab→.agents/→ClawScore 逐步推进
4. **自举设计** - 用烛龙开发烛龙（.agents/ 理念）
