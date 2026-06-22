package gateway

import (
	"testing"
	"time"
)

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	if r == nil {
		t.Fatal("expected non-nil registry")
	}
}

func TestRegisterAndGet(t *testing.T) {
	r := NewRegistry()
	r.Register(NewCLIAdapter())

	a, ok := r.Get(PlatformCLI)
	if !ok {
		t.Fatal("expected CLI adapter")
	}
	if a.Name() != "CLI" {
		t.Errorf("expected name=CLI, got %s", a.Name())
	}
}

func TestListPlatforms(t *testing.T) {
	r := NewRegistry()
	r.Register(NewCLIAdapter())

	platforms := r.ListPlatforms()
	if len(platforms) != 1 {
		t.Errorf("expected 1 platform, got %d", len(platforms))
	}
}

func TestDispatch(t *testing.T) {
	r := NewRegistry()
	received := make(chan Message, 1)

	r.OnMessage(func(msg Message) {
		received <- msg
	})

	r.Dispatch(Message{
		Platform: PlatformCLI, Content: "hello",
		Timestamp: time.Now(),
	})

	select {
	case msg := <-received:
		if msg.Content != "hello" {
			t.Errorf("expected content=hello, got %s", msg.Content)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}

func TestCLIAdapter(t *testing.T) {
	a := NewCLIAdapter()
	if !a.IsAvailable() {
		t.Error("CLI should always be available")
	}

	err := a.Start(nil)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	err = a.Send(OutgoingMessage{Content: "test"})
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	a.Stop()
}

func TestNewRunner(t *testing.T) {
	runner := NewRunner()
	if runner == nil {
		t.Fatal("expected non-nil runner")
	}
}

func TestRunner_RegisterAdapter(t *testing.T) {
	runner := NewRunner()
	runner.RegisterAdapter(NewCLIAdapter())

	platforms := runner.Registry().ListPlatforms()
	if len(platforms) != 1 {
		t.Errorf("expected 1 platform, got %d", len(platforms))
	}
}

func TestRunner_StartStop(t *testing.T) {
	runner := NewRunner()
	runner.RegisterAdapter(NewCLIAdapter())

	err := runner.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	runner.Stop()
}

func TestNewSessionStore(t *testing.T) {
	s := NewSessionStore()
	if s == nil {
		t.Fatal("expected non-nil store")
	}
}

func TestSessionStore_Create(t *testing.T) {
	s := NewSessionStore()
	ses := s.GetOrCreate(PlatformCLI, "user1", "channel1")
	if ses.UserID != "user1" {
		t.Errorf("expected user1, got %s", ses.UserID)
	}
}

func TestSessionStore_Reuse(t *testing.T) {
	s := NewSessionStore()
	s1 := s.GetOrCreate(PlatformCLI, "user1", "ch1")
	s2 := s.GetOrCreate(PlatformCLI, "user1", "ch1")
	if s1.ID != s2.ID {
		t.Error("expected same session for same user+channel")
	}
}

func TestSessionStore_Count(t *testing.T) {
	s := NewSessionStore()
	s.GetOrCreate(PlatformCLI, "u1", "c1")
	s.GetOrCreate(PlatformTelegram, "u2", "c2")

	if s.Count() != 2 {
		t.Errorf("expected 2 sessions, got %d", s.Count())
	}
}
