package scheduler

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewScheduler(t *testing.T) {
	s := New(nil, nil)
	if s == nil {
		t.Fatal("expected non-nil scheduler")
	}
}

func TestOnEvent(t *testing.T) {
	s := New(nil, nil)
	received := make(chan Event, 1)

	s.On("test_event", func(event Event) {
		received <- event
	})

	s.Emit(Event{Type: "test_event", TaskID: "task-1", Payload: "hello"})

	select {
	case event := <-received:
		if event.TaskID != "task-1" {
			t.Errorf("expected taskID=task-1, got %s", event.TaskID)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for event")
	}
}

func TestMultipleHandlers(t *testing.T) {
	s := New(nil, nil)
	var count1, count2 int32

	s.On("test", func(event Event) { atomic.AddInt32(&count1, 1) })
	s.On("test", func(event Event) { atomic.AddInt32(&count2, 1) })

	s.Emit(Event{Type: "test"})

	time.Sleep(100 * time.Millisecond)

	if atomic.LoadInt32(&count1) != 1 {
		t.Errorf("expected handler1 called 1 time, got %d", count1)
	}
	if atomic.LoadInt32(&count2) != 1 {
		t.Errorf("expected handler2 called 1 time, got %d", count2)
	}
}

func TestTickEvent(t *testing.T) {
	// 使用短tick周期
	s := New(&Config{TickPeriod: 10 * time.Millisecond}, nil)
	tickReceived := make(chan Event, 1)

	s.On("tick", func(event Event) {
		tickReceived <- event
	})

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	go s.Start(ctx)

	select {
	case event := <-tickReceived:
		if event.Type != "tick" {
			t.Errorf("expected tick event, got %s", event.Type)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for tick")
	}
}

func TestStop(t *testing.T) {
	s := New(nil, nil)

	stopped := make(chan struct{})
	go func() {
		s.Start(context.Background())
		close(stopped)
	}()

	s.Stop()

	select {
	case <-stopped:
		// 正常停止
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for scheduler to stop")
	}
}

func TestDefaultSpawn(t *testing.T) {
	pid, err := defaultSpawn("task-1", "/tmp")
	if err != nil {
		// 没有zhulong命令是正常的，跳过
		t.Skip("zhulong command not available")
	}
	if pid <= 0 {
		t.Errorf("expected positive PID, got %d", pid)
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.TickPeriod != 60*time.Second {
		t.Errorf("expected TickPeriod=60s, got %s", cfg.TickPeriod)
	}
	if cfg.MaxWorkers != 5 {
		t.Errorf("expected MaxWorkers=5, got %d", cfg.MaxWorkers)
	}
}

func TestSpawnResult(t *testing.T) {
	result := SpawnWorker("task-1", "default", "/tmp")
	_ = result
}
