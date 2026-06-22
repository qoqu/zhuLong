// 主动层调度器 - 事件总线 + 调度tick + Worker spawn（参考Hermes）
package scheduler

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"sync"
	"time"
)

// Event 调度事件
type Event struct {
	Type      string // task_ready / task_done / task_blocked / task_crashed
	TaskID    string
	Payload   string
	Timestamp time.Time
}

// EventHandler 事件处理器
type EventHandler func(event Event)

// Scheduler 调度器
type Scheduler struct {
	mu          sync.RWMutex
	handlers    map[string][]EventHandler
	tickPeriod  time.Duration
	spawnFn     SpawnFunc
	done        chan struct{}
	eventBuffer []Event
}

// SpawnFunc Worker生成函数
type SpawnFunc func(taskID, workspace string) (int, error)

// Config 调度器配置
type Config struct {
	TickPeriod time.Duration // 调度tick间隔（默认60s）
	MaxWorkers int           // 最大并发Worker（默认5）
}

// DefaultConfig 默认配置
func DefaultConfig() *Config {
	return &Config{
		TickPeriod: 60 * time.Second,
		MaxWorkers: 5,
	}
}

// New 创建调度器
func New(cfg *Config, spawn SpawnFunc) *Scheduler {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	if spawn == nil {
		spawn = defaultSpawn
	}

	return &Scheduler{
		handlers:   make(map[string][]EventHandler),
		tickPeriod: cfg.TickPeriod,
		spawnFn:    spawn,
		done:       make(chan struct{}),
	}
}

// On 注册事件处理器
func (s *Scheduler) On(eventType string, handler EventHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[eventType] = append(s.handlers[eventType], handler)
}

// Emit 发送事件
func (s *Scheduler) Emit(event Event) {
	s.mu.RLock()
	handlers := s.handlers[event.Type]
	s.mu.RUnlock()

	event.Timestamp = time.Now()

	for _, handler := range handlers {
		handler(event)
	}
}

// Start 启动调度器（阻塞）
func (s *Scheduler) Start(ctx context.Context) {
	log.Printf("scheduler started, tick=%s", s.tickPeriod)
	ticker := time.NewTicker(s.tickPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.done:
			return
		case <-ticker.C:
			s.tick()
		}
	}
}

// Stop 停止调度器
func (s *Scheduler) Stop() {
	close(s.done)
}

// ========== 调度逻辑 ==========

func (s *Scheduler) tick() {
	s.Emit(Event{Type: "tick"})
}

// GetEventBuffer 获取事件缓冲区
func (s *Scheduler) GetEventBuffer() []Event {
	s.mu.RLock()
	defer s.mu.RUnlock()
	buf := make([]Event, len(s.eventBuffer))
	copy(buf, s.eventBuffer)
	return buf
}

// ========== Worker Spawn ==========

// defaultSpawn 默认Worker生成器
// 参考Hermes: spawn "hermes -p <profile> chat -q work kanban task <id>"
func defaultSpawn(taskID, workspace string) (int, error) {
	cmd := exec.Command("zhulong",
		"--workspace", workspace,
		"kanban", "work", taskID,
	)

	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("spawn worker for task %s: %w", taskID, err)
	}

	return cmd.Process.Pid, nil
}

// SpawnResult spawn结果
type SpawnResult struct {
	TaskID string
	PID    int
	Error  error
}

// SpawnWorker 生成Worker并注入任务上下文
// 参考Hermes环境变量注入：
// HERMES_KANBAN_TASK, HERMES_KANBAN_RUN_ID, HERMES_KANBAN_CLAIM_LOCK
func SpawnWorker(taskID, profile, workspace string) *SpawnResult {
	cmd := exec.Command("zhulong",
		"-p", profile,
		"kanban", "work", taskID,
	)
	cmd.Env = append(cmd.Env,
		fmt.Sprintf("ZHULONG_KANBAN_TASK=%s", taskID),
		fmt.Sprintf("ZHULONG_KANBAN_WORKSPACE=%s", workspace),
		fmt.Sprintf("ZHULONG_KANBAN_CLAIM_LOCK=%s-%d", taskID, time.Now().Unix()),
	)

	pid := 0
	var spawnErr error
	if err := cmd.Start(); err != nil {
		spawnErr = fmt.Errorf("spawn failed: %w", err)
	} else {
		pid = cmd.Process.Pid
	}

	return &SpawnResult{
		TaskID: taskID,
		PID:    pid,
		Error:  spawnErr,
	}
}
