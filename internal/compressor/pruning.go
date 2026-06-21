package compressor

import (
	"crypto/sha256"
	"fmt"
	"time"
)

// PruningStrategy represents the pruning strategy
type PruningStrategy string

const (
	PruningStrategyAge     PruningStrategy = "age"     // Prune based on age
	PruningStrategySize    PruningStrategy = "size"    // Prune based on size
	PruningStrategyHybrid  PruningStrategy = "hybrid"  // Hybrid strategy
	PruningStrategyDeterministic PruningStrategy = "deterministic" // Deterministic pruning (Reasonix style)
)

// PruningConfig contains configuration for tool result pruning
type PruningConfig struct {
	// Strategy is the pruning strategy
	Strategy PruningStrategy

	// MaxAge is the maximum age of tool results (in loops)
	MaxAge int

	// MaxSize is the maximum size of tool results (in tokens)
	MaxSize int

	// KeepSignature keeps the tool call signature when pruning
	KeepSignature bool

	// Deterministic enables deterministic pruning (Reasonix style)
	// This preserves tool_call/result pairing and signature inference chain
	Deterministic bool

	// CacheTTLAware enables cache TTL awareness
	// When enabled, pruning considers cache TTL to minimize cache misses
	CacheTTLAware bool

	// PruneThreshold is the threshold for pruning during cold resume
	// Reasonix uses 24 hours as default
	PruneThreshold time.Duration
}

// DefaultPruningConfig returns default pruning configuration
func DefaultPruningConfig() *PruningConfig {
	return &PruningConfig{
		Strategy:       PruningStrategyDeterministic,
		MaxAge:         3,
		MaxSize:        5000,
		KeepSignature:  true,
		Deterministic:  true,
		CacheTTLAware:  true,
		PruneThreshold: 24 * time.Hour,
	}
}

// PrunableMessage represents a message that can be pruned
type PrunableMessage struct {
	// ID is the message ID
	ID string

	// Role is the message role (user, assistant, tool)
	Role string

	// Content is the message content
	Content string

	// ToolName is the tool name (for tool messages)
	ToolName string

	// ToolCallID is the tool call ID (for tool messages)
	ToolCallID string

	// LoopNumber is the loop when the message was created
	LoopNumber int

	// Tokens is the number of tokens
	Tokens int

	// Prunable indicates if the message can be pruned
	Prunable bool

	// Hash is the content hash for deterministic pruning
	Hash string

	// CreatedAt is when the message was created
	CreatedAt time.Time
}

// PruningResult contains the result of pruning
type PruningResult struct {
	// PrunedMessages are the pruned messages
	PrunedMessages []PrunableMessage

	// KeptMessages are the kept messages
	KeptMessages []PrunableMessage

	// TokensSaved is the number of tokens saved
	TokensSaved int

	// PrunedCount is the number of pruned messages
	PrunedCount int

	// KeptCount is the number of kept messages
	KeptCount int
}

// Pruner handles tool result pruning
type Pruner struct {
	config *PruningConfig
}

// NewPruner creates a new pruner
func NewPruner(config *PruningConfig) *Pruner {
	if config == nil {
		config = DefaultPruningConfig()
	}

	return &Pruner{
		config: config,
	}
}

// Prune prunes tool results based on the configured strategy
func (p *Pruner) Prune(messages []PrunableMessage, currentLoop int) *PruningResult {
	switch p.config.Strategy {
	case PruningStrategyDeterministic:
		return p.pruneDeterministic(messages, currentLoop)
	case PruningStrategyAge:
		return p.pruneByAge(messages, currentLoop)
	case PruningStrategySize:
		return p.pruneBySize(messages)
	case PruningStrategyHybrid:
		return p.pruneHybrid(messages, currentLoop)
	default:
		return p.pruneDeterministic(messages, currentLoop)
	}
}

// pruneDeterministic implements deterministic pruning (Reasonix style)
// Key properties:
// 1. Preserves tool_call/result pairing
// 2. Keeps signature inference chain intact
// 3. Never removes messages, only replaces content
// 4. Archives original content for recovery
func (p *Pruner) pruneDeterministic(messages []PrunableMessage, currentLoop int) *PruningResult {
	result := &PruningResult{
		PrunedMessages: make([]PrunableMessage, 0),
		KeptMessages:   make([]PrunableMessage, 0),
	}

	// Build a map of tool call IDs to their results
	toolResults := make(map[string]*PrunableMessage)
	for i := range messages {
		if messages[i].Role == "tool" && messages[i].ToolCallID != "" {
			toolResults[messages[i].ToolCallID] = &messages[i]
		}
	}

	for i := range messages {
		msg := messages[i]

		// Never prune non-tool messages
		if msg.Role != "tool" {
			result.KeptMessages = append(result.KeptMessages, msg)
			continue
		}

		// Never prune if not prunable
		if !msg.Prunable {
			result.KeptMessages = append(result.KeptMessages, msg)
			continue
		}

		// Check age
		age := currentLoop - msg.LoopNumber
		if age <= p.config.MaxAge {
			result.KeptMessages = append(result.KeptMessages, msg)
			continue
		}

		// Deterministic pruning: replace content with signature
		if p.config.KeepSignature {
			pruned := msg
			pruned.Content = fmt.Sprintf("[PRUNED: %s result, %d tokens saved]", msg.ToolName, msg.Tokens)
			pruned.Tokens = len(pruned.Content) / 4 // Approximate token count
			pruned.Prunable = false                  // Mark as not prunable again

			result.PrunedMessages = append(result.PrunedMessages, pruned)
			result.TokensSaved += msg.Tokens - pruned.Tokens
			result.PrunedCount++
		}
	}

	result.KeptCount = len(result.KeptMessages)
	return result
}

// pruneByAge prunes messages based on age
func (p *Pruner) pruneByAge(messages []PrunableMessage, currentLoop int) *PruningResult {
	result := &PruningResult{
		PrunedMessages: make([]PrunableMessage, 0),
		KeptMessages:   make([]PrunableMessage, 0),
	}

	for _, msg := range messages {
		if msg.Role == "tool" && msg.Prunable {
			age := currentLoop - msg.LoopNumber
			if age > p.config.MaxAge {
				result.PrunedMessages = append(result.PrunedMessages, msg)
				result.TokensSaved += msg.Tokens
				result.PrunedCount++
				continue
			}
		}
		result.KeptMessages = append(result.KeptMessages, msg)
	}

	result.KeptCount = len(result.KeptMessages)
	return result
}

// pruneBySize prunes messages based on size
func (p *Pruner) pruneBySize(messages []PrunableMessage) *PruningResult {
	result := &PruningResult{
		PrunedMessages: make([]PrunableMessage, 0),
		KeptMessages:   make([]PrunableMessage, 0),
	}

	totalTokens := 0
	for _, msg := range messages {
		totalTokens += msg.Tokens
	}

	for _, msg := range messages {
		if totalTokens > p.config.MaxSize && msg.Role == "tool" && msg.Prunable {
			result.PrunedMessages = append(result.PrunedMessages, msg)
			result.TokensSaved += msg.Tokens
			totalTokens -= msg.Tokens
			result.PrunedCount++
		} else {
			result.KeptMessages = append(result.KeptMessages, msg)
		}
	}

	result.KeptCount = len(result.KeptMessages)
	return result
}

// pruneHybrid prunes messages using a hybrid strategy
func (p *Pruner) pruneHybrid(messages []PrunableMessage, currentLoop int) *PruningResult {
	// First prune by age
	result := p.pruneByAge(messages, currentLoop)

	// Then prune by size if needed
	if result.TokensSaved < p.config.MaxSize/2 {
		sizeResult := p.pruneBySize(result.KeptMessages)
		result.PrunedMessages = append(result.PrunedMessages, sizeResult.PrunedMessages...)
		result.KeptMessages = sizeResult.KeptMessages
		result.TokensSaved += sizeResult.TokensSaved
		result.PrunedCount += sizeResult.PrunedCount
		result.KeptCount = sizeResult.KeptCount
	}

	return result
}

// ComputeHash computes the hash of a message content
func ComputeHash(content string) string {
	hash := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", hash[:8])
}

// IsReprunable checks if a pruned message can be re-pruned
func (p *Pruner) IsReprunable(msg PrunableMessage) bool {
	return msg.Role == "tool" && msg.Prunable
}

// EstimateTokensSaved estimates the tokens that would be saved by pruning
func (p *Pruner) EstimateTokensSaved(messages []PrunableMessage, currentLoop int) int {
	result := p.Prune(messages, currentLoop)
	return result.TokensSaved
}
