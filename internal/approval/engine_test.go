package approval

import (
	"testing"
)

func TestExecutionMode_String(t *testing.T) {
	tests := []struct {
		mode     ExecutionMode
		expected string
	}{
		{ModeAsk, "ask"},
		{ModeAuto, "auto"},
		{ModeYolo, "yolo"},
		{ExecutionMode(99), "auto"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.mode.String(); got != tt.expected {
				t.Errorf("ExecutionMode.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestPermission_String(t *testing.T) {
	tests := []struct {
		permission Permission
		expected   string
	}{
		{PermissionDeny, "deny"},
		{PermissionAsk, "ask"},
		{PermissionAllow, "allow"},
		{Permission(99), "ask"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.permission.String(); got != tt.expected {
				t.Errorf("Permission.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestNewApprovalEngine(t *testing.T) {
	ae := NewApprovalEngine(ModeAuto, nil)

	if ae == nil {
		t.Error("NewApprovalEngine() should not return nil")
	}
	if ae.GetMode() != ModeAuto {
		t.Errorf("Mode = %v, want ModeAuto", ae.GetMode())
	}
	if len(ae.GetRules()) == 0 {
		t.Error("Should have default rules")
	}
}

func TestApprovalEngine_SetMode(t *testing.T) {
	ae := NewApprovalEngine(ModeAuto, nil)

	ae.SetMode(ModeYolo)
	if ae.GetMode() != ModeYolo {
		t.Errorf("Mode = %v, want ModeYolo", ae.GetMode())
	}
}

func TestApprovalEngine_CheckPermission_YoloMode(t *testing.T) {
	ae := NewApprovalEngine(ModeYolo, nil)

	// YOLO mode: always allow
	if ae.CheckPermission("read_file", nil) != PermissionAllow {
		t.Error("YOLO mode should allow read_file")
	}
	if ae.CheckPermission("write_file", nil) != PermissionAllow {
		t.Error("YOLO mode should allow write_file")
	}
	if ae.CheckPermission("execute_command", nil) != PermissionAllow {
		t.Error("YOLO mode should allow execute_command")
	}
	if ae.CheckPermission("rm -rf", nil) != PermissionAllow {
		t.Error("YOLO mode should allow rm -rf")
	}
}

func TestApprovalEngine_CheckPermission_AskMode(t *testing.T) {
	ae := NewApprovalEngine(ModeAsk, nil)

	// Ask mode: follow rules
	if ae.CheckPermission("read_file", nil) != PermissionAllow {
		t.Error("Ask mode should allow read_file")
	}
	if ae.CheckPermission("write_file", nil) != PermissionAsk {
		t.Error("Ask mode should ask for write_file")
	}
	if ae.CheckPermission("execute_command", nil) != PermissionAsk {
		t.Error("Ask mode should ask for execute_command")
	}
	if ae.CheckPermission("rm -rf", nil) != PermissionDeny {
		t.Error("Ask mode should deny rm -rf")
	}
}

func TestApprovalEngine_CheckPermission_AutoMode(t *testing.T) {
	ae := NewApprovalEngine(ModeAuto, nil)

	// Auto mode: deny stays deny, ask becomes allow
	if ae.CheckPermission("read_file", nil) != PermissionAllow {
		t.Error("Auto mode should allow read_file")
	}
	if ae.CheckPermission("write_file", nil) != PermissionAllow {
		t.Error("Auto mode should allow write_file")
	}
	if ae.CheckPermission("execute_command", nil) != PermissionAllow {
		t.Error("Auto mode should allow execute_command")
	}
	if ae.CheckPermission("rm -rf", nil) != PermissionDeny {
		t.Error("Auto mode should deny rm -rf")
	}
}

func TestApprovalEngine_RequestApproval_Allow(t *testing.T) {
	ae := NewApprovalEngine(ModeAuto, nil)

	approved, err := ae.RequestApproval("read_file", nil)
	if err != nil {
		t.Errorf("RequestApproval() error = %v", err)
	}
	if !approved {
		t.Error("RequestApproval() should approve read_file")
	}
}

func TestApprovalEngine_RequestApproval_Deny(t *testing.T) {
	ae := NewApprovalEngine(ModeAsk, nil)

	approved, err := ae.RequestApproval("rm -rf", nil)
	if err == nil {
		t.Error("RequestApproval() should return error for denied tool")
	}
	if approved {
		t.Error("RequestApproval() should not approve rm -rf")
	}
}

func TestApprovalEngine_RequestApproval_Ask(t *testing.T) {
	callback := func(toolName string, params map[string]interface{}) bool {
		return toolName == "write_file"
	}

	ae := NewApprovalEngine(ModeAsk, callback)

	// Should be approved by callback
	approved, err := ae.RequestApproval("write_file", nil)
	if err != nil {
		t.Errorf("RequestApproval() error = %v", err)
	}
	if !approved {
		t.Error("RequestApproval() should approve write_file")
	}

	// Should be rejected by callback
	approved, err = ae.RequestApproval("execute_command", nil)
	if err != nil {
		t.Errorf("RequestApproval() error = %v", err)
	}
	if approved {
		t.Error("RequestApproval() should reject execute_command")
	}
}

func TestApprovalEngine_RequestApproval_AskNoCallback(t *testing.T) {
	ae := NewApprovalEngine(ModeAsk, nil)

	_, err := ae.RequestApproval("write_file", nil)
	if err == nil {
		t.Error("RequestApproval() should return error when no callback set")
	}
}

func TestApprovalEngine_AddRule(t *testing.T) {
	ae := NewApprovalEngine(ModeAsk, nil)

	initialRules := len(ae.GetRules())

	ae.AddRule(Rule{
		ToolPattern: "custom_tool",
		Permission:  PermissionAllow,
		Description: "Custom tool",
	})

	if len(ae.GetRules()) != initialRules+1 {
		t.Errorf("Rules count = %v, want %v", len(ae.GetRules()), initialRules+1)
	}
}

func TestMatchPattern(t *testing.T) {
	tests := []struct {
		pattern  string
		toolName string
		expected bool
	}{
		{"read_file", "read_file", true},
		{"read_file", "write_file", false},
		{"bash", "bash", true},
		{"bash", "bash(ls)", true},
		{"bash", "cat", false},
		{"*.py", "script.py", true},
		{"*.py", "script.js", false},
		{"rm -rf", "rm -rf /", true},
		{"rm -rf", "rm -rf", true},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.toolName, func(t *testing.T) {
			if got := matchPattern(tt.pattern, tt.toolName); got != tt.expected {
				t.Errorf("matchPattern(%v, %v) = %v, want %v", tt.pattern, tt.toolName, got, tt.expected)
			}
		})
	}
}

func TestDefaultRules(t *testing.T) {
	rules := DefaultRules()

	if len(rules) == 0 {
		t.Error("DefaultRules() should not be empty")
	}

	// Check some default rules
	hasReadFile := false
	hasWriteFile := false
	hasRmRf := false

	for _, rule := range rules {
		switch rule.ToolPattern {
		case "read_file":
			hasReadFile = true
			if rule.Permission != PermissionAllow {
				t.Error("read_file should be allowed")
			}
		case "write_file":
			hasWriteFile = true
			if rule.Permission != PermissionAsk {
				t.Error("write_file should be ask")
			}
		case "rm -rf":
			hasRmRf = true
			if rule.Permission != PermissionDeny {
				t.Error("rm -rf should be denied")
			}
		}
	}

	if !hasReadFile {
		t.Error("Should have read_file rule")
	}
	if !hasWriteFile {
		t.Error("Should have write_file rule")
	}
	if !hasRmRf {
		t.Error("Should have rm -rf rule")
	}
}
