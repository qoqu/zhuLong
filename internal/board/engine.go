// 看板引擎 - 任务分解/委派/状态流转（参考Hermes Kanban）
package board

import (
	"fmt"
	"sync"
	"time"
)

// Status 看板节点状态
type Status string

const (
	StatusTriage   Status = "triage"   // 粗略想法/未规划
	StatusTodo     Status = "todo"     // 已规划，等待依赖就绪
	StatusReady    Status = "ready"    // 依赖已完成，可执行
	StatusRunning  Status = "running"  // 正在执行（已claim）
	StatusBlocked  Status = "blocked"  // 阻塞，等待解阻
	StatusReview   Status = "review"   // 待审查
	StatusDone     Status = "done"     // 完成
	StatusArchived Status = "archived" // 归档（终态）
)

// BoardTask 看板任务
type BoardTask struct {
	ID               string
	Title            string
	Body             string
	Status           Status
	Priority         int
	Assignee         string   // Worker标识
	DependsOn        []string // 依赖的任务ID
	Children         []string // 子任务ID
	Attachments      []string // 附件文件路径
	ClaimLock        string   // 当前持有者
	ClaimExpiresAt   int64    // claim过期时间
	ConsecutiveFails int      // 连续失败计数
	MaxRetries       int      // 最大重试（断路器阈值）
	WorkerPID        int      // Worker进程ID
	Result           string   // 执行结果摘要
	CreatedBy        string
	CreatedAt        int64
	StartedAt        int64
	CompletedAt      int64
}

// TaskEvent 任务事件
type TaskEvent struct {
	TaskID    string
	Kind      string // created / status_changed / claimed / completed / blocked / crashed
	From      Status
	To        Status
	Payload   string
	Timestamp int64
}

// Engine 看板引擎
type Engine struct {
	mu     sync.RWMutex
	tasks  map[string]*BoardTask
	events []TaskEvent
}

// New 创建看板引擎
func New() *Engine {
	return &Engine{
		tasks:  make(map[string]*BoardTask),
		events: make([]TaskEvent, 0),
	}
}

// CreateTask 创建任务
func (e *Engine) CreateTask(title, body, createdBy string, dependsOn []string) *BoardTask {
	e.mu.Lock()
	defer e.mu.Unlock()

	now := time.Now().Unix()
	id := fmt.Sprintf("task-%d-%d", now, len(e.tasks))

	// 计算初始状态：有未完成的parent→todo，否则ready
	status := StatusReady
	for _, dep := range dependsOn {
		if parent, ok := e.tasks[dep]; ok {
			if parent.Status != StatusDone && parent.Status != StatusArchived {
				status = StatusTodo
				break
			}
		}
	}

	task := &BoardTask{
		ID:        id,
		Title:     title,
		Body:      body,
		Status:    status,
		DependsOn: dependsOn,
		CreatedBy: createdBy,
		CreatedAt: now,
		MaxRetries: 3,
	}

	e.tasks[id] = task
	e.emitEvent(id, "created", "", status, "")

	// 如果状态是ready，检查是否有parent刚完成需要提升
	if status == StatusReady {
		e.tryPromoteChildrenUnsafe(id)
	}

	return task
}

// GetTask 获取任务
func (e *Engine) GetTask(id string) (*BoardTask, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	task, ok := e.tasks[id]
	if !ok {
		return nil, fmt.Errorf("task %s not found", id)
	}
	return task, nil
}

// ClaimTask claim任务（ready→running）
func (e *Engine) ClaimTask(id, claimer string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	task, ok := e.tasks[id]
	if !ok {
		return fmt.Errorf("task %s not found", id)
	}
	if task.Status != StatusReady && task.Status != StatusBlocked {
		return fmt.Errorf("task %s cannot be claimed from status %s", id, task.Status)
	}
	if task.ClaimLock != "" && task.ClaimExpiresAt > time.Now().Unix() {
		return fmt.Errorf("task %s already claimed by %s", id, task.ClaimLock)
	}

	oldStatus := task.Status
	task.Status = StatusRunning
	task.ClaimLock = claimer
	task.ClaimExpiresAt = time.Now().Add(5 * time.Minute).Unix()
	task.StartedAt = time.Now().Unix()

	e.emitEvent(id, "claimed", oldStatus, StatusRunning, claimer)
	return nil
}

// CompleteTask 完成任务（running/ready→done）
func (e *Engine) CompleteTask(id, result string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	task, ok := e.tasks[id]
	if !ok {
		return fmt.Errorf("task %s not found", id)
	}
	if task.Status != StatusRunning && task.Status != StatusReady {
		return fmt.Errorf("task %s cannot be completed from status %s", id, task.Status)
	}

	oldStatus := task.Status
	task.Status = StatusDone
	task.Result = result
	task.CompletedAt = time.Now().Unix()
	task.ConsecutiveFails = 0
	task.ClaimLock = ""

	e.emitEvent(id, "completed", oldStatus, StatusDone, result)

	// 提升依赖此任务的子任务
	e.tryPromoteChildrenUnsafe(id)

	return nil
}

// BlockTask 阻塞任务（running→blocked）
func (e *Engine) BlockTask(id, reason string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	task, ok := e.tasks[id]
	if !ok {
		return fmt.Errorf("task %s not found", id)
	}
	if task.Status != StatusRunning && task.Status != StatusReady {
		return fmt.Errorf("task %s cannot be blocked from status %s", id, task.Status)
	}

	oldStatus := task.Status
	task.Status = StatusBlocked
	task.ConsecutiveFails++
	task.ClaimLock = ""

	e.emitEvent(id, "blocked", oldStatus, StatusBlocked, reason)

	// 断路器：连续失败超限
	if task.ConsecutiveFails >= task.MaxRetries {
		e.emitEvent(id, "gave_up", StatusBlocked, StatusBlocked,
			fmt.Sprintf("exceeded max retries (%d)", task.MaxRetries))
	}

	return nil
}

// UnblockTask 解阻任务（blocked→ready）
func (e *Engine) UnblockTask(id string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	task, ok := e.tasks[id]
	if !ok {
		return fmt.Errorf("task %s not found", id)
	}
	if task.Status != StatusBlocked {
		return fmt.Errorf("task %s is not blocked", id)
	}

	oldStatus := task.Status
	task.Status = StatusReady

	e.emitEvent(id, "unblocked", oldStatus, StatusReady, "")
	return nil
}

// LinkTasks 添加依赖关系
func (e *Engine) LinkTasks(parentID, childID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	parent, ok := e.tasks[parentID]
	if !ok {
		return fmt.Errorf("parent task %s not found", parentID)
	}
	child, ok := e.tasks[childID]
	if !ok {
		return fmt.Errorf("child task %s not found", childID)
	}

	// 环检测
	if e.hasCycleUnsafe(childID, parentID) {
		return fmt.Errorf("linking %s->%s would create a cycle", parentID, childID)
	}

	parent.Children = append(parent.Children, childID)
	child.DependsOn = append(child.DependsOn, parentID)

	// 如果parent已done，尝试提升child
	if parent.Status == StatusDone || parent.Status == StatusArchived {
		if child.Status == StatusTodo {
			child.Status = StatusReady
		}
	}

	return nil
}

// ListReady 列出所有ready状态的任务
func (e *Engine) ListReady() []*BoardTask {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var ready []*BoardTask
	for _, task := range e.tasks {
		if task.Status == StatusReady && task.ClaimLock == "" {
			ready = append(ready, task)
		}
	}
	return ready
}

// ListByStatus 按状态列出任务
func (e *Engine) ListByStatus(status Status) []*BoardTask {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var result []*BoardTask
	for _, task := range e.tasks {
		if task.Status == status {
			result = append(result, task)
		}
	}
	return result
}

// GetEvents 获取事件列表
func (e *Engine) GetEvents(n int) []TaskEvent {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if n <= 0 || n > len(e.events) {
		n = len(e.events)
	}
	events := make([]TaskEvent, n)
	copy(events, e.events[len(e.events)-n:])
	return events
}

// ListAll 列出所有任务
func (e *Engine) ListAll() []*BoardTask {
	e.mu.RLock()
	defer e.mu.RUnlock()

	tasks := make([]*BoardTask, 0, len(e.tasks))
	for _, task := range e.tasks {
		tasks = append(tasks, task)
	}
	return tasks
}

// ========== 内部方法 ==========

func (e *Engine) emitEvent(taskID, kind string, from, to Status, payload string) {
	e.events = append(e.events, TaskEvent{
		TaskID:    taskID,
		Kind:      kind,
		From:      from,
		To:        to,
		Payload:   payload,
		Timestamp: time.Now().Unix(),
	})
}

func (e *Engine) hasCycleUnsafe(from, to string) bool {
	visited := make(map[string]bool)
	return e.dfsCycleUnsafe(from, to, visited)
}

func (e *Engine) dfsCycleUnsafe(current, target string, visited map[string]bool) bool {
	if current == target {
		return true
	}
	if visited[current] {
		return false
	}
	visited[current] = true

	task := e.tasks[current]
	for _, child := range task.Children {
		if e.dfsCycleUnsafe(child, target, visited) {
			return true
		}
	}
	return false
}

func (e *Engine) tryPromoteChildrenUnsafe(taskID string) {
	task := e.tasks[taskID]
	for _, childID := range task.Children {
		child := e.tasks[childID]
		if child.Status == StatusTodo {
			// 检查child的所有parent是否都done了
			allDone := true
			for _, depID := range child.DependsOn {
				dep := e.tasks[depID]
				if dep.Status != StatusDone && dep.Status != StatusArchived {
					allDone = false
					break
				}
			}
			if allDone {
				oldStatus := child.Status
				child.Status = StatusReady
				e.emitEvent(childID, "promoted", oldStatus, StatusReady, "")
			}
		}
	}
}
