package controller

import (
	"context"
	"testing"
	"time"
)

func TestDefaultLoopConfig(t *testing.T) {
	config := DefaultLoopConfig()

	if config.MaxLoops != 50 {
		t.Errorf("DefaultLoopConfig().MaxLoops = %v, want 50", config.MaxLoops)
	}
	if config.MaxTokens != 500000 {
		t.Errorf("DefaultLoopConfig().MaxTokens = %v, want 500000", config.MaxTokens)
	}
	if config.MaxCost != 10.0 {
		t.Errorf("DefaultLoopConfig().MaxCost = %v, want 10.0", config.MaxCost)
	}
	if config.MaxWallTime != 30*time.Minute {
		t.Errorf("DefaultLoopConfig().MaxWallTime = %v, want 30m", config.MaxWallTime)
	}
	if config.CheckpointEvery != 3 {
		t.Errorf("DefaultLoopConfig().CheckpointEvery = %v, want 3", config.CheckpointEvery)
	}
}

func TestNewController(t *testing.T) {
	config := &LoopConfig{
		MaxLoops:    10,
		MaxTokens:   1000,
		MaxCost:     5.0,
		MaxWallTime: 5 * time.Minute,
	}

	controller := NewController(config)

	if controller.config != config {
		t.Error("Controller config should match provided config")
	}
}

func TestNewController_NilConfig(t *testing.T) {
	controller := NewController(nil)

	if controller.config == nil {
		t.Error("Controller config should not be nil")
	}
	if controller.config.MaxLoops != 50 {
		t.Errorf("Default MaxLoops = %v, want 50", controller.config.MaxLoops)
	}
}

func TestController_Run_EmptyGoal(t *testing.T) {
	controller := NewController(nil)
	ctx := context.Background()

	result, _ := controller.Run(ctx, "")
	// The current implementation doesn't check for empty goal in Run
	// but the session is created with the goal
	if controller.session == nil {
		t.Error("Session should be created")
	}
	if result == nil {
		t.Error("Result should not be nil")
	}
}

func TestController_Run_CancelledContext(t *testing.T) {
	controller := NewController(&LoopConfig{
		MaxLoops:    1,
		MaxTokens:   1000,
		MaxCost:     1.0,
		MaxWallTime: 1 * time.Second,
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	result, err := controller.Run(ctx, "test goal")

	if err != nil {
		t.Errorf("Run() error = %v", err)
	}
	if result.FinalState != StateCancelled {
		t.Errorf("Result.FinalState = %v, want %v", result.FinalState, StateCancelled)
	}
}

func TestController_isExceeded(t *testing.T) {
	config := &LoopConfig{
		MaxLoops:  5,
		MaxTokens: 100,
		MaxCost:   1.0,
	}

	controller := NewController(config)
	controller.session = NewSession("test", "goal")

	// Not exceeded initially
	if controller.isExceeded() {
		t.Error("Should not be exceeded initially")
	}

	// Exceed loops
	controller.session.LoopCount = 5
	if !controller.isExceeded() {
		t.Error("Should be exceeded when loops >= MaxLoops")
	}

	// Reset and exceed tokens
	controller.session.LoopCount = 0
	controller.session.TokensUsed = 100
	if !controller.isExceeded() {
		t.Error("Should be exceeded when tokens >= MaxTokens")
	}

	// Reset and exceed cost
	controller.session.TokensUsed = 0
	controller.session.Cost = 1.0
	if !controller.isExceeded() {
		t.Error("Should be exceeded when cost >= MaxCost")
	}
}
