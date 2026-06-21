package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Skill represents a skill with metadata
type Skill struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Path        string `json:"-"`
	Content     string `json:"-"` // Loaded on demand
	Loaded      bool   `json:"-"`
}

// SkillInfo contains skill metadata (for system prompt)
type SkillInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// SkillManager manages skills with progressive disclosure
type SkillManager struct {
	skills      map[string]*Skill
	searchPaths []string
}

// NewSkillManager creates a new skill manager
func NewSkillManager(searchPaths []string) *SkillManager {
	return &SkillManager{
		skills:      make(map[string]*Skill),
		searchPaths: searchPaths,
	}
}

// Discover discovers all skills (only loads metadata)
func (sm *SkillManager) Discover() error {
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
			if _, err := os.Stat(skillPath); err != nil {
				continue
			}

			// Only load metadata (name + description)
			skill, err := sm.loadMetadata(skillPath)
			if err != nil {
				continue
			}

			skill.Path = skillPath
			sm.skills[skill.Name] = skill
		}
	}

	return nil
}

// loadMetadata loads only the metadata from SKILL.md
func (sm *SkillManager) loadMetadata(path string) (*Skill, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Parse YAML front matter
	skill := &Skill{}
	lines := strings.Split(string(content), "\n")

	// Simple parsing: look for name and description in front matter
	inFrontMatter := false
	for _, line := range lines {
		line = strings.TrimSpace(line)

		if line == "---" {
			inFrontMatter = !inFrontMatter
			continue
		}

		if inFrontMatter {
			if strings.HasPrefix(line, "name:") {
				skill.Name = strings.TrimSpace(strings.TrimPrefix(line, "name:"))
				// Remove quotes if present
				skill.Name = strings.Trim(skill.Name, "\"'")
			} else if strings.HasPrefix(line, "description:") {
				skill.Description = strings.TrimSpace(strings.TrimPrefix(line, "description:"))
				// Remove quotes if present
				skill.Description = strings.Trim(skill.Description, "\"'")
				// Handle multi-line description (YAML >- syntax)
				if skill.Description == ">-" || skill.Description == ">" || skill.Description == "|" {
					skill.Description = ""
				}
			}
		}
	}

	// Fallback to directory name if name not found
	if skill.Name == "" {
		skill.Name = filepath.Base(filepath.Dir(path))
	}

	// Fallback description
	if skill.Description == "" {
		skill.Description = "Skill: " + skill.Name
	}

	return skill, nil
}

// GetSkillList returns the list of discovered skills (for system prompt)
func (sm *SkillManager) GetSkillList() []SkillInfo {
	list := make([]SkillInfo, 0, len(sm.skills))
	for _, skill := range sm.skills {
		list = append(list, SkillInfo{
			Name:        skill.Name,
			Description: skill.Description,
		})
	}
	return list
}

// ViewSkill loads the full content of a skill (Activation phase)
func (sm *SkillManager) ViewSkill(name string) (*Skill, error) {
	skill, ok := sm.skills[name]
	if !ok {
		return nil, fmt.Errorf("skill not found: %s", name)
	}

	// Load full content if not already loaded
	if !skill.Loaded {
		content, err := os.ReadFile(skill.Path)
		if err != nil {
			return nil, err
		}
		skill.Content = string(content)
		skill.Loaded = true
	}

	return skill, nil
}

// SearchSkills searches skills by query
func (sm *SkillManager) SearchSkills(query string) []*Skill {
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

// GetSkill returns a skill by name (without loading content)
func (sm *SkillManager) GetSkill(name string) (*Skill, bool) {
	skill, ok := sm.skills[name]
	return skill, ok
}

// GetAllSkills returns all discovered skills
func (sm *SkillManager) GetAllSkills() []*Skill {
	skills := make([]*Skill, 0, len(sm.skills))
	for _, skill := range sm.skills {
		skills = append(skills, skill)
	}
	return skills
}

// Reload rediscovers all skills
func (sm *SkillManager) Reload() error {
	sm.skills = make(map[string]*Skill)
	return sm.Discover()
}
