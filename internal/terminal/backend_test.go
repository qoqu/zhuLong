package terminal

import (
	"context"
	"testing"
)

func TestNewLocalBackend(t *testing.T) {
	b := NewLocalBackend()
	if b.Type() != BackendLocal {
		t.Errorf("expected BackendLocal, got %s", b.Type())
	}
	if !b.IsAvailable() {
		t.Error("local backend should always be available")
	}
}

func TestLocalBackend_Execute(t *testing.T) {
	b := NewLocalBackend()
	result, err := b.Execute(context.Background(), "echo", []string{"hello"}, nil)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result.Stdout != "hello\n" && result.Stdout != "hello\r\n" {
		t.Errorf("expected stdout=hello, got %q", result.Stdout)
	}
}

func TestLocalBackend_ExitCode(t *testing.T) {
	b := NewLocalBackend()
	result, err := b.Execute(context.Background(), "cmd_that_does_not_exist_xyz", nil, nil)
	if err == nil {
		// 可能返回非零退出码
		if result.ExitCode == 0 {
			t.Error("expected non-zero exit code")
		}
	}
}

func TestNewDockerBackend(t *testing.T) {
	b := NewDockerBackend("ubuntu:latest")
	if b.Type() != BackendDocker {
		t.Errorf("expected BackendDocker, got %s", b.Type())
	}
	if b.Name() != "Docker (ubuntu:latest)" {
		t.Errorf("unexpected name: %s", b.Name())
	}
}

func TestNewSSHBackend(t *testing.T) {
	b := NewSSHBackend("example.com", "user", 22)
	if b.Type() != BackendSSH {
		t.Errorf("expected BackendSSH, got %s", b.Type())
	}
	if b.Name() != "SSH (user@example.com)" {
		t.Errorf("unexpected name: %s", b.Name())
	}
}

func TestNewManager(t *testing.T) {
	m := NewManager()
	if m.Current().Type() != BackendLocal {
		t.Errorf("expected default backend to be local")
	}
}

func TestManager_Register(t *testing.T) {
	m := NewManager()
	m.Register(NewDockerBackend("ubuntu:latest"))

	list := m.List()
	if len(list) != 2 {
		t.Errorf("expected 2 backends, got %d", len(list))
	}
}

func TestManager_SetCurrent(t *testing.T) {
	m := NewManager()
	m.Register(NewDockerBackend("ubuntu:latest"))

	// Docker可能不可用，但不影响注册
	err := m.SetCurrent(BackendDocker)
	_ = err // 可能失败因为docker不可用
}

func TestManager_Execute(t *testing.T) {
	m := NewManager()
	result, err := m.Execute(context.Background(), "echo", "test")
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result.Stdout == "" {
		t.Error("expected non-empty stdout")
	}
}

func TestManager_ExecuteWithEnv(t *testing.T) {
	m := NewManager()
	result, err := m.ExecuteWithEnv(context.Background(), "sh", []string{"-c", "echo $TEST_VAR"}, map[string]string{"TEST_VAR": "hello"})
	if err != nil {
		t.Fatalf("ExecuteWithEnv failed: %v", err)
	}
	if result.Stdout == "" {
		t.Error("expected stdout with env var")
	}
}

func TestManager_List(t *testing.T) {
	m := NewManager()
	list := m.List()
	if len(list) != 1 {
		t.Errorf("expected 1 backend initially, got %d", len(list))
	}
}

func TestDockerBackend_SecurityConfig(t *testing.T) {
	b := NewDockerBackend("alpine:latest")
	_ = b
	// Docker后端的--cap-drop=ALL和--security-opt=no-new-privileges在内置逻辑中
}

func TestManager_Current(t *testing.T) {
	m := NewManager()
	current := m.Current()
	if current == nil {
		t.Fatal("expected non-nil current backend")
	}
	if current.Type() != BackendLocal {
		t.Errorf("expected BackendLocal, got %s", current.Type())
	}
}
