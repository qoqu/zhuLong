package compressor

import (
	"testing"
	"time"
)

func TestNewPruner(t *testing.T) {
	pruner := NewPruner(nil)
	if pruner == nil {
		t.Fatal("NewPruner returned nil")
	}
	if pruner.config == nil {
		t.Fatal("config is nil")
	}
}

func TestDefaultPruningConfig(t *testing.T) {
	config := DefaultPruningConfig()
	if config == nil {
		t.Fatal("DefaultPruningConfig returned nil")
	}
	if config.Strategy != PruningStrategyDeterministic {
		t.Errorf("expected strategy deterministic, got %s", config.Strategy)
	}
	if config.MaxAge != 3 {
		t.Errorf("expected max age 3, got %d", config.MaxAge)
	}
	if !config.KeepSignature {
		t.Error("expected keep signature to be true")
	}
	if !config.Deterministic {
		t.Error("expected deterministic to be true")
	}
	if !config.CacheTTLAware {
		t.Error("expected cache TTL aware to be true")
	}
}

func TestPrunerPruneDeterministic(t *testing.T) {
	config := &PruningConfig{
		Strategy:       PruningStrategyDeterministic,
		MaxAge:         2,
		KeepSignature:  true,
		Deterministic:  true,
		CacheTTLAware:  true,
		PruneThreshold: 24 * time.Hour,
	}
	pruner := NewPruner(config)

	messages := []PrunableMessage{
		{
			ID:         "1",
			Role:       "user",
			Content:    "Hello",
			LoopNumber: 1,
			Tokens:     10,
			Prunable:   false,
		},
		{
			ID:         "2",
			Role:       "tool",
			Content:    "Tool result 1",
			ToolName:   "read_file",
			LoopNumber: 1,
			Tokens:     100,
			Prunable:   true,
		},
		{
			ID:         "3",
			Role:       "tool",
			Content:    "Tool result 2",
			ToolName:   "write_file",
			LoopNumber: 5,
			Tokens:     200,
			Prunable:   true,
		},
	}

	result := pruner.Prune(messages, 5)
	if result == nil {
		t.Fatal("Prune returned nil")
	}

	// User message should be kept
	if result.KeptCount < 1 {
		t.Error("expected at least 1 kept message")
	}

	// Old tool message should be pruned
	if result.PrunedCount < 1 {
		t.Error("expected at least 1 pruned message")
	}
}

func TestPrunerPruneByAge(t *testing.T) {
	config := &PruningConfig{
		Strategy: PruningStrategyAge,
		MaxAge:   2,
	}
	pruner := NewPruner(config)

	messages := []PrunableMessage{
		{
			ID:         "1",
			Role:       "tool",
			Content:    "Old result",
			LoopNumber: 1,
			Tokens:     100,
			Prunable:   true,
		},
		{
			ID:         "2",
			Role:       "tool",
			Content:    "New result",
			LoopNumber: 5,
			Tokens:     200,
			Prunable:   true,
		},
	}

	result := pruner.Prune(messages, 5)
	if result == nil {
		t.Fatal("Prune returned nil")
	}

	if result.PrunedCount != 1 {
		t.Errorf("expected 1 pruned message, got %d", result.PrunedCount)
	}
	if result.KeptCount != 1 {
		t.Errorf("expected 1 kept message, got %d", result.KeptCount)
	}
}

func TestComputeHash(t *testing.T) {
	hash1 := ComputeHash("hello")
	hash2 := ComputeHash("hello")
	hash3 := ComputeHash("world")

	if hash1 != hash2 {
		t.Error("expected same hash for same content")
	}
	if hash1 == hash3 {
		t.Error("expected different hash for different content")
	}
	if len(hash1) != 16 {
		t.Errorf("expected hash length 16, got %d", len(hash1))
	}
}

func TestPruningStrategyString(t *testing.T) {
	tests := []struct {
		strategy PruningStrategy
		expected string
	}{
		{PruningStrategyAge, "age"},
		{PruningStrategySize, "size"},
		{PruningStrategyHybrid, "hybrid"},
		{PruningStrategyDeterministic, "deterministic"},
	}

	for _, tt := range tests {
		if string(tt.strategy) != tt.expected {
			t.Errorf("expected %s, got %s", tt.expected, string(tt.strategy))
		}
	}
}
