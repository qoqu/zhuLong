// Circuit Breaker - 连续无改善自动暂停
package breaker

import (
	"fmt"
	"sync"
	"time"
)

// BreakerState 断路器状态
type BreakerState string

const (
	StateClosed   BreakerState = "closed"    // 正常
	StateOpen     BreakerState = "open"      // 断开
	StateHalfOpen BreakerState = "half_open" // 半开（允许一次尝试）
)

// BreakerConfig 断路器配置
type BreakerConfig struct {
	MaxFailures        int           // 最大连续失败次数（默认3）
	CooldownPeriod     time.Duration // 冷却期（默认5分钟）
	HalfOpenMaxRetries int           // 半开状态最大重试次数（默认1）
}

// DefaultBreakerConfig 默认配置
func DefaultBreakerConfig() *BreakerConfig {
	return &BreakerConfig{
		MaxFailures:        3,
		CooldownPeriod:     5 * time.Minute,
		HalfOpenMaxRetries: 1,
	}
}

// Breaker 断路器
type Breaker struct {
	mu               sync.RWMutex
	config           *BreakerConfig
	failures         int
	consecutiveFails int
	lastFailTime     time.Time
	state            BreakerState
	halfOpenRetries  int
}

// NewBreaker 创建断路器
func NewBreaker(config *BreakerConfig) *Breaker {
	if config == nil {
		config = DefaultBreakerConfig()
	}
	return &Breaker{
		config: config,
		state:  StateClosed,
	}
}

// Allow 检查是否允许执行
func (b *Breaker) Allow() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()

	switch b.state {
	case StateClosed:
		return true
	case StateOpen:
		// 冷却期已过，切换到半开状态
		if time.Since(b.lastFailTime) > b.config.CooldownPeriod {
			return true
		}
		return false
	case StateHalfOpen:
		return b.halfOpenRetries < b.config.HalfOpenMaxRetries
	default:
		return false
	}
}

// Success 记录成功
func (b *Breaker) Success() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.consecutiveFails = 0
	b.failures = 0
	b.halfOpenRetries = 0
	b.state = StateClosed
}

// Failure 记录失败
func (b *Breaker) Failure() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.consecutiveFails++
	b.failures++
	b.lastFailTime = time.Now()

	if b.state == StateHalfOpen {
		b.halfOpenRetries++
		if b.halfOpenRetries >= b.config.HalfOpenMaxRetries {
			b.state = StateOpen
		}
		return
	}

	if b.consecutiveFails >= b.config.MaxFailures {
		b.state = StateOpen
	}
}

// State 获取当前状态
func (b *Breaker) State() BreakerState {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.state
}

// Reset 重置断路器
func (b *Breaker) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.consecutiveFails = 0
	b.failures = 0
	b.halfOpenRetries = 0
	b.state = StateClosed
}

// Stats 返回统计信息
func (b *Breaker) Stats() map[string]interface{} {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return map[string]interface{}{
		"state":            string(b.state),
		"total_failures":   b.failures,
		"consecutive_fails": b.consecutiveFails,
		"last_fail_time":   b.lastFailTime.Format(time.RFC3339),
		"max_failures":     b.config.MaxFailures,
		"cooldown_period":  b.config.CooldownPeriod.String(),
	}
}

// Do 执行受断路器保护的操作
func (b *Breaker) Do(fn func() error) error {
	if !b.Allow() {
		return fmt.Errorf("circuit breaker open: too many failures, cooldown until %s",
			b.lastFailTime.Add(b.config.CooldownPeriod).Format(time.RFC3339))
	}

	err := fn()

	if err != nil {
		b.Failure()
		return fmt.Errorf("circuit breaker: operation failed: %w", err)
	}

	b.Success()
	return nil
}
