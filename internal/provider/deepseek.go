package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// DeepSeekProvider implements the LLM provider for DeepSeek
type DeepSeekProvider struct {
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
}

// Config contains DeepSeek provider configuration
type Config struct {
	APIKey  string
	BaseURL string
	Model   string
	Timeout time.Duration
}

// DefaultConfig returns default DeepSeek provider configuration
func DefaultConfig() *Config {
	return &Config{
		BaseURL: "https://api.deepseek.com",
		Model:   "deepseek-chat",
		Timeout: 60 * time.Second,
	}
}

// NewDeepSeekProvider creates a new DeepSeek provider
func NewDeepSeekProvider(config *Config) *DeepSeekProvider {
	if config == nil {
		config = DefaultConfig()
	}

	return &DeepSeekProvider{
		apiKey:  config.APIKey,
		baseURL: config.BaseURL,
		model:   config.Model,
		client: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest represents a chat completion request
type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
}

// ChatResponse represents a chat completion response
type ChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens        int `json:"prompt_tokens"`
		CompletionTokens    int `json:"completion_tokens"`
		TotalTokens         int `json:"total_tokens"`
		PromptTokensDetails struct {
			CachedTokens int `json:"cached_tokens"`
		} `json:"prompt_tokens_details"`
	} `json:"usage"`
}

// Chat sends a chat completion request to DeepSeek
func (p *DeepSeekProvider) Chat(ctx context.Context, messages []Message) (string, error) {
	// Build request
	req := ChatRequest{
		Model:       p.model,
		Messages:    messages,
		Temperature: 0.7,
	}

	// Marshal request
	reqBody, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/v1/chat/completions", bytes.NewReader(reqBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	// Send request
	httpResp, err := p.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer httpResp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Check status code
	if httpResp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error: %s - %s", httpResp.Status, string(respBody))
	}

	// Parse response
	var resp ChatResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Check if we have choices
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return resp.Choices[0].Message.Content, nil
}
