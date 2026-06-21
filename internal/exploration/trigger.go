package exploration

import (
	"math/rand"
)

// ExplorationAction represents an exploration action
type ExplorationAction struct {
	Type        string
	Temperature float64
	ToolName    string
	Strategy    string
}

// Trigger triggers exploration when stagnation is detected
type Trigger struct {
	baseTemperature    float64
	maxTemperature     float64
	explorationTools   []string
	toolUsage          map[string]int
}

// NewTrigger creates a new exploration trigger
func NewTrigger(baseTemperature float64, maxTemperature float64, explorationTools []string) *Trigger {
	if baseTemperature <= 0 {
		baseTemperature = 0.7
	}
	if maxTemperature <= 0 {
		maxTemperature = 1.5
	}

	return &Trigger{
		baseTemperature:  baseTemperature,
		maxTemperature:   maxTemperature,
		explorationTools: explorationTools,
		toolUsage:        make(map[string]int),
	}
}

// ShouldExplore returns true if exploration should be triggered
func (t *Trigger) ShouldExplore(stagnationType string) bool {
	return stagnationType != "none"
}

// GenerateExploration generates an exploration action
func (t *Trigger) GenerateExploration(stagnationType string) ExplorationAction {
	switch stagnationType {
	case "info_starved":
		return t.increaseTemperature()
	case "looping":
		return t.tryNewTool()
	case "blocked":
		return t.changeStrategy()
	default:
		return t.increaseTemperature()
	}
}

// increaseTemperature increases the temperature for more randomness
func (t *Trigger) increaseTemperature() ExplorationAction {
	newTemp := t.baseTemperature * 1.5
	if newTemp > t.maxTemperature {
		newTemp = t.maxTemperature
	}

	return ExplorationAction{
		Type:        "increase_temperature",
		Temperature: newTemp,
	}
}

// tryNewTool tries a tool that hasn't been used much
func (t *Trigger) tryNewTool() ExplorationAction {
	// Find the least used tool
	leastUsed := ""
	minUsage := int(^uint(0) >> 1) // Max int

	for _, tool := range t.explorationTools {
		usage := t.toolUsage[tool]
		if usage < minUsage {
			minUsage = usage
			leastUsed = tool
		}
	}

	if leastUsed == "" && len(t.explorationTools) > 0 {
		leastUsed = t.explorationTools[rand.Intn(len(t.explorationTools))]
	}

	return ExplorationAction{
		Type:     "try_new_tool",
		ToolName: leastUsed,
	}
}

// changeStrategy suggests changing the strategy
func (t *Trigger) changeStrategy() ExplorationAction {
	strategies := []string{
		"Try a completely different approach",
		"Break the problem into smaller parts",
		"Look for alternative tools",
		"Ask for human guidance",
	}

	return ExplorationAction{
		Type:     "change_strategy",
		Strategy: strategies[rand.Intn(len(strategies))],
	}
}

// RecordToolUsage records tool usage
func (t *Trigger) RecordToolUsage(toolName string) {
	t.toolUsage[toolName]++
}

// GetToolUsage returns tool usage
func (t *Trigger) GetToolUsage() map[string]int {
	return t.toolUsage
}

// Reset resets the trigger
func (t *Trigger) Reset() {
	t.toolUsage = make(map[string]int)
}
