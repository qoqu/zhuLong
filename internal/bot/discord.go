package bot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// DiscordAdapter implements the Adapter interface for Discord.
type DiscordAdapter struct {
	*BaseAdapter
	botToken  string
	guildID   string
	client    *http.Client
	stopCh    chan struct{}
	mu        sync.Mutex
}

// NewDiscordAdapter creates a new Discord adapter.
func NewDiscordAdapter(config AdapterConfig) Adapter {
	return &DiscordAdapter{
		BaseAdapter: NewBaseAdapter(config),
		botToken:    config.Token,
		guildID:     config.Extra["guildId"],
		client:      &http.Client{Timeout: 30 * time.Second},
		stopCh:      make(chan struct{}),
	}
}

// Connect starts the Discord adapter.
func (d *DiscordAdapter) Connect() error {
	if d.botToken == "" {
		return fmt.Errorf("discord bot token is required")
	}

	// Verify bot token
	if err := d.verifyToken(); err != nil {
		return fmt.Errorf("invalid bot token: %w", err)
	}

	// 启动消息轮询
	go d.pollMessages()

	d.SetConnected(true)
	return nil
}

// Disconnect stops the Discord adapter.
func (d *DiscordAdapter) Disconnect() error {
	if !d.IsConnected() {
		return nil
	}

	close(d.stopCh)
	d.SetConnected(false)
	return nil
}

// Send sends a text message to a Discord channel.
func (d *DiscordAdapter) Send(chatID string, text string) (*SendResult, error) {
	url := fmt.Sprintf("https://discord.com/api/v10/channels/%s/messages", chatID)

	payload := map[string]interface{}{
		"content": text,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bot "+d.botToken)

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		ID string `json:"id"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &SendResult{
		MessageID: result.ID,
	}, nil
}

// SendImage sends an image to a Discord channel.
func (d *DiscordAdapter) SendImage(chatID string, imageURL string, caption string) (*SendResult, error) {
	url := fmt.Sprintf("https://discord.com/api/v10/channels/%s/messages", chatID)

	payload := map[string]interface{}{
		"content": caption,
		"embeds": []map[string]interface{}{
			{
				"image": map[string]string{
					"url": imageURL,
				},
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bot "+d.botToken)

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		ID string `json:"id"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &SendResult{
		MessageID: result.ID,
	}, nil
}

// GetChatInfo returns information about a Discord channel.
func (d *DiscordAdapter) GetChatInfo(chatID string) (*ChatInfo, error) {
	url := fmt.Sprintf("https://discord.com/api/v10/channels/%s", chatID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bot "+d.botToken)

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		GuildID string `json:"guild_id"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &ChatInfo{
		ID:   result.ID,
		Name: result.Name,
		Type: ChatTypeChannel,
	}, nil
}

// SendRichText sends rich text to a Discord channel.
func (d *DiscordAdapter) SendRichText(chatID string, richText *RichTextContent) (*SendResult, error) {
	// Discord uses markdown, so concatenate text elements
	text := ""
	for _, elem := range richText.Elements {
		if elem.Type == "text" {
			text += elem.Text
		} else if elem.Type == "mention" {
			text += fmt.Sprintf("<@%s>", elem.MentionID)
		} else if elem.Type == "link" {
			text += elem.URL
		} else if elem.Type == "code" {
			text += fmt.Sprintf("`%s`", elem.Text)
		}
	}

	return d.Send(chatID, text)
}

// SendInteractiveCard sends an interactive card (embed) to a Discord channel.
func (d *DiscordAdapter) SendInteractiveCard(chatID string, card *InteractiveCard) (*SendResult, error) {
	url := fmt.Sprintf("https://discord.com/api/v10/channels/%s/messages", chatID)

	// Build embed
	embed := map[string]interface{}{
		"title": card.Title,
	}

	// Add description from elements
	description := ""
	for _, elem := range card.Elements {
		if elem.Type == "text" {
			description += elem.Text + "\n"
		}
	}
	if description != "" {
		embed["description"] = description
	}

	// Add fields from actions
	if len(card.Actions) > 0 {
		fields := make([]map[string]interface{}, 0)
		for _, action := range card.Actions {
			fields = append(fields, map[string]interface{}{
				"name":  action.Text,
				"value": action.Value,
				"inline": true,
			})
		}
		embed["fields"] = fields
	}

	payload := map[string]interface{}{
		"embeds": []map[string]interface{}{embed},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bot "+d.botToken)

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		ID string `json:"id"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &SendResult{
		MessageID: result.ID,
	}, nil
}

// verifyToken verifies the bot token by calling the users/@me endpoint.
func (d *DiscordAdapter) verifyToken() error {
	url := "https://discord.com/api/v10/users/@me"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bot "+d.botToken)

	resp, err := d.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result struct {
		ID string `json:"id"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	if result.ID == "" {
		return fmt.Errorf("invalid bot token")
	}

	return nil
}

// HandleWebhook handles an incoming webhook event from Discord.
func (d *DiscordAdapter) HandleWebhook(payload []byte) error {
	var event discordEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return err
	}

	// Handle message create events
	if event.Type == 0 { // MESSAGE_CREATE
		msg := event.Data

		// Determine message type
		msgType := MessageTypeText
		if len(msg.Attachments) > 0 {
			att := msg.Attachments[0]
			if att.ContentType != "" {
				switch {
				case len(att.ContentType) > 5 && att.ContentType[:5] == "image":
					msgType = MessageTypeImage
				case len(att.ContentType) > 5 && att.ContentType[:5] == "video":
					msgType = MessageTypeVideo
				case len(att.ContentType) > 5 && att.ContentType[:5] == "audio":
					msgType = MessageTypeAudio
				default:
					msgType = MessageTypeFile
				}
			}
		}

		// Build message event
		msgEvent := &MessageEvent{
			Text:        msg.Content,
			MessageType: msgType,
			Source: SessionSource{
				Platform:  PlatformDiscord,
				ChatID:    msg.ChannelID,
				ChatType:  ChatTypeChannel,
				UserID:    msg.Author.ID,
				UserName:  msg.Author.Username,
				GuildID:   msg.GuildID,
				MessageID: msg.ID,
			},
			MessageID: msg.ID,
			Timestamp: time.Now(),
		}

		d.HandleMessage(msgEvent)
	}

	return nil
}

// Discord event types
type discordEvent struct {
	Type int `json:"t"`
	Data struct {
		ID        string `json:"id"`
		Content   string `json:"content"`
		ChannelID string `json:"channel_id"`
		GuildID   string `json:"guild_id"`
		Author    struct {
			ID       string `json:"id"`
			Username string `json:"username"`
		} `json:"author"`
		Attachments []struct {
			ID          string `json:"id"`
			URL         string `json:"url"`
			ContentType string `json:"content_type"`
		} `json:"attachments"`
	} `json:"d"`
}

// Placeholder for websocket
type websocket struct{}

func init() {
	DefaultRegistry.Register(&PlatformEntry{
		Platform: PlatformDiscord,
		Label:    "Discord",
		Factory: func(config AdapterConfig) (Adapter, error) {
			return NewDiscordAdapter(config), nil
		},
		Description:    "Discord Bot API (Gateway + HTTP)",
		RequiredFields: []string{"token"},
	})
}

// pollMessages polls for new messages from Discord (simplified)
func (d *DiscordAdapter) pollMessages() {
	for {
		select {
		case <-d.stopCh:
			return
		default:
		}

		// Discord 推荐使用 Gateway WebSocket 连接
		// 这里使用简化的 HTTP 轮询作为临时方案
		// 生产环境应使用 Discord Gateway
		time.Sleep(5 * time.Second)
	}
}
