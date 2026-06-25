package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSaveLoadSession_RoundTrip(t *testing.T) {
	// 创建一个带 messages 的 SessionState
	now := time.Now()
	s := &SessionState{
		Info: SessionInfo{
			ID:           "s-test-001",
			Title:        "测试会话",
			AgentID:      "auto",
			ProjectID:    "p1",
			MessageCount: 3,
			ToolCount:     1,
			UpdatedAt:    "刚刚",
			Preview:       "测试预览",
		},
		Goal:   "测试目标",
		Status: "done",
		Mode:   "auto",
		Model:  "deepseek-v4-flash",
		Messages: []MessageDTO{
			{ID: "m1", Role: "user", Content: "你好", Time: now},
			{ID: "m2", Role: "assistant", Content: "你好！有什么可以帮你？", Time: now},
			{ID: "m3", Role: "tool", Content: "执行结果：ok", Time: now, ToolName: "execute_command"},
		},
		Logs:      []LogDTO{},
		Plan:      []PlanStepDTO{},
		Created:   now,
		Updated:   now,
	}

	// 序列化
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// 反序列化
	var s2 SessionState
	if err := json.Unmarshal(data, &s2); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// 验证 messages 往返
	if len(s2.Messages) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(s2.Messages))
	}
	if s2.Messages[0].Role != "user" {
		t.Errorf("expected first message role 'user', got %q", s2.Messages[0].Role)
	}
	if s2.Messages[1].Content != "你好！有什么可以帮你？" {
		t.Errorf("unexpected second message content: %q", s2.Messages[1].Content)
	}
	if s2.Messages[2].ToolName != "execute_command" {
		t.Errorf("expected tool name 'execute_command', got %q", s2.Messages[2].ToolName)
	}

	t.Log("✓ Message round-trip OK")
}

func TestSaveSession_ToFile(t *testing.T) {
	app := &App{}
	app.sessions = map[string]*SessionState{}

	now := time.Now()
	s := &SessionState{
		Info: SessionInfo{
			ID:        "s-test-002",
			Title:     "文件保存测试",
			MessageCount: 2,
		},
		Status: "done",
		Messages: []MessageDTO{
			{ID: "m1", Role: "user", Content: "测试消息1", Time: now},
			{ID: "m2", Role: "assistant", Content: "回复1", Time: now},
		},
		Created: now,
		Updated: now,
	}
	app.sessions["s-test-002"] = s

	// 调用 saveSession（新签名：直接传 *SessionState）
	app.saveSession(s)

	// 验证文件已写入
	dir := filepath.Join(zhulongDir(), "sessions")
	file := filepath.Join(dir, "s-test-002.json")
	if _, err := os.Stat(file); os.IsNotExist(err) {
		t.Fatalf("session file not created: %s", file)
	}

	// 读取文件并验证 messages
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("failed to read session file: %v", err)
	}

	var s2 SessionState
	if err := json.Unmarshal(data, &s2); err != nil {
		t.Fatalf("failed to unmarshal session file: %v", err)
	}

	if len(s2.Messages) != 2 {
		t.Fatalf("expected 2 messages in saved file, got %d", len(s2.Messages))
	}

	t.Logf("✓ Session file saved with %d messages", len(s2.Messages))

	// 清理
	os.Remove(file)
}

func TestLoadSession_FromFile(t *testing.T) {
	app := &App{}
	app.sessions = map[string]*SessionState{}

	// 先准备一个 session 文件
	now := time.Now()
	s := &SessionState{
		Info: SessionInfo{
			ID:        "s-test-003",
			Title:     "加载测试",
			MessageCount: 1,
		},
		Status: "idle",
		Messages: []MessageDTO{
			{ID: "m1", Role: "user", Content: "测试加载", Time: now},
		},
		Created: now,
		Updated: now,
	}

	dir := filepath.Join(zhulongDir(), "sessions")
	os.MkdirAll(dir, 0755)
	data, _ := json.MarshalIndent(s, "", "  ")
	file := filepath.Join(dir, "s-test-003.json")
	os.WriteFile(file, data, 0644)

	// 调用 loadSession
	s2 := app.loadSession("s-test-003")
	if s2 == nil {
		t.Fatal("loadSession returned nil")
	}
	if len(s2.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(s2.Messages))
	}
	if s2.Messages[0].Content != "测试加载" {
		t.Errorf("unexpected message content: %q", s2.Messages[0].Content)
	}

	t.Log("✓ Load session from file OK")

	// 验证已加载到内存
	if _, ok := app.sessions["s-test-003"]; !ok {
		t.Error("session not loaded into memory after loadSession")
	}

	// 清理
	os.Remove(file)
}
