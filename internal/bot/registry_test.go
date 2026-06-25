package bot

import (
	"testing"
)

func TestNewRegistry(t *testing.T) {
	reg := NewRegistry()

	if reg == nil {
		t.Fatal("NewRegistry returned nil")
	}

	if reg.entries == nil {
		t.Error("entries map should be initialized")
	}
}

func TestRegistryRegister(t *testing.T) {
	reg := NewRegistry()

	entry := &PlatformEntry{
		Platform: PlatformTelegram,
		Label:    "Telegram",
		Factory: func(config AdapterConfig) (Adapter, error) {
			return NewTelegramAdapter(config), nil
		},
		Description:    "Telegram Bot API",
		RequiredFields: []string{"token"},
	}

	reg.Register(entry)

	// 验证注册成功
	got, err := reg.GetEntry(PlatformTelegram)
	if err != nil {
		t.Fatalf("GetEntry failed: %v", err)
	}

	if got.Label != "Telegram" {
		t.Errorf("Expected label='Telegram', got %q", got.Label)
	}
}

func TestRegistryCreate(t *testing.T) {
	reg := NewRegistry()

	entry := &PlatformEntry{
		Platform: PlatformTelegram,
		Label:    "Telegram",
		Factory: func(config AdapterConfig) (Adapter, error) {
			return NewTelegramAdapter(config), nil
		},
	}

	reg.Register(entry)

	config := AdapterConfig{
		Platform: PlatformTelegram,
		Name:     "test-tg",
		Token:    "test-token",
	}

	adapter, err := reg.Create(config)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if adapter.Platform() != PlatformTelegram {
		t.Errorf("Expected platform='telegram', got %q", adapter.Platform())
	}
}

func TestRegistryCreateUnsupported(t *testing.T) {
	reg := NewRegistry()

	config := AdapterConfig{
		Platform: "unsupported",
		Name:     "test",
	}

	_, err := reg.Create(config)
	if err == nil {
		t.Error("Expected error for unsupported platform, got nil")
	}
}

func TestRegistryGetEntryNotFound(t *testing.T) {
	reg := NewRegistry()

	_, err := reg.GetEntry(PlatformTelegram)
	if err == nil {
		t.Error("Expected error for non-existent platform, got nil")
	}
}

func TestRegistryListPlatforms(t *testing.T) {
	reg := NewRegistry()

	reg.Register(&PlatformEntry{
		Platform: PlatformTelegram,
		Label:    "Telegram",
	})

	reg.Register(&PlatformEntry{
		Platform: PlatformDiscord,
		Label:    "Discord",
	})

	platforms := reg.ListPlatforms()
	if len(platforms) != 2 {
		t.Errorf("Expected 2 platforms, got %d", len(platforms))
	}
}

func TestDefaultRegistry(t *testing.T) {
	// DefaultRegistry 应该已经被 init() 初始化
	if DefaultRegistry == nil {
		t.Fatal("DefaultRegistry should not be nil")
	}

	// 验证所有内置平台已注册
	platforms := []Platform{
		PlatformTelegram,
		PlatformFeishu,
		PlatformDingTalk,
		PlatformDiscord,
		PlatformSlack,
		PlatformWeCom,
		PlatformWebhook,
	}

	for _, p := range platforms {
		_, err := DefaultRegistry.GetEntry(p)
		if err != nil {
			t.Errorf("Platform %q should be registered in DefaultRegistry", p)
		}
	}
}
