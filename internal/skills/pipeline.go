// 技能管道 - 串联技能目录加载 → 渐进披露 → 骨架压缩 → Agent上下文
package skills

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/qoqu/zhuLong/internal/compressor"
)

// Pipeline 技能管道
type Pipeline struct {
	manager   *SkillManager
	generator *compressor.SkeletonGenerator
	mu        sync.RWMutex
}

// NewPipeline 创建技能管道
func NewPipeline(searchPaths []string) *Pipeline {
	return &Pipeline{
		manager: NewSkillManager(searchPaths),
		generator: compressor.NewSkeletonGenerator(&compressor.SkeletonConfig{
			Preset:    compressor.PresetCodebase,
			FocusMode: compressor.FocusModeOutline,
			Density:   compressor.SkeletonDensityCompact,
		}),
	}
}

// Init 初始化管道：发现所有技能（Tier 1）
func (p *Pipeline) Init() error {
	return p.manager.Discover()
}

// GetSkillList 获取技能清单（注入系统提示词用）
func (p *Pipeline) GetSkillList() string {
	skills := p.manager.GetSkillList()
	if len(skills) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("\n## Available Skills\n\n")
	b.WriteString("The following skills are available. Use `skill_view` to load a skill's content.\n\n")
	b.WriteString("| Name | Description | Tags |\n")
	b.WriteString("|------|-------------|------|\n")
	for _, s := range skills {
		tags := strings.Join(s.Tags, ", ")
		if tags == "" {
			tags = "-"
		}
		b.WriteString(fmt.Sprintf("| %s | %s | %s |\n", s.Name, s.Description, tags))
	}
	return b.String()
}

// LoadAndCompress 加载技能并压缩（Tier 1→2 + 骨架压缩）
// 返回压缩后的技能内容，适合注入Agent上下文
func (p *Pipeline) LoadAndCompress(name string) (string, error) {
	// 渐进披露 Tier 1→2
	skill, err := p.manager.ViewSkill(name)
	if err != nil {
		return "", fmt.Errorf("load skill %s: %w", name, err)
	}

	// 提取YAML后的纯正文（Tier 2精要）
	body := skill.Content
	parts := strings.SplitN(body, "---", 3)
	if len(parts) >= 3 {
		body = strings.TrimSpace(parts[2])
	}

	// 骨架压缩
	compressed := p.generator.GenerateSkeleton(body)

	// 装配完整技能包：元数据 + 压缩内容
	var b strings.Builder
	b.WriteString(fmt.Sprintf("### Skill: %s\n\n", skill.Name))
	b.WriteString(fmt.Sprintf("**Description**: %s\n", skill.Description))
	if skill.Version != "" {
		b.WriteString(fmt.Sprintf("**Version**: %s\n", skill.Version))
	}
	if len(skill.Tags) > 0 {
		b.WriteString(fmt.Sprintf("**Tags**: %s\n", strings.Join(skill.Tags, ", ")))
	}
	b.WriteString("\n**Content**:\n")
	b.WriteString(compressed)
	b.WriteString("\n\nTo load the full skill content, use `skill_view`.\n")

	return b.String(), nil
}

// LoadFull 加载完整技能（Tier 1→3）
func (p *Pipeline) LoadFull(name string) (string, error) {
	skill, err := p.manager.ViewSkillFull(name)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("# Skill: %s\n\n", skill.Name))
	b.WriteString(skill.Content)
	b.WriteString("\n")

	// 附加支持文件清单
	if len(skill.SupportFiles) > 0 {
		b.WriteString("\n## Support Files\n\n")
		for path := range skill.SupportFiles {
			b.WriteString(fmt.Sprintf("- %s\n", path))
		}
	}

	return b.String(), nil
}

// Search 搜索技能
func (p *Pipeline) Search(query string) string {
	results := p.manager.SearchSkills(query)
	if len(results) == 0 {
		return "No skills found matching: " + query
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Found %d skill(s):\n\n", len(results)))
	for _, s := range results {
		b.WriteString(fmt.Sprintf("- **%s**: %s\n", s.Name, s.Description))
	}
	return b.String()
}

// Stats 返回管道统计
func (p *Pipeline) Stats() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	s := p.manager.Stats()
	s["pipeline"] = "skills → skeleton → agent"
	return s
}

// EvictStale 卸载闲置技能
func (p *Pipeline) EvictStale() int {
	return p.manager.EvictStale()
}

// Refresh 重新发现技能
func (p *Pipeline) Refresh() error {
	return p.manager.Reload()
}

// ========== 技能工具函数（供Agent调用） ==========

// SkillViewTool 技能查看工具
// Agent通过此工具加载技能内容
func (p *Pipeline) SkillViewTool(name string) (string, error) {
	return p.LoadAndCompress(name)
}

// SkillViewFullTool 技能完整查看工具
func (p *Pipeline) SkillViewFullTool(name string) (string, error) {
	return p.LoadFull(name)
}

// SkillSearchTool 技能搜索工具
func (p *Pipeline) SkillSearchTool(query string) string {
	return p.Search(query)
}

// BackgroundReview 后台审查（简单版）
// 定期检查技能使用情况，清理不用的技能
func (p *Pipeline) BackgroundReview() {
	interval := 30 * time.Minute
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		evicted := p.EvictStale()
		if evicted > 0 {
			fmt.Printf("[skill-pipeline] evicted %d stale skills\n", evicted)
		}
	}
}
