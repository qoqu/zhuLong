package bot

import (
	"testing"
)

func TestNewManager(t *testing.T) {
	mgr := NewManager()

	if mgr == nil {
		t.Fatal("NewManager returned nil")
	}

	if mgr.adapters == nil {
		t.Error("adapters map should be initialized")
	}

	if mgr.dedup == nil {
		t.Error("deduplicator should be initialized")
	}
}

func TestManagerAddAdapter(t *testing.T) {
	mgr := NewManager()

	config := AdapterConfig{
		Platform: PlatformTelegram,
		Name:     "test-telegram",
		Enabled:  true,
		Token:    "test-token",
	}

	// 注册适配器工厂
	DefaultRegistry.Register(&PlatformEntry{
		Platform: PlatformTelegram,
		Label:    "Telegram",
		Factory: func(config AdapterConfig) (Adapter, error) {
			return NewTelegramAdapter(config), nil
		},
	})

	// 添加适配器
	err := mgr.AddAdapter(config)
	if err != nil {
		t.Fatalf("AddAdapter failed: %v", err)
	}

	// 验证适配器已添加
	adapter, err := mgr.GetAdapter("test-telegram")
	if err != nil {
		t.Fatalf("GetAdapter failed: %v", err)
	}

	if adapter.Platform() != PlatformTelegram {
		t.Errorf("Expected platform 'telegram', got %q", adapter.Platform())
	}

	if adapter.Name() != "test-telegram" {
		t.Errorf("Expected name 'test-telegram', got %q", adapter.Name())
	}
}

func TestManagerAddDuplicateAdapter(t *testing.T) {
	mgr := NewManager()

	config := AdapterConfig{
		Platform: PlatformTelegram,
		Name:     "test-telegram-dup",
		Enabled:  true,
		Token:    "test-token",
	}

	// 注册适配器工厂
	DefaultRegistry.Register(&PlatformEntry{
		Platform: PlatformTelegram,
		Label:    "Telegram",
		Factory: func(config AdapterConfig) (Adapter, error) {
			return NewTelegramAdapter(config), nil
		},
	})

	// 第一次添加
	err := mgr.AddAdapter(config)
	if err != nil {
		t.Fatalf("First AddAdapter failed: %v", err)
	}

	// 第二次添加应该失败
	err = mgr.AddAdapter(config)
	if err == nil {
		t.Error("Expected error for duplicate adapter, got nil")
	}
}

func TestManagerGetNonExistentAdapter(t *testing.T) {
	mgr := NewManager()

	_, err := mgr.GetAdapter("non-existent")
	if err == nil {
		t.Error("Expected error for non-existent adapter, got nil")
	}
}

func TestManagerRemoveAdapter(t *testing.T) {
	mgr := NewManager()

	config := AdapterConfig{
		Platform: PlatformTelegram,
		Name:     "test-telegram-remove",
		Enabled:  true,
		Token:    "test-token",
	}

	// 注册适配器工厂
	DefaultRegistry.Register(&PlatformEntry{
		Platform: PlatformTelegram,
		Label:    "Telegram",
		Factory: func(config AdapterConfig) (Adapter, error) {
			return NewTelegramAdapter(config), nil
		},
	})

	// 添加适配器
	err := mgr.AddAdapter(config)
	if err != nil {
		t.Fatalf("AddAdapter failed: %v", err)
	}

	// 移除适配器
	err = mgr.RemoveAdapter("test-telegram-remove")
	if err != nil {
		t.Fatalf("RemoveAdapter failed: %v", err)
	}

	// 验证适配器已移除
	_, err = mgr.GetAdapter("test-telegram-remove")
	if err == nil {
		t.Error("Expected error after removal, got nil")
	}
}

func TestManagerListAdapters(t *testing.T) {
	mgr := NewManager()

	config1 := AdapterConfig{
		Platform: PlatformTelegram,
		Name:     "test-telegram-list",
		Enabled:  true,
		Token:    "test-token",
	}

	config2 := AdapterConfig{
		Platform: PlatformDiscord,
		Name:     "test-discord-list",
		Enabled:  true,
		Token:    "test-token",
	}

	// 注册适配器工厂
	DefaultRegistry.Register(&PlatformEntry{
		Platform: PlatformTelegram,
		Label:    "Telegram",
		Factory: func(config AdapterConfig) (Adapter, error) {
			return NewTelegramAdapter(config), nil
		},
	})

	DefaultRegistry.Register(&PlatformEntry{
		Platform: PlatformDiscord,
		Label:    "Discord",
		Factory: func(config AdapterConfig) (Adapter, error) {
			return NewDiscordAdapter(config), nil
		},
	})

	// 添加适配器
	mgr.AddAdapter(config1)
	mgr.AddAdapter(config2)

	// 列出适配器
	adapters := mgr.ListAdapters()
	if len(adapters) != 2 {
		t.Errorf("Expected 2 adapters, got %d", len(adapters))
	}
}

func TestManagerSetMessageHandler(t *testing.T) {
	mgr := NewManager()

	handler := func(event *MessageEvent) {
		// handler implementation
	}

	mgr.SetMessageHandler(handler)

	if mgr.handler == nil {
		t.Error("Handler should be set")
	}
}

func TestManagerIsConnected(t *testing.T) {
	mgr := NewManager()

	config := AdapterConfig{
		Platform: PlatformTelegram,
		Name:     "test-telegram-connected",
		Enabled:  true,
		Token:    "test-token",
	}

	// 注册适配器工厂
	DefaultRegistry.Register(&PlatformEntry{
		Platform: PlatformTelegram,
		Label:    "Telegram",
		Factory: func(config AdapterConfig) (Adapter, error) {
			return NewTelegramAdapter(config), nil
		},
	})

	// 添加适配器
	mgr.AddAdapter(config)

	// 初始状态应该是未连接
	if mgr.IsConnected("test-telegram-connected") {
		t.Error("Adapter should not be connected initially")
	}
}

func TestManagerListConnectedAdapters(t *testing.T) {
	mgr := NewManager()

	// 初始状态应该没有已连接的适配器
	connected := mgr.ListConnectedAdapters()
	if len(connected) != 0 {
		t.Errorf("Expected 0 connected adapters, got %d", len(connected))
	}
}
