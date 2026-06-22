// 插件系统 - 动态加载/卸载
package plugins

import (
	"fmt"
	"sync"
)

// Plugin 插件接口
type Plugin interface {
	Name() string
	Version() string
	Init() error
	Shutdown() error
}

// Type 插件类型
type Type string

const (
	TypeTool   Type = "tool"
	TypeHook   Type = "hook"
	TypeMemory Type = "memory"
	TypeCron   Type = "cron"
	TypeAuth   Type = "auth"
)

// Info 插件信息
type Info struct {
	Name    string
	Version string
	Type    Type
	Status  string // active / inactive / error
}

// Manager 插件管理器
type Manager struct {
	mu      sync.RWMutex
	plugins map[string]Plugin
	infos   map[string]Info
}

func NewManager() *Manager {
	return &Manager{
		plugins: make(map[string]Plugin),
		infos:   make(map[string]Info),
	}
}

func (m *Manager) Register(p Plugin, ptype Type) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.plugins[p.Name()]; ok {
		return fmt.Errorf("plugin already registered: %s", p.Name())
	}

	if err := p.Init(); err != nil {
		return fmt.Errorf("init %s: %w", p.Name(), err)
	}

	m.plugins[p.Name()] = p
	m.infos[p.Name()] = Info{
		Name: p.Name(), Version: p.Version(),
		Type: ptype, Status: "active",
	}
	return nil
}

func (m *Manager) Unregister(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	p, ok := m.plugins[name]
	if !ok {
		return fmt.Errorf("plugin not found: %s", name)
	}

	if err := p.Shutdown(); err != nil {
		return err
	}

	delete(m.plugins, name)
	delete(m.infos, name)
	return nil
}

func (m *Manager) Get(name string) (Plugin, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.plugins[name]
	return p, ok
}

func (m *Manager) List() []Info {
	m.mu.RLock()
	defer m.mu.RUnlock()
	list := make([]Info, 0, len(m.infos))
	for _, info := range m.infos {
		list = append(list, info)
	}
	return list
}

func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.plugins)
}
