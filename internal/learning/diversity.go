package learning

import (
	"math"
)

// DiversityReport contains diversity metrics
type DiversityReport struct {
	ToolDiversity     float64
	StrategyDiversity float64
	IsHealthy         bool
	Recommendations   []string
}

// DiversityManager manages diversity
type DiversityManager struct {
	toolUsage     map[string]int
	strategyUsage map[string]int
	threshold     float64
}

// NewDiversityManager creates a new diversity manager
func NewDiversityManager(threshold float64) *DiversityManager {
	if threshold <= 0 {
		threshold = 1.0
	}

	return &DiversityManager{
		toolUsage:     make(map[string]int),
		strategyUsage: make(map[string]int),
		threshold:     threshold,
	}
}

// RecordToolUsage records tool usage
func (dm *DiversityManager) RecordToolUsage(toolName string) {
	dm.toolUsage[toolName]++
}

// RecordStrategyUsage records strategy usage
func (dm *DiversityManager) RecordStrategyUsage(strategyName string) {
	dm.strategyUsage[strategyName]++
}

// CheckDiversity checks diversity
func (dm *DiversityManager) CheckDiversity() DiversityReport {
	toolDiversity := dm.calculateDiversity(dm.toolUsage)
	strategyDiversity := dm.calculateDiversity(dm.strategyUsage)

	isHealthy := toolDiversity >= dm.threshold && strategyDiversity >= dm.threshold

	recommendations := dm.generateRecommendations(toolDiversity, strategyDiversity)

	return DiversityReport{
		ToolDiversity:     toolDiversity,
		StrategyDiversity: strategyDiversity,
		IsHealthy:         isHealthy,
		Recommendations:   recommendations,
	}
}

// calculateDiversity calculates diversity using Shannon entropy
func (dm *DiversityManager) calculateDiversity(usage map[string]int) float64 {
	total := 0
	for _, count := range usage {
		total += count
	}

	if total == 0 {
		return 0
	}

	entropy := 0.0
	for _, count := range usage {
		p := float64(count) / float64(total)
		if p > 0 {
			entropy -= p * math.Log2(p)
		}
	}

	return entropy
}

// generateRecommendations generates recommendations for improving diversity
func (dm *DiversityManager) generateRecommendations(toolDiv, strategyDiv float64) []string {
	recommendations := make([]string, 0)

	if toolDiv < dm.threshold {
		recommendations = append(recommendations, "Try using different tools to increase diversity")
	}

	if strategyDiv < dm.threshold {
		recommendations = append(recommendations, "Try using different strategies to increase diversity")
	}

	// Find underused tools
	underused := dm.getUnderusedTools()
	if len(underused) > 0 {
		recommendations = append(recommendations, "Consider using these tools: "+joinStrings(underused))
	}

	return recommendations
}

// getUnderusedTools gets tools that have been used less
func (dm *DiversityManager) getUnderusedTools() []string {
	if len(dm.toolUsage) == 0 {
		return nil
	}

	// Find minimum usage
	minUsage := int(^uint(0) >> 1)
	for _, count := range dm.toolUsage {
		if count < minUsage {
			minUsage = count
		}
	}

	// Get tools with minimum usage
	underused := make([]string, 0)
	for tool, count := range dm.toolUsage {
		if count == minUsage {
			underused = append(underused, tool)
		}
	}

	return underused
}

// GetToolUsage returns tool usage
func (dm *DiversityManager) GetToolUsage() map[string]int {
	return dm.toolUsage
}

// GetStrategyUsage returns strategy usage
func (dm *DiversityManager) GetStrategyUsage() map[string]int {
	return dm.strategyUsage
}

// Reset resets the diversity manager
func (dm *DiversityManager) Reset() {
	dm.toolUsage = make(map[string]int)
	dm.strategyUsage = make(map[string]int)
}

// joinStrings joins strings with comma
func joinStrings(strs []string) string {
	result := ""
	for i, s := range strs {
		if i > 0 {
			result += ", "
		}
		result += s
	}
	return result
}
