package loop

import (
	"testing"
	"time"
)

func TestLoopEngine_Register(t *testing.T) {
	engine := NewLoopEngine()
	task := GCScanTask(1 * time.Hour)

	engine.Register(task)

	got, err := engine.GetState(task.ID)
	if err != nil {
		t.Fatalf("GetState failed: %v", err)
	}

	if got.State != StateRunning {
		t.Errorf("expected StateRunning, got %v", got.State)
	}
}

func TestLoopEngine_RecordResult(t *testing.T) {
	engine := NewLoopEngine()
	task := GCScanTask(1 * time.Hour)
	engine.Register(task)

	result := LoopResult{
		TaskID:    task.ID,
		Success:   true,
		Output:    "scan completed",
		Timestamp: time.Now(),
	}
	engine.RecordResult(result)

	got, _ := engine.GetState(task.ID)
	if got.RunCount != 1 {
		t.Errorf("expected RunCount=1, got %d", got.RunCount)
	}
	if got.FailCount != 0 {
		t.Errorf("expected FailCount=0, got %d", got.FailCount)
	}
}

func TestLoopEngine_CircuitBreaker(t *testing.T) {
	engine := NewLoopEngine()
	task := &LoopTask{
		ID:          "breaker-test",
		Name:        "Breaker Test",
		Interval:    1 * time.Minute,
		Command:     "test",
		MaxFailures: 3,
		State:       StateRunning,
	}
	engine.Register(task)

	// 连续失败3次
	for i := 0; i < 3; i++ {
		engine.RecordResult(LoopResult{
			TaskID:    task.ID,
			Success:   false,
			Output:    "failed",
			Timestamp: time.Now(),
		})
	}

	got, _ := engine.GetState(task.ID)
	if got.State != StatePaused {
		t.Errorf("expected StatePaused after 3 failures, got %v", got.State)
	}
}

func TestLoopEngine_Recovery(t *testing.T) {
	engine := NewLoopEngine()
	task := &LoopTask{
		ID:          "recovery-test",
		Name:        "Recovery Test",
		Interval:    1 * time.Minute,
		Command:     "test",
		MaxFailures: 3,
		State:       StateRunning,
	}
	engine.Register(task)

	// 失败2次后成功1次
	engine.RecordResult(LoopResult{TaskID: task.ID, Success: false, Timestamp: time.Now()})
	engine.RecordResult(LoopResult{TaskID: task.ID, Success: false, Timestamp: time.Now()})
	engine.RecordResult(LoopResult{TaskID: task.ID, Success: true, Timestamp: time.Now()})

	got, _ := engine.GetState(task.ID)
	if got.FailCount != 0 {
		t.Errorf("expected FailCount reset to 0 after success, got %d", got.FailCount)
	}
	if got.State != StateRunning {
		t.Errorf("expected StateRunning after success, got %v", got.State)
	}
}

func TestLoopEngine_GetLogs(t *testing.T) {
	engine := NewLoopEngine()
	task := GCScanTask(1 * time.Hour)
	engine.Register(task)

	engine.RecordResult(LoopResult{TaskID: task.ID, Success: true, Timestamp: time.Now()})
	engine.RecordResult(LoopResult{TaskID: task.ID, Success: true, Timestamp: time.Now()})

	logs := engine.GetLogs()
	if len(logs) != 2 {
		t.Errorf("expected 2 logs, got %d", len(logs))
	}
}
