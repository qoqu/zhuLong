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

// Controller controls the main agent loop
type Controller struct {
	config  *LoopConfig
	session *Session
	// TODO: Add planner, executor, reflector, memory, etc.
}

// NewController creates a new controller
func NewController(config *LoopConfig) *Controller {
	if config == nil {
		config = DefaultLoopConfig()
	}

	return &Controller{
		config: config,
	}
}

// Run runs the main loop
func (c *Controller) Run(ctx context.Context, goal string) (*LoopResult, error) {
	// Create session
	session := NewSession("session-1", goal)
	c.session = session

	startTime := time.Now()

	// Main loop
	for {
		// Check context cancellation
		select {
		case <-ctx.Done():
			session.UpdateState(StateCancelled)
			return c.finalize(startTime)
		default:
		}

		// Check limits
		if c.isExceeded() {
			session.UpdateState(StateWaitingHuman)
			return c.finalize(startTime)
		}

		// State machine
		switch session.State {
		case StateIdle:
			session.UpdateState(StatePlanning)

		case StatePlanning:
			// TODO: Call planner
			fmt.Println("[Planning] Generating plan...")
			session.UpdateState(StateExecuting)

		case StateExecuting:
			// TODO: Call executor
			fmt.Println("[Executing] Executing step...")
			session.UpdateState(StateReflecting)

		case StateReflecting:
			// TODO: Call reflector
			fmt.Println("[Reflecting] Reflecting on results...")
			session.UpdateState(StateDone)

		case StateReplanning:
			// TODO: Call planner for replanning
			fmt.Println("[Replanning] Replanning...")
			session.UpdateState(StateExecuting)

		case StateWaitingHuman:
			// TODO: Wait for human input
			fmt.Println("[WaitingHuman] Waiting for human input...")
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

// finalize finalizes the session
func (c *Controller) finalize(startTime time.Time) (*LoopResult, error) {
	duration := time.Since(startTime)

	answer := ""
	if c.session.State == StateDone {
		answer = "Agent completed successfully"
	} else if c.session.State == StateError {
		answer = fmt.Sprintf("Agent failed: %v", c.session.Error)
	} else {
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
