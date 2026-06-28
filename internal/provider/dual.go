package provider

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// DualModelConfig contains configuration for dual model collaboration
type DualModelConfig struct {
	// ExecutorModel is the model for executing tasks
	ExecutorModel string

	// PlannerModel is the model for planning tasks
	PlannerModel string

	// SharedCache enables shared cache between models
	SharedCache bool

	// IndependentSessions enables independent sessions for each model
	IndependentSessions bool
}

// DefaultDualModelConfig returns default dual model configuration
func DefaultDualModelConfig() *DualModelConfig {
	return &DualModelConfig{
		ExecutorModel:       "deepseek-chat",
		PlannerModel:        "deepseek-reasoner",
		SharedCache:         true,
		IndependentSessions: true,
	}
}

// Provider interface for LLM providers
type Provider interface {
	Chat(ctx context.Context, system, user string) (string, error)
}

// DualProvider manages two models for different tasks
// This implements the dual model collaboration from Reasonix:
// "one executor model and one planner model in independent,
// cache-stable sessions"
type DualProvider struct {
	config   *DualModelConfig
	executor *DeepSeekProvider
	planner  *DeepSeekProvider
	mu       sync.RWMutex
}

// NewDualProvider creates a new dual provider
func NewDualProvider(config *DualModelConfig, executorKey, plannerKey string) *DualProvider {
	if config == nil {
		config = DefaultDualModelConfig()
	}

	executorConfig := &Config{
		APIKey:  executorKey,
		BaseURL: "https://api.deepseek.com",
		Model:   config.ExecutorModel,
		Timeout: 120 * time.Second,
	}

	plannerConfig := &Config{
		APIKey:  plannerKey,
		BaseURL: "https://api.deepseek.com",
		Model:   config.PlannerModel,
		Timeout: 120 * time.Second,
	}

	return &DualProvider{
		config:   config,
		executor: NewDeepSeekProvider(executorConfig),
		planner:  NewDeepSeekProvider(plannerConfig),
	}
}

// GetExecutor returns the executor provider
func (dp *DualProvider) GetExecutor() *DeepSeekProvider {
	return dp.executor
}

// GetPlanner returns the planner provider
func (dp *DualProvider) GetPlanner() *DeepSeekProvider {
	return dp.planner
}

// Execute executes a task using the executor model
func (dp *DualProvider) Execute(ctx context.Context, system, user string) (string, error) {
	dp.mu.RLock()
	defer dp.mu.RUnlock()

	result, err := dp.executor.Chat(ctx, []Message{
		{Role: "system", Content: system},
		{Role: "user", Content: user},
	})
	if err != nil {
		return "", err
	}
	return result.Content, nil
}

// Plan plans a task using the planner model
func (dp *DualProvider) Plan(ctx context.Context, system, user string) (string, error) {
	dp.mu.RLock()
	defer dp.mu.RUnlock()

	result, err := dp.planner.Chat(ctx, []Message{
		{Role: "system", Content: system},
		{Role: "user", Content: user},
	})
	if err != nil {
		return "", err
	}
	return result.Content, nil
}

// isPlanningPrompt checks if the system prompt is for planning
func isPlanningPrompt(system string) bool {
	// Check for planning-related keywords
	planningKeywords := []string{
		"plan",
		"planner",
		"task planner",
		"create a plan",
		"design a plan",
	}

	lowerSystem := strings.ToLower(system)
	for _, keyword := range planningKeywords {
		if strings.Contains(lowerSystem, keyword) {
			return true
		}
	}

	return false
}

// GetConfig returns the dual model configuration
func (dp *DualProvider) GetConfig() *DualModelConfig {
	return dp.config
}

// String returns a string representation
func (dp *DualProvider) String() string {
	return fmt.Sprintf("DualProvider(executor=%s, planner=%s)",
		dp.config.ExecutorModel, dp.config.PlannerModel)
}
