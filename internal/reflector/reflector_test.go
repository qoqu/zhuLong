package reflector

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

func TestDecision_String(t *testing.T) {
	tests := []struct {
		decision Decision
		expected string
	}{
		{DecisionComplete, "complete"},
		{DecisionContinue, "continue"},
		{DecisionReplan, "replan"},
		{DecisionFail, "fail"},
		{Decision(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.decision.String(); got != tt.expected {
				t.Errorf("Decision.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Model != "deepseek-chat" {
		t.Errorf("DefaultConfig().Model = %v, want deepseek-chat", config.Model)
	}
	if config.Temperature != 0.7 {
		t.Errorf("DefaultConfig().Temperature = %v, want 0.7", config.Temperature)
	}
	if config.ConfidenceThreshold != 0.7 {
		t.Errorf("DefaultConfig().ConfidenceThreshold = %v, want 0.7", config.ConfidenceThreshold)
	}
	if config.AutoFailThreshold != 0.3 {
		t.Errorf("DefaultConfig().AutoFailThreshold = %v, want 0.3", config.AutoFailThreshold)
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
			input:    `{"decision": "complete", "reason": "done"}`,
			expected: `{"decision": "complete", "reason": "done"}`,
		},
		{
			name:     "JSON with surrounding text",
			input:    `Here is the assessment: {"decision": "continue"} end`,
			expected: `{"decision": "continue"}`,
		},
		{
			name:     "no JSON",
			input:    "no json here",
			expected: "",
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

func TestParseAssessment(t *testing.T) {
	provider := &MockLLMProvider{}
	reflector := NewLLMReflector(provider, nil)

	tests := []struct {
		name       string
		input      string
		wantErr    bool
		decision   Decision
		confidence float64
	}{
		{
			name:       "complete decision",
			input:      `{"decision": "complete", "reason": "Goal achieved", "confidence": 0.95}`,
			wantErr:    false,
			decision:   DecisionComplete,
			confidence: 0.95,
		},
		{
			name:       "continue decision",
			input:      `{"decision": "continue", "reason": "Progress is good", "confidence": 0.8}`,
			wantErr:    false,
			decision:   DecisionContinue,
			confidence: 0.8,
		},
		{
			name:       "replan decision",
			input:      `{"decision": "replan", "reason": "Need to adjust", "confidence": 0.6}`,
			wantErr:    false,
			decision:   DecisionReplan,
			confidence: 0.6,
		},
		{
			name:       "fail decision",
			input:      `{"decision": "fail", "reason": "Cannot complete", "confidence": 0.9}`,
			wantErr:    false,
			decision:   DecisionFail,
			confidence: 0.9,
		},
		{
			name:    "unknown decision",
			input:   `{"decision": "unknown", "reason": "test"}`,
			wantErr: true,
		},
		{
			name:    "invalid JSON",
			input:   `{invalid json}`,
			wantErr: true,
		},
		{
			name:    "no JSON",
			input:   "no json",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assessment, err := reflector.parseAssessment(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseAssessment() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if assessment.Decision != tt.decision {
					t.Errorf("parseAssessment() decision = %v, want %v", assessment.Decision, tt.decision)
				}
				if assessment.Confidence != tt.confidence {
					t.Errorf("parseAssessment() confidence = %v, want %v", assessment.Confidence, tt.confidence)
				}
			}
		})
	}
}

func TestParseAssessment_WithFindings(t *testing.T) {
	provider := &MockLLMProvider{}
	reflector := NewLLMReflector(provider, nil)

	input := `{
		"decision": "continue",
		"reason": "Progress is good",
		"confidence": 0.8,
		"findings": ["Found file X", "Error in line 42"],
		"suggestions": ["Fix the error", "Try alternative approach"]
	}`

	assessment, err := reflector.parseAssessment(input)
	if err != nil {
		t.Fatalf("parseAssessment() error = %v", err)
	}

	if len(assessment.Findings) != 2 {
		t.Errorf("parseAssessment() findings length = %v, want 2", len(assessment.Findings))
	}
	if len(assessment.Suggestions) != 2 {
		t.Errorf("parseAssessment() suggestions length = %v, want 2", len(assessment.Suggestions))
	}
}
