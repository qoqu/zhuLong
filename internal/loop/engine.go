// Loop自治循环系统
package loop

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// LoopState 循环状态
type LoopState string

const (
	StateRunning  LoopState = "running"
	StatePaused   LoopState = "paused"
	StateStopped  LoopState = "stopped"
	StateError    LoopState = "error"
)

// LoopTask 循环任务
type LoopTask struct {
	ID          string
	Name        string
	Interval    time.Duration
	Command     string
	State       LoopState
	LastRun     time.Time
	NextRun     time.Time
	RunCount    int
	FailCount   int
	MaxFailures int // 最大失败次数，超过则暂停
}

// LoopResult 循环执行结果
type LoopResult struct {
	TaskID    string
	Success   bool
	Output    string
	Duration  time.Duration
	Timestamp time.Time
}

// LoopEngine 循环引擎
type LoopEngine struct {
	mu     sync.RWMutex
	tasks  map[string]*LoopTask
	logs   []LoopResult
	stopCh chan struct{}
}

// NewLoopEngine 创建循环引擎
func NewLoopEngine() *LoopEngine {
	return &LoopEngine{
		tasks:  make(map[string]*LoopTask),
		logs:   make([]LoopResult, 0),
		stopCh: make(chan struct{}),
	}
}

// Register 注册循环任务
func (e *LoopEngine) Register(task *LoopTask) {
	e.mu.Lock()
	defer e.mu.Unlock()

	task.State = StateRunning
	task.NextRun = time.Now().Add(task.Interval)
	e.tasks[task.ID] = task
}

// Start 启动循环引擎
func (e *LoopEngine) Start(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-e.stopCh:
			return
		case <-ticker.C:
			e.checkAndRun(ctx)
		}
	}
}

// Stop 停止循环引擎
func (e *LoopEngine) Stop() {
	close(e.stopCh)
}

// GetState 获取任务状态
func (e *LoopEngine) GetState(taskID string) (*LoopTask, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	task, ok := e.tasks[taskID]
	if !ok {
		return nil, fmt.Errorf("task %s not found", taskID)
	}
	return task, nil
}

// RecordResult 记录执行结果
func (e *LoopEngine) RecordResult(result LoopResult) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.logs = append(e.logs, result)

	if task, ok := e.tasks[result.TaskID]; ok {
		task.LastRun = result.Timestamp
		task.RunCount++
		task.NextRun = time.Now().Add(task.Interval)

		if !result.Success {
			task.FailCount++
			// Circuit Breaker：连续失败超过阈值则暂停
			if task.FailCount >= task.MaxFailures && task.MaxFailures > 0 {
				task.State = StatePaused
			}
		} else {
			task.FailCount = 0
			task.State = StateRunning
		}
	}
}

// GetLogs 获取执行日志
func (e *LoopEngine) GetLogs() []LoopResult {
	e.mu.RLock()
	defer e.mu.RUnlock()

	logs := make([]LoopResult, len(e.logs))
	copy(logs, e.logs)
	return logs
}

// ========== 预置循环任务 ==========

// GCScanTask 创建GC扫描循环任务
// 参考Harness-Starter的 /loop 24h "node scripts/gc-scan.mjs"
func GCScanTask(interval time.Duration) *LoopTask {
	return &LoopTask{
		ID:          "gc-scan",
		Name:        "GC自治扫描",
		Interval:    interval,
		Command:     "gc-scan",
		MaxFailures: 3,
	}
}

// QualityCheckTask 创建质量检查循环任务
func QualityCheckTask(interval time.Duration) *LoopTask {
	return &LoopTask{
		ID:          "quality-check",
		Name:        "质量检查",
		Interval:    interval,
		Command:     "quality-check",
		MaxFailures: 3,
	}
}

func (e *LoopEngine) checkAndRun(ctx context.Context) {
	e.mu.RLock()
	tasks := make([]*LoopTask, 0, len(e.tasks))
	for _, task := range e.tasks {
		if task.State == StateRunning && time.Now().After(task.NextRun) {
			tasks = append(tasks, task)
		}
	}
	e.mu.RUnlock()

	for _, task := range tasks {
		go e.runTask(ctx, task)
	}
}

func (e *LoopEngine) runTask(ctx context.Context, task *LoopTask) {
	start := time.Now()
	result := LoopResult{
		TaskID:    task.ID,
		Timestamp: start,
	}

	// 模拟执行
	output := fmt.Sprintf("Task %s executed at %s", task.ID, start.Format(time.RFC3339))
	result.Success = true
	result.Output = output
	result.Duration = time.Since(start)

	e.RecordResult(result)
}
