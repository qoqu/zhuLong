package plugins

import (
	"testing"
)

type mockPlugin struct{}

func (m *mockPlugin) Name() string    { return "mock" }
func (m *mockPlugin) Version() string { return "1.0.0" }
func (m *mockPlugin) Init() error     { return nil }
func (m *mockPlugin) Shutdown() error { return nil }

type failPlugin struct{}

func (m *failPlugin) Name() string    { return "fail" }
func (m *failPlugin) Version() string { return "0.1.0" }
func (m *failPlugin) Init() error     { return nil }
func (m *failPlugin) Shutdown() error { return nil }

func TestNewManager(t *testing.T) {
	m := NewManager()
	if m == nil {
		t.Fatal("expected non-nil manager")
	}
}

func TestRegister(t *testing.T) {
	m := NewManager()
	err := m.Register(&mockPlugin{}, TypeTool)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if m.Count() != 1 {
		t.Errorf("expected 1 plugin, got %d", m.Count())
	}
}

func TestDuplicate(t *testing.T) {
	m := NewManager()
	m.Register(&mockPlugin{}, TypeTool)
	err := m.Register(&mockPlugin{}, TypeTool)
	if err == nil {
		t.Error("expected error for duplicate registration")
	}
}

func TestUnregister(t *testing.T) {
	m := NewManager()
	m.Register(&mockPlugin{}, TypeTool)
	err := m.Unregister("mock")
	if err != nil {
		t.Fatalf("Unregister failed: %v", err)
	}
	if m.Count() != 0 {
		t.Errorf("expected 0 after unregister, got %d", m.Count())
	}
}

func TestGet(t *testing.T) {
	m := NewManager()
	m.Register(&mockPlugin{}, TypeTool)
	p, ok := m.Get("mock")
	if !ok {
		t.Fatal("expected to find plugin")
	}
	if p.Name() != "mock" {
		t.Errorf("expected name=mock, got %s", p.Name())
	}
}

func TestList(t *testing.T) {
	m := NewManager()
	m.Register(&mockPlugin{}, TypeTool)
	list := m.List()
	if len(list) != 1 {
		t.Errorf("expected 1 in list, got %d", len(list))
	}
}
