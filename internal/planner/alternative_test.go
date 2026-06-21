package planner

import (
	"testing"
)

func TestNewAlternativePlanner(t *testing.T) {
	mainPlanner := NewLLMPlanner(nil, nil)
	ap := NewAlternativePlanner(mainPlanner)

	if ap == nil {
		t.Error("NewAlternativePlanner() should not return nil")
	}
	if ap.mainPlanner != mainPlanner {
		t.Error("AlternativePlanner should store main planner")
	}
}

func TestAlternativePlanner_GenerateAlternatives(t *testing.T) {
	mainPlanner := NewLLMPlanner(nil, nil)
	ap := NewAlternativePlanner(mainPlanner)

	plan := &Plan{
		ID: "plan-1",
		Steps: []Step{
			{
				ID:          "step-1",
				Description: "Read the file",
				Action: Action{
					Type: "tool_call",
					Tool: "read_file",
					Params: map[string]interface{}{
						"path": "main.go",
					},
				},
			},
			{
				ID:          "step-2",
				Description: "Write the file",
				Action: Action{
					Type: "tool_call",
					Tool: "write_file",
					Params: map[string]interface{}{
						"path":    "output.txt",
						"content": "hello",
					},
				},
			},
		},
	}

	alternatives, err := ap.GenerateAlternatives(plan, nil)

	if err != nil {
		t.Errorf("GenerateAlternatives() error = %v", err)
	}

	if len(alternatives) != 2 {
		t.Errorf("GenerateAlternatives() returned %v alternatives, want 2", len(alternatives))
	}
}

func TestAlternativePlanner_GenerateAlternatives_NilPlan(t *testing.T) {
	mainPlanner := NewLLMPlanner(nil, nil)
	ap := NewAlternativePlanner(mainPlanner)

	_, err := ap.GenerateAlternatives(nil, nil)

	if err == nil {
		t.Error("GenerateAlternatives() should return error for nil plan")
	}
}

func TestAlternativePlanner_SelectBestAlternative(t *testing.T) {
	mainPlanner := NewLLMPlanner(nil, nil)
	ap := NewAlternativePlanner(mainPlanner)

	alternatives := []AlternativePath{
		{
			ID: "alt-1",
			Steps: []Step{
				{ID: "step-1-alt", Description: "Alternative for step 1"},
			},
		},
		{
			ID: "alt-2",
			Steps: []Step{
				{ID: "step-2-alt", Description: "Alternative for step 2"},
			},
		},
	}

	// Select alternative for step-1
	failedStep := Step{ID: "step-1"}
	selected := ap.SelectBestAlternative(alternatives, failedStep)

	if selected == nil {
		t.Error("SelectBestAlternative() should not return nil")
	}
	if selected.ID != "alt-1" {
		t.Errorf("SelectBestAlternative() selected %v, want alt-1", selected.ID)
	}
}

func TestAlternativePlanner_SelectBestAlternative_NoMatch(t *testing.T) {
	mainPlanner := NewLLMPlanner(nil, nil)
	ap := NewAlternativePlanner(mainPlanner)

	alternatives := []AlternativePath{
		{
			ID: "alt-1",
			Steps: []Step{
				{ID: "step-1-alt", Description: "Alternative for step 1"},
			},
		},
	}

	// Select alternative for step-2 (no match)
	failedStep := Step{ID: "step-2"}
	selected := ap.SelectBestAlternative(alternatives, failedStep)

	// Should return first alternative as fallback
	if selected == nil {
		t.Error("SelectBestAlternative() should not return nil")
	}
	if selected.ID != "alt-1" {
		t.Errorf("SelectBestAlternative() selected %v, want alt-1", selected.ID)
	}
}

func TestAlternativePlanner_SelectBestAlternative_Empty(t *testing.T) {
	mainPlanner := NewLLMPlanner(nil, nil)
	ap := NewAlternativePlanner(mainPlanner)

	alternatives := []AlternativePath{}

	failedStep := Step{ID: "step-1"}
	selected := ap.SelectBestAlternative(alternatives, failedStep)

	if selected != nil {
		t.Error("SelectBestAlternative() should return nil for empty alternatives")
	}
}

func TestAlternativePlanner_SuggestAlternativeTool(t *testing.T) {
	mainPlanner := NewLLMPlanner(nil, nil)
	ap := NewAlternativePlanner(mainPlanner)

	// Test read_file -> execute_command
	action := Action{
		Type: "tool_call",
		Tool: "read_file",
		Params: map[string]interface{}{
			"path": "main.go",
		},
	}

	altAction := ap.suggestAlternativeTool(action)

	if altAction.Tool != "execute_command" {
		t.Errorf("suggestAlternativeTool() tool = %v, want execute_command", altAction.Tool)
	}

	// Test write_file -> execute_command
	action = Action{
		Type: "tool_call",
		Tool: "write_file",
		Params: map[string]interface{}{
			"path":    "output.txt",
			"content": "hello",
		},
	}

	altAction = ap.suggestAlternativeTool(action)

	if altAction.Tool != "execute_command" {
		t.Errorf("suggestAlternativeTool() tool = %v, want execute_command", altAction.Tool)
	}
}
