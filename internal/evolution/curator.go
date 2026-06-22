// 守卫者 - 定期审查技能库，标记过时/归档/合并
// 参考Hermes curator.py
package evolution

import (
	"fmt"
	"sync"
	"time"
)

// SkillState 技能状态
type SkillState string

const (
	StateActive   SkillState = "active"   // 活跃
	StateStale    SkillState = "stale"    // 过时（30天未使用）
	StateArchived SkillState = "archived" // 归档（90天未使用）
)

// SkillRecord 技能记录（供守卫者扫描）
type SkillRecord struct {
	Name      string
	LastUsed  time.Time
	UseCount  int
	Pinned    bool
	State     SkillState
	CreatedAt time.Time
	Tags      []string
}

// Curator 守卫者
type Curator struct {
	mu               sync.Mutex
	staleAfterDays   int  // 标记过时的天数（默认30）
	archiveAfterDays int  // 归档的天数（默认90）
	consolidate      bool // 是否合并相似技能（默认关闭）
	interval         time.Duration
}

// CuratorConfig 守卫者配置
type CuratorConfig struct {
	StaleAfterDays   int
	ArchiveAfterDays int
	Consolidate      bool
	Interval         time.Duration
}

// DefaultCuratorConfig 默认配置
func DefaultCuratorConfig() *CuratorConfig {
	return &CuratorConfig{
		StaleAfterDays:   30,
		ArchiveAfterDays: 90,
		Consolidate:      false,
		Interval:         24 * 7 * time.Hour, // 每周一次
	}
}

// NewCurator 创建守卫者
func NewCurator(cfg *CuratorConfig) *Curator {
	if cfg == nil {
		cfg = DefaultCuratorConfig()
	}
	return &Curator{
		staleAfterDays:   cfg.StaleAfterDays,
		archiveAfterDays: cfg.ArchiveAfterDays,
		consolidate:      cfg.Consolidate,
		interval:         cfg.Interval,
	}
}

// CuratorAction 守卫者执行的动作
type CuratorAction struct {
	SkillName string
	Action    string // mark_stale / mark_archived / suggest_merge
	Reason    string
}

// Run 执行一次审查
func (c *Curator) Run(skills []SkillRecord) []CuratorAction {
	c.mu.Lock()
	defer c.mu.Unlock()

	var actions []CuratorAction
	now := time.Now()

	for _, skill := range skills {
		if skill.Pinned {
			continue // 钉住的技能永远不动
		}

		daysSinceUse := now.Sub(skill.LastUsed).Hours() / 24

		switch {
		case daysSinceUse >= float64(c.archiveAfterDays):
			if skill.State != StateArchived {
				actions = append(actions, CuratorAction{
					SkillName: skill.Name,
					Action:    "mark_archived",
					Reason:    fmt.Sprintf("已归档：%.0f天未使用（阈值：%d天）", daysSinceUse, c.archiveAfterDays),
				})
			}

		case daysSinceUse >= float64(c.staleAfterDays):
			if skill.State != StateStale {
				actions = append(actions, CuratorAction{
					SkillName: skill.Name,
					Action:    "mark_stale",
					Reason:    fmt.Sprintf("已过时：%.0f天未使用（阈值：%d天）", daysSinceUse, c.staleAfterDays),
				})
			}
		}
	}

	// 合并相似技能（可选，默认关闭）
	if c.consolidate && len(skills) > 1 {
		consolidations := c.findConsolidations(skills)
		actions = append(actions, consolidations...)
	}

	return actions
}

// ShouldRunNow 判断是否应该运行
func (c *Curator) ShouldRunNow(lastRun time.Time) bool {
	return time.Since(lastRun) >= c.interval
}

// Start 启动定时审查
func (c *Curator) Start(skillsFn func() []SkillRecord, callback func([]CuratorAction)) {
	ticker := time.NewTicker(c.interval)
	go func() {
		for range ticker.C {
			skills := skillsFn()
			if len(skills) == 0 {
				continue
			}

			actions := c.Run(skills)
			if len(actions) > 0 && callback != nil {
				callback(actions)
			}
		}
	}()
}

func (c *Curator) findConsolidations(skills []SkillRecord) []CuratorAction {
	var actions []CuratorAction
	// 简单的启发式：同标签且很少使用的技能建议合并
	tagGroups := make(map[string][]string)
	for _, s := range skills {
		for _, tag := range s.Tags {
			tagGroups[tag] = append(tagGroups[tag], s.Name)
		}
	}

	for tag, names := range tagGroups {
		if len(names) >= 3 {
			actions = append(actions, CuratorAction{
				SkillName: names[0],
				Action:    "suggest_merge",
				Reason:    fmt.Sprintf("标签 '%s' 下有 %d 个技能，建议合并", tag, len(names)),
			})
		}
	}

	return actions
}
