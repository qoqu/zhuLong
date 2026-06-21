package information

import (
	"strings"
)

// Message represents a message in the context
type Message struct {
	Role    string
	Content string
	Tokens  int
}

// DensityCalculator calculates information density
type DensityCalculator struct {
	tokenCounter TokenCounter
}

// TokenCounter interface for counting tokens
type TokenCounter interface {
	Count(text string) int
}

// NewDensityCalculator creates a new density calculator
func NewDensityCalculator(tokenCounter TokenCounter) *DensityCalculator {
	return &DensityCalculator{
		tokenCounter: tokenCounter,
	}
}

// Density calculates the information density of messages
// Density = useful information / total tokens
func (dc *DensityCalculator) Density(messages []Message) float64 {
	usefulInfo := 0.0
	totalTokens := 0

	for _, msg := range messages {
		tokens := dc.tokenCounter.Count(msg.Content)
		totalTokens += tokens

		switch msg.Role {
		case "system":
			usefulInfo += float64(tokens) * 1.0 // System prompt is high density
		case "tool":
			usefulInfo += float64(tokens) * 0.5 // Tool results are medium density
		case "user":
			usefulInfo += float64(tokens) * 0.8 // User input is high density
		case "assistant":
			usefulInfo += float64(tokens) * 0.6 // Assistant output is medium density
		default:
			usefulInfo += float64(tokens) * 0.5
		}
	}

	if totalTokens == 0 {
		return 0
	}

	return usefulInfo / float64(totalTokens)
}

// OptimizeDensity optimizes message density by removing low-density messages
func (dc *DensityCalculator) OptimizeDensity(messages []Message, maxTokens int) []Message {
	if len(messages) == 0 {
		return messages
	}

	// Calculate density for each message
	type scoredMessage struct {
		message Message
		density float64
	}

	scored := make([]scoredMessage, len(messages))
	for i, msg := range messages {
		density := 0.5
		switch msg.Role {
		case "system":
			density = 1.0
		case "user":
			density = 0.8
		case "assistant":
			density = 0.6
		case "tool":
			density = 0.4
		}
		scored[i] = scoredMessage{
			message: msg,
			density: density,
		}
	}

	// Sort by density (highest first)
	for i := 0; i < len(scored); i++ {
		for j := i + 1; j < len(scored); j++ {
			if scored[j].density > scored[i].density {
				scored[i], scored[j] = scored[j], scored[i]
			}
		}
	}

	// Select messages that fit within token limit
	result := make([]Message, 0)
	usedTokens := 0

	for _, sm := range scored {
		tokens := dc.tokenCounter.Count(sm.message.Content)
		if usedTokens+tokens <= maxTokens {
			result = append(result, sm.message)
			usedTokens += tokens
		}
	}

	return result
}

// SimpleTokenCounter implements TokenCounter using simple word counting
type SimpleTokenCounter struct{}

// Count counts tokens (simplified: count words)
func (c *SimpleTokenCounter) Count(text string) int {
	return len(strings.Fields(text))
}
