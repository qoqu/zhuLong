package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	if config == nil {
		t.Fatal("DefaultConfig returned nil")
	}
	if config.BaseURL != "https://api.deepseek.com" {
		t.Errorf("expected BaseURL https://api.deepseek.com, got %s", config.BaseURL)
	}
	if config.Model != "deepseek-chat" {
		t.Errorf("expected Model deepseek-chat, got %s", config.Model)
	}
	if config.Timeout != 60*time.Second {
		t.Errorf("expected Timeout 60s, got %s", config.Timeout)
	}
}

func TestNewDeepSeekProvider(t *testing.T) {
	config := &Config{
		APIKey:  "test-key",
		BaseURL: "https://api.test.com",
		Model:   "test-model",
		Timeout: 30 * time.Second,
	}
	provider := NewDeepSeekProvider(config)
	if provider == nil {
		t.Fatal("NewDeepSeekProvider returned nil")
	}
	if provider.apiKey != "test-key" {
		t.Errorf("expected apiKey test-key, got %s", provider.apiKey)
	}
	if provider.baseURL != "https://api.test.com" {
		t.Errorf("expected baseURL https://api.test.com, got %s", provider.baseURL)
	}
	if provider.model != "test-model" {
		t.Errorf("expected model test-model, got %s", provider.model)
	}
	if provider.client == nil {
		t.Fatal("expected client to be set")
	}
	if provider.client.Timeout != 30*time.Second {
		t.Errorf("expected client timeout 30s, got %s", provider.client.Timeout)
	}
}

func TestNewDeepSeekProviderNilConfig(t *testing.T) {
	provider := NewDeepSeekProvider(nil)
	if provider == nil {
		t.Fatal("NewDeepSeekProvider returned nil")
	}
	if provider.baseURL != "https://api.deepseek.com" {
		t.Errorf("expected baseURL https://api.deepseek.com, got %s", provider.baseURL)
	}
	if provider.model != "deepseek-chat" {
		t.Errorf("expected model deepseek-chat, got %s", provider.model)
	}
}

func TestChatSuccess(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("expected path /v1/chat/completions, got %s", r.URL.Path)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("expected Authorization Bearer test-key, got %s", r.Header.Get("Authorization"))
		}

		// Parse request
		var req ChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		if req.Model != "test-model" {
			t.Errorf("expected model test-model, got %s", req.Model)
		}
		if len(req.Messages) != 2 {
			t.Errorf("expected 2 messages, got %d", len(req.Messages))
		}

		// Send response
		resp := ChatResponse{
			Choices: []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			}{
				{
					Message: struct {
						Content string `json:"content"`
					}{
						Content: "Hello, world!",
					},
				},
			},
			Usage: struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
				TotalTokens      int `json:"total_tokens"`
			}{
				PromptTokens:     10,
				CompletionTokens: 5,
				TotalTokens:      15,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Create provider
	config := &Config{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "test-model",
		Timeout: 10 * time.Second,
	}
	provider := NewDeepSeekProvider(config)

	// Test chat
	messages := []Message{
		{Role: "system", Content: "You are a helpful assistant"},
		{Role: "user", Content: "Hello"},
	}
	response, err := provider.Chat(context.Background(), messages)
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}
	if response != "Hello, world!" {
		t.Errorf("expected response 'Hello, world!', got '%s'", response)
	}
}

func TestChatAPIError(t *testing.T) {
	// Create mock server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("bad request"))
	}))
	defer server.Close()

	// Create provider
	config := &Config{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "test-model",
		Timeout: 10 * time.Second,
	}
	provider := NewDeepSeekProvider(config)

	// Test chat
	messages := []Message{
		{Role: "user", Content: "Hello"},
	}
	_, err := provider.Chat(context.Background(), messages)
	if err == nil {
		t.Error("expected error for API error")
	}
}

func TestChatNoChoices(t *testing.T) {
	// Create mock server that returns no choices
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := ChatResponse{
			Choices: []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			}{},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Create provider
	config := &Config{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "test-model",
		Timeout: 10 * time.Second,
	}
	provider := NewDeepSeekProvider(config)

	// Test chat
	messages := []Message{
		{Role: "user", Content: "Hello"},
	}
	_, err := provider.Chat(context.Background(), messages)
	if err == nil {
		t.Error("expected error for no choices")
	}
}

func TestChatCancelledContext(t *testing.T) {
	// Create provider
	config := &Config{
		APIKey:  "test-key",
		BaseURL: "https://api.test.com",
		Model:   "test-model",
		Timeout: 10 * time.Second,
	}
	provider := NewDeepSeekProvider(config)

	// Test chat with cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	messages := []Message{
		{Role: "user", Content: "Hello"},
	}
	_, err := provider.Chat(ctx, messages)
	if err == nil {
		t.Error("expected error for cancelled context")
	}
}

func TestMessageStruct(t *testing.T) {
	msg := Message{
		Role:    "user",
		Content: "Hello",
	}
	if msg.Role != "user" {
		t.Errorf("expected role user, got %s", msg.Role)
	}
	if msg.Content != "Hello" {
		t.Errorf("expected content Hello, got %s", msg.Content)
	}
}

func TestChatRequestStruct(t *testing.T) {
	req := ChatRequest{
		Model:       "test-model",
		Messages:    []Message{{Role: "user", Content: "Hello"}},
		Temperature: 0.7,
		MaxTokens:   100,
	}
	if req.Model != "test-model" {
		t.Errorf("expected model test-model, got %s", req.Model)
	}
	if len(req.Messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(req.Messages))
	}
	if req.Temperature != 0.7 {
		t.Errorf("expected temperature 0.7, got %f", req.Temperature)
	}
	if req.MaxTokens != 100 {
		t.Errorf("expected max_tokens 100, got %d", req.MaxTokens)
	}
}
