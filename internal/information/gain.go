package information

import (
	"math"
)

// ToolResult contains the result of a tool call
type ToolResult struct {
	ToolName string
	Output   string
	Tokens   int
}

// InformationGain calculates information gain for tool selection
type InformationGain struct {
	toolHistory map[string][]ToolResult
}

// NewInformationGain creates a new information gain calculator
func NewInformationGain() *InformationGain {
	return &InformationGain{
		toolHistory: make(map[string][]ToolResult),
	}
}

// AddResult adds a tool result to the history
func (ig *InformationGain) AddResult(result ToolResult) {
	ig.toolHistory[result.ToolName] = append(ig.toolHistory[result.ToolName], result)
}

// EstimateGain estimates the information gain of calling a tool
// Higher gain = more new information expected
func (ig *InformationGain) EstimateGain(toolName string) float64 {
	results := ig.toolHistory[toolName]

	if len(results) == 0 {
		// New tool, maximum expected gain
		return 1.0
	}

	// Existing tool, diminishing returns
	return ig.diminishingGain(results)
}

// diminishingGain calculates diminishing gain based on usage count
func (ig *InformationGain) diminishingGain(results []ToolResult) float64 {
	baseGain := 1.0
	decay := 0.6 // Each call reduces gain by 40%
	return baseGain * math.Pow(decay, float64(len(results)))
}

// GetToolUsage returns the usage count for each tool
func (ig *InformationGain) GetToolUsage() map[string]int {
	usage := make(map[string]int)
	for toolName, results := range ig.toolHistory {
		usage[toolName] = len(results)
	}
	return usage
}

// GetToolGain returns the current gain for each tool
func (ig *InformationGain) GetToolGain() map[string]float64 {
	gains := make(map[string]float64)
	for toolName := range ig.toolHistory {
		gains[toolName] = ig.EstimateGain(toolName)
	}
	return gains
}

// Reset resets the history
func (ig *InformationGain) Reset() {
	ig.toolHistory = make(map[string][]ToolResult)
}
