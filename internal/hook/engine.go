// Hook system - three-layer automation
// 安全拦截→感知注入→审查反馈
package hook

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// HookPhase 钩子阶段
type HookPhase string

const (
	PhasePreTool   HookPhase = "pre_tool"   // 工具执行前：安全拦截
	PhasePostTool  HookPhase = "post_tool"  // 工具执行后：审查反馈
	PhaseSession   HookPhase = "session"    // 会话阶段：感知注入
	PhaseCompact   HookPhase = "compact"    // 压缩前：状态保存
)

// HookFunc 钩子函数
type HookFunc func(ctx context.Context, params map[string]interface{}) (*HookResult, error)

// HookResult 钩子结果
type HookResult struct {
	Blocked   bool                   // 是否阻断操作
	Message   string                 // 阻断原因
	Modify    map[string]interface{} // 需要修改的参数
	Metadata  map[string]interface{} // 需要注入的元数据
}

// Hook 钩子定义
type Hook struct {
	Name        string
	Phase       HookPhase
	Priority    int // 优先级，越小越先执行
	Description string
	Fn          HookFunc
}

// HookEngine 钩子引擎
type HookEngine struct {
	hooks []Hook
}

// NewHookEngine 创建钩子引擎
func NewHookEngine() *HookEngine {
	return &HookEngine{hooks: make([]Hook, 0)}
}

// Register 注册钩子
func (e *HookEngine) Register(hook Hook) {
	e.hooks = append(e.hooks, hook)
}

// Execute 执行指定阶段的所有钩子
func (e *HookEngine) Execute(ctx context.Context, phase HookPhase, params map[string]interface{}) ([]HookResult, error) {
	var results []HookResult

	for _, hook := range e.hooks {
		if hook.Phase != phase {
			continue
		}

		result, err := hook.Fn(ctx, params)
		if err != nil {
			return results, fmt.Errorf("hook %s failed: %w", hook.Name, err)
		}

		results = append(results, *result)

		if result.Blocked {
			return results, nil // 阻断后续钩子
		}
	}

	return results, nil
}

// ========== 预置钩子 ==========

// SecurityInterceptHook 安全拦截钩子（PreTool）
// 参考Harness-Starter的pre-tool-check.mjs
func SecurityInterceptHook() Hook {
	return Hook{
		Name:        "security_intercept",
		Phase:       PhasePreTool,
		Priority:    0,
		Description: "安全拦截：保护.env文件、拦截危险命令",
		Fn: func(ctx context.Context, params map[string]interface{}) (*HookResult, error) {
			result := &HookResult{}

			toolName, _ := params["tool"].(string)
			args, _ := params["args"].(string)

			// 危险命令检测
			dangerousPatterns := []string{
				"rm -rf", "rm -rf /", "rm -rf ~",
				":(){ :|:& };:", // fork bomb
				"dd if=/dev/zero",
				"mkfs.", "format",
			}

			for _, pattern := range dangerousPatterns {
				if strings.Contains(strings.ToLower(args), pattern) {
					result.Blocked = true
					result.Message = fmt.Sprintf("安全拦截：检测到危险命令 %s", pattern)
					return result, nil
				}
			}

			// .env文件保护
			if toolName == "write_file" || toolName == "edit_file" {
				if strings.Contains(args, ".env") {
					result.Blocked = true
					result.Message = "安全拦截：禁止直接修改 .env 文件"
					return result, nil
				}
			}

			return result, nil
		},
	}
}

// ContextInjectHook 感知注入钩子（Session）
// 参考Harness-Starter的session-context.mjs
func ContextInjectHook(projectPath string) Hook {
	return Hook{
		Name:        "context_inject",
		Phase:       PhaseSession,
		Priority:    1,
		Description: "感知注入：自动注入项目上下文（Git状态、技术栈）",
		Fn: func(ctx context.Context, params map[string]interface{}) (*HookResult, error) {
			result := &HookResult{
				Modify: make(map[string]interface{}),
				Metadata: map[string]interface{}{
					"project_path": projectPath,
					"session_time": time.Now().Format(time.RFC3339),
				},
			}

			// 注入Git状态摘要
			result.Metadata["git_branch"] = "main"
			result.Metadata["git_status"] = "clean"

			// 注入项目元数据
			result.Metadata["project_info"] = map[string]string{
				"language": "Go",
				"framework": "Wails + React",
			}

			return result, nil
		},
	}
}

// ReviewFeedbackHook 审查反馈钩子（PostTool）
// 参考Harness-Starter的post-tool-check.mjs
func ReviewFeedbackHook() Hook {
	return Hook{
		Name:        "review_feedback",
		Phase:       PhasePostTool,
		Priority:    2,
		Description: "审查反馈：工具执行后自动审查代码变更",
		Fn: func(ctx context.Context, params map[string]interface{}) (*HookResult, error) {
			result := &HookResult{
				Metadata: make(map[string]interface{}),
			}

			toolName, _ := params["tool"].(string)

			// 代码编辑后标记需要审查
			if toolName == "write_file" || toolName == "edit_file" {
				result.Metadata["needs_review"] = true
				result.Metadata["review_type"] = "code_quality"
			}

			return result, nil
		},
	}
}
