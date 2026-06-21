package planner

import (
	"context"
	"testing"
)

// MockMemoryReader implements MemoryReader for testing
type MockMemoryReader struct {
	sessionSummary string
	stepResults    []StepResult
}

func (m *MockMemoryReader) GetSessionSummary() string {
	return m.sessionSummary
}

func (m *MockMemoryReader) GetStepResults() []StepResult {
	return m.stepResults
}

// MockLLMProvider implements LLMProvider for testing
type MockLLMProvider struct {
	response string
	err      error
}

func (m *MockLLMProvider) Chat(ctx context.Context, messages []Message) (string, error) {
	return m.response, m.err
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Model != "deepseek-chat" {
		t.Errorf("DefaultConfig().Model = %v, want deepseek-chat", config.Model)
	}
	if config.MaxSteps != 15 {
		t.Errorf("DefaultConfig().MaxSteps = %v, want 15", config.MaxSteps)
	}
	if config.Temperature != 0.7 {
		t.Errorf("DefaultConfig().Temperature = %v, want 0.7", config.Temperature)
	}
}

func TestExtractJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple JSON",
			input:    `{"key": "value"}`,
			expected: `{"key": "value"}`,
		},
		{
			name:     "JSON with surrounding text",
			input:    `Here is the plan: {"key": "value"} end`,
			expected: `{"key": "value"}`,
		},
		{
			name:     "no JSON",
			input:    "no json here",
			expected: "",
		},
		{
			name:     "nested JSON",
			input:    `{"outer": {"inner": "value"}}`,
			expected: `{"outer": {"inner": "value"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractJSON(tt.input)
			if result != tt.expected {
				t.Errorf("extractJSON() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestParsePlan(t *testing.T) {
	provider := &MockLLMProvider{}
	planner := NewLLMPlanner(provider, nil)

	tests := []struct {
		name     string
		input    string
		wantErr  bool
		numSteps int
	}{
		{
			name: "valid plan",
			input: `{
				"id": "plan-1",
				"steps": [
					{"id": "step-1", "description": "Step 1", "action": {"type": "tool_call"}},
					{"id": "step-2", "description": "Step 2", "action": {"type": "llm_generate"}}
				],
				"rationale": "Test plan"
			}`,
			wantErr:  false,
			numSteps: 2,
		},
		{
			name:     "empty steps",
			input:    `{"id": "plan-1", "steps": [], "rationale": "Empty"}`,
			wantErr:  false, // parsePlan doesn't validate, validatePlan does
			numSteps: 0,
		},
		{
			name:     "invalid JSON",
			input:    `{invalid json}`,
			wantErr:  true,
			numSteps: 0,
		},
		{
			name:     "no JSON",
			input:    "no json",
			wantErr:  true,
			numSteps: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan, err := planner.parsePlan(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parsePlan() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(plan.Steps) != tt.numSteps {
				t.Errorf("parsePlan() steps = %v, want %v", len(plan.Steps), tt.numSteps)
			}
		})
	}
}

func TestValidatePlan(t *testing.T) {
	provider := &MockLLMProvider{}
	planner := NewLLMPlanner(provider, nil)

	tests := []struct {
		name    string
		plan    *Plan
		wantErr bool
	}{
		{
			name: "valid plan",
			plan: &Plan{
				ID: "plan-1",
				Steps: []Step{
					{ID: "step-1", Description: "Step 1"},
					{ID: "step-2", Description: "Step 2", DependsOn: []string{"step-1"}},
				},
			},
			wantErr: false,
		},
		{
			name:    "empty plan",
			plan:    &Plan{ID: "plan-1", Steps: []Step{}},
			wantErr: true,
		},
		{
			name: "too many steps",
			plan: &Plan{
				ID: "plan-1",
				Steps: func() []Step {
					steps := make([]Step, 20)
					for i := range steps {
						steps[i] = Step{ID: "step-" + string(rune('A'+i))}
					}
					return steps
				}(),
			},
			wantErr: true,
		},
		{
			name: "invalid dependency",
			plan: &Plan{
				ID: "plan-1",
				Steps: []Step{
					{ID: "step-1", Description: "Step 1", DependsOn: []string{"nonexistent"}},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := planner.validatePlan(tt.plan)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePlan() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
