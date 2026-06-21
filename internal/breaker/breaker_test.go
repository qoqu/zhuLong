package breaker

import (
	"errors"
	"testing"
	"time"
)

func TestNewBreaker(t *testing.T) {
	b := NewBreaker(nil)
	if b.State() != StateClosed {
		t.Errorf("expected closed state, got %v", b.State())
	}
}

func TestBreaker_Allow(t *testing.T) {
	b := NewBreaker(nil)
	if !b.Allow() {
		t.Error("expected allow when state is closed")
	}
}

func TestBreaker_Open(t *testing.T) {
	b := NewBreaker(&BreakerConfig{
		MaxFailures:        2,
		CooldownPeriod:     1 * time.Hour,
		HalfOpenMaxRetries: 1,
	})

	b.Failure()
	b.Failure()

	if b.State() != StateOpen {
		t.Errorf("expected open state, got %v", b.State())
	}

	if b.Allow() {
		t.Error("expected not allow when state is open")
	}
}

func TestBreaker_Success(t *testing.T) {
	b := NewBreaker(nil)

	b.Failure()
	b.Failure()
	b.Success()

	if b.State() != StateClosed {
		t.Errorf("expected closed state after success, got %v", b.State())
	}
}

func TestBreaker_HalfOpen(t *testing.T) {
	// 使用短冷却期，让断路器快速进入半开状态
	b := NewBreaker(&BreakerConfig{
		MaxFailures:        2,
		CooldownPeriod:     1 * time.Millisecond,
		HalfOpenMaxRetries: 1,
	})

	b.Failure()
	b.Failure()

	// 等待冷却期过去
	time.Sleep(2 * time.Millisecond)

	// 应允许一次尝试（半开状态）
	if !b.Allow() {
		t.Error("expected allow during half-open state")
	}

	// 再次失败，应回到打开状态
	b.Failure()
	if b.State() != StateOpen {
		t.Errorf("expected open state after half-open failure, got %v", b.State())
	}
}

func TestBreaker_Do_Success(t *testing.T) {
	b := NewBreaker(nil)
	err := b.Do(func() error {
		return nil
	})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestBreaker_Do_Failure(t *testing.T) {
	b := NewBreaker(&BreakerConfig{
		MaxFailures:    1,
		CooldownPeriod: 1 * time.Hour,
	})

	err := b.Do(func() error {
		return errors.New("test error")
	})
	if err == nil {
		t.Fatal("expected error")
	}

	// 第二次尝试应被阻止
	err = b.Do(func() error {
		return nil
	})
	if err == nil {
		t.Error("expected breaker to block second attempt")
	}
}

func TestBreaker_Reset(t *testing.T) {
	b := NewBreaker(&BreakerConfig{
		MaxFailures:    2,
		CooldownPeriod: 1 * time.Hour,
	})

	b.Failure()
	b.Failure()

	b.Reset()
	if b.State() != StateClosed {
		t.Errorf("expected closed state after reset, got %v", b.State())
	}
}

func TestBreaker_Stats(t *testing.T) {
	b := NewBreaker(nil)
	stats := b.Stats()
	if stats == nil {
		t.Error("expected stats")
	}
	if stats["state"] != string(StateClosed) {
		t.Errorf("expected state=closed, got %v", stats["state"])
	}
}
