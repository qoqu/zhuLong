// Profile隔离 - 独立环境并发运行
package profile

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Profile 配置隔离
type Profile struct {
	Name    string
	HomeDir string
	Config  map[string]string
}

// Manager Profile管理器
type Manager struct {
	mu       sync.Mutex
	baseDir  string
	profiles map[string]*Profile
	current  string
}

func NewManager(baseDir string) *Manager {
	return &Manager{
		baseDir:  baseDir,
		profiles: make(map[string]*Profile),
		current:  "default",
	}
}

func (m *Manager) Create(name string) (*Profile, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.profiles[name]; ok {
		return nil, fmt.Errorf("profile already exists: %s", name)
	}

	home := filepath.Join(m.baseDir, name)
	os.MkdirAll(home, 0755)

	p := &Profile{
		Name: name, HomeDir: home,
		Config: make(map[string]string),
	}
	m.profiles[name] = p
	return p, nil
}

func (m *Manager) Switch(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.profiles[name]; !ok {
		return fmt.Errorf("profile not found: %s", name)
	}
	m.current = name
	return nil
}

func (m *Manager) Current() *Profile {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.profiles[m.current]
}

func (m *Manager) List() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	var names []string
	for n := range m.profiles {
		names = append(names, n)
	}
	return names
}
