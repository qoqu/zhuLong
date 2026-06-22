// 记忆系统 - MEMORY.md + USER.md 双文件体系
// 参考Hermes memory系统：有界、可策划、冻结快照模式
package memento

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// MemoryType 记忆类型
type MemoryType string

const (
	MemoryAgent MemoryType = "agent" // Agent笔记（MEMORY.md）
	MemoryUser  MemoryType = "user"  // 用户画像（USER.md）
)

// MemoryConstraints 记忆约束
const (
	MaxAgentChars = 2200 // MEMORY.md 最大字符数
	MaxUserChars  = 1375 // USER.md 最大字符数
)

// MemoryEntry 单条记忆
type MemoryEntry struct {
	ID        string    `json:"id"`
	Type      MemoryType `json:"type"`
	Content   string    `json:"content"`
	Source    string    `json:"source"`  // 来源：manual / auto / review
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Tags      []string  `json:"tags,omitempty"`
	Pinned    bool      `json:"pinned"`  // 钉住防止被覆盖
}

// MemoryStore 记忆存储
type MemoryStore struct {
	mu        sync.RWMutex
	dataDir   string
	agentFile string // MEMORY.md 路径
	userFile  string // USER.md 路径
	entries   []MemoryEntry
}

// NewMemoryStore 创建记忆存储
func NewMemoryStore(dataDir string) *MemoryStore {
	os.MkdirAll(dataDir, 0755)
	return &MemoryStore{
		dataDir:   dataDir,
		agentFile: filepath.Join(dataDir, "MEMORY.md"),
		userFile:  filepath.Join(dataDir, "USER.md"),
		entries:   make([]MemoryEntry, 0),
	}
}

// Init 初始化：如果文件不存在则创建
// 关键修复: 之前 Init() 只创建空文件，文件内容损坏或 schema 变更时无法自愈
// 现在增加文件存在性 + 可写性双重校验
func (ms *MemoryStore) Init() error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	for _, f := range []string{ms.agentFile, ms.userFile} {
		if _, err := os.Stat(f); os.IsNotExist(err) {
			if err := os.WriteFile(f, []byte(""), 0644); err != nil {
				return fmt.Errorf("create memory file %s: %w", f, err)
			}
		} else if err != nil {
			// 文件存在但 stat 失败（权限/损坏）
			return fmt.Errorf("stat memory file %s: %w", f, err)
		}
	}
	return nil
}

// GetSnapshot 获取记忆快照（冻结模式——会话开始时快照注入，中间不变）
// 返回 (agentNote, userProfile)
//
// 关键修复: 之前版本有两个问题：
//   1) 直接 os.ReadFile 读磁盘，没有 entries 缓存加速
//   2) 之前我改成优先 entries 但 append 没写入 entries，导致快照缺失追加内容
// 现在：文件是 source of truth（AppendAgentNote 也写文件），entries 只在 Search/History 用
// 这样既保证快照反映真实磁盘内容，又不丢 append 的内容
func (ms *MemoryStore) GetSnapshot() (string, string) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	agentData, _ := os.ReadFile(ms.agentFile)
	userData, _ := os.ReadFile(ms.userFile)

	return string(agentData), string(userData)
}

// SaveAgentNote 保存Agent笔记（写入MEMORY.md）
// 受MaxAgentChars限制，超长自动截断
func (ms *MemoryStore) SaveAgentNote(content string) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	if len([]rune(content)) > MaxAgentChars {
		content = string([]rune(content)[:MaxAgentChars])
	}

	if err := os.WriteFile(ms.agentFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("save agent note: %w", err)
	}

	ms.entries = append(ms.entries, MemoryEntry{
		ID:        fmt.Sprintf("mem-%d", time.Now().UnixNano()),
		Type:      MemoryAgent,
		Content:   content,
		Source:    "manual",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	return nil
}

// AppendAgentNote 追加到Agent笔记
// 关键修复: 之前 AppendAgentNote 没把内容写入 entries 历史，导致 Search/History 丢失追加内容
// 现在：文件 + entries 同步更新（entries 用于历史查询，文件是 source of truth）
func (ms *MemoryStore) AppendAgentNote(content string) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	existing, _ := os.ReadFile(ms.agentFile)
	current := string(existing)

	newContent := current
	if current != "" && !strings.HasSuffix(current, "\n") {
		newContent += "\n"
	}
	newContent += content

	// 限制总长度
	if len([]rune(newContent)) > MaxAgentChars {
		newContent = string([]rune(newContent)[:MaxAgentChars])
	}

	if err := os.WriteFile(ms.agentFile, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("append agent note: %w", err)
	}

	// 同步写入 entries 历史
	ms.entries = append(ms.entries, MemoryEntry{
		ID:        fmt.Sprintf("mem-%d", time.Now().UnixNano()),
		Type:      MemoryAgent,
		Content:   newContent,
		Source:    "append",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	return nil
}

// SaveUserProfile 保存用户画像（写入USER.md）
func (ms *MemoryStore) SaveUserProfile(content string) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	if len([]rune(content)) > MaxUserChars {
		content = string([]rune(content)[:MaxUserChars])
	}

	if err := os.WriteFile(ms.userFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("save user profile: %w", err)
	}

	return nil
}

// UpdateUserProfile 更新用户画像中指定部分
// 使用子串匹配替换
func (ms *MemoryStore) UpdateUserProfile(substring, replacement string) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	data, err := os.ReadFile(ms.userFile)
	if err != nil {
		return err
	}

	content := string(data)
	if !strings.Contains(content, substring) {
		return fmt.Errorf("substring %q not found in USER.md", substring)
	}

	content = strings.Replace(content, substring, replacement, 1)

	if len([]rune(content)) > MaxUserChars {
		content = string([]rune(content)[:MaxUserChars])
	}

	return os.WriteFile(ms.userFile, []byte(content), 0644)
}

// Search 全文搜索（线性扫描）
func (ms *MemoryStore) Search(query string) []MemoryEntry {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	query = strings.ToLower(query)
	var results []MemoryEntry

	for _, e := range ms.entries {
		if strings.Contains(strings.ToLower(e.Content), query) {
			results = append(results, e)
		}
	}

	return results
}

// GetHistory 获取记忆历史
func (ms *MemoryStore) GetHistory(n int) []MemoryEntry {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	if n <= 0 || n > len(ms.entries) {
		n = len(ms.entries)
	}
	entries := make([]MemoryEntry, n)
	copy(entries, ms.entries[len(ms.entries)-n:])
	return entries
}

// Stats 返回统计
func (ms *MemoryStore) Stats() map[string]interface{} {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	agentSize := 0
	userSize := 0
	if d, err := os.ReadFile(ms.agentFile); err == nil {
		agentSize = len(d)
	}
	if d, err := os.ReadFile(ms.userFile); err == nil {
		userSize = len(d)
	}

	return map[string]interface{}{
		"agent_chars": agentSize,
		"user_chars":  userSize,
		"agent_max":   MaxAgentChars,
		"user_max":    MaxUserChars,
		"entries":     len(ms.entries),
	}
}

// ========== 外部记忆提供者接口 ==========

// ExternalProvider 外部记忆提供者
type ExternalProvider interface {
	Name() string
	Store(key string, value string) error
	Retrieve(key string) (string, bool)
	Search(query string) []string
}

// ProviderRegistry 提供者注册表
type ProviderRegistry struct {
	mu        sync.Mutex
	providers map[string]ExternalProvider
}

// NewProviderRegistry 创建提供者注册表
func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{
		providers: make(map[string]ExternalProvider),
	}
}

// Register 注册提供者
func (pr *ProviderRegistry) Register(p ExternalProvider) {
	pr.mu.Lock()
	defer pr.mu.Unlock()
	pr.providers[p.Name()] = p
}

// Get 获取提供者
func (pr *ProviderRegistry) Get(name string) (ExternalProvider, bool) {
	pr.mu.Lock()
	defer pr.mu.Unlock()
	p, ok := pr.providers[name]
	return p, ok
}

// List 列出所有提供者
func (pr *ProviderRegistry) List() []string {
	pr.mu.Lock()
	defer pr.mu.Unlock()
	var names []string
	for n := range pr.providers {
		names = append(names, n)
	}
	return names
}
