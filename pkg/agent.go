package pkg

import (
	"context"
	"fmt"
	"time"
)

// Agent is the main entry point for the Zhulong agent
type Agent struct {
	goal    string
	options *Options
}

// Options contains configuration for the agent
type Options struct {
	MaxLoops    int
	MaxTokens   int
	MaxCost     float64
	MaxWallTime time.Duration
	Verbose     bool
}

// Option is a function that configures the agent
type Option func(*Options)

// WithGoal sets the goal for the agent
func WithGoal(goal string) Option {
	return func(o *Options) {
		// Goal is set on the agent, not options
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

// WithVerbose enables verbose output
func WithVerbose(verbose bool) Option {
	return func(o *Options) {
		o.Verbose = verbose
	}
}

// NewAgent creates a new agent
func NewAgent(opts ...Option) (*Agent, error) {
	options := &Options{
		MaxLoops:    50,
		MaxTokens:   500000,
		MaxCost:     10.0,
		MaxWallTime: 30 * time.Minute,
		Verbose:     false,
	}

	for _, opt := range opts {
		opt(options)
	}

	return &Agent{
		options: options,
	}, nil
}

// SetGoal sets the goal for the agent
func (a *Agent) SetGoal(goal string) {
	a.goal = goal
}

// Run runs the agent
func (a *Agent) Run() (*AgentResult, error) {
	if a.goal == "" {
		return nil, fmt.Errorf("goal is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), a.options.MaxWallTime)
	defer cancel()

	startTime := time.Now()

	// TODO: Initialize controller, planner, executor, reflector, etc.
	// TODO: Run the main loop

	_ = ctx

	return &AgentResult{
		Status:   StatusCompleted,
		Answer:   "Agent completed successfully",
		Loops:    0,
		Duration: time.Since(startTime),
	}, nil
}
