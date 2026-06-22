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
// 关键修复: 之前完全没有把 ready 任务 spawn 出 worker，spawnFn 形同虚设
// 现在新增可选 board 引用 + workerTag，使 tick() 能主动 spawn
type Scheduler struct {
	mu          sync.RWMutex
	handlers    map[string][]EventHandler
	tickPeriod  time.Duration
	spawnFn     SpawnFunc
	done        chan struct{}
	eventBuffer []Event

	// 可选注入：board 引擎 + worker 标识
	// 注入后，tick() 会扫描 board 的 ready 任务并自动 spawn
	board      BoardLike
	workerTag  string
}

// BoardLike 调度器需要的看板接口（解耦 internal/board 模块）
type BoardLike interface {
	ListReady() []*BoardTaskView
	ClaimTask(id, claimer string) error
	BlockTask(id, reason string) error
	GetTask(id string) (*BoardTaskView, error)
}

// BoardTaskView 调度器需要的任务视图
type BoardTaskView struct {
	ID        string
	Body      string
	Title     string
	WorkerPID int // 关键修复: 之前 BoardTaskView 缺这个字段，scheduler 写入会编译失败
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
		workerTag:  "scheduler-default",
	}
}

// SetBoard 注入看板引擎（解耦形式：接受 BoardLike 接口）
// 关键修复: 之前 tick() 没有真正消费 board，注入后调度器才能把 ready 任务 spawn 出去
func (s *Scheduler) SetBoard(b BoardLike, workerTag string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.board = b
	if workerTag != "" {
		s.workerTag = workerTag
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

// tick 调度 tick：每 tickPeriod 秒执行一次
// 关键修复: 之前 tick() 只发了个无消费者的事件，调度器没有把 ready 任务 spawn 出 worker
// 现在 tick() 内置两个内置 handler：
//   1. tick 事件 -> 用户可注册的 handler（外部订阅）
//   2. 内置 ready 任务 spawn（如果用户注入了 board 引擎 + spawnFn）
// 同时把每个事件写入 eventBuffer 供 GetEventBuffer 查询（之前是死代码）
func (s *Scheduler) tick() {
	now := time.Now()

	// 1) 写入事件缓冲（修复: 之前从不 append）
	s.mu.Lock()
	s.eventBuffer = append(s.eventBuffer, Event{
		Type:      "tick",
		Timestamp: now,
	})
	// 截断缓冲，避免内存膨胀
	if len(s.eventBuffer) > 1024 {
		s.eventBuffer = s.eventBuffer[len(s.eventBuffer)-1024:]
	}
	s.mu.Unlock()

	// 2) 触发用户订阅的 tick 处理器
	s.Emit(Event{Type: "tick", Timestamp: now})

	// 3) 如果注入了 board 引擎，则遍历 ready 任务 spawn worker
	if s.board != nil && s.spawnFn != nil {
		ready := s.board.ListReady()
		for _, task := range ready {
			// claim 任务（避免两个 worker 抢同一任务）
			if err := s.board.ClaimTask(task.ID, s.workerTag); err != nil {
				continue // 已被其他 worker 拿走
			}
			pid, err := s.spawnFn(task.ID, task.Body)
			if err != nil {
				// 失败: 释放 claim 并 Block
				_ = s.board.BlockTask(task.ID, err.Error())
				s.Emit(Event{Type: "task_crashed", TaskID: task.ID, Payload: err.Error(), Timestamp: now})
				continue
			}
			// 记录 worker PID 到 task
			if b := s.board; b != nil {
				if t, _ := b.GetTask(task.ID); t != nil {
					t.WorkerPID = pid
				}
			}
			s.Emit(Event{Type: "task_ready", TaskID: task.ID, Payload: fmt.Sprintf("pid=%d", pid), Timestamp: now})
		}
	}
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
