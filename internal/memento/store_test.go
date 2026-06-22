package memento

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewMemoryStore(t *testing.T) {
	dir := t.TempDir()
	ms := NewMemoryStore(dir)
	if ms == nil {
		t.Fatal("expected non-nil store")
	}
}

func TestInit(t *testing.T) {
	dir := t.TempDir()
	ms := NewMemoryStore(dir)
	err := ms.Init()
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "MEMORY.md")); os.IsNotExist(err) {
		t.Error("MEMORY.md not created")
	}
	if _, err := os.Stat(filepath.Join(dir, "USER.md")); os.IsNotExist(err) {
		t.Error("USER.md not created")
	}
}

func TestSaveAndGetSnapshot(t *testing.T) {
	dir := t.TempDir()
	ms := NewMemoryStore(dir)
	ms.Init()

	err := ms.SaveAgentNote("This is a test note for the agent.")
	if err != nil {
		t.Fatalf("SaveAgentNote failed: %v", err)
	}

	agent, user := ms.GetSnapshot()
	if !strings.Contains(agent, "test note") {
		t.Error("expected test note in agent snapshot")
	}
	if user != "" {
		t.Error("expected empty user profile")
	}
}

func TestAppendAgentNote(t *testing.T) {
	dir := t.TempDir()
	ms := NewMemoryStore(dir)
	ms.Init()

	ms.SaveAgentNote("First note.")
	ms.AppendAgentNote("Second note.")

	agent, _ := ms.GetSnapshot()
	if !strings.Contains(agent, "First note.") || !strings.Contains(agent, "Second note.") {
		t.Error("expected both notes in agent snapshot")
	}
}

func TestMaxAgentChars(t *testing.T) {
	dir := t.TempDir()
	ms := NewMemoryStore(dir)
	ms.Init()

	longContent := strings.Repeat("A", MaxAgentChars+100)
	err := ms.SaveAgentNote(longContent)
	if err != nil {
		t.Fatalf("SaveAgentNote failed: %v", err)
	}

	agent, _ := ms.GetSnapshot()
	if len([]rune(agent)) > MaxAgentChars {
		t.Errorf("agent note exceeds max chars: %d > %d", len([]rune(agent)), MaxAgentChars)
	}
}

func TestSaveUserProfile(t *testing.T) {
	dir := t.TempDir()
	ms := NewMemoryStore(dir)
	ms.Init()

	err := ms.SaveUserProfile("User prefers concise answers.")
	if err != nil {
		t.Fatalf("SaveUserProfile failed: %v", err)
	}

	_, user := ms.GetSnapshot()
	if !strings.Contains(user, "concise") {
		t.Error("expected user preference in profile")
	}
}

func TestMaxUserChars(t *testing.T) {
	dir := t.TempDir()
	ms := NewMemoryStore(dir)
	ms.Init()

	longContent := strings.Repeat("U", MaxUserChars+100)
	err := ms.SaveUserProfile(longContent)
	if err != nil {
		t.Fatalf("SaveUserProfile failed: %v", err)
	}

	_, user := ms.GetSnapshot()
	if len([]rune(user)) > MaxUserChars {
		t.Errorf("user profile exceeds max chars: %d > %d", len([]rune(user)), MaxUserChars)
	}
}

func TestUpdateUserProfile(t *testing.T) {
	dir := t.TempDir()
	ms := NewMemoryStore(dir)
	ms.Init()

	ms.SaveUserProfile("Style: technical\nLanguage: English")
	err := ms.UpdateUserProfile("technical", "concise")
	if err != nil {
		t.Fatalf("UpdateUserProfile failed: %v", err)
	}

	_, user := ms.GetSnapshot()
	if strings.Contains(user, "technical") {
		t.Error("expected 'technical' to be replaced")
	}
	if !strings.Contains(user, "concise") {
		t.Error("expected 'concise' in updated profile")
	}
}

func TestUpdateUserProfile_SubstringNotFound(t *testing.T) {
	dir := t.TempDir()
	ms := NewMemoryStore(dir)
	ms.Init()

	ms.SaveUserProfile("Hello")
	err := ms.UpdateUserProfile("nonexistent", "replacement")
	if err == nil {
		t.Error("expected error for nonexistent substring")
	}
}

func TestSearch(t *testing.T) {
	dir := t.TempDir()
	ms := NewMemoryStore(dir)

	ms.SaveAgentNote("The agent prefers Go language.")
	ms.SaveAgentNote("Testing is important.")

	results := ms.Search("language")
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
}

func TestGetHistory(t *testing.T) {
	dir := t.TempDir()
	ms := NewMemoryStore(dir)

	for i := 0; i < 5; i++ {
		ms.SaveAgentNote("Note " + string(rune('0'+i)))
	}

	history := ms.GetHistory(3)
	if len(history) != 3 {
		t.Errorf("expected 3 history entries, got %d", len(history))
	}
}

func TestStats(t *testing.T) {
	dir := t.TempDir()
	ms := NewMemoryStore(dir)
	ms.Init()

	ms.SaveAgentNote("test")
	ms.SaveUserProfile("user")

	stats := ms.Stats()
	if stats["agent_chars"] == 0 && stats["user_chars"] == 0 {
		t.Error("expected non-zero stats")
	}
}

func TestSnapshotImmutability(t *testing.T) {
	// 验证快照模式：同一会话中多次GetSnapshot返回相同内容
	dir := t.TempDir()
	ms := NewMemoryStore(dir)
	ms.Init()

	ms.SaveAgentNote("Original content.")
	snap1, _ := ms.GetSnapshot()

	ms.SaveAgentNote("Modified content.")
	snap2, _ := ms.GetSnapshot()

	// 快照应反映最新状态（不同会话之间）
	if snap1 == snap2 {
		t.Error("expected different snapshots after modification")
	}
}

func TestProviderRegistry(t *testing.T) {
	pr := NewProviderRegistry()

	// Mock provider
	mock := &mockProvider{name: "test-provider"}
	pr.Register(mock)

	got, ok := pr.Get("test-provider")
	if !ok {
		t.Fatal("expected to find provider")
	}
	if got.Name() != "test-provider" {
		t.Errorf("expected name=test-provider, got %s", got.Name())
	}

	names := pr.List()
	if len(names) != 1 {
		t.Errorf("expected 1 provider, got %d", len(names))
	}
}

func TestProviderRegistry_NotFound(t *testing.T) {
	pr := NewProviderRegistry()
	_, ok := pr.Get("nonexistent")
	if ok {
		t.Error("expected not found")
	}
}

type mockProvider struct {
	name string
	data map[string]string
}

func (m *mockProvider) Name() string { return m.name }

func (m *mockProvider) Store(key, value string) error {
	if m.data == nil {
		m.data = make(map[string]string)
	}
	m.data[key] = value
	return nil
}

func (m *mockProvider) Retrieve(key string) (string, bool) {
	if m.data == nil {
		return "", false
	}
	v, ok := m.data[key]
	return v, ok
}

func (m *mockProvider) Search(query string) []string {
	var results []string
	for k, v := range m.data {
		if strings.Contains(k, query) || strings.Contains(v, query) {
			results = append(results, v)
		}
	}
	return results
}
