package bot

import (
	"testing"
	"time"
)

func TestMessageEvent(t *testing.T) {
	event := &MessageEvent{
		Text:        "Hello, World!",
		MessageType: MessageTypeText,
		Source: SessionSource{
			Platform:  PlatformTelegram,
			ChatID:    "chat-123",
			ChatName:  "Test Chat",
			ChatType:  ChatTypeDM,
			UserID:    "user-123",
			UserName:  "Test User",
			MessageID: "msg-123",
		},
		MessageID: "msg-123",
		Timestamp: time.Now(),
	}

	if event.Text != "Hello, World!" {
		t.Errorf("Expected Text='Hello, World!', got %q", event.Text)
	}

	if event.Source.Platform != PlatformTelegram {
		t.Errorf("Expected Platform='telegram', got %q", event.Source.Platform)
	}

	if event.Source.ChatType != ChatTypeDM {
		t.Errorf("Expected ChatType='dm', got %q", event.Source.ChatType)
	}
}

func TestRichTextContent(t *testing.T) {
	richText := &RichTextContent{
		Elements: []RichTextElement{
			{Type: "text", Text: "Hello "},
			{Type: "mention", Text: "User", MentionID: "user-123"},
			{Type: "link", Text: "Click here", URL: "https://example.com"},
			{Type: "code", Text: "fmt.Println()", Language: "go"},
		},
	}

	if len(richText.Elements) != 4 {
		t.Errorf("Expected 4 elements, got %d", len(richText.Elements))
	}

	if richText.Elements[0].Type != "text" {
		t.Errorf("Expected first element type='text', got %q", richText.Elements[0].Type)
	}

	if richText.Elements[1].MentionID != "user-123" {
		t.Errorf("Expected mention ID='user-123', got %q", richText.Elements[1].MentionID)
	}
}

func TestInteractiveCard(t *testing.T) {
	card := &InteractiveCard{
		Title: "Test Card",
		Elements: []CardElement{
			{Type: "text", Text: "Card content"},
			{Type: "image", URL: "https://example.com/image.png"},
		},
		Actions: []CardAction{
			{Type: "button", Text: "Click me", Value: "https://example.com", Style: "primary"},
		},
	}

	if card.Title != "Test Card" {
		t.Errorf("Expected Title='Test Card', got %q", card.Title)
	}

	if len(card.Elements) != 2 {
		t.Errorf("Expected 2 elements, got %d", len(card.Elements))
	}

	if len(card.Actions) != 1 {
		t.Errorf("Expected 1 action, got %d", len(card.Actions))
	}

	if card.Actions[0].Style != "primary" {
		t.Errorf("Expected action style='primary', got %q", card.Actions[0].Style)
	}
}

func TestSessionSource(t *testing.T) {
	source := SessionSource{
		Platform:  PlatformDiscord,
		ChatID:    "channel-123",
		ChatName:  "general",
		ChatType:  ChatTypeChannel,
		UserID:    "user-456",
		UserName:  "TestUser",
		GuildID:   "guild-789",
		MessageID: "msg-456",
	}

	if source.Platform != PlatformDiscord {
		t.Errorf("Expected Platform='discord', got %q", source.Platform)
	}

	if source.GuildID != "guild-789" {
		t.Errorf("Expected GuildID='guild-789', got %q", source.GuildID)
	}
}

func TestSendResult(t *testing.T) {
	result := &SendResult{
		MessageID: "msg-789",
	}

	if result.MessageID != "msg-789" {
		t.Errorf("Expected MessageID='msg-789', got %q", result.MessageID)
	}
}

func TestChatInfo(t *testing.T) {
	info := &ChatInfo{
		ID:   "chat-123",
		Name: "Test Chat",
		Type: ChatTypeGroup,
	}

	if info.ID != "chat-123" {
		t.Errorf("Expected ID='chat-123', got %q", info.ID)
	}

	if info.Type != ChatTypeGroup {
		t.Errorf("Expected Type='group', got %q", info.Type)
	}
}

func TestAdapterConfig(t *testing.T) {
	config := AdapterConfig{
		Platform:   PlatformFeishu,
		Name:       "feishu-bot",
		Enabled:    true,
		Token:      "test-token",
		AppID:      "app-123",
		AppSecret:  "secret-456",
		WebhookURL: "https://example.com/webhook",
		Extra: map[string]string{
			"key1": "value1",
		},
	}

	if config.Platform != PlatformFeishu {
		t.Errorf("Expected Platform='feishu', got %q", config.Platform)
	}

	if config.Extra["key1"] != "value1" {
		t.Errorf("Expected Extra['key1']='value1', got %q", config.Extra["key1"])
	}
}

func TestPlatformConstants(t *testing.T) {
	platforms := []Platform{
		PlatformFeishu,
		PlatformDingTalk,
		PlatformTelegram,
		PlatformDiscord,
		PlatformSlack,
		PlatformWeCom,
		PlatformWeChat,
		PlatformWebhook,
	}

	for _, p := range platforms {
		if p == "" {
			t.Error("Platform constant should not be empty")
		}
	}
}

func TestMessageTypeConstants(t *testing.T) {
	types := []MessageType{
		MessageTypeText,
		MessageTypeImage,
		MessageTypeVideo,
		MessageTypeAudio,
		MessageTypeVoice,
		MessageTypeDocument,
		MessageTypeSticker,
		MessageTypeCommand,
		MessageTypeFile,
	}

	for _, mt := range types {
		if mt == "" {
			t.Error("MessageType constant should not be empty")
		}
	}
}

func TestChatTypeConstants(t *testing.T) {
	types := []ChatType{
		ChatTypeDM,
		ChatTypeGroup,
		ChatTypeChannel,
		ChatTypeThread,
	}

	for _, ct := range types {
		if ct == "" {
			t.Error("ChatType constant should not be empty")
		}
	}
}
