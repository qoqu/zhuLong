package bot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// TelegramAdapter implements the Adapter interface for Telegram.
type TelegramAdapter struct {
	*BaseAdapter
	botToken string
	client   *http.Client
	offset   int
	stopCh   chan struct{}
}

// NewTelegramAdapter creates a new Telegram adapter.
func NewTelegramAdapter(config AdapterConfig) Adapter {
	return &TelegramAdapter{
		BaseAdapter: NewBaseAdapter(config),
		botToken:    config.Token,
		client:      &http.Client{Timeout: 30 * time.Second},
		stopCh:      make(chan struct{}),
	}
}

// Connect starts polling for updates.
func (t *TelegramAdapter) Connect() error {
	if t.botToken == "" {
		return fmt.Errorf("telegram bot token is required")
	}

	// Verify bot token
	if err := t.verifyToken(); err != nil {
		return fmt.Errorf("invalid bot token: %w", err)
	}

	t.SetConnected(true)

	// Start polling
	go t.pollUpdates()

	return nil
}

// Disconnect stops polling.
func (t *TelegramAdapter) Disconnect() error {
	if !t.IsConnected() {
		return nil
	}

	close(t.stopCh)
	t.SetConnected(false)
	return nil
}

// Send sends a text message to a Telegram chat.
func (t *TelegramAdapter) Send(chatID string, text string) (*SendResult, error) {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.botToken)

	payload := map[string]interface{}{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": "Markdown",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	resp, err := t.client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		OK     bool `json:"ok"`
		Result struct {
			MessageID int `json:"message_id"`
		} `json:"result"`
		Description string `json:"description"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if !result.OK {
		return nil, fmt.Errorf("telegram error: %s", result.Description)
	}

	return &SendResult{
		MessageID: fmt.Sprintf("%d", result.Result.MessageID),
	}, nil
}

// SendImage sends an image to a Telegram chat.
func (t *TelegramAdapter) SendImage(chatID string, imageURL string, caption string) (*SendResult, error) {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendPhoto", t.botToken)

	payload := map[string]interface{}{
		"chat_id": chatID,
		"photo":   imageURL,
	}
	if caption != "" {
		payload["caption"] = caption
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	resp, err := t.client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		OK     bool `json:"ok"`
		Result struct {
			MessageID int `json:"message_id"`
		} `json:"result"`
		Description string `json:"description"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if !result.OK {
		return nil, fmt.Errorf("telegram error: %s", result.Description)
	}

	return &SendResult{
		MessageID: fmt.Sprintf("%d", result.Result.MessageID),
	}, nil
}

// GetChatInfo returns information about a Telegram chat.
func (t *TelegramAdapter) GetChatInfo(chatID string) (*ChatInfo, error) {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/getChat?chat_id=%s", t.botToken, chatID)

	resp, err := t.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		OK     bool `json:"ok"`
		Result struct {
			ID    int64  `json:"id"`
			Type  string `json:"type"`
			Title string `json:"title"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if !result.OK {
		return nil, fmt.Errorf("failed to get chat info")
	}

	chatType := ChatTypeDM
	if result.Result.Type == "group" || result.Result.Type == "supergroup" {
		chatType = ChatTypeGroup
	} else if result.Result.Type == "channel" {
		chatType = ChatTypeChannel
	}

	return &ChatInfo{
		ID:   fmt.Sprintf("%d", result.Result.ID),
		Name: result.Result.Title,
		Type: chatType,
	}, nil
}

// SendRichText sends rich text to a Telegram chat.
func (t *TelegramAdapter) SendRichText(chatID string, richText *RichTextContent) (*SendResult, error) {
	// Build markdown from rich text elements
	text := ""
	for _, elem := range richText.Elements {
		if elem.Type == "text" {
			text += elem.Text
		} else if elem.Type == "mention" {
			text += fmt.Sprintf("[%s](tg://user?id=%s)", elem.Text, elem.MentionID)
		} else if elem.Type == "link" {
			text += fmt.Sprintf("[%s](%s)", elem.Text, elem.URL)
		} else if elem.Type == "code" {
			text += fmt.Sprintf("`%s`", elem.Text)
		}
	}

	return t.Send(chatID, text)
}

// SendInteractiveCard sends an interactive card (inline keyboard) to a Telegram chat.
func (t *TelegramAdapter) SendInteractiveCard(chatID string, card *InteractiveCard) (*SendResult, error) {
	// Build inline keyboard
	keyboard := make([][]map[string]interface{}, 0)
	for _, action := range card.Actions {
		row := []map[string]interface{}{
			{
				"text": action.Text,
				"url":  action.Value,
			},
		}
		keyboard = append(keyboard, row)
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.botToken)

	payload := map[string]interface{}{
		"chat_id":    chatID,
		"text":       card.Title,
		"reply_markup": map[string]interface{}{
			"inline_keyboard": keyboard,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	resp, err := t.client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		OK     bool `json:"ok"`
		Result struct {
			MessageID int `json:"message_id"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if !result.OK {
		return nil, fmt.Errorf("failed to send interactive card")
	}

	return &SendResult{
		MessageID: fmt.Sprintf("%d", result.Result.MessageID),
	}, nil
}

// verifyToken verifies the bot token by calling getMe.
func (t *TelegramAdapter) verifyToken() error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/getMe", t.botToken)

	resp, err := t.client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result struct {
		OK bool `json:"ok"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	if !result.OK {
		return fmt.Errorf("invalid bot token")
	}

	return nil
}

// pollUpdates polls for new updates from Telegram.
func (t *TelegramAdapter) pollUpdates() {
	for {
		select {
		case <-t.stopCh:
			return
		default:
		}

		updates, err := t.getUpdates()
		if err != nil {
			time.Sleep(5 * time.Second)
			continue
		}

		for _, update := range updates {
			t.processUpdate(update)
		}
	}
}

// getUpdates gets new updates from Telegram.
func (t *TelegramAdapter) getUpdates() ([]telegramUpdate, error) {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates?offset=%d&timeout=30", t.botToken, t.offset)

	resp, err := t.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		OK     bool             `json:"ok"`
		Result []telegramUpdate `json:"result"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	if !result.OK {
		return nil, fmt.Errorf("failed to get updates")
	}

	// Update offset
	for _, update := range result.Result {
		if update.UpdateID >= t.offset {
			t.offset = update.UpdateID + 1
		}
	}

	return result.Result, nil
}

// processUpdate processes a single Telegram update.
func (t *TelegramAdapter) processUpdate(update telegramUpdate) {
	if update.Message == nil {
		return
	}

	msg := update.Message

	// Determine message type
	msgType := MessageTypeText
	if msg.Photo != nil {
		msgType = MessageTypeImage
	} else if msg.Document != nil {
		msgType = MessageTypeDocument
	} else if msg.Voice != nil {
		msgType = MessageTypeVoice
	} else if msg.Video != nil {
		msgType = MessageTypeVideo
	}

	// Get text
	text := msg.Text
	if text == "" && msg.Caption != "" {
		text = msg.Caption
	}

	// Determine chat type
	chatType := ChatTypeDM
	if msg.Chat.Type == "group" || msg.Chat.Type == "supergroup" {
		chatType = ChatTypeGroup
	} else if msg.Chat.Type == "channel" {
		chatType = ChatTypeChannel
	}

	// Build message event
	event := &MessageEvent{
		Text:        text,
		MessageType: msgType,
		Source: SessionSource{
			Platform:  PlatformTelegram,
			ChatID:    fmt.Sprintf("%d", msg.Chat.ID),
			ChatName:  msg.Chat.Title,
			ChatType:  chatType,
			UserID:    fmt.Sprintf("%d", msg.From.ID),
			UserName:  msg.From.FirstName + " " + msg.From.LastName,
			MessageID: fmt.Sprintf("%d", msg.MessageID),
		},
		MessageID: fmt.Sprintf("%d", msg.MessageID),
		Timestamp: time.Unix(int64(msg.Date), 0),
	}

	// Handle reply
	if msg.ReplyToMessage != nil {
		event.ReplyToMessageID = fmt.Sprintf("%d", msg.ReplyToMessage.MessageID)
		event.ReplyToText = msg.ReplyToMessage.Text
	}

	t.HandleMessage(event)
}

// Telegram API types
type telegramUpdate struct {
	UpdateID int               `json:"update_id"`
	Message  *telegramMessage  `json:"message"`
}

type telegramMessage struct {
	MessageID int              `json:"message_id"`
	From      telegramUser     `json:"from"`
	Chat      telegramChat     `json:"chat"`
	Date      int              `json:"date"`
	Text      string           `json:"text"`
	Caption   string           `json:"caption"`
	Photo     []telegramPhoto  `json:"photo"`
	Document  *telegramDocument `json:"document"`
	Voice     *telegramVoice   `json:"voice"`
	Video     *telegramVideo   `json:"video"`
	ReplyToMessage *telegramMessage `json:"reply_to_message"`
}

type telegramUser struct {
	ID        int    `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type telegramChat struct {
	ID    int64  `json:"id"`
	Type  string `json:"type"`
	Title string `json:"title"`
}

type telegramPhoto struct {
	FileID string `json:"file_id"`
}

type telegramDocument struct {
	FileID   string `json:"file_id"`
	FileName string `json:"file_name"`
}

type telegramVoice struct {
	FileID string `json:"file_id"`
}

type telegramVideo struct {
	FileID string `json:"file_id"`
}

func init() {
	DefaultRegistry.Register(&PlatformEntry{
		Platform: PlatformTelegram,
		Label:    "Telegram",
		Factory: func(config AdapterConfig) (Adapter, error) {
			return NewTelegramAdapter(config), nil
		},
		Description:    "Telegram Bot API",
		RequiredFields: []string{"token"},
	})
}
