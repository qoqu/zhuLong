package voice

import (
	"testing"
)

type mockSTT struct{}

func (m *mockSTT) Name() string                       { return "mock-stt" }
func (m *mockSTT) Transcribe(data []byte) (string, error) { return "hello", nil }

type mockTTS struct{}

func (m *mockTTS) Name() string                         { return "mock-tts" }
func (m *mockTTS) Synthesize(text string) ([]byte, error) { return []byte("audio-data"), nil }

func TestNewEngine(t *testing.T) {
	e := NewEngine()
	if e == nil {
		t.Fatal("expected non-nil engine")
	}
}

func TestSTT(t *testing.T) {
	e := NewEngine()
	e.SetSTT(&mockSTT{})

	text, err := e.Transcribe([]byte("audio"))
	if err != nil {
		t.Fatalf("Transcribe failed: %v", err)
	}
	if text != "hello" {
		t.Errorf("expected hello, got %s", text)
	}
}

func TestTTS(t *testing.T) {
	e := NewEngine()
	e.SetTTS(&mockTTS{})

	data, err := e.Synthesize("hello")
	if err != nil {
		t.Fatalf("Synthesize failed: %v", err)
	}
	if string(data) != "audio-data" {
		t.Errorf("expected audio-data, got %s", string(data))
	}
}

func TestHasSTT(t *testing.T) {
	e := NewEngine()
	if e.HasSTT() {
		t.Error("expected no STT initially")
	}
	e.SetSTT(&mockSTT{})
	if !e.HasSTT() {
		t.Error("expected STT after set")
	}
}

func TestEmptyTranscribe(t *testing.T) {
	e := NewEngine()
	text, err := e.Transcribe(nil)
	if err != nil {
		t.Fatalf("Transcribe failed: %v", err)
	}
	if text != "" {
		t.Errorf("expected empty, got %s", text)
	}
}
