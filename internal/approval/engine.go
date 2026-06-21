package approval

import (
	"fmt"
	"strings"
)

// ExecutionMode represents the execution mode
type ExecutionMode int

const (
	ModeAsk  ExecutionMode = iota // Ask mode: ask for risky operations
	ModeAuto                       // Auto mode: auto-approve low risk, ask for high risk
	ModeYolo                       // YOLO mode: auto-approve everything
)

// String returns the string representation
func (m ExecutionMode) String() string {
	switch m {
	case ModeAsk:
		return "ask"
	case ModeAuto:
		return "auto"
	case ModeYolo:
		return "yolo"
	default:
		return "auto"
	}
}

// Permission represents the permission level
type Permission int

const (
	PermissionDeny  Permission = iota // Deny: always deny
	PermissionAsk                      // Ask: ask for approval
	PermissionAllow                    // Allow: always allow
)

// String returns the string representation
func (p Permission) String() string {
	switch p {
	case PermissionDeny:
		return "deny"
	case PermissionAsk:
		return "ask"
	case PermissionAllow:
		return "allow"
	default:
		return "ask"
	}
}

// Rule represents an approval rule
type Rule struct {
	ToolPattern string     // Tool pattern (supports wildcards)
	Permission  Permission
	Description string
}

// DynamicRule represents a dynamic approval rule (NB-Agent style)
// Dynamic rules can evaluate tool name and parameters to make decisions
type DynamicRule struct {
	// Name is the rule name
	Name string

	// Description describes what the rule does
	Description string

	// Evaluator evaluates the tool call and returns (matches, permission)
	// matches=true means this rule applies
	// permission is the permission to grant if matches=true
	Evaluator func(toolName string, params map[string]interface{}) (matches bool, permission Permission)
}

// ApprovalCallback is called when approval is needed
type ApprovalCallback func(toolName string, params map[string]interface{}) bool

// ApprovalEngine manages tool execution approval
type ApprovalEngine struct {
	mode         ExecutionMode
	rules        []Rule
	dynamicRules []DynamicRule
	callback     ApprovalCallback
}

// NewApprovalEngine creates a new approval engine
func NewApprovalEngine(mode ExecutionMode, callback ApprovalCallback) *ApprovalEngine {
	return &ApprovalEngine{
		mode:         mode,
		rules:        DefaultRules(),
		dynamicRules: make([]DynamicRule, 0),
		callback:     callback,
	}
}

// AddDynamicRule adds a dynamic rule
func (ae *ApprovalEngine) AddDynamicRule(rule DynamicRule) {
	ae.dynamicRules = append(ae.dynamicRules, rule)
}

// RemoveDynamicRule removes a dynamic rule by name
func (ae *ApprovalEngine) RemoveDynamicRule(name string) {
	for i, rule := range ae.dynamicRules {
		if rule.Name == name {
			ae.dynamicRules = append(ae.dynamicRules[:i], ae.dynamicRules[i+1:]...)
			return
		}
	}
}

// GetDynamicRules returns all dynamic rules
func (ae *ApprovalEngine) GetDynamicRules() []DynamicRule {
	return ae.dynamicRules
}

// SetMode changes the execution mode
func (ae *ApprovalEngine) SetMode(mode ExecutionMode) {
	ae.mode = mode
}

// GetMode returns the current execution mode
func (ae *ApprovalEngine) GetMode() ExecutionMode {
	return ae.mode
}

// AddRule adds a rule
func (ae *ApprovalEngine) AddRule(rule Rule) {
	ae.rules = append(ae.rules, rule)
}

// SetRules sets all rules
func (ae *ApprovalEngine) SetRules(rules []Rule) {
	ae.rules = rules
}

// GetRules returns all rules
func (ae *ApprovalEngine) GetRules() []Rule {
	return ae.rules
}

// CheckPermission checks the permission for a tool
func (ae *ApprovalEngine) CheckPermission(toolName string, params map[string]interface{}) Permission {
	// YOLO mode: always allow
	if ae.mode == ModeYolo {
		return PermissionAllow
	}

	// Check dynamic rules first (NB-Agent style)
	for _, rule := range ae.dynamicRules {
		if matches, permission := rule.Evaluator(toolName, params); matches {
			return permission
		}
	}

	// Find matching static rule
	for _, rule := range ae.rules {
		if matchPattern(rule.ToolPattern, toolName) {
			// Ask mode: follow rule exactly
			if ae.mode == ModeAsk {
				return rule.Permission
			}

			// Auto mode: deny stays deny, ask becomes allow (except dangerous)
			if ae.mode == ModeAuto {
				if rule.Permission == PermissionDeny {
					return PermissionDeny
				}
				return PermissionAllow
			}
		}
	}

	// Default: ask mode asks, auto/yolo allows
	if ae.mode == ModeAsk {
		return PermissionAsk
	}
	return PermissionAllow
}

// RequestApproval requests approval for a tool call
// Returns: (approved, error)
// - approved=true: execute the tool
// - approved=false: skip the tool, step fails
func (ae *ApprovalEngine) RequestApproval(toolName string, params map[string]interface{}) (bool, error) {
	permission := ae.CheckPermission(toolName, params)

	switch permission {
	case PermissionDeny:
		return false, fmt.Errorf("tool %s is denied by policy", toolName)

	case PermissionAllow:
		return true, nil

	case PermissionAsk:
		// Need human approval
		if ae.callback == nil {
			return false, fmt.Errorf("no approval callback set for ask mode")
		}
		approved := ae.callback(toolName, params)
		return approved, nil
	}

	return false, fmt.Errorf("unknown permission level")
}

// matchPattern matches a tool name against a pattern (supports wildcards)
func matchPattern(pattern string, toolName string) bool {
	// Exact match
	if pattern == toolName {
		return true
	}

	// Suffix match: *.py matches file.py
	if strings.HasPrefix(pattern, "*.") {
		suffix := strings.TrimPrefix(pattern, "*")
		return strings.HasSuffix(toolName, suffix)
	}

	// Wildcard match: bash(rm -rf*)
	if strings.Contains(pattern, "*") {
		prefix := strings.Split(pattern, "*")[0]
		if prefix == "" {
			return true // Pattern is just "*"
		}
		return strings.HasPrefix(toolName, prefix)
	}

	// Prefix match: bash matches bash(ls), bash(cat), etc.
	if strings.HasPrefix(toolName, pattern) {
		return true
	}

	return false
}

// DefaultRules returns default approval rules
func DefaultRules() []Rule {
	return []Rule{
		// File operations: allow read, ask for write/delete
		{ToolPattern: "read_file", Permission: PermissionAllow, Description: "Read file"},
		{ToolPattern: "write_file", Permission: PermissionAsk, Description: "Write file"},
		{ToolPattern: "delete_file", Permission: PermissionAsk, Description: "Delete file"},

		// Search: allow
		{ToolPattern: "search_file", Permission: PermissionAllow, Description: "Search file"},
		{ToolPattern: "web_search", Permission: PermissionAllow, Description: "Web search"},

		// Command execution: ask
		{ToolPattern: "execute_command", Permission: PermissionAsk, Description: "Execute command"},

		// Dangerous commands: deny
		{ToolPattern: "rm -rf", Permission: PermissionDeny, Description: "Dangerous: rm -rf"},
		{ToolPattern: "format", Permission: PermissionDeny, Description: "Dangerous: format"},
		{ToolPattern: "mkfs", Permission: PermissionDeny, Description: "Dangerous: mkfs"},
		{ToolPattern: "dd if=", Permission: PermissionDeny, Description: "Dangerous: dd"},
	}
}
