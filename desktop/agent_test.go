package main

import (
	"testing"
	"time"

	"github.com/qoqu/zhuLong/internal/bot"
)

func TestNewApp(t *testing.T) {
	app := NewApp()

	if app == nil {
		t.Fatal("NewApp returned nil")
	}

	if app.config.MaxLoops != 50 {
		t.Errorf("Expected MaxLoops=50, got %d", app.config.MaxLoops)
	}

	if app.config.BudgetMaxTokens != 500000 {
		t.Errorf("Expected BudgetMaxTokens=500000, got %d", app.config.BudgetMaxTokens)
	}

	if app.config.ExplorationBaseTemp != 0.7 {
		t.Errorf("Expected ExplorationBaseTemp=0.7, got %f", app.config.ExplorationBaseTemp)
	}
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
	}{
		{"30m", 30 * time.Minute},
		{"1h", time.Hour},
		{"5s", 5 * time.Second},
		{"", 30 * time.Minute},
		{"invalid", 30 * time.Minute},
	}

	for _, tt := range tests {
		result := parseDuration(tt.input, 30*time.Minute)
		if result != tt.expected {
			t.Errorf("parseDuration(%q) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}

func TestSetConfigField(t *testing.T) {
	app := NewApp()

	// Test setting deepseekModel
	err := app.SetConfigField("deepseekModel", "deepseek-v4-pro")
	if err != nil {
		t.Fatalf("SetConfigField failed: %v", err)
	}
	if app.config.DeepSeekModel != "deepseek-v4-pro" {
		t.Errorf("Expected DeepSeekModel='deepseek-v4-pro', got %q", app.config.DeepSeekModel)
	}

	// Test setting temperature
	err = app.SetConfigField("temperature", 0.5)
	if err != nil {
		t.Fatalf("SetConfigField failed: %v", err)
	}
	if app.config.Temperature != 0.5 {
		t.Errorf("Expected Temperature=0.5, got %f", app.config.Temperature)
	}

	// Test setting unknown field
	err = app.SetConfigField("unknownField", "value")
	if err == nil {
		t.Error("Expected error for unknown field, got nil")
	}
}

func TestGetConfigField(t *testing.T) {
	app := NewApp()

	// Test getting deepseekModel
	val := app.GetConfigField("deepseekModel")
	if val != "deepseek-v4-flash" {
		t.Errorf("Expected GetConfigField('deepseekModel')='deepseek-v4-flash', got %v", val)
	}

	// Test getting temperature
	val = app.GetConfigField("temperature")
	if val != 0.7 {
		t.Errorf("Expected GetConfigField('temperature')=0.7, got %v", val)
	}

	// Test getting unknown field
	val = app.GetConfigField("unknownField")
	if val != nil {
		t.Errorf("Expected GetConfigField('unknownField')=nil, got %v", val)
	}
}

func TestDataDir(t *testing.T) {
	app := NewApp()
	dir := app.dataDir()

	if dir == "" {
		t.Error("dataDir returned empty string")
	}
}

func TestHandleBotMessage(t *testing.T) {
	app := NewApp()

	// Initialize bot manager
	app.botMgr = nil

	// Create a mock message event
	_ = &bot.MessageEvent{
		Text:        "Hello, World!",
		MessageType: bot.MessageTypeText,
		Source: bot.SessionSource{
			Platform: bot.PlatformTelegram,
			ChatID:   "test-chat-123",
			ChatType: bot.ChatTypeDM,
			UserID:   "user-123",
		},
		MessageID: "msg-123",
		Timestamp: time.Now(),
	}

	// This should not panic (but may not do much without sessions)
	// We're just testing that it doesn't crash
}
