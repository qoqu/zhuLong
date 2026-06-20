# Zhulong（烛龙） 详细设计文档

> 版本: v0.1-draft
> 日期: 2026-06-20
> 语言: Go
> 状态: 设计阶段

---

## 1. 项目定位

Zhulong（烛龙） 是一个**通用自主循环 Agent 框架**，核心特性：

- **自主多轮循环**：规划 → 执行 → 反省 → 重新规划，无需每轮人工触发
- **高效上下文管理**：融合 Reasonix 工具结果裁剪 + Ailoom 骨架压缩，最大化 LLM prefix-cache 命中率
- **生产级可靠性**：检查点恢复、成本控制、人机协作断点、完整可观测性

### 1.1 设计目标

| 目标 | 说明 |
|------|------|
| 通用性 | 不绑定特定场景（编程/写作/数据分析），通过工具和 prompt 定制 |
| 高缓存命中率 | 保持 Reasonix 的 prefix-cache 优化水平，上下文布局只追加不重写 |
| 崩溃可恢复 | 任意时刻中断都能从最近检查点恢复 |
| 成本可控 | 多层预算机制，防止无限循环烧 token |
| 可观测 | 完整 trace，每步决策可追溯 |
| 多模型 | 不绑定 DeepSeek，支持 OpenAI / Anthropic / 任意 OpenAI 兼容 API |

### 1.2 参考项目

> **原则：学习设计思路，从零实现。不 fork、不复制、不引入外部 License 依赖。**

| 项目 | 学习内容 |
|------|---------|
| [DeepSeek-Reasonix](https://github.com/esengine/DeepSeek-Reasonix) | MCP 工具协议规范、确定性工具结果裁剪策略、prefix-cache 稳定性设计思路 |
| [Ailoom-Context](https://github.com/EvanLyu-oss/Ailoom-Context) | 骨架压缩结构设计（skeleton + restore 分离）、焦点模式语义、增量压缩思路 |

两个项目均为 MIT License（允许商业使用），但 Zhulong 选择完全独立实现，确保零外部依赖。

---

## 2. 整体架构

```
┌──────────────────────────────────────────────────────────────────┐
│                          Zhulong（烛龙）                               │
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
│  │  │   Budget   │ │   Trace    │ │   Human    │ │Scheduler ││    │
│  │  │  成本控制   │ │  可观测性   │ │  人机协作   │ │ 并行调度  ││    │
│  │  └────────────┘ └────────────┘ └────────────┘ └──────────┘│    │
│  └────────────────────────────────────────────────────────────┘    │
│                                                                    │
│  ┌────────────────────────────────────────────────────────────┐    │
│  │                    Providers 接口层                         │    │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐     │    │
│  │  │  LLM Provider│  │  MCP Client  │  │ Custom Tools │     │    │
│  │  │  多模型适配    │  │  工具协议     │  │  自定义工具   │     │    │
│  │  └──────────────┘  └──────────────┘  └──────────────┘     │    │
│  └────────────────────────────────────────────────────────────┘    │
└──────────────────────────────────────────────────────────────────┘
```

---

## 2.5 工程控制论基础

> 钱学森《工程控制论》(1954) 的核心思想应用于 Agent 系统设计

### 2.5.1 控制论视角下的 Agent 系统

```
┌─────────────────────────────────────────────────────────────────┐
│              Agent 作为闭环控制系统                                │
│                                                                 │
│  ┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐      │
│  │  Goal   │───→│Planner  │───→│Executor │───→│  Plant  │      │
│  │ (目标)   │    │(控制器)  │    │(执行器)  │    │(被控对象)│      │
│  └─────────┘    └─────────┘    └─────────┘    └────┬────┘      │
│       ↑                                            │           │
│       │            ┌─────────┐    ┌─────────┐      │           │
│       └────────────│Reflector│◄───│ Sensor  │◄─────┘           │
│                    │(反馈器)  │    │(传感器)  │                   │
│                    └─────────┘    └─────────┘                   │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  稳定性保障层                                            │    │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐              │    │
│  │  │ 振荡检测  │  │ 发散检测  │  │ 收敛判定  │              │    │
│  │  │ (阻尼器)  │  │ (安全阀)  │  │ (终止器)  │              │    │
│  │  └──────────┘  └──────────┘  └──────────┘              │    │
│  └─────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
```

### 2.5.2 三大控制论原理在 Zhulong 中的应用

#### 原理一：负反馈控制

**工程控制论原话**：负反馈是稳定、抗干扰、高精度控制的根本机制。

**在 Zhulong 中的体现**：

```
传统 Agent（开环控制）：
  Goal → Plan → Execute → Output（无反馈，无法纠正偏差）

Zhulong（闭环负反馈）：
  Goal → Plan → Execute → Output
    ↑                         │
    │      ┌─────────────────┘
    │      ▼
    │   Reflector（计算误差信号）
    │      │
    │      ▼
    └── RePlanner（根据误差纠正）
```

**误差信号定义**：

```go
// 误差 = 目标状态 - 当前状态
type ErrorSignal struct {
    GoalProgress    float64  // 目标完成度 0-1
    QualityDelta    float64  // 质量偏差（预期 vs 实际）
    CostDelta       float64  // 成本偏差（预期 vs 实际）
    DirectionError  string   // 方向性错误（如用错了工具）
}

// 误差信号驱动修正行为
func (r *Reflector) ComputeError(goal string, currentState State) *ErrorSignal {
    return &ErrorSignal{
        GoalProgress:   r.assessProgress(goal, currentState),
        QualityDelta:   r.expectedQuality - r.actualQuality,
        CostDelta:      r.expectedCost - r.actualCost,
        DirectionError: r.detectDirectionError(goal, currentState),
    }
}
```

**阻尼机制**（防止振荡）：

```go
// 阻尼器：限制修正幅度，防止过度修正导致振荡
type Damper struct {
    MaxCorrectionMagnitude float64  // 最大修正幅度（0-1）
    CorrectionHistory      []Correction
    OscillationDetector    *OscillationDetector
}

// 如果检测到振荡（连续3次相似的失败），增大阻尼
func (d *Damper) ApplyCorrection(err *ErrorSignal) *Correction {
    if d.OscillationDetector.IsOscillating() {
        // 振荡检测：增大阻尼，减小修正幅度
        d.MaxCorrectionMagnitude *= 0.5
        return &Correction{
            Magnitude: d.MaxCorrectionMagnitude,
            Strategy:  "conservative",  // 保守策略
        }
    }
    return &Correction{
        Magnitude: math.Min(err.Magnitude(), d.MaxCorrectionMagnitude),
        Strategy:  "normal",
    }
}
```

#### 原理二：系统稳定性理论

**工程控制论原话**：系统稳定性是全书重中之重，区分绝对稳定与相对稳定。

**在 Zhulong 中的体现**：

| 稳定性类型 | 控制论含义 | Zhulong 实现 |
|-----------|-----------|-------------|
| **绝对稳定** | 系统不会发散 | 最大循环次数 + 成本上限 + 超时机制 |
| **相对稳定** | 振荡强弱、响应快慢 | 收敛速度指标 + 振荡检测 + 阻尼控制 |

**稳定性判据**：

```go
// 稳定性分析器
type StabilityAnalyzer struct {
    history []LoopSnapshot
}

// 绝对稳定性：系统是否在安全边界内
func (s *StabilityAnalyzer) IsAbsolutelyStable() bool {
    if len(s.history) < 2 {
        return true
    }
    latest := s.history[len(s.history)-1]

    // 发散检测：连续N次循环进度不增反降
    if s.isDiverging() {
        return false
    }

    // 振荡检测：连续N次循环在相同状态附近摆动
    if s.isOscillating() {
        return false
    }

    // 资源耗尽检测
    if latest.CostUsed >= latest.CostLimit * 0.95 {
        return false
    }

    return true
}

// 相对稳定性：系统收敛速度和振荡程度
func (s *StabilityAnalyzer) RelativeStability() StabilityMetrics {
    return StabilityMetrics{
        ConvergenceRate:  s.convergenceRate(),      // 收敛速率
        OvershootAmount:  s.maxOvershoot(),          // 超调量
        OscillationFreq:  s.oscillationFrequency(),  // 振荡频率
        SettlingTime:     s.settlingTime(),           // 稳定时间
    }
}

// 发散检测：进度是否在倒退
func (s *StabilityAnalyzer) isDiverging() bool {
    if len(s.history) < 3 {
        return false
    }
    // 最近3次循环的完成度递减
    n := len(s.history)
    return s.history[n-3].Progress > s.history[n-2].Progress &&
           s.history[n-2].Progress > s.history[n-1].Progress
}

// 振荡检测：是否在重复相同的失败模式
func (s *StabilityAnalyzer) isOscillating() bool {
    if len(s.history) < 4 {
        return false
    }
    // 检测 A→B→A→B 模式
    n := len(s.history)
    return s.history[n-4].State == s.history[n-2].State &&
           s.history[n-3].State == s.history[n-1].State
}
```

#### 原理三：最优控制与系统综合

**工程控制论原话**：先给定性能指标，再反推控制器结构参数。

**在 Zhulong 中的体现**：

```
传统思路（分析式）：
  已有系统 → 观察行为 → 分析性能

控制论思路（综合式）：
  定义性能指标 → 设计控制器 → 实现目标
```

**性能指标定义**：

```go
// 性能指标（用户可配置）
type PerformanceIndex struct {
    // 动态品质
    MaxCompletionTime  time.Duration  // 最大完成时间
    MinQualityScore    float64        // 最低质量分数

    // 资源约束
    MaxTokenBudget     int            // 最大 token 消耗
    MaxCostBudget      float64        // 最大费用（元）

    // 优先级权重（多目标优化）
    WeightSpeed        float64        // 速度权重
    WeightQuality      float64        // 质量权重
    WeightCost         float64        // 成本权重

    // 约束条件
    HardConstraints    []Constraint   // 硬约束（不可违反）
    SoftConstraints    []Constraint   // 软约束（尽量满足）
}

// 系统综合：根据性能指标反推控制器参数
func SynthesizeController(index PerformanceIndex) ControllerConfig {
    config := ControllerConfig{}

    // 根据速度权重调整并行度
    if index.WeightSpeed > 0.7 {
        config.MaxParallel = 5
    } else if index.WeightSpeed > 0.4 {
        config.MaxParallel = 3
    } else {
        config.MaxParallel = 1
    }

    // 根据成本权重调整阻尼
    if index.WeightCost > 0.7 {
        config.DamperLevel = "aggressive"  // 积极阻尼，减少浪费
    } else {
        config.DamperLevel = "normal"
    }

    // 根据质量权重调整反省频率
    if index.WeightQuality > 0.7 {
        config.ReflectEveryStep = true
    } else {
        config.ReflectEveryNSteps = 3
    }

    return config
}
```

### 2.5.3 时滞、非线性与耦合处理

**工程控制论原话**：专门研究工程普遍存在的滞后环节、饱和摩擦非线性、多变量互相耦合问题。

**在 Zhulong 中的体现**：

| 工程问题 | Agent 对应问题 | 解决方案 |
|---------|---------------|---------|
| **时滞** | LLM 推理延迟、工具调用延迟 | 异步执行 + 超时机制 + 延迟补偿 |
| **非线性** | LLM 输出不确定性、工具结果不可预测 | 置信度评估 + 降级策略 |
| **耦合** | 行动影响未来上下文，上下文影响未来决策 | 耦合度评估 + 解耦策略 |

**时滞处理**：

```go
// 时滞补偿器
type DelayCompensator struct {
    AvgLLMLatency    time.Duration
    AvgToolLatency   time.Duration
    TimeoutBudget    time.Duration
}

// 预估步骤执行时间，决定是否并行
func (d *DelayCompensator) EstimateStepDuration(step Step) time.Duration {
    switch step.Action.Type {
    case "llm_generate":
        return d.AvgLLMLatency
    case "tool_call":
        return d.AvgToolLatency
    default:
        return 0
    }
}

// 如果单步超时风险高，拆分为更小的步骤
func (d *DelayCompensator) ShouldSplit(step Step) bool {
    return d.EstimateStepDuration(step) > d.TimeoutBudget * 0.8
}
```

**非线性处理**：

```go
// 非线性补偿：LLM 输出不确定性处理
type NonlinearCompensator struct {
    ConfidenceThreshold float64
    FallbackStrategy    string
}

// 当 LLM 输出置信度低时，启用降级策略
func (n *NonlinearCompensator) HandleUncertainty(output LLMOutput) *Action {
    if output.Confidence < n.ConfidenceThreshold {
        switch n.FallbackStrategy {
        case "retry":
            // 重试，降低 temperature
            return &Action{Type: "retry", Temperature: output.Temperature * 0.5}
        case "simplify":
            // 简化任务，拆分为更小步骤
            return &Action{Type: "split", Steps: n.splitTask(output.Task)}
        case "human":
            // 请求人工介入
            return &Action{Type: "human_input", Question: output.Ambiguity}
        }
    }
    return &Action{Type: "proceed"}
}
```

**耦合处理**：

```go
// 耦合度评估：评估当前行动对未来状态的影响
type CouplingAnalyzer struct {
    StateHistory []StateSnapshot
}

// 评估行动的耦合影响
func (c *CouplingAnalyzer) AssessCoupling(action Action, currentState State) CouplingReport {
    return CouplingReport{
        // 上下文耦合：此行动会改变多少上下文
        ContextImpact: c.estimateContextChange(action),

        // 工具耦合：此行动是否会影响后续工具调用
        ToolImpact: c.estimateToolDependency(action),

        // 目标耦合：此行动是否会影响其他目标
        GoalImpact: c.estimateGoalDependency(action),

        // 建议：是否需要先执行其他行动来解耦
        Recommendation: c.suggestDecoupling(action, currentState),
    }
}
```

### 2.5.4 整体性系统思维

**工程控制论原话**：反对孤立看待元件，强调系统整体性能优先于单个零部件。

**在 Zhulong 中的体现**：

```
错误思路（孤立优化）：
  - 优化 Planner prompt → 规划更好
  - 优化 Executor → 执行更快
  - 优化 Reflector → 反省更准
  （各模块独立优化，可能互相冲突）

正确思路（系统整体优化）：
  - 定义系统级性能指标
  - 评估闭环整体响应
  - 模块间协调优化
```

**系统级评估**：

```go
// 系统级性能评估（不是单模块评估）
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

// 系统级优化建议
func (s *SystemPerformance) OptimizationSuggestions() []Suggestion {
    var suggestions []Suggestion

    if s.TokenEfficiency < 0.6 {
        suggestions = append(suggestions, Suggestion{
            Module:  "compressor",
            Issue:   "token 效率低",
            Action:  "增强压缩策略，裁剪更多冗余上下文",
        })
    }

    if s.OscillationCount > 3 {
        suggestions = append(suggestions, Suggestion{
            Module:  "damper",
            Issue:   "振荡频繁",
            Action:  "增大阻尼系数，减少修正幅度",
        })
    }

    if s.ReplanCount > 5 {
        suggestions = append(suggestions, Suggestion{
            Module:  "planner",
            Issue:   "重规划过多",
            Action:  "提高初始规划质量，或降低触发重规划的阈值",
        })
    }

    return suggestions
}
```

---

## 3. 核心模块设计

### 3.1 Controller — 状态机驱动的循环控制

Controller 是 Zhulong（烛龙） 的大脑，通过有限状态机（FSM）驱动整个循环。

#### 3.1.1 状态定义

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

#### 3.1.2 状态转移规则

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

**转移条件表：**

| 当前状态 | 下一状态 | 触发条件 |
|---------|---------|---------|
| Idle | Planning | 收到目标（Goal）|
| Planning | Executing | 计划生成成功 |
| Planning | Error | LLM 调用失败 / 生成无效计划 |
| Executing | Reflecting | 当前步骤执行完成 |
| Executing | Executing | 当前步骤完成，还有下一步（并行场景）|
| Executing | Error | 工具调用失败 / 超时 |
| Reflecting | Done | Reflector 判定目标完成 |
| Reflecting | Replanning | Reflector 判定需要调整计划 |
| Reflecting | Executing | Reflector 判定继续执行下一步 |
| Reflecting | Error | Reflector 判定目标无法完成 |
| Replanning | Executing | 新计划生成成功 |
| Replanning | Error | 重规划失败 |
| 任意 | WaitingHuman | 命中人工断点 / 成本超限 |
| 任意 | Cancelled | 用户主动取消 |

#### 3.1.3 核心循环实现

```go
// internal/controller/loop.go

package controller

import (
    "context"
    "fmt"
    "time"

    "zhulong/internal/budget"
    "zhulong/internal/checkpoint"
    "zhulong/internal/trace"
)

type LoopConfig struct {
    MaxLoops    int           // 最大循环次数，默认 50
    MaxTokens   int           // 最大 token 消耗
    MaxCost     float64       // 最大费用（元）
    MaxWallTime time.Duration // 最大运行时间
    CheckpointEvery int       // 每 N 次状态转移保存一次检查点
}

type LoopResult struct {
    FinalState LoopState
    Answer     string        // 最终输出
    Trace      trace.TraceLog
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
                // 所有步骤执行完毕，进入反省
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
            // 阻塞等待人工输入
            input, err := c.human.WaitForInput(ctx, session)
            if err != nil {
                session.State = StateError
                session.Error = err
                break
            }
            // 根据人工输入恢复状态
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

#### 3.1.4 检查点恢复逻辑

```go
// internal/checkpoint/store.go

type Checkpoint struct {
    ID          string
    SessionID   string
    State       controller.LoopState
    Goal        string
    Plan        *Plan
    CurrentStep int
    Memory      MemorySnapshot
    BudgetUsed  BudgetSnapshot
    CreatedAt   time.Time
}

// 持久化到本地文件（JSON 或 SQLite）
type Store interface {
    Save(cp *Checkpoint) error
    LoadLatest() (*Checkpoint, error)
    LoadByID(id string) (*Checkpoint, error)
    List(sessionID string) ([]Checkpoint, error)
    Delete(id string) error
}

// 文件存储实现
type FileStore struct {
    dir string
}

func (fs *FileStore) Save(cp *Checkpoint) error {
    data, _ := json.Marshal(cp)
    path := filepath.Join(fs.dir, cp.SessionID, fmt.Sprintf("%s.json", cp.ID))
    return os.WriteFile(path, data, 0644)
}
```

---

### 3.2 Planner — 规划器

负责将用户目标分解为可执行的步骤序列。

#### 3.2.1 接口定义

```go
// internal/planner/planner.go

package planner

import (
    "context"
)

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
    Estimated   Estimate  // 预估成本
    CreatedAt   time.Time
}

type Step struct {
    ID          string
    Description string
    Action      Action     // 要执行的动作
    DependsOn   []string   // 依赖的步骤 ID
    Parallel    bool       // 是否可并行
    Breakpoint  bool       // 是否需要执行前人工确认
}

type Action struct {
    Type    string                 // "tool_call" | "llm_generate" | "human_input"
    Tool    string                 // 工具名称（tool_call 类型时）
    Params  map[string]interface{} // 工具参数
    Prompt  string                 // LLM 提示（llm_generate 类型时）
}

type Estimate struct {
    TokenEstimate int
    CostEstimate  float64
    DurationEst   time.Duration
}
```

#### 3.2.2 Planner Prompt 设计

```go
// internal/planner/prompts.go

var planSystemPrompt = `你是一个任务规划器。你的职责是将用户目标分解为清晰、可执行的步骤。

## 输出格式
返回 JSON 格式的计划，包含：
- steps: 步骤列表，每个步骤包含 id, description, action, depends_on, parallel
- rationale: 规划理由

## 规划原则
1. 每个步骤应该是原子的、可独立验证的
2. 明确标注步骤间的依赖关系
3. 无依赖的步骤标记 parallel=true
4. 涉及高风险操作（删除、发送、公开）的步骤标记 breakpoint=true
5. 控制总步骤数在合理范围内（默认 3-15 步）
6. 第一步通常是"收集信息/理解上下文"

## 可用工具
{{.AvailableTools}}

## 当前上下文
{{.MemorySummary}}`

var replanSystemPrompt = `你是一个任务重规划器。根据执行历史和反省结果，调整当前计划。

## 当前计划
{{.CurrentPlan}}

## 执行历史
{{.ExecutionHistory}}

## 反省结论
{{.Assessment}}

## 重规划原则
1. 保留已完成且成功的步骤
2. 修复失败步骤的策略
3. 根据新发现调整后续步骤
4. 如果原计划方向错误，可以大幅调整，但要说明理由
5. 避免重复已完成的工作`
```

#### 3.2.3 LLM 调用与解析

```go
// internal/planner/llm_planner.go

package planner

type LLMPlanner struct {
    provider provider.LLMProvider
    config   PlannerConfig
}

type PlannerConfig struct {
    Model          string  // 使用的模型
    MaxSteps       int     // 单次计划最大步骤数
    Temperature    float64
    JSONMode       bool    // 强制 JSON 输出
}

func (p *LLMPlanner) Plan(ctx context.Context, goal string, memory MemoryReader) (*Plan, error) {
    // 1. 构建 prompt
    prompt := p.buildPlanPrompt(goal, memory)

    // 2. 调用 LLM
    resp, err := p.provider.Chat(ctx, &provider.Request{
        Model: p.config.Model,
        Messages: []provider.Message{
            {Role: "system", Content: planSystemPrompt},
            {Role: "user", Content: prompt},
        },
        Temperature: p.config.Temperature,
        JSONMode:    p.config.JSONMode,
    })
    if err != nil {
        return nil, fmt.Errorf("planner LLM call failed: %w", err)
    }

    // 3. 解析 JSON 输出
    plan, err := p.parsePlan(resp.Content)
    if err != nil {
        return nil, fmt.Errorf("failed to parse plan: %w", err)
    }

    // 4. 校验计划合法性
    if err := p.validatePlan(plan); err != nil {
        return nil, fmt.Errorf("invalid plan: %w", err)
    }

    return plan, nil
}
```

---

### 3.3 Executor — 执行器

负责执行计划中的单个步骤。

#### 3.3.1 接口定义

```go
// internal/executor/executor.go

package executor

import (
    "context"
)

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
    Prunable   bool   // 结果是否可裁剪（Reasonix 策略）
    CacheKey   string // 用于判断是否可重新获取
}
```

#### 3.3.2 执行流程

```go
// 执行一个步骤的完整流程

func (e *LLMExecutor) Execute(ctx context.Context, step Step, memory MemoryReader) (*StepResult, error) {
    startTime := time.Now()

    switch step.Action.Type {
    case "tool_call":
        return e.executeToolCall(ctx, step, memory)
    case "llm_generate":
        return e.executeLLMGenerate(ctx, step, memory)
    case "human_input":
        return e.executeHumanInput(ctx, step, memory)
    default:
        return nil, fmt.Errorf("unknown action type: %s", step.Action.Type)
    }
}

func (e *LLMExecutor) executeToolCall(ctx context.Context, step Step, memory MemoryReader) (*StepResult, error) {
    // 1. 解析工具名称和参数
    toolName := step.Action.Tool
    params := step.Action.Params

    // 2. 查找工具
    tool, err := e.toolRegistry.Get(toolName)
    if err != nil {
        return &StepResult{StepID: step.ID, Success: false, Error: err}, nil
    }

    // 3. 执行工具
    output, err := tool.Call(ctx, params)
    if err != nil {
        return &StepResult{StepID: step.ID, Success: false, Error: err}, nil
    }

    // 4. 记录工具调用（用于后续 Reasonix 裁剪决策）
    record := ToolCallRecord{
        ToolName: toolName,
        Input:    params,
        Output:   output,
        Prunable: tool.IsPrunable(), // 工具是否支持结果裁剪
        CacheKey: tool.CacheKey(params), // 用于判断重新获取的可行性
    }

    return &StepResult{
        StepID:    step.ID,
        Success:   true,
        Output:    output,
        ToolCalls: []ToolCallRecord{record},
        Duration:  time.Since(startTime),
    }, nil
}
```

#### 3.3.3 并行调度器

```go
// internal/executor/scheduler.go

package executor

// Scheduler 根据步骤依赖关系决定执行顺序
type Scheduler struct {
    maxParallel int // 最大并行数
}

// Schedule 根据计划生成执行批次
// 返回的每个 batch 内的步骤可以并行执行，batch 间必须串行
func (s *Scheduler) Schedule(plan *Plan) [][]Step {
    // 拓扑排序 + 分层
    // 无依赖的步骤在同一层（可并行）
    // 有依赖的步骤在不同层（必须串行）

    remaining := make(map[string]Step)
    for _, step := range plan.Steps {
        remaining[step.ID] = step
    }

    completed := make(map[string]bool)
    var batches [][]Step

    for len(remaining) > 0 {
        // 找出所有依赖已满足的步骤
        var batch []Step
        for id, step := range remaining {
            allDepsDone := true
            for _, dep := range step.DependsOn {
                if !completed[dep] {
                    allDepsDone = false
                    break
                }
            }
            if allDepsDone {
                batch = append(batch, step)
            }
        }

        if len(batch) == 0 {
            // 存在循环依赖，报错
            break
        }

        // 限制并行数
        if len(batch) > s.maxParallel {
            batch = batch[:s.maxParallel]
        }

        // 标记完成
        for _, step := range batch {
            delete(remaining, step.ID)
            completed[step.ID] = true
        }

        batches = append(batches, batch)
    }

    return batches
}
```

---

### 3.4 Reflector — 反省器（含负反馈控制）

负责评估执行结果，计算误差信号，决定循环的下一步走向。
基于工程控制论的负反馈原理：Reflector 不仅评估，还生成误差信号驱动修正。

#### 3.4.1 接口定义

```go
// internal/reflector/reflector.go

package reflector

import (
    "context"
)

type Reflector interface {
    // Reflect 评估当前状态，返回决策和误差信号
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

    // 负反馈控制新增字段
    ErrorSignal   *ErrorSignal   // 误差信号（驱动修正）
    StabilityInfo *StabilityInfo // 稳定性信息
}

// 误差信号：目标状态与当前状态的偏差
type ErrorSignal struct {
    GoalProgress   float64 // 目标完成度 0-1
    QualityDelta   float64 // 质量偏差（预期 - 实际）
    CostDelta      float64 // 成本偏差（预期 - 实际）
    DirectionError string  // 方向性错误描述
    Magnitude      float64 // 误差总幅度 0-1
}

// 稳定性信息：系统运行状态
type StabilityInfo struct {
    IsOscillating     bool    // 是否在振荡
    IsDiverging       bool    // 是否在发散
    ConvergenceRate   float64 // 收敛速率
    LoopsWithoutProgress int  // 无进展的循环数
}
```

#### 3.4.2 Reflector Prompt

```go
var reflectSystemPrompt = `你是一个任务反省器。评估当前执行进度，决定下一步行动。

## 当前目标
{{.Goal}}

## 当前计划
{{.Plan}}

## 执行历史
{{.ExecutionHistory}}

## 评估维度
1. 目标完成度：当前进度距离目标还有多远？
2. 计划有效性：原计划的策略是否正确？
3. 资源消耗：是否在预算范围内？
4. 风险评估：继续执行是否有风险？

## 输出格式（JSON）
{
  "decision": "complete|continue|replan|fail",
  "reason": "决策理由",
  "confidence": 0.0-1.0,
  "findings": ["发现1", "发现2"],
  "suggestions": ["建议1", "建议2"]
}

## 决策规则
- complete: 目标已完全达成，可以输出最终结果
- continue: 当前进展正常，继续执行下一步
- replan: 发现原计划有重大问题，需要调整
- fail: 确定无法完成目标（工具不足、信息缺失、目标不合理）`
```

---

### 3.5 Memory — 三层记忆系统

#### 3.5.1 架构

```
┌─────────────────────────────────────────────────────────┐
│                    Memory System                         │
│                                                         │
│  ┌─────────────────────────────────────────────────┐    │
│  │  Layer 1: Working Memory (当前循环)              │    │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐        │    │
│  │  │当前计划   │ │步骤结果   │ │工具输出   │        │    │
│  │  │ Plan     │ │ Results  │ │ Outputs  │        │    │
│  │  └──────────┘ └──────────┘ └──────────┘        │    │
│  │  生命周期: 单次循环 | 完整保留，不压缩            │    │
│  └─────────────────────────────────────────────────┘    │
│                         │                               │
│                     循环结束时                            │
│                         ▼                               │
│  ┌─────────────────────────────────────────────────┐    │
│  │  Layer 2: Session Memory (当前会话)              │    │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐        │    │
│  │  │循环摘要   │ │关键决策   │ │反省记录   │        │    │
│  │  │ Summaries│ │ Decisions│ │Assessments│        │    │
│  │  └──────────┘ └──────────┘ └──────────┘        │    │
│  │  生命周期: 整个会话 | 压缩存储（Reasonix裁剪）    │    │
│  └─────────────────────────────────────────────────┘    │
│                         │                               │
│                     会话结束时                            │
│                         ▼                               │
│  ┌─────────────────────────────────────────────────┐    │
│  │  Layer 3: Long-term Memory (跨会话)              │    │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐        │    │
│  │  │用户偏好   │ │项目知识   │ │经验教训   │        │    │
│  │  │ Prefs    │ │ Knowledge│ │ Lessons  │        │    │
│  │  └──────────┘ └──────────┘ └──────────┘        │    │
│  │  生命周期: 永久 | 骨架压缩（Ailoom策略）           │    │
│  └─────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────┘
```

#### 3.5.2 接口定义

```go
// internal/memory/interfaces.go

package memory

// MemoryReader 只读接口，供 Planner/Executor/Reflector 使用
type MemoryReader interface {
    // GetWorkingMemory 获取当前循环的工作记忆
    GetWorkingMemory() *WorkingMemory

    // GetSessionSummary 获取会话摘要（压缩后的历史）
    GetSessionSummary() string

    // GetLongTermMemory 获取长期记忆
    GetLongTermMemory() *LongTermMemory

    // GetContextWindow 获取当前上下文窗口（已压缩，可直接发给 LLM）
    GetContextWindow() []Message

    // Search 搜索记忆（跨层级）
    Search(query string, limit int) []MemoryEntry
}

// MemoryWriter 写入接口，仅供 Controller 使用
type MemoryWriter interface {
    // AddStepResult 添加步骤执行结果到工作记忆
    AddStepResult(step Step, result StepResult)

    // AddAssessment 添加反省评估到工作记忆
    AddAssessment(assessment Assessment)

    // CompressWorkingToSession 将工作记忆压缩到会话记忆
    CompressWorkingToSession() error

    // CompressSessionToLongTerm 将会话记忆压缩到长期记忆
    CompressSessionToLongTerm() error

    // UpdateLongTerm 更新长期记忆
    UpdateLongTerm(entry MemoryEntry) error
}
```

#### 3.5.3 Working Memory 实现

```go
// internal/memory/working.go

package memory

type WorkingMemory struct {
    CurrentPlan   *Plan
    StepResults   []StepRecord
    Assessments   []Assessment
    ToolOutputs   []ToolOutput
    TokenCount    int  // 当前 token 总量
}

type StepRecord struct {
    Step     Step
    Result   StepResult
    Timestamp time.Time
}

type ToolOutput struct {
    CallID   string
    ToolName string
    Input    string
    Output   string
    Tokens   int
    Prunable bool   // Reasonix: 是否可裁剪
}

// GetTokenCount 计算当前工作记忆的 token 总量
func (w *WorkingMemory) GetTokenCount() int {
    // 使用 tokenizer 精确计算
    total := 0
    for _, rec := range w.StepResults {
        total += rec.Result.TokensUsed
    }
    return total
}
```

#### 3.5.4 Session Memory 实现

```go
// internal/memory/session.go

package memory

type SessionMemory struct {
    Goal           string
    LoopSummaries  []LoopSummary   // 每次循环的摘要
    Decisions      []Decision      // 关键决策记录
    Findings       []string        // 累积发现
    TokenCount     int
}

type LoopSummary struct {
    LoopNumber  int
    PlanBrief   string    // 计划摘要（非完整计划）
    StepsDone   int       // 完成步骤数
    Assessment  string    // 反省结论
    Duration    time.Duration
    TokensUsed  int
}

// CompressLoop 将一次循环的工作记忆压缩为摘要
func (s *SessionMemory) CompressLoop(wm *WorkingMemory) {
    summary := LoopSummary{
        LoopNumber:  len(s.LoopSummaries) + 1,
        PlanBrief:   wm.CurrentPlan.Rationale,
        StepsDone:   len(wm.StepResults),
        Assessment:  wm.Assessments[len(wm.Assessments)-1].Reason,
        TokensUsed:  wm.GetTokenCount(),
    }
    s.LoopSummaries = append(s.LoopSummaries, summary)
}
```

---

### 3.6 Compressor — 上下文压缩

这是保持缓存命中率的核心模块。

#### 3.6.1 设计原则

```
┌─────────────────────────────────────────────────────────────┐
│                  Context Layout (铁律)                       │
│                                                             │
│  ┌─────── 固定 Prefix（永不变化）────────────────────────┐   │
│  │  [system prompt]                                      │   │
│  │  [project skeleton]        ← Ailoom 风格，低频刷新     │   │
│  └───────────────────────────────────────────────────────┘   │
│                                                             │
│  ┌─────── 稳定区间（只追加，不修改）──────────────────────┐   │
│  │  [session summary]                                    │   │
│  │  [loop 1 summary]                                     │   │
│  │  [loop 2 summary]                                     │   │
│  │  ...                                                  │   │
│  └───────────────────────────────────────────────────────┘   │
│                                                             │
│  ┌─────── 动态区间（Reasonix 裁剪活跃区）────────────────┐   │
│  │  [current loop: plan + step results]                  │   │
│  │    ↑ 工具结果在此区间内被裁剪                           │   │
│  │  [current turn input]                                 │   │
│  └───────────────────────────────────────────────────────┘   │
│                                                             │
│  ←── 高缓存命中（字节级稳定）──→│←── Reasonix 优化 ──→│      │
└─────────────────────────────────────────────────────────────┘
```

#### 3.6.2 两层压缩策略

```go
// internal/compressor/compressor.go

package compressor

// Compressor 统一压缩接口
type Compressor interface {
    // Prune 执行 Reasonix 风格的工具结果裁剪
    Prune(ctx *ContextWindow) error

    // Skeletonize 执行 Ailoom 风格的骨架压缩
    Skeletonize(input SkeletonInput) (*Skeleton, error)

    // CompressSession 会话级压缩（循环摘要 → 单条 summary）
    CompressSession(session *SessionMemory) (string, error)

    // ShouldRefresh 判断骨架是否需要刷新
    ShouldRefresh(current *Skeleton, projectPath string) bool
}

// Reasonix 风格：确定性工具结果裁剪
type PruneStrategy struct {
    PrunableTools  map[string]bool  // 哪些工具的结果可以裁剪
    MaxAge         int              // 超过 N 轮循环的结果可裁剪
    KeepSignature  bool             // 保留工具调用签名，只裁剪输出
}

// Ailoom 风格：骨架压缩
type SkeletonStrategy struct {
    FocusMode   string  // "symbols" | "imports" | "tree" | "writing-outline"
    Density     string  // "adaptive" | "standard" | "compact"
    MaxTokens   int     // 骨架最大 token 数
    RefreshRate int     // 多少次循环后检查是否需要刷新骨架
}
```

#### 3.6.3 Reasonix 裁剪实现

```go
// internal/compressor/prune.go

package compressor

// PruneToolResults 裁剪可重新获取的工具结果
// 核心逻辑：保留工具调用记录，裁剪输出内容
// 当 LLM 需要该结果时，重新调用工具获取
func (c *ContextCompressor) PruneToolResults(messages []Message, currentLoop int) []Message {
    result := make([]Message, 0, len(messages))

    for _, msg := range messages {
        if msg.Role == "tool" && c.isPrunable(msg, currentLoop) {
            // 裁剪：保留调用签名，替换输出为 placeholder
            pruned := Message{
                Role:    msg.Role,
                Content: fmt.Sprintf("[PRUNED: %s result, %d tokens saved. Re-call if needed.]",
                    msg.ToolName, msg.OrigTokenCount),
                ToolCallID: msg.ToolCallID,
            }
            result = append(result, pruned)
        } else {
            result = append(result, msg)
        }
    }

    return result
}

// isPrunable 判断是否可以裁剪
func (c *ContextCompressor) isPrunable(msg Message, currentLoop int) bool {
    // 1. 工具必须在可裁剪列表中
    if !c.strategy.PrunableTools[msg.ToolName] {
        return false
    }
    // 2. 必须是旧循环的结果（当前循环的不裁剪）
    if msg.LoopNumber >= currentLoop {
        return false
    }
    // 3. 结果必须可以重新获取（有确定性 cache key）
    if msg.CacheKey == "" {
        return false
    }
    return true
}
```

#### 3.6.4 骨架压缩实现（参考 Ailoom）

```go
// internal/compressor/skeleton.go

package compressor

// Skeleton 项目骨架
type Skeleton struct {
    Version     string            `json:"version"`      // "zhulong-skl.v1"
    FocusMode   string            `json:"focus_mode"`
    Density     string            `json:"density"`
    Text        string            `json:"skeleton_text"` // AI 可读的骨架文本
    Manifest    SkeletonManifest  `json:"manifest"`      // 结构元数据
    Fingerprint string            `json:"fingerprint"`   // 项目指纹，用于判断是否需要刷新
    TokenCount  int               `json:"token_count"`
    CreatedAt   time.Time         `json:"created_at"`
}

type SkeletonManifest struct {
    FileCount    int               `json:"file_count"`
    DirCount     int               `json:"dir_count"`
    SymbolCount  int               `json:"symbol_count"`
    TotalSize    int64             `json:"total_size"`
    FileHashes   map[string]string `json:"file_hashes"` // 路径 → hash，用于增量更新
}

// GenerateSkeleton 生成项目骨架
func (c *ContextCompressor) GenerateSkeleton(projectPath string, opts SkeletonOptions) (*Skeleton, error) {
    // 1. 扫描项目结构
    structure := c.scanProject(projectPath, opts.IgnorePatterns)

    // 2. 根据 FocusMode 生成骨架
    var text string
    switch opts.FocusMode {
    case "symbols":
        text = c.generateSymbolSkeleton(structure)
    case "tree":
        text = c.generateTreeSkeleton(structure)
    case "writing-outline":
        text = c.generateWritingOutline(structure)
    default: // "full"
        text = c.generateFullSkeleton(structure)
    }

    // 3. 根据 Density 裁剪
    text = c.applyDensity(text, opts.Density, opts.MaxTokens)

    // 4. 计算指纹
    fingerprint := c.computeFingerprint(structure)

    return &Skeleton{
        Version:     "zhulong-skl.v1",
        FocusMode:   opts.FocusMode,
        Density:     opts.Density,
        Text:        text,
        Manifest:    structure.Manifest,
        Fingerprint: fingerprint,
        TokenCount:  c.countTokens(text),
        CreatedAt:   time.Now(),
    }, nil
}

// ShouldRefresh 检查骨架是否需要刷新
func (c *ContextCompressor) ShouldRefresh(current *Skeleton, projectPath string) bool {
    // 计算当前项目的指纹
    currentFingerprint := c.computeProjectFingerprint(projectPath)
    // 指纹不同 → 项目结构变了 → 需要刷新
    return current.Fingerprint != currentFingerprint
}
```

#### 3.6.5 上下文组装

```go
// internal/compressor/assembler.go

package compressor

// AssembleContext 将所有组件组装成最终的上下文窗口
// 这是整个压缩系统的核心入口
func (c *ContextCompressor) AssembleContext(
    systemPrompt string,
    skeleton *Skeleton,
    session *SessionMemory,
    working *WorkingMemory,
    currentInput string,
    maxTokens int,
) []Message {
    var messages []Message
    usedTokens := 0

    // ──── 固定 Prefix ────
    // 1. System prompt（永不变化）
    messages = append(messages, Message{Role: "system", Content: systemPrompt})
    usedTokens += c.countTokens(systemPrompt)

    // 2. Project skeleton（低频刷新，保持 prefix 稳定）
    if skeleton != nil {
        skeletonMsg := Message{
            Role:    "system",
            Content: fmt.Sprintf("## 项目骨架\n\n%s", skeleton.Text),
        }
        messages = append(messages, skeletonMsg)
        usedTokens += skeleton.TokenCount
    }

    // ──── 稳定区间（只追加） ────
    // 3. Session summary
    if session != nil && len(session.LoopSummaries) > 0 {
        summary := c.buildSessionSummary(session)
        summaryMsg := Message{Role: "system", Content: summary}
        messages = append(messages, summaryMsg)
        usedTokens += c.countTokens(summary)
    }

    // ──── 动态区间（Reasonix 裁剪活跃区） ────
    // 4. Current loop working memory
    remainingTokens := maxTokens - usedTokens - c.countTokens(currentInput)

    // 先添加完整的当前循环内容
    workingMessages := c.buildWorkingMessages(working)
    workingTokens := c.countMessagesTokens(workingMessages)

    if workingTokens > remainingTokens {
        // 需要裁剪：对旧循环的工具结果执行 Reasonix 裁剪
        workingMessages = c.PruneToolResults(workingMessages, working.CurrentLoop)
        workingTokens = c.countMessagesTokens(workingMessages)

        // 如果裁剪后仍然超限，进一步压缩旧循环摘要
        if workingTokens > remainingTokens {
            messages = c.compressSessionPrefix(messages)
        }
    }

    messages = append(messages, workingMessages...)

    // 5. Current turn input
    messages = append(messages, Message{Role: "user", Content: currentInput})

    return messages
}
```

---

### 3.7 Budget — 成本控制

```go
// internal/budget/tracker.go

package budget

type Budget struct {
    MaxLoops    int
    MaxTokens   int
    MaxCost     float64       // 单位：元
    MaxWallTime time.Duration

    // 运行时状态
    currentLoops  int
    currentTokens int
    currentCost   float64
    startTime     time.Time
}

type BudgetConfig struct {
    MaxLoops     int           `yaml:"max_loops"`     // 默认 50
    MaxTokens    int           `yaml:"max_tokens"`    // 默认 500000
    MaxCost      float64       `yaml:"max_cost"`      // 默认 10.0 元
    MaxWallTime  time.Duration `yaml:"max_wall_time"` // 默认 30 分钟
    WarnAt       float64       `yaml:"warn_at"`       // 预警阈值，默认 0.8 (80%)
}

func (b *Budget) Consume(tokens int, cost float64) {
    b.currentTokens += tokens
    b.currentCost += cost
}

func (b *Budget) IsExceeded() bool {
    return b.currentLoops >= b.MaxLoops ||
           b.currentTokens >= b.MaxTokens ||
           b.currentCost >= b.MaxCost ||
           time.Since(b.startTime) >= b.MaxWallTime
}

func (b *Budget) IsWarning() bool {
    return float64(b.currentTokens)/float64(b.MaxTokens) >= b.WarnAt ||
           b.currentCost/b.MaxCost >= b.WarnAt
}

func (b *Budget) Summary() BudgetSummary {
    return BudgetSummary{
        LoopsUsed:     b.currentLoops,
        LoopsMax:      b.MaxLoops,
        TokensUsed:    b.currentTokens,
        TokensMax:     b.MaxTokens,
        CostUsed:      b.currentCost,
        CostMax:       b.MaxCost,
        WallTimeUsed:  time.Since(b.startTime),
        WallTimeMax:   b.MaxWallTime,
    }
}
```

---

### 3.8 Trace — 可观测性

```go
// internal/trace/trace.go

package trace

type TraceLog struct {
    SessionID string
    Entries   []TraceEntry
    StartTime time.Time
}

type TraceEntry struct {
    Timestamp time.Time
    Loop      int
    Phase     string                 // "planning" | "executing" | "reflecting" | "replanning"
    Event     string                 // "plan_created" | "step_executed" | "reflected" | ...
    Data      map[string]interface{} // 事件数据
    Duration  time.Duration
    Tokens    int
}

// Logger 接口
type Logger interface {
    Start(sessionID string)
    Log(phase string, event string, data ...interface{})
    LogError(phase string, err error)
    GetTrace() *TraceLog
    Export(format string) ([]byte, error) // "json" | "text" | "markdown"
}

// 输出示例：
// ┌─ Zhulong（烛龙） Trace ─────────────────────────────────┐
// │ Session: abc123 | Started: 2026-06-20 22:00:00    │
// ├───────────────────────────────────────────────────┤
// │ [Loop 1] Planning                                 │
// │   → Plan created: 5 steps (1.2s, 342 tokens)     │
// │ [Loop 1] Executing Step 1/5                       │
// │   → Tool: search_file("*.go") → 15 files (0.3s)  │
// │ [Loop 1] Reflecting                               │
// │   → Decision: continue (confidence: 0.9)          │
// │ [Loop 1] Executing Step 2/5                       │
// │   → Tool: read_file("main.go") → 234 lines (0.2s)│
// │ [Loop 1] Reflecting                               │
// │   → Decision: replan (new info discovered)        │
// │ [Loop 2] Replanning                               │
// │   → New plan: 4 steps (1.1s, 289 tokens)         │
// │ ...                                               │
// ├───────────────────────────────────────────────────┤
// │ Stats: 3 loops | 12,450 tokens | ¥0.23 | 45.2s   │
// └───────────────────────────────────────────────────┘
```

---

### 3.9 Human — 人机协作断点

```go
// internal/human/breakpoint.go

package human

type BreakpointType int

const (
    BPBeforeExecute   BreakpointType = iota // 每次执行前暂停
    BPCostExceed                             // 成本超限时暂停
    BPOnError                                // 出错时暂停
    BPOnReplan                               // 重规划时暂停
    BPOnReflectFail                          // 反省判定失败时暂停
    BPCustom                                 // 自定义条件
)

type Breakpoint struct {
    Type      BreakpointType
    Condition string   // 自定义条件表达式（BPCustom 类型时）
    Message   string   // 暂停时显示给用户的消息
}

type HumanInput struct {
    Action  string                 // "continue" | "abort" | "modify_plan" | "provide_info"
    Content string                 // 用户输入的内容
    Data    map[string]interface{} // 附加数据（如修改后的计划）
}

// ShouldPause 判断是否需要暂停
func (h *HumanBreakpoint) ShouldPause(step Step, result StepResult) bool {
    for _, bp := range h.breakpoints {
        switch bp.Type {
        case BPBeforeExecute:
            if step.Breakpoint {
                return true
            }
        case BPOnError:
            if !result.Success {
                return true
            }
        case BPCustom:
            if h.evaluateCondition(bp.Condition, step, result) {
                return true
            }
        }
    }
    return false
}
```

---

### 3.10 Tools — 工具抽象层

```go
// internal/tools/interface.go

package tools

// Tool 统一工具接口
type Tool interface {
    // Name 工具名称
    Name() string

    // Description 工具描述（用于 LLM 理解）
    Description() string

    // Schema 参数 JSON Schema
    Schema() map[string]interface{}

    // Call 执行工具
    Call(ctx context.Context, params map[string]interface{}) (string, error)

    // IsPrunable 结果是否可裁剪（Reasonix 策略）
    IsPrunable() bool

    // CacheKey 根据参数生成缓存 key（相同 key = 可重新获取）
    CacheKey(params map[string]interface{}) string
}

// Registry 工具注册表
type Registry struct {
    tools map[string]Tool
}

func (r *Registry) Register(tool Tool) {
    r.tools[tool.Name()] = tool
}

func (r *Registry) Get(name string) (Tool, error) {
    tool, ok := r.tools[name]
    if !ok {
        return nil, fmt.Errorf("tool not found: %s", name)
    }
    return tool, nil
}

// ToToolDescriptions 生成工具描述列表（用于 Planner prompt）
func (r *Registry) ToToolDescriptions() string {
    // 生成类似 Reasonix 的工具列表格式
}
```

---

### 3.11 Provider — LLM 提供商抽象

```go
// internal/provider/interface.go

package provider

type LLMProvider interface {
    Chat(ctx context.Context, req *Request) (*Response, error)
    StreamChat(ctx context.Context, req *Request) (<-chan StreamChunk, error)
}

type Request struct {
    Model       string
    Messages    []Message
    Tools       []ToolDefinition
    Temperature float64
    MaxTokens   int
    JSONMode    bool
    Stream      bool
}

type Message struct {
    Role       string // "system" | "user" | "assistant" | "tool"
    Content    string
    ToolCalls  []ToolCall
    ToolCallID string
}

type Response struct {
    Content      string
    ToolCalls    []ToolCall
    TokensUsed   TokenUsage
    FinishReason string
}

type TokenUsage struct {
    PromptTokens     int
    CompletionTokens int
    TotalTokens      int
    CachedTokens     int  // prefix-cache 命中的 token 数
}

// Provider 实现列表
// - DeepSeekProvider  (复用 Reasonix 的实现)
// - OpenAIProvider
// - AnthropicProvider
// - OpenAICompatProvider (通用 OpenAI 兼容 API)
```

---

## 4. 缓存命中率保障机制

### 4.1 为什么 Zhulong（烛龙） 能保持高缓存命中率

```
传统 Agent（缓存命中率低）:
请求 1: [system][user][tool result 1234 tokens][assistant]
请求 2: [system][user][tool result 1234 tokens][assistant][user][NEW tool result 567 tokens]
                                              ↑ 前缀相同，但中间插入了大块工具结果
                                              请求 2 的 prefix 比请求 1 长
                                              → 缓存部分命中

Zhulong（烛龙）（缓存命中率高）:
请求 1: [system][skeleton][summary][loop1 pruned][current input]
请求 2: [system][skeleton][summary][loop1 pruned][loop2 pruned][current input]
         ↑─────────── 完全相同的 prefix ──────────↑
         → 100% 缓存命中

关键差异：
1. system + skeleton 永不变化 → prefix 稳定
2. 旧循环的工具结果被裁剪 → 中间部分体积小且稳定
3. 新内容只追加在末尾 → prefix 不会被覆盖
```

### 4.2 缓存命中率保障的三条铁律

| 铁律 | 说明 | 实现位置 |
|------|------|---------|
| **Prefix 只追加不修改** | system prompt + skeleton 永不改写 | `Compressor.AssembleContext()` |
| **历史只压缩不重排** | 旧循环压缩为 summary，追加到稳定区间 | `Memory.CompressWorkingToSession()` |
| **裁剪只在动态区间** | Reasonix 裁剪只发生在当前循环的工具结果 | `Compressor.PruneToolResults()` |

### 4.3 与 Reasonix 的缓存效果对比

```
Reasonix（单轮对话）:
  输入: [system][tools][user msg][tool results...][assistant]
  优化: 裁剪 tool results → prefix 稳定
  缓存命中率: ~80-90%

Zhulong（烛龙）（自主循环）:
  输入: [system][skeleton][session summary][loop history][current loop][input]
  优化:
    - skeleton 保持 prefix 稳定（Ailoom 策略）
    - 裁剪旧循环 tool results（Reasonix 策略）
    - 压缩旧循环为 summary（自摘要）
  缓存命中率: ≥ Reasonix（80-90%）

核心原理：
  Zhulong 的 prefix 布局 = Reasonix 的 prefix 布局
  [system prompt] [skeleton/tools] 这段永远不变
  → 缓存命中率不可能低于 Reasonix

Ailoom 骨架压缩的加成效果：
  - 骨架比原始项目上下文更小、更确定性
  - prefix 更稳定 → 更多内容落在缓存窗口内
  - 长循环场景下，缓存命中率可能高于纯 Reasonix
```

---

## 5. 配置系统

```yaml
# config/default.yaml

# LLM 配置
llm:
  provider: "deepseek"          # deepseek | openai | anthropic | openai_compat
  model: "deepseek-chat"
  api_key: "${DEEPSEEK_API_KEY}"
  base_url: ""                  # 自定义 API 地址
  temperature: 0.7
  max_tokens: 4096

# 循环控制
loop:
  max_loops: 50
  max_wall_time: "30m"
  checkpoint_every: 3           # 每 3 次状态转移保存检查点
  checkpoint_dir: ".zhulong/checkpoints"

# 成本控制
budget:
  max_tokens: 500000
  max_cost: 10.0                # 元
  warn_at: 0.8                  # 80% 时预警

# 规划器
planner:
  max_steps: 15
  allow_replan: true
  max_replans: 5                # 最大重规划次数

# 执行器
executor:
  max_parallel: 3               # 最大并行步骤数
  tool_timeout: "30s"           # 单个工具调用超时

# 反省器
reflector:
  confidence_threshold: 0.7     # 置信度低于此值触发重规划
  auto_fail_threshold: 0.3      # 置信度低于此值判定失败

# 压缩
compressor:
  # Reasonix 风格裁剪
  prune:
    enabled: true
    max_age: 2                  # 超过 2 个循环的工具结果可裁剪
    keep_signature: true        # 保留调用签名

  # Ailoom 风格骨架
  skeleton:
    enabled: true
    focus_mode: "auto"          # auto | symbols | tree | writing-outline
    density: "adaptive"         # adaptive | standard | compact
    max_tokens: 2000
    refresh_check_interval: 5   # 每 5 次循环检查是否需要刷新

  # 会话压缩
  session:
    compress_after_loops: 3     # 超过 3 次循环后压缩旧循环为 summary

# 人机协作
human:
  enabled: true
  default_breakpoints:
    - type: "on_error"
    - type: "on_replan"
    - type: "cost_exceed"
  interactive: true             # 是否在终端交互

# 可观测性
trace:
  enabled: true
  output_dir: ".zhulong/traces"
  format: "markdown"            # json | text | markdown
  verbose: false

# 工具
tools:
  mcp_config: "mcp.json"       # MCP 服务器配置
  builtin:
    - "read_file"
    - "write_file"
    - "search_file"
    - "execute_command"
    - "web_search"

# 项目骨架
project:
  path: "."                     # 项目根目录
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
│   ├── controller/                 # 状态机 + 循环控制
│   │   ├── fsm.go                  # 状态机定义 + 转移逻辑
│   │   ├── loop.go                 # 主循环 Run()
│   │   └── session.go              # 会话管理
│   │
│   ├── planner/                    # 规划器
│   │   ├── planner.go              # Planner 接口 + LLM 实现
│   │   ├── prompts.go              # 规划 prompt 模板
│   │   ├── parser.go               # JSON 计划解析 + 校验
│   │   └── replan.go               # 重规划逻辑
│   │
│   ├── executor/                   # 执行器
│   │   ├── executor.go             # Executor 接口 + 实现
│   │   ├── scheduler.go            # 并行调度（拓扑排序）
│   │   └── retry.go                # 工具调用重试策略
│   │
│   ├── reflector/                  # 反省器（含负反馈控制）
│   │   ├── reflector.go            # Reflector 接口 + 实现
│   │   ├── prompts.go              # 反省 prompt 模板
│   │   ├── assessment.go           # 评估结果解析 + 误差信号
│   │   └── damper.go               # 阻尼器（防振荡）
│   │
│   ├── stability/                  # 稳定性分析器（工程控制论）
│   │   ├── analyzer.go             # 稳定性分析主逻辑
│   │   ├── oscillation.go          # 振荡检测
│   │   ├── divergence.go           # 发散检测
│   │   └── convergence.go          # 收敛判定
│   │
│   ├── memory/                     # 三层记忆系统
│   │   ├── interfaces.go           # MemoryReader / MemoryWriter 接口
│   │   ├── working.go              # Working Memory
│   │   ├── session.go              # Session Memory
│   │   ├── longterm.go             # Long-term Memory
│   │   └── store.go                # 持久化（JSON/SQLite）
│   │
│   ├── compressor/                 # 上下文压缩
│   │   ├── compressor.go           # Compressor 接口 + 主逻辑
│   │   ├── prune.go                # Reasonix 风格裁剪
│   │   ├── skeleton.go             # Ailoom 风格骨架压缩
│   │   ├── assembler.go            # 上下文组装（最终输出）
│   │   └── tokenizer.go            # Token 计数工具
│   │
│   ├── checkpoint/                 # 检查点持久化
│   │   ├── checkpoint.go           # Checkpoint 数据结构
│   │   ├── store.go                # 文件存储实现
│   │   └── restore.go              # 恢复逻辑
│   │
│   ├── budget/                     # 成本控制
│   │   ├── budget.go               # Budget 主逻辑
│   │   └── pricing.go              # 各模型定价表
│   │
│   ├── trace/                      # 可观测性
│   │   ├── logger.go               # Trace Logger 实现
│   │   ├── formatter.go            # 输出格式化（JSON/Text/Markdown）
│   │   └── export.go               # 导出功能
│   │
│   ├── human/                      # 人机协作
│   │   ├── breakpoint.go           # 断点管理
│   │   ├── terminal.go             # 终端交互实现
│   │   └── input.go                # 用户输入解析
│   │
│   ├── tools/                      # 工具抽象层
│   │   ├── interface.go            # Tool 接口 + Registry
│   │   ├── mcp.go                  # MCP 协议适配
│   │   ├── builtin.go              # 内置工具（read/write/search/exec）
│   │   └── custom.go               # 自定义工具注册
│   │
│   └── provider/                   # LLM 提供商
│       ├── interface.go            # LLMProvider 接口
│       ├── deepseek.go             # DeepSeek 实现
│       ├── openai.go               # OpenAI 实现
│       ├── anthropic.go            # Anthropic 实现
│       ├── openai_compat.go        # OpenAI 兼容通用实现
│       └── registry.go             # Provider 注册 + 选择
│
├── pkg/                            # 对外公共 API
│   ├── agent.go                    # Agent 构造函数 + Run()
│   ├── options.go                  # 函数式选项
│   └── types.go                    # 公共类型导出
│
├── config/
│   ├── default.yaml                # 默认配置
│   └── schema.json                 # 配置 JSON Schema
│
├── examples/                       # 使用示例
│   ├── coding/main.go              # 编程场景
│   ├── writing/main.go             # 写作场景
│   └── analysis/main.go            # 数据分析场景
│
├── testing/                        # 测试
│   ├── unit/                       # 单元测试
│   ├── integration/                # 集成测试
│   └── benchmark/                  # 缓存命中率基准测试
│
├── docs/
│   ├── design.md                   # 本文档
│   ├── api.md                      # API 文档
│   └── architecture.md             # 架构图
│
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

---

## 7. 开发路线图

### Phase 1: 最小可运行原型（MVP）
- [ ] 项目初始化（go mod, 目录结构）
- [ ] Controller 状态机（核心循环）
- [ ] Planner（LLM 规划，JSON 解析）
- [ ] Executor（单工具调用）
- [ ] Reflector（基础反省）
- [ ] DeepSeek Provider
- [ ] 基础配置系统
- **目标**：能跑通 Plan → Execute → Reflect → Replan 完整循环

### Phase 2: 上下文优化
- [ ] Memory 三层系统实现
- [ ] Reasonix 工具结果裁剪
- [ ] 上下文组装器
- [ ] Token 计数 + 预算控制
- **目标**：缓存命中率可测量，预算控制生效

### Phase 3: 生产级特性
- [ ] 检查点持久化 + 恢复
- [ ] Trace 可观测性
- [ ] 人机协作断点
- [ ] 并行调度器
- [ ] Ailoom 骨架压缩
- **目标**：可崩溃恢复，可调试，可人工介入

### Phase 4: 多模型 + 工具生态
- [ ] OpenAI / Anthropic Provider
- [ ] MCP 工具协议适配
- [ ] 内置工具集
- [ ] 自定义工具注册
- **目标**：不绑定特定 LLM，工具生态可扩展

### Phase 5: 打磨 + 文档
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
| 自主循环死循环 | 无限消耗 token | 最大循环次数 + 成本上限 + 超时机制 |
| 反省器过于保守 | 频繁触发重规划 | 置信度阈值可调 + 重规划次数上限 |
| Ailoom 骨架压缩 Go 移植难度 | 开发周期拉长 | Phase 2 先跳过骨架压缩，Phase 3 再做 |
| 多模型 Prompt 效果不一致 | 不同 LLM 规划质量差异大 | 针对每个模型定制 prompt + 测试用例 |
| 并行执行状态管理复杂 | 死锁、竞态 | 单线程优先，Phase 3 再加并行 |

---

## 附录 A: 与 Reasonix 的学习参考清单

> **策略：学习设计思路，从零实现。不 fork、不复制、不引入外部 License 依赖。**

| 模块 | Reasonix 路径 | 学习内容 | Zhulong 实现方式 |
|------|--------------|---------|-----------------|
| MCP 工具协议 | `internal/mcp/` | MCP 协议规范和消息格式 | 按 MCP 规范自行实现 `tools/mcp.go` |
| 工具结果裁剪逻辑 | `internal/session/pruning/` | 裁剪策略：保留调用签名，裁剪可重新获取的输出 | 自行实现 `compressor/prune.go`，思路相同但代码全新 |
| Prefix-cache 配置 | `internal/llm/config.go` | prefix 稳定性设计思路 | 自行设计 `provider/` 的上下文布局策略 |
| 配置管理 | `config/` | YAML 配置结构参考 | 自行设计配置 schema |
| DeepSeek API 调用 | `internal/llm/deepseek.go` | API 调用格式和错误处理 | 按 DeepSeek API 文档自行实现 |

## 附录 B: 与 Ailoom-Context 的学习参考清单

> **策略：学习算法思路，Go 从零实现。不复制 Python 代码。**

| 模块 | Ailoom 路径 | 学习内容 | Zhulong 实现方式 |
|------|------------|---------|-----------------|
| 骨架生成策略 | `ailoom_core/compress.py` | 骨架结构设计思路（skeleton + restore package 分离） | 自行设计骨架格式，Go 实现 `compressor/skeleton.go` |
| 焦点模式 | `ailoom_core/focus_modes/` | 各焦点模式的语义定义 | 自行定义焦点模式的提取规则 |
| 增量压缩 | `ailoom_core/incremental.py` | 增量 diff 策略思路 | 自行实现增量更新逻辑 |
| 项目扫描 | `ailoom_core/scan.py` | 目录扫描和过滤策略 | Go 标准库实现文件扫描 |
| 配置预设 | `config/presets/` | 预设场景的参数设计 | 自行设计预设配置 |
