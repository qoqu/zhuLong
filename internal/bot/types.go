// Package bot implements a multi-platform messaging gateway for Zhulong.
// It follows the Adapter Pattern inspired by Hermes, normalizing messages
// from different platforms (Feishu, DingTalk, Telegram, etc.) into a unified format.
package bot

import (
	"time"
)

// Platform represents a messaging platform.
type Platform string

const (
	PlatformFeishu   Platform = "feishu"
	PlatformDingTalk Platform = "dingtalk"
	PlatformTelegram Platform = "telegram"
	PlatformDiscord  Platform = "discord"
	PlatformSlack    Platform = "slack"
	PlatformWeCom    Platform = "wecom"
	PlatformWeChat   Platform = "wechat"
	PlatformWebhook  Platform = "webhook"
)

// MessageType represents the type of message content.
type MessageType string

const (
	MessageTypeText     MessageType = "text"
	MessageTypeImage    MessageType = "image"
	MessageTypeVideo    MessageType = "video"
	MessageTypeAudio    MessageType = "audio"
	MessageTypeVoice    MessageType = "voice"
	MessageTypeDocument MessageType = "document"
	MessageTypeSticker  MessageType = "sticker"
	MessageTypeCommand  MessageType = "command"
	MessageTypeFile     MessageType = "file"
)

// ChatType represents the type of chat.
type ChatType string

const (
	ChatTypeDM      ChatType = "dm"
	ChatTypeGroup   ChatType = "group"
	ChatTypeChannel ChatType = "channel"
	ChatTypeThread  ChatType = "thread"
)

// SessionSource identifies where a message comes from.
type SessionSource struct {
	Platform  Platform `json:"platform"`
	ChatID    string   `json:"chatId"`
	ChatName  string   `json:"chatName,omitempty"`
	ChatType  ChatType `json:"chatType"`
	UserID    string   `json:"userId,omitempty"`
	UserName  string   `json:"userName,omitempty"`
	ThreadID  string   `json:"threadId,omitempty"`
	GuildID   string   `json:"guildId,omitempty"`
	MessageID string   `json:"messageId,omitempty"`
}

// MessageEvent is the unified message format after normalization.
// All platform adapters convert their native messages to this format.
type MessageEvent struct {
	Text              string            `json:"text"`
	MessageType       MessageType       `json:"messageType"`
	Source            SessionSource     `json:"source"`
	RawMessage        interface{}       `json:"rawMessage,omitempty"`
	MessageID         string            `json:"messageId,omitempty"`
	MediaURLs         []string          `json:"mediaUrls,omitempty"`
	MediaTypes        []string          `json:"mediaTypes,omitempty"`
	ReplyToMessageID  string            `json:"replyToMessageId,omitempty"`
	ReplyToText       string            `json:"replyToText,omitempty"`
	ChannelPrompt     string            `json:"channelPrompt,omitempty"`
	ChannelContext    string            `json:"channelContext,omitempty"`
	Timestamp         time.Time         `json:"timestamp"`
	Extra             map[string]string `json:"extra,omitempty"`
	// Rich text support
	RichText          *RichTextContent  `json:"richText,omitempty"`
	// Interactive cards
	InteractiveCard   *InteractiveCard  `json:"interactiveCard,omitempty"`
}

// RichTextContent represents rich text content.
type RichTextContent struct {
	Elements []RichTextElement `json:"elements"`
}

// RichTextElement represents a single rich text element.
type RichTextElement struct {
	Type     string `json:"type"` // text, mention, emoji, link, image, code
	Text     string `json:"text,omitempty"`
	MentionID string `json:"mentionId,omitempty"`
	EmojiID  string `json:"emojiId,omitempty"`
	URL      string `json:"url,omitempty"`
	Alt      string `json:"alt,omitempty"`
	Language string `json:"language,omitempty"`
}

// InteractiveCard represents an interactive card message.
type InteractiveCard struct {
	Title    string            `json:"title,omitempty"`
	Elements []CardElement     `json:"elements"`
	Actions  []CardAction      `json:"actions,omitempty"`
	Styles   map[string]string `json:"styles,omitempty"`
}

// CardElement represents an element in an interactive card.
type CardElement struct {
	Type     string `json:"type"` // text, image, divider, action
	Text     string `json:"text,omitempty"`
	URL      string `json:"url,omitempty"`
	Elements []CardElement `json:"elements,omitempty"`
}

// CardAction represents an action button in an interactive card.
type CardAction struct {
	Type  string `json:"type"` // button, select
	Text  string `json:"text,omitempty"`
	Value string `json:"value,omitempty"`
	Style string `json:"style,omitempty"` // primary, danger, default
}

// SendResult is the result of sending a message.
type SendResult struct {
	MessageID string `json:"messageId,omitempty"`
	Error     error  `json:"-"`
}

// ChatInfo contains information about a chat.
type ChatInfo struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Type  ChatType `json:"type"`
}

// AdapterConfig contains configuration for a platform adapter.
type AdapterConfig struct {
	Platform    Platform          `json:"platform"`
	Name        string            `json:"name"`
	Enabled     bool              `json:"enabled"`
	Token       string            `json:"token,omitempty"`
	AppID       string            `json:"appId,omitempty"`
	AppSecret   string            `json:"appSecret,omitempty"`
	WebhookURL  string            `json:"webhookUrl,omitempty"`
	Endpoint    string            `json:"endpoint,omitempty"`
	Extra       map[string]string `json:"extra,omitempty"`
}

// MessageHandler is called when a message is received from a platform.
type MessageHandler func(event *MessageEvent)

// Adapter is the interface that all platform adapters must implement.
type Adapter interface {
	// Platform returns the platform type.
	Platform() Platform

	// Name returns the adapter name.
	Name() string

	// Connect starts the adapter and begins listening for messages.
	Connect() error

	// Disconnect stops the adapter.
	Disconnect() error

	// IsConnected returns whether the adapter is connected.
	IsConnected() bool

	// Send sends a text message to a chat.
	Send(chatID string, text string) (*SendResult, error)

	// SendImage sends an image to a chat.
	SendImage(chatID string, imageURL string, caption string) (*SendResult, error)

	// SendRichText sends rich text to a chat.
	SendRichText(chatID string, richText *RichTextContent) (*SendResult, error)

	// SendInteractiveCard sends an interactive card to a chat.
	SendInteractiveCard(chatID string, card *InteractiveCard) (*SendResult, error)

	// GetChatInfo returns information about a chat.
	GetChatInfo(chatID string) (*ChatInfo, error)

	// SetMessageHandler sets the handler for incoming messages.
	SetMessageHandler(handler MessageHandler)
}
