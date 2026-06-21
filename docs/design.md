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

## 8. 风险与缓解

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
