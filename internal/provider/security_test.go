package provider

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewKeyManager(t *testing.T) {
	km := NewKeyManager(nil)
	if km == nil {
		t.Fatal("NewKeyManager returned nil")
	}
	if km.config == nil {
		t.Fatal("config is nil")
	}
}

func TestDefaultSecurityConfig(t *testing.T) {
	config := DefaultSecurityConfig()
	if config == nil {
		t.Fatal("DefaultSecurityConfig returned nil")
	}
	if config.KeyEnvVar != "DEEPSEEK_API_KEY" {
		t.Errorf("expected KeyEnvVar DEEPSEEK_API_KEY, got %s", config.KeyEnvVar)
	}
	if !config.MaskKey {
		t.Error("expected MaskKey to be true")
	}
}

func TestKeyManagerGetKeyFromEnv(t *testing.T) {
	// Set environment variable
	os.Setenv("TEST_API_KEY", "test-key-12345")
	defer os.Unsetenv("TEST_API_KEY")

	config := &SecurityConfig{
		KeyEnvVar: "TEST_API_KEY",
		MaskKey:   true,
	}
	km := NewKeyManager(config)

	key, source, err := km.GetKey()
	if err != nil {
		t.Fatalf("GetKey failed: %v", err)
	}
	if key != "test-key-12345" {
		t.Errorf("expected key test-key-12345, got %s", key)
	}
	if source != KeySourceEnv {
		t.Errorf("expected source env, got %s", source)
	}
}

func TestKeyManagerGetKeyFromFile(t *testing.T) {
	// Create temp file
	dir := t.TempDir()
	keyFile := filepath.Join(dir, "api_key")
	os.WriteFile(keyFile, []byte("file-key-12345"), 0600)

	config := &SecurityConfig{
		KeyFile: keyFile,
		MaskKey: true,
	}
	km := NewKeyManager(config)

	key, source, err := km.GetKey()
	if err != nil {
		t.Fatalf("GetKey failed: %v", err)
	}
	if key != "file-key-12345" {
		t.Errorf("expected key file-key-12345, got %s", key)
	}
	if source != KeySourceFile {
		t.Errorf("expected source file, got %s", source)
	}
}

func TestKeyManagerMaskKey(t *testing.T) {
	config := &SecurityConfig{
		MaskKey: true,
	}
	km := NewKeyManager(config)

	tests := []struct {
		key      string
		expected string
	}{
		{"12345678", "****"},
		{"123456789", "1234****6789"},
		{"sk-1234567890abcdef", "sk-1****cdef"},
	}

	for _, tt := range tests {
		result := km.MaskKey(tt.key)
		if result != tt.expected {
			t.Errorf("MaskKey(%s) = %s, expected %s", tt.key, result, tt.expected)
		}
	}
}

func TestKeyManagerSetKey(t *testing.T) {
	dir := t.TempDir()
	keyFile := filepath.Join(dir, "api_key")

	config := &SecurityConfig{
		KeyFile: keyFile,
		MaskKey: true,
	}
	km := NewKeyManager(config)

	err := km.SetKey("new-key-12345", KeySourceFile)
	if err != nil {
		t.Fatalf("SetKey failed: %v", err)
	}

	// Read the key back
	key, _, err := km.GetKey()
	if err != nil {
		t.Fatalf("GetKey failed: %v", err)
	}
	if key != "new-key-12345" {
		t.Errorf("expected key new-key-12345, got %s", key)
	}
}

func TestKeySourceString(t *testing.T) {
	tests := []struct {
		source   KeySource
		expected string
	}{
		{KeySourceEnv, "env"},
		{KeySourceFile, "file"},
		{KeySourceKeychain, "keychain"},
		{KeySourcePrompt, "prompt"},
	}

	for _, tt := range tests {
		if string(tt.source) != tt.expected {
			t.Errorf("expected %s, got %s", tt.expected, string(tt.source))
		}
	}
}
