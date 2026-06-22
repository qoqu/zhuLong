// 建议引擎 - 检测重复需求，主动建议创建技能/自动化
// 参考Hermes suggestion_catalog.py
package evolution

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// SuggestionType 建议类型
type SuggestionType string

const (
	SuggestionSkill     SuggestionType = "create_skill"     // 创建技能
	SuggestionBlueprint SuggestionType = "create_blueprint" // 创建自动化蓝图
	SuggestionUpdate    SuggestionType = "update_skill"     // 更新技能
)

// Suggestion 建议
type Suggestion struct {
	ID          string
	Type        SuggestionType
	Title       string
	Description string
	Priority    int    // 1-5
	Source      string // catalog / usage / integration
	CreatedAt   time.Time
	Accepted    bool
}

// SuggestionEngine 建议引擎
type SuggestionEngine struct {
	mu           sync.Mutex
	suggestions  []Suggestion
	maxPending   int // 最多5个待处理建议（防止骚扰）
	usedPatterns map[string]int
}

// NewSuggestionEngine 创建建议引擎
func NewSuggestionEngine() *SuggestionEngine {
	return &SuggestionEngine{
		suggestions:  make([]Suggestion, 0),
		maxPending:   5,
		usedPatterns: make(map[string]int),
	}
}

// FromPattern 从重复模式生成建议
func (e *SuggestionEngine) FromPattern(pattern string, count int) *Suggestion {
	e.mu.Lock()
	defer e.mu.Unlock()

	// 检查是否已存在相同模式建议
	for _, s := range e.suggestions {
		if strings.Contains(s.Description, pattern) && !s.Accepted {
			return nil
		}
	}

	// 检查待处理限制
	pendingCount := 0
	for _, s := range e.suggestions {
		if !s.Accepted {
			pendingCount++
		}
	}
	if pendingCount >= e.maxPending {
		return nil
	}

	e.usedPatterns[pattern] = count

	suggestion := &Suggestion{
		ID:          fmt.Sprintf("sug-%d", time.Now().UnixNano()),
		Type:        SuggestionSkill,
		Title:       fmt.Sprintf("自动创建技能：%s", pattern),
		Description: fmt.Sprintf("检测到模式 '%s' 已出现 %d 次，建议创建为可复用技能", pattern, count),
		Priority:    min(5, count),
		Source:      "usage",
		CreatedAt:   time.Now(),
	}

	e.suggestions = append(e.suggestions, *suggestion)
	return suggestion
}

// FromCatalog 从内置目录生成建议
func (e *SuggestionEngine) FromCatalog(title, desc string) *Suggestion {
	e.mu.Lock()
	defer e.mu.Unlock()

	pendingCount := 0
	for _, s := range e.suggestions {
		if !s.Accepted {
			pendingCount++
		}
	}
	if pendingCount >= e.maxPending {
		return nil
	}

	suggestion := &Suggestion{
		ID:          fmt.Sprintf("sug-%d", time.Now().UnixNano()),
		Type:        SuggestionBlueprint,
		Title:       title,
		Description: desc,
		Priority:    2,
		Source:      "catalog",
		CreatedAt:   time.Now(),
	}

	e.suggestions = append(e.suggestions, *suggestion)
	return suggestion
}

// Accept 接受建议
func (e *SuggestionEngine) Accept(id string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()

	for i, s := range e.suggestions {
		if s.ID == id {
			e.suggestions[i].Accepted = true
			return true
		}
	}
	return false
}

// Dismiss 忽略建议
func (e *SuggestionEngine) Dismiss(id string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()

	for i, s := range e.suggestions {
		if s.ID == id {
			e.suggestions = append(e.suggestions[:i], e.suggestions[i+1:]...)
			return true
		}
	}
	return false
}

// ListPending 列出待处理建议
func (e *SuggestionEngine) ListPending() []Suggestion {
	e.mu.Lock()
	defer e.mu.Unlock()

	var pending []Suggestion
	for _, s := range e.suggestions {
		if !s.Accepted {
			pending = append(pending, s)
		}
	}

	sort.Slice(pending, func(i, j int) bool {
		return pending[i].Priority > pending[j].Priority
	})

	return pending
}

// ListAll 列出所有建议
func (e *SuggestionEngine) ListAll() []Suggestion {
	e.mu.Lock()
	defer e.mu.Unlock()
	suggestions := make([]Suggestion, len(e.suggestions))
	copy(suggestions, e.suggestions)
	return suggestions
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
