package board

import (
	"testing"
)

func TestNewEngine(t *testing.T) {
	e := New()
	if e == nil {
		t.Fatal("expected non-nil engine")
	}
}

func TestCreateTask(t *testing.T) {
	e := New()
	task := e.CreateTask("test task", "test body", "user", nil)

	if task.Title != "test task" {
		t.Errorf("expected title=test task, got %s", task.Title)
	}
	if task.Status != StatusReady {
		t.Errorf("expected StatusReady for task without deps, got %s", task.Status)
	}
}

func TestCreateTaskWithDeps(t *testing.T) {
	e := New()
	parent := e.CreateTask("parent", "", "user", nil)
	child := e.CreateTask("child", "", "user", []string{parent.ID})

	if child.Status != StatusTodo {
		t.Errorf("expected StatusTodo when parent not done, got %s", child.Status)
	}
}

func TestClaimTask(t *testing.T) {
	e := New()
	task := e.CreateTask("test", "", "user", nil)

	err := e.ClaimTask(task.ID, "worker-1")
	if err != nil {
		t.Fatalf("ClaimTask failed: %v", err)
	}

	got, _ := e.GetTask(task.ID)
	if got.Status != StatusRunning {
		t.Errorf("expected StatusRunning, got %s", got.Status)
	}
	if got.ClaimLock != "worker-1" {
		t.Errorf("expected claim_lock=worker-1, got %s", got.ClaimLock)
	}
}

func TestCompleteTask(t *testing.T) {
	e := New()
	task := e.CreateTask("test", "", "user", nil)
	e.ClaimTask(task.ID, "worker")

	err := e.CompleteTask(task.ID, "done")
	if err != nil {
		t.Fatalf("CompleteTask failed: %v", err)
	}

	got, _ := e.GetTask(task.ID)
	if got.Status != StatusDone {
		t.Errorf("expected StatusDone, got %s", got.Status)
	}
}

func TestBlockTask(t *testing.T) {
	e := New()
	task := e.CreateTask("test", "", "user", nil)
	e.ClaimTask(task.ID, "worker")

	err := e.BlockTask(task.ID, "waiting for input")
	if err != nil {
		t.Fatalf("BlockTask failed: %v", err)
	}

	got, _ := e.GetTask(task.ID)
	if got.Status != StatusBlocked {
		t.Errorf("expected StatusBlocked, got %s", got.Status)
	}
}

func TestUnblockTask(t *testing.T) {
	e := New()
	task := e.CreateTask("test", "", "user", nil)
	e.ClaimTask(task.ID, "worker")
	e.BlockTask(task.ID, "blocked")

	err := e.UnblockTask(task.ID)
	if err != nil {
		t.Fatalf("UnblockTask failed: %v", err)
	}

	got, _ := e.GetTask(task.ID)
	if got.Status != StatusReady {
		t.Errorf("expected StatusReady after unblock, got %s", got.Status)
	}
}

func TestCircuitBreaker(t *testing.T) {
	e := New()
	task := e.CreateTask("test", "", "user", nil)

	for i := 0; i < 5; i++ {
		e.ClaimTask(task.ID, "worker")
		e.BlockTask(task.ID, "fail")
	}

	got, _ := e.GetTask(task.ID)
	if got.ConsecutiveFails != 5 {
		t.Errorf("expected 5 consecutive fails, got %d", got.ConsecutiveFails)
	}
}

func TestLinkTasks(t *testing.T) {
	e := New()
	parent := e.CreateTask("parent", "", "user", nil)
	child := e.CreateTask("child", "", "user", nil)

	err := e.LinkTasks(parent.ID, child.ID)
	if err != nil {
		t.Fatalf("LinkTasks failed: %v", err)
	}

	// 完成parent，child应自动提升为ready
	e.CompleteTask(parent.ID, "done")

	got, _ := e.GetTask(child.ID)
	if got.Status != StatusReady {
		t.Errorf("expected StatusReady after parent done, got %s", got.Status)
	}
}

func TestCycleDetection(t *testing.T) {
	e := New()
	a := e.CreateTask("a", "", "user", nil)
	b := e.CreateTask("b", "", "user", nil)
	c := e.CreateTask("c", "", "user", nil)

	e.LinkTasks(a.ID, b.ID)
	e.LinkTasks(b.ID, c.ID)

	// c→a 会形成环
	err := e.LinkTasks(c.ID, a.ID)
	if err == nil {
		t.Error("expected cycle detection error")
	}
}

func TestListReady(t *testing.T) {
	e := New()
	e.CreateTask("t1", "", "user", nil)
	e.CreateTask("t2", "", "user", nil)

	ready := e.ListReady()
	if len(ready) != 2 {
		t.Errorf("expected 2 ready tasks, got %d", len(ready))
	}
}

func TestGetEvents(t *testing.T) {
	e := New()
	task := e.CreateTask("test", "", "user", nil)
	e.ClaimTask(task.ID, "worker")
	e.CompleteTask(task.ID, "done")

	events := e.GetEvents(10)
	if len(events) < 3 {
		t.Errorf("expected at least 3 events, got %d", len(events))
	}
}

func TestListAll(t *testing.T) {
	e := New()
	e.CreateTask("t1", "", "user", nil)
	e.CreateTask("t2", "", "user", nil)
	e.CreateTask("t3", "", "user", nil)

	tasks := e.ListAll()
	if len(tasks) != 3 {
		t.Errorf("expected 3 tasks, got %d", len(tasks))
	}
}
