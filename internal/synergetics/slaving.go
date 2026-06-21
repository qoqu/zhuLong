package synergetics

import (
	"fmt"
	"strings"
)

// Action represents an action to be evaluated
type Action struct {
	Description string
	Type        string
	Tool        string
	Priority    float64
}

// EnforcedAction represents an action enforced by the slaving principle
type EnforcedAction struct {
	Action   *Action
	Priority float64
	Aligned  bool
}

// SlavingPrinciple implements the slaving principle
type SlavingPrinciple struct {
	orderParameter *OrderParameter
}

// NewSlavingPrinciple creates a new slaving principle
func NewSlavingPrinciple(orderParameter *OrderParameter) *SlavingPrinciple {
	return &SlavingPrinciple{
		orderParameter: orderParameter,
	}
}

// EnforceSlaving enforces the slaving principle on an action
func (sp *SlavingPrinciple) EnforceSlaving(action *Action) (*EnforcedAction, error) {
	if action == nil {
		return nil, fmt.Errorf("action cannot be nil")
	}

	// Calculate relevance to goal
	relevance := sp.assessRelevance(action)

	// Reject actions with very low relevance
	if relevance < 0.3 {
		return nil, fmt.Errorf("action '%s' is not relevant to goal '%s' (relevance: %.2f)",
			action.Description, sp.orderParameter.Goal, relevance)
	}

	// Adjust priority based on relevance
	adjustedPriority := action.Priority * relevance

	// Determine if action is aligned with goal
	aligned := relevance >= 0.7

	return &EnforcedAction{
		Action:   action,
		Priority: adjustedPriority,
		Aligned:  aligned,
	}, nil
}

// assessRelevance assesses how relevant an action is to the goal
func (sp *SlavingPrinciple) assessRelevance(action *Action) float64 {
	goal := strings.ToLower(sp.orderParameter.Goal)
	description := strings.ToLower(action.Description)
	tool := strings.ToLower(action.Tool)

	// Simple keyword matching
	score := 0.0
	matches := 0

	// Check if action description contains goal keywords
	goalWords := strings.Fields(goal)
	for _, word := range goalWords {
		if len(word) > 3 && strings.Contains(description, word) {
			matches++
		}
	}

	if len(goalWords) > 0 {
		score = float64(matches) / float64(len(goalWords))
	}

	// Bonus for using relevant tools
	if strings.Contains(goal, "file") && strings.Contains(tool, "file") {
		score += 0.2
	}
	if strings.Contains(goal, "search") && strings.Contains(tool, "search") {
		score += 0.2
	}

	// Cap at 1.0
	if score > 1.0 {
		score = 1.0
	}

	// Minimum relevance for any action
	if score < 0.5 {
		score = 0.5
	}

	return score
}

// GetOrderParameter returns the order parameter
func (sp *SlavingPrinciple) GetOrderParameter() *OrderParameter {
	return sp.orderParameter
}

// UpdateOrderParameter updates the order parameter
func (sp *SlavingPrinciple) UpdateOrderParameter(op *OrderParameter) {
	sp.orderParameter = op
}
