package budget

import (
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.MaxLoops != 50 {
		t.Errorf("DefaultConfig().MaxLoops = %v, want 50", config.MaxLoops)
	}
	if config.MaxTokens != 500000 {
		t.Errorf("DefaultConfig().MaxTokens = %v, want 500000", config.MaxTokens)
	}
	if config.MaxCost != 10.0 {
		t.Errorf("DefaultConfig().MaxCost = %v, want 10.0", config.MaxCost)
	}
	if config.MaxWallTime != 30*time.Minute {
		t.Errorf("DefaultConfig().MaxWallTime = %v, want 30m", config.MaxWallTime)
	}
	if config.WarnAt != 0.8 {
		t.Errorf("DefaultConfig().WarnAt = %v, want 0.8", config.WarnAt)
	}
}

func TestNewBudget(t *testing.T) {
	config := &Config{
		MaxLoops:    10,
		MaxTokens:   1000,
		MaxCost:     5.0,
		MaxWallTime: 5 * time.Minute,
		WarnAt:      0.8,
	}

	budget := NewBudget(config)

	if budget == nil {
		t.Error("NewBudget() should not return nil")
	}
	if budget.config != config {
		t.Error("Budget config should match provided config")
	}
}

func TestNewBudget_NilConfig(t *testing.T) {
	budget := NewBudget(nil)

	if budget == nil {
		t.Error("NewBudget() should not return nil")
	}
	if budget.config == nil {
		t.Error("Budget config should not be nil")
	}
	if budget.config.MaxLoops != 50 {
		t.Errorf("Default MaxLoops = %v, want 50", budget.config.MaxLoops)
	}
}

func TestBudget_Reset(t *testing.T) {
	budget := NewBudget(nil)

	budget.ConsumeLoop()
	budget.ConsumeTokens(100)
	budget.ConsumeCost(1.0)

	budget.Reset()

	if budget.GetLoops() != 0 {
		t.Errorf("After Reset(), loops = %v, want 0", budget.GetLoops())
	}
	if budget.GetTokensUsed() != 0 {
		t.Errorf("After Reset(), tokensUsed = %v, want 0", budget.GetTokensUsed())
	}
	if budget.GetCost() != 0 {
		t.Errorf("After Reset(), cost = %v, want 0", budget.GetCost())
	}
}

func TestBudget_ConsumeLoop(t *testing.T) {
	budget := NewBudget(nil)

	budget.ConsumeLoop()
	if budget.GetLoops() != 1 {
		t.Errorf("After ConsumeLoop(), loops = %v, want 1", budget.GetLoops())
	}

	budget.ConsumeLoop()
	if budget.GetLoops() != 2 {
		t.Errorf("After 2x ConsumeLoop(), loops = %v, want 2", budget.GetLoops())
	}
}

func TestBudget_ConsumeTokens(t *testing.T) {
	budget := NewBudget(nil)

	budget.ConsumeTokens(100)
	if budget.GetTokensUsed() != 100 {
		t.Errorf("After ConsumeTokens(100), tokensUsed = %v, want 100", budget.GetTokensUsed())
	}

	budget.ConsumeTokens(200)
	if budget.GetTokensUsed() != 300 {
		t.Errorf("After ConsumeTokens(200), tokensUsed = %v, want 300", budget.GetTokensUsed())
	}
}

func TestBudget_ConsumeCost(t *testing.T) {
	budget := NewBudget(nil)

	budget.ConsumeCost(1.5)
	if budget.GetCost() != 1.5 {
		t.Errorf("After ConsumeCost(1.5), cost = %v, want 1.5", budget.GetCost())
	}

	budget.ConsumeCost(2.5)
	if budget.GetCost() != 4.0 {
		t.Errorf("After ConsumeCost(2.5), cost = %v, want 4.0", budget.GetCost())
	}
}

func TestBudget_IsExceeded(t *testing.T) {
	config := &Config{
		MaxLoops:    5,
		MaxTokens:   100,
		MaxCost:     1.0,
		MaxWallTime: 1 * time.Second,
		WarnAt:      0.8,
	}

	budget := NewBudget(config)

	// Not exceeded initially
	if budget.IsExceeded() {
		t.Error("Should not be exceeded initially")
	}

	// Exceed loops
	budget.ConsumeLoop()
	budget.ConsumeLoop()
	budget.ConsumeLoop()
	budget.ConsumeLoop()
	budget.ConsumeLoop()
	if !budget.IsExceeded() {
		t.Error("Should be exceeded when loops >= MaxLoops")
	}

	// Reset and exceed tokens
	budget.Reset()
	budget.ConsumeTokens(100)
	if !budget.IsExceeded() {
		t.Error("Should be exceeded when tokens >= MaxTokens")
	}

	// Reset and exceed cost
	budget.Reset()
	budget.ConsumeCost(1.0)
	if !budget.IsExceeded() {
		t.Error("Should be exceeded when cost >= MaxCost")
	}
}

func TestBudget_IsWarning(t *testing.T) {
	config := &Config{
		MaxLoops:    10,
		MaxTokens:   1000,
		MaxCost:     10.0,
		MaxWallTime: 10 * time.Minute,
		WarnAt:      0.8,
	}

	budget := NewBudget(config)

	// Not warning initially
	if budget.IsWarning() {
		t.Error("Should not be warning initially")
	}

	// Warning when loops >= 80%
	for i := 0; i < 8; i++ {
		budget.ConsumeLoop()
	}
	if !budget.IsWarning() {
		t.Error("Should be warning when loops >= 80%")
	}

	// Reset and test tokens
	budget.Reset()
	budget.ConsumeTokens(800)
	if !budget.IsWarning() {
		t.Error("Should be warning when tokens >= 80%")
	}

	// Reset and test cost
	budget.Reset()
	budget.ConsumeCost(8.0)
	if !budget.IsWarning() {
		t.Error("Should be warning when cost >= 80%")
	}
}

func TestBudget_Summary(t *testing.T) {
	budget := NewBudget(nil)

	budget.ConsumeLoop()
	budget.ConsumeTokens(100)
	budget.ConsumeCost(1.5)

	summary := budget.Summary()

	if summary == "" {
		t.Error("Summary() should not be empty")
	}
}
