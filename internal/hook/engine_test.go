package hook

import (
	"context"
	"testing"
)

func TestHookEngine_Register(t *testing.T) {
	engine := NewHookEngine()

	h := Hook{
		Name:        "test",
		Phase:       PhasePreTool,
		Priority:    0,
		Description: "test hook",
		Fn: func(ctx context.Context, params map[string]interface{}) (*HookResult, error) {
			return &HookResult{}, nil
		},
	}

	engine.Register(h)

	if len(engine.hooks) != 1 {
		t.Errorf("expected 1 hook, got %d", len(engine.hooks))
	}
}

func TestSecurityInterceptHook_BlockDangerous(t *testing.T) {
	engine := NewHookEngine()
	engine.Register(SecurityInterceptHook())

	tests := []struct {
		name     string
		params   map[string]interface{}
		blocked  bool
	}{
		{"rm -rf /", map[string]interface{}{"tool": "execute_command", "args": "rm -rf /"}, true},
		{"safe command", map[string]interface{}{"tool": "execute_command", "args": "ls -la"}, false},
		{"write .env", map[string]interface{}{"tool": "write_file", "args": ".env"}, true},
		{"write normal", map[string]interface{}{"tool": "write_file", "args": "main.go"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := engine.Execute(context.Background(), PhasePreTool, tt.params)
			if err != nil {
				t.Fatalf("Execute failed: %v", err)
			}

			blocked := len(results) > 0 && results[0].Blocked
			if blocked != tt.blocked {
				t.Errorf("expected blocked=%v, got blocked=%v", tt.blocked, blocked)
			}
		})
	}
}

func TestContextInjectHook(t *testing.T) {
	engine := NewHookEngine()
	engine.Register(ContextInjectHook("/test/project"))

	results, err := engine.Execute(context.Background(), PhaseSession, map[string]interface{}{})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("expected results, got none")
	}

	meta := results[0].Metadata
	if meta["project_path"] != "/test/project" {
		t.Errorf("expected project_path=/test/project, got %v", meta["project_path"])
	}
	if meta["git_branch"] == "" {
		t.Error("expected git_branch to be set")
	}
}

func TestReviewFeedbackHook(t *testing.T) {
	engine := NewHookEngine()
	engine.Register(ReviewFeedbackHook())

	tests := []struct {
		name       string
		params     map[string]interface{}
		needsReview bool
	}{
		{"write file", map[string]interface{}{"tool": "write_file"}, true},
		{"read file", map[string]interface{}{"tool": "read_file"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := engine.Execute(context.Background(), PhasePostTool, tt.params)
			if err != nil {
				t.Fatalf("Execute failed: %v", err)
			}

			needsReview := false
			if len(results) > 0 {
				if v, ok := results[0].Metadata["needs_review"]; ok {
					needsReview = v.(bool)
				}
			}

			if needsReview != tt.needsReview {
				t.Errorf("expected needsReview=%v, got %v", tt.needsReview, needsReview)
			}
		})
	}
}

func TestMultipleHooks(t *testing.T) {
	engine := NewHookEngine()
	engine.Register(SecurityInterceptHook())
	engine.Register(ContextInjectHook("/test"))

	// 安全拦截应阻断后续钩子
	results, err := engine.Execute(context.Background(), PhasePreTool, map[string]interface{}{
		"tool": "execute_command",
		"args": "rm -rf /",
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if len(results) > 0 && !results[0].Blocked {
		t.Error("expected blocked for dangerous command")
	}
}
