package pkg

import "time"

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
func WithDataDir(d string) Option {
	return func(o *Options) {
		o.DataDir = d
	}
}
