// 后台审查器 - 对话后自动审查，判断是否需要创建/更新技能
package evolution

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// ReviewSignal 审查信号类型
type ReviewSignal string

const (
	SignalStyleCorrection ReviewSignal = "style_correction"  // 风格/语气修正
	SignalWorkflowChange  ReviewSignal = "workflow_change"   // 工作流方法修正
	SignalNewTechnique    ReviewSignal = "new_technique"     // 非平凡技术/技巧
	SignalSkillOutdated   ReviewSignal = "skill_outdated"    // 技能过时
)

// ReviewResult 审查结果
type ReviewResult struct {
	HasChanges  bool
	Signal      ReviewSignal
	SkillName   string   // 建议创建/修改的技能名
	Description string   // 技能描述
	Content     string   // 技能内容
	Priority    int      // 优先级 1-5
}

// BackgroundReviewer 后台审查器
type BackgroundReviewer struct {
	mu          sync.Mutex
	provider    ReviewProvider
	sessionLogs []SessionSnapshot
	maxSessions int
}

// SessionSnapshot 会话快照
type SessionSnapshot struct {
	Goal     string
	Messages []MessagePair
	Duration time.Duration
}

// MessagePair 消息对
type MessagePair struct {
	Role    string // user / assistant / tool
	Content string
}

// ReviewProvider 审查用LLM提供者（权限受限，只允许memory和skill_manage）
type ReviewProvider interface {
	Review(snapshot SessionSnapshot) (*ReviewResult, error)
}

// NewBackgroundReviewer 创建后台审查器
func NewBackgroundReviewer(provider ReviewProvider) *BackgroundReviewer {
	return &BackgroundReviewer{
		provider:    provider,
		sessionLogs: make([]SessionSnapshot, 0),
		maxSessions: 50,
	}
}

// RecordSession 记录会话
func (r *BackgroundReviewer) RecordSession(snapshot SessionSnapshot) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.sessionLogs = append(r.sessionLogs, snapshot)
	if len(r.sessionLogs) > r.maxSessions {
		r.sessionLogs = r.sessionLogs[len(r.sessionLogs)-r.maxSessions:]
	}
}

// Review 审查最近一次会话
func (r *BackgroundReviewer) Review() (*ReviewResult, error) {
	r.mu.Lock()
	if len(r.sessionLogs) == 0 {
		r.mu.Unlock()
		return &ReviewResult{HasChanges: false}, nil
	}
	last := r.sessionLogs[len(r.sessionLogs)-1]
	r.mu.Unlock()

	return r.provider.Review(last)
}

// AnalyzePatterns 分析历史会话中的模式
// 检测重复出现的需求，建议创建技能
func (r *BackgroundReviewer) AnalyzePatterns() []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.sessionLogs) < 3 {
		return nil
	}

	var suggestions []string
	goalCount := make(map[string]int)

	for _, s := range r.sessionLogs {
		key := normalizeGoal(s.Goal)
		goalCount[key]++
	}

	for goal, count := range goalCount {
		if count >= 3 {
			suggestions = append(suggestions,
				fmt.Sprintf("检测到重复需求 '%s' 出现 %d 次，建议创建自动化技能", goal, count))
		}
	}

	return suggestions
}

// HeuristicReview 启发式审查（无LLM，纯规则判断）
// 用于离线/测试模式
func HeuristicReview(snapshot SessionSnapshot) *ReviewResult {
	result := &ReviewResult{HasChanges: false}

	// 检测风格修正
	for i := len(snapshot.Messages) - 1; i >= 0; i-- {
		msg := snapshot.Messages[i]
		if msg.Role == "user" {
			lower := strings.ToLower(msg.Content)

			// 风格修正信号
			if strings.Contains(lower, "简洁") || strings.Contains(lower, "简短") ||
				strings.Contains(lower, "详细") || strings.Contains(lower, "格式") {
				result.HasChanges = true
				result.Signal = SignalStyleCorrection
				result.Description = "用户对输出格式/风格有明确偏好，建议更新技能"
				result.Priority = 3
				return result
			}

			// 工作流修正信号
			if strings.Contains(lower, "步骤") || strings.Contains(lower, "先") ||
				strings.Contains(lower, "顺序") {
				result.HasChanges = true
				result.Signal = SignalWorkflowChange
				result.Description = "用户纠正了工作流程，建议更新技能"
				result.Priority = 4
				return result
			}

			// 新技巧信号
			if strings.Contains(lower, "技巧") || strings.Contains(lower, "方法") ||
				strings.Contains(lower, "更好的") {
				result.HasChanges = true
				result.Signal = SignalNewTechnique
				result.Description = "用户提到了非平凡的技术方法，建议创建新技能"
				result.Priority = 2
				return result
			}
		}
	}

	return result
}

// Start 启动后台审查循环
func (r *BackgroundReviewer) Start(interval time.Duration, callback func(*ReviewResult)) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			r.mu.Lock()
			count := len(r.sessionLogs)
			r.mu.Unlock()

			if count == 0 {
				continue
			}

			result, err := r.Review()
			if err != nil {
				continue
			}

			if result.HasChanges && callback != nil {
				callback(result)
			}
		}
	}()
}

func normalizeGoal(goal string) string {
	goal = strings.ToLower(strings.TrimSpace(goal))
	if len(goal) > 50 {
		goal = goal[:50]
	}
	return goal
}
