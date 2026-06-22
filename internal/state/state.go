// 状态持久化系统
// 热状态+冷存储分离
package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// HotState 热状态 - 会话自动加载
type HotState struct {
	Phase          string            `json:"phase"`          // design/fix
	Mode           string            `json:"mode"`           // full/hotfix/tweak
	LastScanAt     string            `json:"last_scan_at"`   // 最后扫描时间
	LastScanScore  float64           `json:"last_scan_score"` // 最后扫描得分
	ActiveGoals    []string          `json:"active_goals"`   // 活跃目标列表
	CircuitState   string            `json:"circuit_state"`  // breaker状态
	SessionChanges int               `json:"session_changes"`// 本次会话变更数
	UpdatedAt      string            `json:"updated_at"`
}

// ColdLog 冷存储记录
type ColdLog struct {
	Type      string `json:"type"`      // gc_scan / review / upgrade
	Summary   string `json:"summary"`
	Score     float64 `json:"score"`
	Timestamp string `json:"timestamp"`
	Details   string `json:"details,omitempty"`
}

// Store 状态管理器
type Store struct {
	mu       sync.RWMutex
	dataDir  string
	hotState *HotState
	coldLogs []ColdLog
}

// New 创建状态管理器
func New(dataDir string) *Store {
	s := &Store{
		dataDir:  dataDir,
		hotState: &HotState{Phase: "design", Mode: "full", UpdatedAt: time.Now().Format(time.RFC3339)},
		coldLogs: make([]ColdLog, 0),
	}
	s.load()
	return s
}

// GetHot 获取热状态
func (s *Store) GetHot() *HotState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cp := *s.hotState
	return &cp
}

// UpdateHot 更新热状态
func (s *Store) UpdateHot(update func(h *HotState)) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	update(s.hotState)
	s.hotState.UpdatedAt = time.Now().Format(time.RFC3339)
	return s.save()
}

// AppendCold 追加冷存储记录
func (s *Store) AppendCold(log ColdLog) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.coldLogs = append(s.coldLogs, log)
	// 只保留最近100条
	if len(s.coldLogs) > 100 {
		s.coldLogs = s.coldLogs[len(s.coldLogs)-100:]
	}
	return s.saveCold()
}

// GetCold 获取冷存储记录
func (s *Store) GetCold(n int) []ColdLog {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if n <= 0 || n > len(s.coldLogs) {
		n = len(s.coldLogs)
	}
	logs := make([]ColdLog, n)
	copy(logs, s.coldLogs[len(s.coldLogs)-n:])
	return logs
}

// ========== 持久化 ==========

func (s *Store) hotPath() string  { return filepath.Join(s.dataDir, ".zhulong-state") }
func (s *Store) coldPath() string { return filepath.Join(s.dataDir, "zhulong-log.json") }

func (s *Store) load() {
	os.MkdirAll(s.dataDir, 0755)

	// 加载热状态
	if data, err := os.ReadFile(s.hotPath()); err == nil {
		json.Unmarshal(data, s.hotState)
	}

	// 加载冷存储
	if data, err := os.ReadFile(s.coldPath()); err == nil {
		json.Unmarshal(data, &s.coldLogs)
	}
}

func (s *Store) save() error {
	data, err := json.MarshalIndent(s.hotState, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal hot state: %w", err)
	}
	return os.WriteFile(s.hotPath(), data, 0644)
}

func (s *Store) saveCold() error {
	data, err := json.MarshalIndent(s.coldLogs, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal cold log: %w", err)
	}
	return os.WriteFile(s.coldPath(), data, 0644)
}
