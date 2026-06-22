package compressor

import (
	"fmt"
	"strings"
)

// Compressor interface for compressing context
type Compressor interface {
	// Prune prunes stale tool results
	Prune(messages []Message, currentLoop int) []Message

	// AssembleContext assembles the final context
	AssembleContext(systemPrompt string, skeleton string, sessionSummary string, workingMessages []Message, currentInput string, maxTokens int) []Message
}

// Message represents a message in the context
type Message struct {
	Role       string
	Content    string
	ToolName   string
	LoopNumber int
	Tokens     int
	Prunable   bool
}

// TokenCounter interface for counting tokens
type TokenCounter interface {
	Count(text string) int
}

// CharCounter 是 TokenCounter 的字符计数实现（粗略近似）
// 中文按 1 字符 1 token，英文按 4 字符 1 token 估算
// 关键修复: 之前 Compressor 必须传 TokenCounter 但没有提供默认实现
// pkg/agent.go 中 NewSimpleCompressor 必须传 &CharCounter{}
type CharCounter struct{}

// Count 估算 token 数（英文 4 字符 ≈ 1 token，中文 1 字符 ≈ 1 token）
// 实际 DeepSeek 真实计费以服务端为准，这里仅用于本地预算估算
func (c *CharCounter) Count(text string) int {
	// 简单启发式：4 字符 = 1 token
	tokens := 0
	cjkChars := 0
	otherChars := 0

	for _, r := range text {
		if r >= 0x4E00 && r <= 0x9FFF {
			// CJK Unified Ideographs
			cjkChars++
		} else {
			otherChars++
		}
	}

	// 中文每字符约 1 token，英文每 4 字符约 1 token
	tokens = cjkChars + (otherChars + 3) / 4
	return tokens
}

// SimpleCompressor implements Compressor
type SimpleCompressor struct {
	tokenCounter TokenCounter
	config       *Config
}

// Config contains compressor configuration
type Config struct {
	PruneEnabled    bool
	PruneMaxAge     int
	KeepSignature   bool
	MaxTokens       int
}

// DefaultConfig returns default compressor configuration
func DefaultConfig() *Config {
	return &Config{
		PruneEnabled:    true,
		PruneMaxAge:     2,
		KeepSignature:   true,
		MaxTokens:       8000,
	}
}

// NewSimpleCompressor creates a new simple compressor
func NewSimpleCompressor(tokenCounter TokenCounter, config *Config) *SimpleCompressor {
	if config == nil {
		config = DefaultConfig()
	}

	return &SimpleCompressor{
		tokenCounter: tokenCounter,
		config:       config,
	}
}

// Prune prunes stale tool results
func (c *SimpleCompressor) Prune(messages []Message, currentLoop int) []Message {
	if !c.config.PruneEnabled {
		return messages
	}

	result := make([]Message, 0, len(messages))

	for _, msg := range messages {
		// Only prune tool messages
		if msg.Role == "tool" && msg.Prunable {
			// Check if the message is old enough to prune
			if currentLoop-msg.LoopNumber > c.config.PruneMaxAge {
				// Prune: keep signature, replace content
				if c.config.KeepSignature {
					pruned := Message{
						Role:       msg.Role,
						Content:    fmt.Sprintf("[PRUNED: %s result, %d tokens saved]", msg.ToolName, msg.Tokens),
						ToolName:   msg.ToolName,
						LoopNumber: msg.LoopNumber,
						Tokens:     c.tokenCounter.Count(fmt.Sprintf("[PRUNED: %s result, %d tokens saved]", msg.ToolName, msg.Tokens)),
						Prunable:   false,
					}
					result = append(result, pruned)
				}
				// If not keeping signature, skip the message entirely
			} else {
				result = append(result, msg)
			}
		} else {
			result = append(result, msg)
		}
	}

	return result
}

// AssembleContext assembles the final context
func (c *SimpleCompressor) AssembleContext(
	systemPrompt string,
	skeleton string,
	sessionSummary string,
	workingMessages []Message,
	currentInput string,
	maxTokens int,
) []Message {
	var messages []Message
	usedTokens := 0

	// 1. System prompt (fixed prefix)
	systemMsg := Message{
		Role:    "system",
		Content: systemPrompt,
		Tokens:  c.tokenCounter.Count(systemPrompt),
	}
	messages = append(messages, systemMsg)
	usedTokens += systemMsg.Tokens

	// 2. Skeleton (low-frequency refresh)
	if skeleton != "" {
		skeletonMsg := Message{
			Role:    "system",
			Content: fmt.Sprintf("## Project Skeleton\n\n%s", skeleton),
			Tokens:  c.tokenCounter.Count(skeleton),
		}
		messages = append(messages, skeletonMsg)
		usedTokens += skeletonMsg.Tokens
	}

	// 3. Session summary (stable zone)
	if sessionSummary != "" {
		summaryMsg := Message{
			Role:    "system",
			Content: sessionSummary,
			Tokens:  c.tokenCounter.Count(sessionSummary),
		}
		messages = append(messages, summaryMsg)
		usedTokens += summaryMsg.Tokens
	}

	// 4. Working messages (dynamic zone)
	remainingTokens := maxTokens - usedTokens - c.tokenCounter.Count(currentInput)

	// Add working messages if they fit
	for _, msg := range workingMessages {
		if msg.Tokens <= remainingTokens {
			messages = append(messages, msg)
			usedTokens += msg.Tokens
			remainingTokens -= msg.Tokens
		}
	}

	// 5. Current input
	inputMsg := Message{
		Role:    "user",
		Content: currentInput,
		Tokens:  c.tokenCounter.Count(currentInput),
	}
	messages = append(messages, inputMsg)

	return messages
}

// SimpleTokenCounter implements TokenCounter using simple word counting
type SimpleTokenCounter struct{}

// Count counts tokens (simplified: 1 token ≈ 4 characters)
func (c *SimpleTokenCounter) Count(text string) int {
	// Simple approximation: 1 token ≈ 4 characters
	return len(text) / 4
}

// StringTokenCounter implements TokenCounter using string length
type StringTokenCounter struct{}

// Count counts tokens (simplified: count words)
func (c *StringTokenCounter) Count(text string) int {
	// Count words as a simple approximation
	return len(strings.Fields(text))
}
