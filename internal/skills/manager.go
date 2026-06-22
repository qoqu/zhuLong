// 增强技能管理器 - 渐进式披露 + 目录加载 + 热重载 + 骨架压缩
package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Tier 暴露层级
type Tier int

const (
	Tier1Name     Tier = 1 // 仅名称和描述
	Tier2Content  Tier = 2 // 完整SKILL.md内容
	Tier3Full     Tier = 3 // 完整内容 + 支持文件
)

// Skill 技能
type Skill struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Version     string            `json:"version,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	Path        string            `json:"-"`
	Dir         string            `json:"-"`
	Content     string            `json:"-"`      // SKILL.md 完整内容
	Loaded      bool              `json:"-"`      // 内容是否已加载
	LoadedTier  Tier              `json:"-"`      // 当前加载到的层级
	SupportFiles map[string]string `json:"-"`      // references/templates/scripts/assets
	LastUsed    time.Time         `json:"-"`
	UseCount    int               `json:"-"`
	Pinned      bool              `json:"-"`      // 钉住防止被归档
	CreatedAt   time.Time         `json:"-"`
}

// SkillInfo 元数据（Tier 1）
type SkillInfo struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Version     string   `json:"version,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

// SkillManager 技能管理器
type SkillManager struct {
	mu          sync.RWMutex
	skills      map[string]*Skill
	searchPaths []string
	maxSkills   int           // 最大技能数（0=无限制）
	evictAfter  time.Duration // 闲置多久后卸载内容
}

// NewSkillManager 创建技能管理器
func NewSkillManager(searchPaths []string) *SkillManager {
	return &SkillManager{
		skills:      make(map[string]*Skill),
		searchPaths: searchPaths,
		maxSkills:   100,
		evictAfter:  30 * time.Minute,
	}
}

// ========== 生命周��� ==========

// Discover 发现所有技能（仅加载元数据 - Tier 1）
func (sm *SkillManager) Discover() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for _, searchPath := range sm.searchPaths {
		entries, err := os.ReadDir(searchPath)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			skillPath := filepath.Join(searchPath, entry.Name(), "SKILL.md")
			info, err := os.Stat(skillPath)
			if err != nil {
				continue
			}

			// 只加载元数据（渐进式披露阶段1）
			skill, err := sm.loadMetadata(skillPath)
			if err != nil {
				continue
			}

			skill.Path = skillPath
			skill.Dir = filepath.Dir(skillPath)
			skill.CreatedAt = info.ModTime()
			skill.LoadedTier = Tier1Name
			sm.skills[skill.Name] = skill
		}
	}

	return nil
}

// GetSkillList 获取技能列表（Tier 1 - 用于系统提示词）
func (sm *SkillManager) GetSkillList() []SkillInfo {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	list := make([]SkillInfo, 0, len(sm.skills))
	for _, skill := range sm.skills {
		list = append(list, SkillInfo{
			Name:        skill.Name,
			Description: skill.Description,
			Version:     skill.Version,
			Tags:        skill.Tags,
		})
	}

	// 按使用次数排序，常用优先
	sort.Slice(list, func(i, j int) bool {
		return sm.skills[list[i].Name].UseCount > sm.skills[list[j].Name].UseCount
	})

	return list
}

// ViewSkill 查看技能完整内容（Tier 1→2 - 激活阶段）
func (sm *SkillManager) ViewSkill(name string) (*Skill, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	skill, ok := sm.skills[name]
	if !ok {
		return nil, fmt.Errorf("skill not found: %s", name)
	}

	// 加载完整内容
	if !skill.Loaded || skill.LoadedTier < Tier2Content {
		content, err := os.ReadFile(skill.Path)
		if err != nil {
			return nil, err
		}
		skill.Content = string(content)
		skill.Loaded = true
		skill.LoadedTier = Tier2Content
	}

	skill.LastUsed = time.Now()
	skill.UseCount++

	return skill, nil
}

// ViewSkillFull 加载完整技能包（Tier 1→3 - 含支持文件）
func (sm *SkillManager) ViewSkillFull(name string) (*Skill, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	skill, ok := sm.skills[name]
	if !ok {
		return nil, fmt.Errorf("skill not found: %s", name)
	}

	// 加载完整内容
	if !skill.Loaded || skill.LoadedTier < Tier2Content {
		content, err := os.ReadFile(skill.Path)
		if err != nil {
			return nil, err
		}
		skill.Content = string(content)
		skill.Loaded = true
	}

	// 加载支持文件（仅当需要Tier 3时）
	if skill.LoadedTier < Tier3Full {
		skill.SupportFiles = make(map[string]string)
		subDirs := []string{"references", "templates", "scripts", "assets"}
		for _, sub := range subDirs {
			subPath := filepath.Join(skill.Dir, sub)
			entries, err := os.ReadDir(subPath)
			if err != nil {
				continue
			}
			for _, e := range entries {
				if !e.IsDir() {
					data, err := os.ReadFile(filepath.Join(subPath, e.Name()))
					if err == nil {
						skill.SupportFiles[sub+"/"+e.Name()] = string(data)
					}
				}
			}
		}
		skill.LoadedTier = Tier3Full
	}

	skill.LastUsed = time.Now()
	skill.UseCount++

	return skill, nil
}

// Evict 卸载技能内容（释放内存），保留元数据
func (sm *SkillManager) Evict(name string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	skill, ok := sm.skills[name]
	if !ok {
		return fmt.Errorf("skill not found: %s", name)
	}
	if skill.Pinned {
		return fmt.Errorf("skill %s is pinned, cannot evict", name)
	}

	skill.Content = ""
	skill.SupportFiles = nil
	skill.Loaded = false
	skill.LoadedTier = Tier1Name
	return nil
}

// EvictStale 卸载所有闲置超时的技能
func (sm *SkillManager) EvictStale() int {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	count := 0
	threshold := time.Now().Add(-sm.evictAfter)

	for _, skill := range sm.skills {
		if skill.Pinned {
			continue
		}
		if skill.LastUsed.Before(threshold) && skill.Loaded {
			skill.Content = ""
			skill.SupportFiles = nil
			skill.Loaded = false
			skill.LoadedTier = Tier1Name
			count++
		}
	}

	return count
}

// Pin 钉住技能（防止被Evict/归档）
func (sm *SkillManager) Pin(name string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	skill, ok := sm.skills[name]
	if !ok {
		return fmt.Errorf("skill not found: %s", name)
	}
	skill.Pinned = true
	return nil
}

// Unpin 取消钉住
func (sm *SkillManager) Unpin(name string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	skill, ok := sm.skills[name]
	if !ok {
		return fmt.Errorf("skill not found: %s", name)
	}
	skill.Pinned = false
	return nil
}

// GetTier2Content 获取Tier 2内容摘要（压缩后）
// 与骨架压缩集成：只保留核心结构
func (sm *SkillManager) GetTier2Content(name string) (string, error) {
	skill, err := sm.ViewSkill(name)
	if err != nil {
		return "", err
	}

	// 提取YAML前面的markdown正文作为精要
	parts := strings.SplitN(skill.Content, "---", 3)
	if len(parts) >= 3 {
		return strings.TrimSpace(parts[2]), nil
	}
	return skill.Content, nil
}

// ListByTag 按标签列出技能
func (sm *SkillManager) ListByTag(tag string) []SkillInfo {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	var result []SkillInfo
	for _, skill := range sm.skills {
		for _, t := range skill.Tags {
			if t == tag {
				result = append(result, SkillInfo{
					Name: skill.Name, Description: skill.Description,
					Version: skill.Version, Tags: skill.Tags,
				})
				break
			}
		}
	}
	return result
}

// Stats 返回统计信息
func (sm *SkillManager) Stats() map[string]interface{} {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	loaded := 0
	pinned := 0
	for _, s := range sm.skills {
		if s.Loaded { loaded++ }
		if s.Pinned { pinned++ }
	}

	return map[string]interface{}{
		"total":     len(sm.skills),
		"loaded":    loaded,
		"pinned":    pinned,
		"paths":     len(sm.searchPaths),
	}
}

// ========== 原有方法保留 ==========

func (sm *SkillManager) SearchSkills(query string) []*Skill {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	results := make([]*Skill, 0)
	query = strings.ToLower(query)

	for _, skill := range sm.skills {
		if strings.Contains(strings.ToLower(skill.Name), query) ||
			strings.Contains(strings.ToLower(skill.Description), query) {
			results = append(results, skill)
		}
	}
	return results
}

func (sm *SkillManager) GetSkill(name string) (*Skill, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	skill, ok := sm.skills[name]
	return skill, ok
}

func (sm *SkillManager) GetAllSkills() []*Skill {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	skills := make([]*Skill, 0, len(sm.skills))
	for _, skill := range sm.skills {
		skills = append(skills, skill)
	}
	return skills
}

func (sm *SkillManager) Reload() error {
	sm.mu.Lock()
	sm.skills = make(map[string]*Skill)
	sm.mu.Unlock()
	return sm.Discover()
}

// ========== 内部方法 ==========

func (sm *SkillManager) loadMetadata(path string) (*Skill, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	skill := &Skill{}
	lines := strings.Split(string(content), "\n")
	inFrontMatter := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "---" {
			inFrontMatter = !inFrontMatter
			continue
		}
		if inFrontMatter {
			switch {
			case strings.HasPrefix(line, "name:"):
				skill.Name = strings.Trim(strings.TrimSpace(strings.TrimPrefix(line, "name:")), "\"'")
			case strings.HasPrefix(line, "description:"):
				skill.Description = strings.Trim(strings.TrimSpace(strings.TrimPrefix(line, "description:")), "\"'")
			case strings.HasPrefix(line, "version:"):
				skill.Version = strings.Trim(strings.TrimSpace(strings.TrimPrefix(line, "version:")), "\"'")
			case strings.HasPrefix(line, "tags:"):
				// 解析 tags: [tag1, tag2]
				tagStr := strings.TrimSpace(strings.TrimPrefix(line, "tags:"))
				tagStr = strings.Trim(tagStr, "[]")
				for _, t := range strings.Split(tagStr, ",") {
					t = strings.TrimSpace(t)
					t = strings.Trim(t, "\"'")
					if t != "" {
						skill.Tags = append(skill.Tags, t)
					}
				}
			}
		}
	}

	if skill.Name == "" {
		skill.Name = filepath.Base(filepath.Dir(path))
	}
	if skill.Description == "" {
		skill.Description = "Skill: " + skill.Name
	}

	return skill, nil
}
