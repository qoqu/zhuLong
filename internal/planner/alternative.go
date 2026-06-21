package planner

import (
	"fmt"
)

// AlternativePath represents an alternative path
type AlternativePath struct {
	ID          string   `json:"id"`
	Description string   `json:"description"`
	Steps       []Step   `json:"steps"`
	WhenToUse   string   `json:"when_to_use"`
	Confidence  float64  `json:"confidence"`
}

// AlternativePlanner generates alternative paths
type AlternativePlanner struct {
	mainPlanner *LLMPlanner
}

// NewAlternativePlanner creates a new alternative planner
func NewAlternativePlanner(mainPlanner *LLMPlanner) *AlternativePlanner {
	return &AlternativePlanner{
		mainPlanner: mainPlanner,
	}
}

// GenerateAlternatives generates alternative paths for a plan
func (ap *AlternativePlanner) GenerateAlternatives(plan *Plan, memory MemoryReader) ([]AlternativePath, error) {
	if plan == nil {
		return nil, fmt.Errorf("plan cannot be nil")
	}

	alternatives := make([]AlternativePath, 0)

	// Generate alternatives based on the main plan
	// For now, we create simple alternatives by modifying steps
	for i, step := range plan.Steps {
		alt := ap.generateAlternativeForStep(plan, i, step, memory)
		if alt != nil {
			alternatives = append(alternatives, *alt)
		}
	}

	return alternatives, nil
}

// generateAlternativeForStep generates an alternative for a specific step
func (ap *AlternativePlanner) generateAlternativeForStep(plan *Plan, stepIndex int, step Step, memory MemoryReader) *AlternativePath {
	// Create alternative by trying a different approach
	altSteps := make([]Step, len(plan.Steps))
	copy(altSteps, plan.Steps)

	// Modify the step to use a different approach
	altStep := step
	altStep.ID = fmt.Sprintf("%s-alt", step.ID)
	altStep.Description = fmt.Sprintf("Alternative: %s", step.Description)

	// Try different tool or approach
	if step.Action.Type == "tool_call" {
		altStep.Action = ap.suggestAlternativeTool(step.Action)
	}

	altSteps[stepIndex] = altStep

	return &AlternativePath{
		ID:          fmt.Sprintf("alt-%s", step.ID),
		Description: fmt.Sprintf("Alternative approach for: %s", step.Description),
		Steps:       altSteps,
		WhenToUse:   fmt.Sprintf("When %s fails", step.Description),
		Confidence:  0.7,
	}
}

// suggestAlternativeTool suggests an alternative tool
func (ap *AlternativePlanner) suggestAlternativeTool(action Action) Action {
	altAction := action

	// Simple tool substitution logic
	switch action.Tool {
	case "read_file":
		altAction.Tool = "execute_command"
		altAction.Params = map[string]interface{}{
			"command": fmt.Sprintf("cat %v", action.Params["path"]),
		}
	case "write_file":
		altAction.Tool = "execute_command"
		altAction.Params = map[string]interface{}{
			"command": fmt.Sprintf("echo %v > %v", action.Params["content"], action.Params["path"]),
		}
	case "execute_command":
		// Try a different command approach
		altAction.Tool = "execute_command"
		altAction.Params = map[string]interface{}{
			"command": fmt.Sprintf("bash -c %v", action.Params["command"]),
		}
	}

	return altAction
}

// SelectBestAlternative selects the best alternative based on context
func (ap *AlternativePlanner) SelectBestAlternative(alternatives []AlternativePath, failedStep Step) *AlternativePath {
	if len(alternatives) == 0 {
		return nil
	}

	// Find alternative that addresses the failed step
	for _, alt := range alternatives {
		for _, step := range alt.Steps {
			if step.ID == fmt.Sprintf("%s-alt", failedStep.ID) {
				return &alt
			}
		}
	}

	// Return first alternative as fallback
	return &alternatives[0]
}

// PlanWithAlternatives generates a plan with alternatives
func (ap *AlternativePlanner) PlanWithAlternatives(goal string, memory MemoryReader) (*Plan, []AlternativePath, error) {
	// Generate main plan
	mainPlan, err := ap.mainPlanner.Plan(nil, goal, memory)
	if err != nil {
		return nil, nil, err
	}

	// Generate alternatives
	alternatives, err := ap.GenerateAlternatives(mainPlan, memory)
	if err != nil {
		return mainPlan, nil, err
	}

	return mainPlan, alternatives, nil
}
