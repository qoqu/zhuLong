# Zhulong（烛龙） 详细设计文档

> 版本: v0.2-draft
> 日期: 2026-06-20
> 语言: Go
> 状态: 设计阶段（按优先级整理）

---

## 1. 项目定位

Zhulong（烛龙）是一个**基于 DeepSeek 的通用自主循环 Agent 框架**，核心特性：

- **自主多轮循环**：规划 → 执行 → 反省 → 重新规划，无需每轮人工触发
- **高效上下文管理**：深度优化 DeepSeek prefix-cache，最大化缓存命中率
- **自适应学习**：成功策略复用、认知模型更新、探索/利用平衡
- **生产级可靠性**：检查点恢复、成本控制、人机协作断点、完整可观测性

### 1.1 设计目标

| 目标 | 说明 |
|------|------|
| 通用性 | 不绑定特定场景（编程/写作/数据分析），通过工具和 prompt 定制 |
| 高缓存命中率 | 深度优化 DeepSeek prefix-cache，上下文布局只追加不重写 |
| 自适应学习 | 成功策略复用、认知模型更新、探索/利用平衡 |
| 崩溃可恢复 | 任意时刻中断都能从最近检查点恢复 |
| 成本可控 | 多层预算机制，防止无限循环烧 token |
| 可观测 | 完整 trace，每步决策可追溯 |

### 1.2 技术约束

| 约束 | 说明 |
|------|------|
| LLM | **仅支持 DeepSeek**（prefix-cache 优化深度绑定 DeepSeek API） |
| 语言 | Go |
| 工具协议 | MCP（Model Context Protocol） |

### 1.3 参考项目

> **原则：学习设计思路，从零实现。不 fork、不复制、不引入外部 License 依赖。**

| 项目 | 学习内容 |
|------|---------|
| [DeepSeek-Reasonix](https://github.com/esengine/DeepSeek-Reasonix) | MCP 工具协议规范、确定性工具结果裁剪策略、prefix-cache 稳定性设计思路 |
| [Ailoom-Context](https://github.com/EvanLyu-oss/Ailoom-Context) | 骨架压缩结构设计（skeleton + restore 分离）、焦点模式语义、增量压缩思路 |

两个项目均为 MIT License（允许商业使用），但 Zhulong 选择完全独立实现，确保零外部依赖。

### 1.4 理论基础

Zhulong 融合六大系统科学理论：

| 理论 | 贡献 | 核心应用 |
|------|------|---------|
| 一般系统论（贝塔朗菲） | 系统是什么 | 开放系统、备选路径 |
| 工程控制论（钱学森） | 系统怎么控 | 振荡/发散检测、性能指标 |
| 信息论（香农） | 信息怎么传 | 信息增益、信息密度、噪声处理 |
| 耗散结构理论（普利高津） | 系统怎么活 | 停滞检测、探索触发 |
| 协同学（哈肯） | 系统怎么协同 | 序参量识别、役使原理 |
| 复杂适应系统（霍兰德） | 系统怎么学 | 积木块、认知模型、多样性 |

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

**核心循环实现**：

```go
// internal/controller/loop.go

package controller

import (
    "context"
    "fmt"
    "time"
)

type LoopConfig struct {
    MaxLoops        int           // 最大循环次数，默认 50
    MaxTokens       int           // 最大 token 消耗
    MaxCost         float64       // 最大费用（元）
    MaxWallTime     time.Duration // 最大运行时间
    CheckpointEvery int           // 每 N 次状态转移保存一次检查点
}

type LoopResult struct {
    FinalState LoopState
    Answer     string        // 最终输出
    Trace      TraceLog
    Stats      LoopStats
}

type LoopStats struct {
    TotalLoops    int
    TotalTokens   int
    TotalCost     float64
    TotalDuration time.Duration
    PlanChanges   int         // 重规划次数
}

// Run 是主循环入口
func (c *Controller) Run(ctx context.Context, goal string, opts ...RunOption) (*LoopResult, error) {
    // 1. 初始化
    session := c.newSession(goal)
    c.budget.Reset()
    c.trace.Start(session.ID)

    // 2. 检查是否有未完成的检查点
    if cp, err := c.checkpoint.LoadLatest(); err == nil && cp != nil {
        session = c.restoreFromCheckpoint(cp)
        c.trace.Log("checkpoint_restored", cp.ID)
    }

    // 3. 主循环
    for {
        // 3.1 检查 context 取消
        select {
        case <-ctx.Done():
            session.State = StateCancelled
            return c.finalize(session)
        default:
        }

        // 3.2 检查预算
        if c.budget.IsExceeded() {
            session.State = StateWaitingHuman
            c.trace.Log("budget_exceeded", c.budget.Summary())
            return c.finalize(session)
        }

        // 3.3 状态转移
        switch session.State {
        case StateIdle:
            session.State = StatePlanning

        case StatePlanning:
            plan, err := c.planner.Plan(ctx, session.Goal, session.Memory)
            if err != nil {
                session.State = StateError
                session.Error = err
                break
            }
            session.Plan = plan
            session.CurrentStep = 0
            session.State = StateExecuting
            c.trace.Log("plan_created", plan)

        case StateExecuting:
            if session.CurrentStep >= len(session.Plan.Steps) {
                session.State = StateReflecting
                break
            }

            step := session.Plan.Steps[session.CurrentStep]
            result, err := c.executor.Execute(ctx, step, session.Memory)
            if err != nil {
                session.State = StateError
                session.Error = err
                break
            }

            session.Memory.AddStepResult(step, result)
            session.CurrentStep++
            c.budget.Consume(result.TokensUsed)
            c.trace.Log("step_executed", step.ID, result)

            // 检查是否需要人工介入
            if c.human.ShouldPause(step, result) {
                session.State = StateWaitingHuman
                break
            }

            session.State = StateReflecting

        case StateReflecting:
            assessment, err := c.reflector.Reflect(ctx, session.Goal, session.Plan, session.Memory)
            if err != nil {
                session.State = StateError
                session.Error = err
                break
            }

            session.Memory.AddAssessment(assessment)
            c.trace.Log("reflected", assessment)

            switch assessment.Decision {
            case DecisionComplete:
                session.State = StateDone
            case DecisionContinue:
                session.State = StateExecuting
            case DecisionReplan:
                session.State = StateReplanning
            case DecisionFail:
                session.State = StateError
                session.Error = fmt.Errorf("reflector判定失败: %s", assessment.Reason)
            }

        case StateReplanning:
            newPlan, err := c.planner.Replan(ctx, session.Goal, session.Plan, session.Memory)
            if err != nil {
                session.State = StateError
                session.Error = err
                break
            }
            session.Plan = newPlan
            session.CurrentStep = 0
            session.PlanChanges++
            session.State = StateExecuting
            c.trace.Log("replanned", newPlan)

        case StateWaitingHuman:
            input, err := c.human.WaitForInput(ctx, session)
            if err != nil {
                session.State = StateError
                session.Error = err
                break
            }
            session = c.resumeFromHuman(session, input)
            c.trace.Log("human_resumed", input)

        case StateDone, StateError, StateCancelled:
            return c.finalize(session)
        }

        // 3.4 保存检查点
        if c.shouldCheckpoint(session) {
            c.checkpoint.Save(session.ToCheckpoint())
        }
    }
}
```

#### 3.1.2 Planner — 规划器

负责将用户目标分解为可执行的步骤序列。

**接口定义**：

```go
// internal/planner/planner.go

package planner

import "context"

type Planner interface {
    // Plan 根据目标和当前记忆生成初始计划
    Plan(ctx context.Context, goal string, memory MemoryReader) (*Plan, error)

    // Replan 根据反省结果调整计划
    Replan(ctx context.Context, goal string, currentPlan *Plan, memory MemoryReader) (*Plan, error)
}

type Plan struct {
    ID          string
    Steps       []Step
    Rationale   string    // 为什么这样规划
    CreatedAt   time.Time
}

type Step struct {
    ID          string
    Description string
    Action      Action     // 要执行的动作
    DependsOn   []string   // 依赖的步骤 ID
    Breakpoint  bool       // 是否需要执行前人工确认
}

type Action struct {
    Type    string                 // "tool_call" | "llm_generate" | "human_input"
    Tool    string                 // 工具名称（tool_call 类型时）
    Params  map[string]interface{} // 工具参数
    Prompt  string                 // LLM 提示（llm_generate 类型时）
}
```

#### 3.1.3 Executor — 执行器

负责执行计划中的单个步骤。

**接口定义**：

```go
// internal/executor/executor.go

package executor

import "context"

type Executor interface {
    // Execute 执行单个步骤，返回结果
    Execute(ctx context.Context, step Step, memory MemoryReader) (*StepResult, error)
}

type StepResult struct {
    StepID      string
    Success     bool
    Output      string                 // 执行输出
    ToolCalls   []ToolCallRecord       // 工具调用记录
    TokensUsed  int                    // 本步骤消耗的 token
    Duration    time.Duration
    Error       error                  // 如果失败，错误信息
}

type ToolCallRecord struct {
    ToolName   string
    Input      map[string]interface{}
    Output     string
    Duration   time.Duration
    Prunable   bool   // 结果是否可裁剪
    CacheKey   string // 用于判断是否可重新获取
}
```

#### 3.1.4 Reflector — 反省器

负责评估执行结果，决定循环的下一步走向。

**接口定义**：

```go
// internal/reflector/reflector.go

package reflector

import "context"

type Reflector interface {
    // Reflect 评估当前状态，返回决策
    Reflect(ctx context.Context, goal string, plan *Plan, memory MemoryReader) (*Assessment, error)
}

type Decision int

const (
    DecisionComplete Decision = iota // 目标已完成
    DecisionContinue                  // 继续执行下一步
    DecisionReplan                    // 需要重新规划
    DecisionFail                      // 目标无法完成
)

type Assessment struct {
    Decision    Decision
    Reason      string   // 决策理由
    Confidence  float64  // 置信度 0-1
    Findings    []string // 本轮发现的新信息
    Suggestions []string // 给 Replanner 的建议
}
```

---

### 3.2 P1：核心增强模块（实现基础功能后优先实现）

#### 3.2.1 振荡/发散检测（工程控制论）

**问题**：Agent 卡在重复失败循环，或进度倒退。

**解决方案**：

```go
// internal/stability/analyzer.go

package stability

// StabilityAnalyzer 稳定性分析器
type StabilityAnalyzer struct {
    history []LoopSnapshot
}

// IsOscillating 检测是否在振荡（重复相同失败模式）
func (s *StabilityAnalyzer) IsOscillating() bool {
    if len(s.history) < 4 {
        return false
    }
    // 检测 A→B→A→B 模式
    n := len(s.history)
    return s.history[n-4].State == s.history[n-2].State &&
           s.history[n-3].State == s.history[n-1].State
}

// IsDiverging 检测是否在发散（进度倒退）
func (s *StabilityAnalyzer) IsDiverging() bool {
    if len(s.history) < 3 {
        return false
    }
    // 最近3次循环的完成度递减
    n := len(s.history)
    return s.history[n-3].Progress > s.history[n-2].Progress &&
           s.history[n-2].Progress > s.history[n-1].Progress
}

// LoopSnapshot 循环快照
type LoopSnapshot struct {
    LoopNumber int
    State      string
    Progress   float64
    Timestamp  time.Time
}
```

#### 3.2.2 系统级性能指标（工程控制论）

**问题**：需要量化评估 Agent 运行效率。

**解决方案**：

```go
// internal/metrics/performance.go

package metrics

// SystemPerformance 系统级性能指标
type SystemPerformance struct {
    // 端到端指标
    TotalTime       time.Duration  // 总耗时
    TotalCost       float64        // 总成本
    GoalAchievement float64        // 目标达成度

    // 效率指标
    TokenEfficiency  float64  // 有效 token / 总 token
    LoopEfficiency   float64  // 有效循环 / 总循环
    ToolEfficiency   float64  // 有效工具调用 / 总工具调用

    // 稳定性指标
    OscillationCount int      // 振荡次数
    ReplanCount      int      // 重规划次数
    ErrorCount       int      // 错误次数
}
```

#### 3.2.3 信息增益工具选择（信息论）

**问题**：选择工具时，应该选能提供最多新信息的工具。

**解决方案**：

```go
// internal/information/gain.go

package information

import "math"

// InformationGain 信息增益计算器
type InformationGain struct {
    toolHistory map[string][]ToolResult
}

// EstimateGain 估算工具调用的信息增益
func (ig *InformationGain) EstimateGain(tool Tool, currentContext string) float64 {
    // 如果这个工具之前调用过且结果已知，信息增益低
    priorResults := ig.toolHistory[tool.Name()]
    if len(priorResults) > 0 {
        return ig.diminishingGain(priorResults)
    }

    // 新工具，预期信息增益高
    return 0.8
}

// diminishingGain 边际信息递减
func (ig *InformationGain) diminishingGain(results []ToolResult) float64 {
    baseGain := 1.0
    decay := 0.6 // 每次调用衰减 40%
    return baseGain * math.Pow(decay, float64(len(results)))
}
```

#### 3.2.4 信息密度优化（信息论）

**问题**：上下文窗口有容量上限，必须最大化每 token 携带的有用信息。

**解决方案**：

```go
// internal/information/density.go

package information

// InformationDensity 信息密度计算器
type InformationDensity struct {
    tokenizer Tokenizer
}

// Density 信息密度 = 有用信息量 / token 数
func (id *InformationDensity) Density(messages []Message) float64 {
    usefulInfo := 0.0
    totalTokens := 0

    for _, msg := range messages {
        tokens := id.tokenizer.Count(msg.Content)
        totalTokens += tokens

        switch {
        case msg.Role == "system" && isSystemPrompt(msg):
            usefulInfo += float64(tokens) * 1.0 // 系统提示信息密度高
        case msg.Role == "tool" && isPrunable(msg):
            usefulInfo += float64(tokens) * 0.1 // 可裁剪的工具结果信息密度低
        case msg.Role == "tool" && !isPrunable(msg):
            usefulInfo += float64(tokens) * 0.8 // 不可裁剪的工具结果信息密度高
        default:
            usefulInfo += float64(tokens) * 0.5
        }
    }

    if totalTokens == 0 {
        return 0
    }
    return usefulInfo / float64(totalTokens)
}
```

#### 3.2.5 停滞检测（耗散结构理论）

**问题**：Agent 卡死时需要自动发现。

**解决方案**：

```go
// internal/stagnation/detector.go

package stagnation

// Detector 停滞检测器
type Detector struct {
    windowSize       int     // 检测窗口大小（最近 N 步）
    entropyThreshold float64 // 信息增益阈值
    history          []StepInfo
}

type StepInfo struct {
    StepNumber    int
    NewInfoScore  float64 // 本步获取的新信息量（0-1）
    ToolUsed      string
    ProgressDelta float64 // 进展变化（-1 到 1）
}

// IsStagnating 检测是否停滞
func (d *Detector) IsStagnating() bool {
    if len(d.history) < d.windowSize {
        return false
    }

    recent := d.history[len(d.history)-d.windowSize:]
    for _, step := range recent {
        if step.NewInfoScore >= d.entropyThreshold {
            return false // 还有新信息输入，未停滞
        }
    }
    return true // 连续 N 步无新信息，停滞
}

// StagnationType 停滞类型
type StagnationType int

const (
    StagnationInfoStarved StagnationType = iota // 信息饥饿
    StagnationLooping                            // 循环卡死
    StagnationBlocked                            // 路径阻塞
)

// DiagnoseStagnation 诊断停滞原因
func (d *Detector) DiagnoseStagnation() StagnationType {
    recent := d.history[len(d.history)-d.windowSize:]

    // 检测循环卡死：相同工具重复调用
    tools := make(map[string]int)
    for _, step := range recent {
        tools[step.ToolUsed]++
    }
    for _, count := range tools {
        if count >= d.windowSize/2 {
            return StagnationLooping
        }
    }

    // 检测路径阻塞：进展持续为负
    negativeCount := 0
    for _, step := range recent {
        if step.ProgressDelta < 0 {
            negativeCount++
        }
    }
    if negativeCount >= d.windowSize/2 {
        return StagnationBlocked
    }

    return StagnationInfoStarved
}
```

#### 3.2.6 探索触发（耗散结构理论）

**问题**：停滞时需要增加随机性来突破。

**解决方案**：

```go
// internal/exploration/trigger.go

package exploration

// Trigger 探索触发器
type Trigger struct {
    stagnationDetector *stagnation.Detector
    baseTemperature    float64
    explorationTools   []Tool
}

// ShouldExplore 是否应该触发探索
func (t *Trigger) ShouldExplore() bool {
    return t.stagnationDetector.IsStagnating()
}

// ExplorationAction 探索行动
type ExplorationAction struct {
    Type        string  // "increase_temperature" | "try_new_tool"
    Temperature float64
    Tool        Tool
}

// GenerateExploration 生成探索行动
func (t *Trigger) GenerateExploration() ExplorationAction {
    stagnationType := t.stagnationDetector.DiagnoseStagnation()

    switch stagnationType {
    case stagnation.StagnationInfoStarved:
        return ExplorationAction{
            Type:        "increase_temperature",
            Temperature: t.baseTemperature * 1.5,
        }

    case stagnation.StagnationLooping:
        newTool := t.selectUnexpectedTool()
        return ExplorationAction{
            Type: "try_new_tool",
            Tool: newTool,
        }

    case stagnation.StagnationBlocked:
        return ExplorationAction{
            Type:        "increase_temperature",
            Temperature: t.baseTemperature * 2.0,
        }
    }

    return ExplorationAction{Type: "increase_temperature", Temperature: t.baseTemperature * 1.2}
}
```

#### 3.2.7 序参量识别（协同学）

**问题**：识别真正驱动系统的目标/策略，确保所有行动服务于它。

**解决方案**：

```go
// internal/synergetics/order_parameter.go

package synergetics

// OrderParameter 序参量：驱动系统的慢变量
type OrderParameter struct {
    Goal      string    // 核心目标（最慢的变量）
    Strategy  string    // 当前策略（次慢的变量）
    Priority  float64   // 优先级（0-1）
    Stability float64   // 稳定性
}

// IdentifyOrderParameter 识别序参量
func IdentifyOrderParameter(session *Session) *OrderParameter {
    return &OrderParameter{
        Goal:      session.Goal,
        Strategy:  session.CurrentStrategy,
        Priority:  1.0,
        Stability: 1.0 / (float64(session.StrategyChanges) + 1.0),
    }
}
```

#### 3.2.8 役使原理（协同学）

**问题**：行动必须服务于目标，拒绝无关行动。

**解决方案**：

```go
// internal/synergetics/slaving.go

package synergetics

// SlavingPrinciple 役使原理实现
type SlavingPrinciple struct {
    orderParameter *OrderParameter
}

// EnforceSlaving 强制役使：确保行动服务于目标
func (sp *SlavingPrinciple) EnforceSlaving(action Action) (*EnforcedAction, error) {
    relevance := sp.assessRelevance(action, sp.orderParameter.Goal)

    if relevance < 0.3 {
        return nil, fmt.Errorf("行动 '%s' 与目标 '%s' 无关（相关度 %.2f）",
            action.Description, sp.orderParameter.Goal, relevance)
    }

    return &EnforcedAction{
        Action:   action,
        Priority: action.Priority * relevance,
        Aligned:  relevance >= 0.7,
    }, nil
}
```

#### 3.2.9 积木块（CAS）

**问题**：成功策略应该被保存和复用。

**解决方案**：

```go
// internal/learning/building_block.go

package learning

// BuildingBlock 积木块：可复用的策略单元
type BuildingBlock struct {
    ID          string
    Name        string
    Description string
    Pattern     StrategyPattern
    Context     TaskContext
    SuccessRate float64
    UsageCount  int
    CreatedAt   time.Time
    LastUsedAt  time.Time
}

// StrategyPattern 策略模式
type StrategyPattern struct {
    Steps      []StepTemplate
    ToolsUsed  []string
    KeyInsight string
    Conditions []string
}

// BuildingBlockStore 积木块存储
type BuildingBlockStore struct {
    blocks map[string]*BuildingBlock
}

// SaveBlock 保存成功的策略为积木块
func (store *BuildingBlockStore) SaveBlock(strategy Strategy, result Result) error {
    if !result.Success {
        return nil
    }

    block := &BuildingBlock{
        ID:          generateID(),
        Name:        strategy.Name,
        Description: strategy.Description,
        Pattern:     extractPattern(strategy),
        Context:     result.TaskContext,
        SuccessRate: 1.0,
        UsageCount:  1,
        CreatedAt:   time.Now(),
        LastUsedAt:  time.Now(),
    }

    store.blocks[block.ID] = block
    return nil
}

// FindMatchingBlocks 查找匹配当前任务的积木块
func (store *BuildingBlockStore) FindMatchingBlocks(task TaskContext) []*BuildingBlock {
    matches := make([]*BuildingBlock, 0)

    for _, block := range store.blocks {
        similarity := calculateSimilarity(task, block.Context)
        if similarity > 0.7 {
            matches = append(matches, block)
        }
    }

    sort.Slice(matches, func(i, j int) bool {
        return matches[i].SuccessRate > matches[j].SuccessRate
    })

    return matches
}
```

#### 3.2.10 内部模型（CAS）

**问题**：Agent 应该有认知模型，不是无状态执行器。

**解决方案**：

```go
// internal/learning/internal_model.go

package learning

// InternalModel 内部模型：Agent 的认知模型
type InternalModel struct {
    WorldModel WorldModel
    TaskModel  TaskModel
    ToolModel  ToolModel
    Updates    []ModelUpdate
}

// WorldModel 世界模型
type WorldModel struct {
    KnownFacts  map[string]Fact
    Assumptions map[string]float64
    LastUpdated time.Time
}

// TaskModel 任务模型
type TaskModel struct {
    TaskPatterns map[string]TaskPattern
    Strategies   map[string]Strategy
    LastUpdated  time.Time
}

// ToolModel 工具模型
type ToolModel struct {
    ToolCapabilities map[string]ToolCapability
    ToolReliability  map[string]float64
    LastUpdated      time.Time
}

// UpdateModel 根据经验更新认知模型
func (model *InternalModel) UpdateModel(experience Experience) {
    if experience.NewFact != nil {
        model.WorldModel.KnownFacts[experience.NewFact.Key] = *experience.NewFact
    }

    if experience.TaskPattern != nil {
        model.TaskModel.TaskPatterns[experience.TaskPattern.Name] = *experience.TaskPattern
    }

    if experience.ToolResult != nil {
        model.ToolModel.ToolReliability[experience.ToolResult.Tool] =
            model.updateReliability(experience.ToolResult)
    }

    model.Updates = append(model.Updates, ModelUpdate{
        Timestamp:  time.Now(),
        Experience: experience,
    })
}
```

#### 3.2.11 多样性管理（CAS）

**问题**：避免过度依赖单一策略。

**解决方案**：

```go
// internal/learning/diversity.go

package learning

// DiversityManager 多样性管理器
type DiversityManager struct {
    toolUsage     map[string]int
    strategyUsage map[string]int
    threshold     float64
}

// CheckDiversity 检查多样性是否足够
func (dm *DiversityManager) CheckDiversity() DiversityReport {
    toolDiversity := dm.calculateDiversity(dm.toolUsage)
    strategyDiversity := dm.calculateDiversity(dm.strategyUsage)

    return DiversityReport{
        ToolDiversity:     toolDiversity,
        StrategyDiversity: strategyDiversity,
        IsHealthy:         toolDiversity > dm.threshold &&
                          strategyDiversity > dm.threshold,
    }
}

// calculateDiversity 计算多样性指数（香农熵）
func (dm *DiversityManager) calculateDiversity(usage map[string]int) float64 {
    total := 0
    for _, count := range usage {
        total += count
    }

    if total == 0 {
        return 0
    }

    entropy := 0.0
    for _, count := range usage {
        p := float64(count) / float64(total)
        if p > 0 {
            entropy -= p * math.Log2(p)
        }
    }

    return entropy
}
```

#### 3.2.12 混沌边缘（CAS）

**问题**：在"利用已知"和"探索未知"之间保持平衡。

**解决方案**：

```go
// internal/learning/edge_of_chaos.go

package learning

// EdgeOfChaos 混沌边缘管理器
type EdgeOfChaos struct {
    balance float64 // 0=纯利用，1=纯探索
}

// CalculateBalance 计算最佳平衡点
func (eoc *EdgeOfChaos) CalculateBalance(history []Experience) float64 {
    recentSuccess := eoc.calculateRecentSuccess(history)

    if recentSuccess > 0.8 {
        return 0.6 // 成功率高，增加探索
    }

    if recentSuccess < 0.4 {
        return 0.3 // 成功率低，增加利用
    }

    return 0.5
}

// ShouldExplore 是否应该探索
func (eoc *EdgeOfChaos) ShouldExplore() bool {
    return rand.Float64() < eoc.balance
}
```

---

### 3.3 P2：扩展模块（基础稳定后实现）

#### 3.3.1 环境感知器（一般系统论）

**问题**：文件被外部修改时，Agent 需要感知。

**解决方案**：

```go
// internal/environment/monitor.go

package environment

// Monitor 持续监控外部环境变化
type Monitor struct {
    watchers []Watcher
    changes  chan Change
}

type Watcher interface {
    Watch(ctx context.Context, resource string) (<-chan Change, error)
    Stop() error
}

type Change struct {
    Resource  string
    Type      ChangeType
    Timestamp time.Time
}

type ChangeType int

const (
    ChangeModified  ChangeType = iota
    ChangeAdded
    ChangeDeleted
    ChangeUnavailable
)
```

#### 3.3.2 备选路径规划（一般系统论）

**问题**：主路径失败时需要 Plan B。

**解决方案**：

```go
// internal/planner/alternative.go

package planner

// AlternativePath 备选路径
type AlternativePath struct {
    ID          string
    Description string
    Steps       []Step
    WhenToUse   string
    Confidence  float64
}

// PlanWithAlternatives 生成带备选路径的计划
func (p *LLMPlanner) PlanWithAlternatives(ctx context.Context, goal string, memory MemoryReader) (*Plan, error) {
    mainPlan, err := p.Plan(ctx, goal, memory)
    if err != nil {
        return nil, err
    }

    alternatives := p.generateAlternatives(ctx, goal, mainPlan, memory)
    mainPlan.Alternatives = alternatives

    return mainPlan, nil
}
```

#### 3.3.3 噪声处理（信息论）

**问题**：LLM 输出不稳定时需要验证。

**解决方案**：

```go
// internal/executor/noise_handler.go

package executor

// NoiseHandler 噪声处理器
type NoiseHandler struct {
    maxRetries       int
    confidenceThresh float64
}

// HandleNoise 处理 LLM 输出的不确定性
func (n *NoiseHandler) HandleNoise(output LLMOutput, task string) (*VerifiedOutput, error) {
    if output.Confidence >= n.confidenceThresh {
        return &VerifiedOutput{
            Output:   output,
            Verified: false,
            Method:   "confidence_pass",
        }, nil
    }

    verified, err := n.verifyWithRetry(output, task)
    if err == nil {
        return verified, nil
    }

    return &VerifiedOutput{
        Output:   output,
        Verified: false,
        Method:   "degraded",
        Warning:  "低置信度输出，建议人工确认",
    }, nil
}
```

#### 3.3.4 冗余管理（信息论）

**问题**：区分可压缩冗余和必要冗余。

**解决方案**：

```go
// internal/compressor/redundancy.go

package compressor

// RedundancyType 冗余类型
type RedundancyType int

const (
    RedundancyCompressible RedundancyType = iota
    RedundancyNecessary
)

// ClassifyRedundancy 分类冗余
func ClassifyRedundancy(msg Message) RedundancyType {
    if msg.Role == "tool" && msg.LoopNumber < currentLoop-1 {
        return RedundancyCompressible
    }
    if msg.Role == "tool" && isVerification(msg) {
        return RedundancyNecessary
    }
    return RedundancyCompressible
}
```

---

## 4. 缓存命中率保障机制

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

### 4.2 三条铁律

| 铁律 | 说明 |
|------|------|
| **Prefix 只追加不修改** | system prompt + skeleton 永不改写 |
| **历史只压缩不重排** | 旧循环压缩为 summary，追加到稳定区间 |
| **裁剪只在动态区间** | 工具结果裁剪只发生在当前循环 |

### 4.3 缓存命中率

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
│   │   ├── analyzer.go
│   │   ├── oscillation.go
│   │   ├── divergence.go
│   │   └── convergence.go
│   │
│   ├── information/                # P1: 信息论模块
│   │   ├── gain.go
│   │   └── density.go
│   │
│   ├── stagnation/                 # P1: 停滞检测器
│   │   ├── detector.go
│   │   ├── diagnosis.go
│   │   └── metrics.go
│   │
│   ├── exploration/                # P1: 探索触发器
│   │   ├── trigger.go
│   │   ├── temperature.go
│   │   └── tool_roulette.go
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
│   ├── environment/                # P2: 环境感知器
│   │   ├── monitor.go
│   │   ├── watcher.go
│   │   └── file_watcher.go
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
├── pkg/                            # 对外公共 API
│   ├── agent.go
│   ├── options.go
│   └── types.go
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

### Phase 1: P0 基础模块
- [ ] 项目初始化（go mod, 目录结构）
- [ ] Controller 状态机（核心循环）
- [ ] Planner（LLM 规划，JSON 解析）
- [ ] Executor（单工具调用）
- [ ] Reflector（基础反省）
- [ ] DeepSeek Provider（深度优化 prefix-cache）
- [ ] Memory 三层系统
- [ ] Compressor（上下文压缩）
- [ ] Checkpoint（检查点）
- [ ] Budget（成本控制）
- [ ] Trace（可观测性）
- [ ] Human（人机协作）
- [ ] Tools（MCP 工具层）
- **目标**：能跑通 Plan → Execute → Reflect → Replan 完整循环

### Phase 2: P1 核心增强
- [ ] 振荡/发散检测
- [ ] 系统级性能指标
- [ ] 信息增益工具选择
- [ ] 信息密度优化
- [ ] 停滞检测
- [ ] 探索触发
- [ ] 序参量识别
- [ ] 役使原理
- [ ] 积木块
- [ ] 内部模型
- [ ] 多样性管理
- [ ] 混沌边缘
- **目标**：Agent 具备自适应学习能力

### Phase 3: P2 扩展模块
- [ ] 环境感知器
- [ ] 备选路径规划
- [ ] 噪声处理
- [ ] 冗余管理
- **目标**：Agent 具备环境感知和容错能力

### Phase 4: 打磨 + 文档
- [ ] 缓存命中率基准测试
- [ ] 使用示例
- [ ] API 文档
- [ ] README + 贡献指南
- **目标**：可开源发布

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
