package budget

import (
	"fmt"
	"time"
)

// Budget tracks resource consumption
type Budget struct {
	config      *Config
	startTime   time.Time
	loops       int
	tokensUsed  int
	cost        float64
}

// Config contains budget configuration
type Config struct {
	MaxLoops    int
	MaxTokens   int
	MaxCost     float64
	MaxWallTime time.Duration
	WarnAt      float64
}

// DefaultConfig returns default budget configuration
func DefaultConfig() *Config {
	return &Config{
		MaxLoops:    50,
		MaxTokens:   500000,
		MaxCost:     10.0,
		MaxWallTime: 30 * time.Minute,
		WarnAt:      0.8,
	}
}

// NewBudget creates a new budget
func NewBudget(config *Config) *Budget {
	if config == nil {
		config = DefaultConfig()
	}

	return &Budget{
		config:    config,
		startTime: time.Now(),
	}
}

// Reset resets the budget
func (b *Budget) Reset() {
	b.startTime = time.Now()
	b.loops = 0
	b.tokensUsed = 0
	b.cost = 0
}

// ConsumeLoop consumes a loop
func (b *Budget) ConsumeLoop() {
	b.loops++
}

// ConsumeTokens consumes tokens
func (b *Budget) ConsumeTokens(tokens int) {
	b.tokensUsed += tokens
}

// ConsumeCost consumes cost
func (b *Budget) ConsumeCost(cost float64) {
	b.cost += cost
}

// IsExceeded returns true if any limit is exceeded
func (b *Budget) IsExceeded() bool {
	if b.loops >= b.config.MaxLoops {
		return true
	}
	if b.tokensUsed >= b.config.MaxTokens {
		return true
	}
	if b.cost >= b.config.MaxCost {
		return true
	}
	if time.Since(b.startTime) >= b.config.MaxWallTime {
		return true
	}
	return false
}

// IsWarning returns true if any limit is near
func (b *Budget) IsWarning() bool {
	loopRatio := float64(b.loops) / float64(b.config.MaxLoops)
	tokenRatio := float64(b.tokensUsed) / float64(b.config.MaxTokens)
	costRatio := b.cost / b.config.MaxCost

	return loopRatio >= b.config.WarnAt ||
		tokenRatio >= b.config.WarnAt ||
		costRatio >= b.config.WarnAt
}

// Summary returns a summary of the budget
func (b *Budget) Summary() string {
	return fmt.Sprintf("Loops: %d/%d, Tokens: %d/%d, Cost: %.2f/%.2f, Time: %s/%s",
		b.loops, b.config.MaxLoops,
		b.tokensUsed, b.config.MaxTokens,
		b.cost, b.config.MaxCost,
		time.Since(b.startTime).Round(time.Second), b.config.MaxWallTime)
}

// GetLoops returns the number of loops consumed
func (b *Budget) GetLoops() int {
	return b.loops
}

// GetTokensUsed returns the number of tokens consumed
func (b *Budget) GetTokensUsed() int {
	return b.tokensUsed
}

// GetCost returns the cost consumed
func (b *Budget) GetCost() float64 {
	return b.cost
}

// GetDuration returns the duration
func (b *Budget) GetDuration() time.Duration {
	return time.Since(b.startTime)
}
