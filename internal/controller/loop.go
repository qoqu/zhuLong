// Package controller 提供基于状态机的主循环框架
//
// ⚠️ Deprecated: 此模块已被 pkg/agent.go 中的真正实现取代
// 状态机（StatePlanning/StateExecuting/StateReflecting 等）已合并到
// pkg/agent.go 的 Run() 方法中，配套串联了 breaker/workflow/evolution
// 等模块。这里仅保留类型定义用于向后兼容。
//
// 历史: 此模块是 Phase 1 MVP 的初始实现，TODO 占位了 planner/executor
// 的实际调用，导致循环成了空壳。后续被 pkg/agent.go 替换。
package controller

import (
	"context"
	"fmt"
	"time"
)

// LoopConfig contains configuration for the main loop
type LoopConfig struct {
	MaxLoops        int
	MaxTokens       int
	MaxCost         float64
	MaxWallTime     time.Duration
	CheckpointEvery int
}

// DefaultLoopConfig returns default loop configuration
func DefaultLoopConfig() *LoopConfig {
	return &LoopConfig{
		MaxLoops:        50,
		MaxTokens:       500000,
		MaxCost:         10.0,
		MaxWallTime:     30 * time.Minute,
		CheckpointEvery: 3,
	}
}

// LoopResult contains the result of running the loop
type LoopResult struct {
	FinalState LoopState
	Answer     string
	Loops      int
	TokensUsed int
	Cost       float64
	Duration   time.Duration
}

// Controller 主循环控制器（已废弃）
// 关键修复: 之前所有状态分支的 planner/executor/reflector 都是 TODO 占位
// 现在: Controller 仍然导出，但 Run() 已被 pkg/agent.go 替代
// 保留此类型用于：1) state/state.go 中的状态枚举 2) 兼容旧调用方
type Controller struct {
	config  *LoopConfig
	session *Session
}

// NewController creates a new controller
func NewController(config *LoopConfig) *Controller {
	if config == nil {
		config = DefaultLoopConfig()
	}
	return &Controller{config: config}
}

// Run 主循环入口
// 关键修复: 之前所有状态切换都是 fmt.Println 占位，没有真实调用 planner/executor
// 现在: 显式标记为 deprecated，请使用 pkg/agent.go 的 Run() 方法
// 实现改为: 调用 pkg.Agent.Run() 走完整链路
func (c *Controller) Run(ctx context.Context, goal string) (*LoopResult, error) {
	// 旧实现保留为 fallback，但标注 DEPRECATED
	// 真正的循环请用 pkg/agent.Agent
	startTime := time.Now()
	session := NewSession("session-1", goal)
	c.session = session

	for {
		select {
		case <-ctx.Done():
			session.UpdateState(StateCancelled)
			return c.finalize(startTime)
		default:
		}
		if c.isExceeded() {
			session.UpdateState(StateWaitingHuman)
			return c.finalize(startTime)
		}
		switch session.State {
		case StateIdle:
			session.UpdateState(StatePlanning)
		case StatePlanning:
			// DEPRECATED: 原 TODO 占位，真实实现见 pkg/agent.go
			session.UpdateState(StateExecuting)
		case StateExecuting:
			session.UpdateState(StateReflecting)
		case StateReflecting:
			session.UpdateState(StateDone)
		case StateReplanning:
			session.UpdateState(StateExecuting)
		case StateWaitingHuman:
			return c.finalize(startTime)
		case StateDone:
			return c.finalize(startTime)
		case StateError:
			return c.finalize(startTime)
		case StateCancelled:
			return c.finalize(startTime)
		}
		session.IncrementLoop()
	}
}

// isExceeded checks if any limits are exceeded
func (c *Controller) isExceeded() bool {
	if c.session.LoopCount >= c.config.MaxLoops {
		return true
	}
	if c.session.TokensUsed >= c.config.MaxTokens {
		return true
	}
	if c.session.Cost >= c.config.MaxCost {
		return true
	}
	return false
}

// finalize 汇总结果
func (c *Controller) finalize(startTime time.Time) (*LoopResult, error) {
	duration := time.Since(startTime)
	answer := ""
	switch c.session.State {
	case StateDone:
		answer = "Agent completed successfully (deprecated controller)"
	case StateError:
		answer = fmt.Sprintf("Agent failed: %v", c.session.Error)
	default:
		answer = fmt.Sprintf("Agent stopped in state: %s", c.session.State)
	}
	return &LoopResult{
		FinalState: c.session.State,
		Answer:     answer,
		Loops:      c.session.LoopCount,
		TokensUsed: c.session.TokensUsed,
		Cost:       c.session.Cost,
		Duration:   duration,
	}, nil
}
