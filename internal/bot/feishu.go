package bot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"sync"
	"time"
)

// FeishuAdapter implements the Adapter interface for Feishu/Lark.
type FeishuAdapter struct {
	*BaseAdapter
	appID     string
	appSecret string
	token     string
	tokenExp  time.Time
	client    *http.Client
	server    *http.Server
	stopCh    chan struct{}
	mu        sync.Mutex
}

// NewFeishuAdapter creates a new Feishu adapter.
func NewFeishuAdapter(config AdapterConfig) Adapter {
	return &FeishuAdapter{
		BaseAdapter: NewBaseAdapter(config),
		appID:       config.AppID,
		appSecret:   config.AppSecret,
		client:      &http.Client{Timeout: 30 * time.Second},
		stopCh:      make(chan struct{}),
	}
}

// Connect starts the Feishu adapter.
func (f *FeishuAdapter) Connect() error {
	if f.appID == "" || f.appSecret == "" {
		return fmt.Errorf("feishu app_id and app_secret are required")
	}

	// Get tenant access token
	if err := f.refreshToken(); err != nil {
		return fmt.Errorf("failed to get access token: %w", err)
	}

	f.SetConnected(true)

	// Start token refresh goroutine
	go f.tokenRefreshLoop()

	return nil
}

// Disconnect stops the Feishu adapter.
func (f *FeishuAdapter) Disconnect() error {
	if !f.IsConnected() {
		return nil
	}

	close(f.stopCh)
	f.SetConnected(false)
	return nil
}

// Send sends a text message to a Feishu chat.
func (f *FeishuAdapter) Send(chatID string, text string) (*SendResult, error) {
	url := "https://open.feishu.cn/open-apis/im/v1/messages"

	payload := map[string]interface{}{
		"receive_id": chatID,
		"msg_type":   "text",
		"content":    fmt.Sprintf(`{"text":"%s"}`, escapeJSON(text)),
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
	req.Header.Set("Authorization", "Bearer "+f.token)

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			MessageID string `json:"message_id"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result.Code != 0 {
		return nil, fmt.Errorf("feishu error: %s", result.Msg)
	}

	return &SendResult{
		MessageID: result.Data.MessageID,
	}, nil
}

// SendImage sends an image to a Feishu chat.
func (f *FeishuAdapter) SendImage(chatID string, imageURL string, caption string) (*SendResult, error) {
	// First, upload image to get image_key
	imageKey, err := f.uploadImage(imageURL)
	if err != nil {
		return nil, fmt.Errorf("failed to upload image: %w", err)
	}

	url := "https://open.feishu.cn/open-apis/im/v1/messages"

	payload := map[string]interface{}{
		"receive_id": chatID,
		"msg_type":   "image",
		"content":    fmt.Sprintf(`{"image_key":"%s"}`, imageKey),
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
	req.Header.Set("Authorization", "Bearer "+f.token)

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			MessageID string `json:"message_id"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result.Code != 0 {
		return nil, fmt.Errorf("feishu error: %s", result.Msg)
	}

	return &SendResult{
		MessageID: result.Data.MessageID,
	}, nil
}

// GetChatInfo returns information about a Feishu chat.
func (f *FeishuAdapter) GetChatInfo(chatID string) (*ChatInfo, error) {
	url := fmt.Sprintf("https://open.feishu.cn/open-apis/im/v1/chats/%s", chatID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+f.token)

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Code int    `json:"code"`
		Data struct {
			ChatID   string `json:"chat_id"`
			Name     string `json:"name"`
			ChatType string `json:"chat_type"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result.Code != 0 {
		return nil, fmt.Errorf("failed to get chat info")
	}

	chatType := ChatTypeGroup
	if result.Data.ChatType == "p2p" {
		chatType = ChatTypeDM
	}

	return &ChatInfo{
		ID:   result.Data.ChatID,
		Name: result.Data.Name,
		Type: chatType,
	}, nil
}

// SendRichText sends rich text to a Feishu chat.
func (f *FeishuAdapter) SendRichText(chatID string, richText *RichTextContent) (*SendResult, error) {
	// Build Feishu rich text (post) message
	content := ""
	for _, elem := range richText.Elements {
		if elem.Type == "text" {
			content += elem.Text
		} else if elem.Type == "mention" {
			content += fmt.Sprintf("<at user_id=\"%s\">@%s</at>", elem.MentionID, elem.Text)
		}
	}

	url := "https://open.feishu.cn/open-apis/im/v1/messages"

	payload := map[string]interface{}{
		"receive_id": chatID,
		"msg_type":   "text",
		"content":    fmt.Sprintf(`{"text":"%s"}`, escapeJSON(content)),
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
	req.Header.Set("Authorization", "Bearer "+f.token)

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			MessageID string `json:"message_id"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result.Code != 0 {
		return nil, fmt.Errorf("feishu error: %s", result.Msg)
	}

	return &SendResult{
		MessageID: result.Data.MessageID,
	}, nil
}

// SendInteractiveCard sends an interactive card to a Feishu chat.
func (f *FeishuAdapter) SendInteractiveCard(chatID string, card *InteractiveCard) (*SendResult, error) {
	// Build Feishu interactive card JSON
	cardJSON := map[string]interface{}{
		"config": map[string]interface{}{
			"wide_screen_mode": true,
		},
	}

	if card.Title != "" {
		cardJSON["header"] = map[string]interface{}{
			"title": map[string]interface{}{
				"tag":     "plain_text",
				"content": card.Title,
			},
		}
	}

	// Add elements
	elements := make([]map[string]interface{}, 0)
	for _, elem := range card.Elements {
		if elem.Type == "text" {
			elements = append(elements, map[string]interface{}{
				"tag": "div",
				"text": map[string]interface{}{
					"tag":     "plain_text",
					"content": elem.Text,
				},
			})
		}
	}

	// Add actions
	if len(card.Actions) > 0 {
		actionElements := make([]map[string]interface{}, 0)
		for _, action := range card.Actions {
			actionElements = append(actionElements, map[string]interface{}{
				"tag": "button",
				"text": map[string]interface{}{
					"tag":     "plain_text",
					"content": action.Text,
				},
				"type": action.Style,
				"url":  action.Value,
			})
		}
		elements = append(elements, map[string]interface{}{
			"tag":     "action",
			"actions": actionElements,
		})
	}

	cardJSON["elements"] = elements

	cardData, _ := json.Marshal(cardJSON)

	url := "https://open.feishu.cn/open-apis/im/v1/messages"

	payload := map[string]interface{}{
		"receive_id": chatID,
		"msg_type":   "interactive",
		"content":    string(cardData),
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
	req.Header.Set("Authorization", "Bearer "+f.token)

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			MessageID string `json:"message_id"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result.Code != 0 {
		return nil, fmt.Errorf("feishu error: %s", result.Msg)
	}

	return &SendResult{
		MessageID: result.Data.MessageID,
	}, nil
}

// HandleWebhook handles an incoming webhook event from Feishu.
func (f *FeishuAdapter) HandleWebhook(payload []byte) error {
	var event feishuEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return err
	}

	// Handle URL verification challenge
	if event.Challenge != "" {
		return nil // Response handled by HTTP handler
	}

	// Handle message events
	if event.Header.EventType == "im.message.receive_v1" {
		return f.handleMessageEvent(event)
	}

	return nil
}

// handleMessageEvent handles a message receive event.
func (f *FeishuAdapter) handleMessageEvent(event feishuEvent) error {
	msg := event.Event.Message

	// Determine message type
	msgType := MessageTypeText
	switch msg.MessageType {
	case "image":
		msgType = MessageTypeImage
	case "audio":
		msgType = MessageTypeAudio
	case "video":
		msgType = MessageTypeVideo
	case "file":
		msgType = MessageTypeFile
	}

	// Parse text content
	text := ""
	if msg.MessageType == "text" {
		var content struct {
			Text string `json:"text"`
		}
		if err := json.Unmarshal([]byte(msg.Content), &content); err == nil {
			text = content.Text
		}
	}

	// Determine chat type
	chatType := ChatTypeGroup
	if msg.ChatType == "p2p" {
		chatType = ChatTypeDM
	}

	// Build message event
	msgEvent := &MessageEvent{
		Text:        text,
		MessageType: msgType,
		Source: SessionSource{
			Platform:  PlatformFeishu,
			ChatID:    msg.ChatID,
			ChatType:  chatType,
			UserID:    event.Event.Sender.SenderID.OpenID,
			MessageID: msg.MessageID,
		},
		MessageID: msg.MessageID,
		Timestamp: time.Now(),
	}

	f.HandleMessage(msgEvent)
	return nil
}

// refreshToken refreshes the tenant access token.
func (f *FeishuAdapter) refreshToken() error {
	url := "https://open.feishu.cn/open-apis/auth/v3/tenant_access_token/internal"

	payload := map[string]string{
		"app_id":     f.appID,
		"app_secret": f.appSecret,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := f.client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result struct {
		Code              int    `json:"code"`
		Msg               string `json:"msg"`
		TenantAccessToken string `json:"tenant_access_token"`
		Expire            int    `json:"expire"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	if result.Code != 0 {
		return fmt.Errorf("feishu auth error: %s", result.Msg)
	}

	f.mu.Lock()
	f.token = result.TenantAccessToken
	f.tokenExp = time.Now().Add(time.Duration(result.Expire-60) * time.Second)
	f.mu.Unlock()

	return nil
}

// tokenRefreshLoop refreshes the token before it expires.
func (f *FeishuAdapter) tokenRefreshLoop() {
	for {
		select {
		case <-f.stopCh:
			return
		default:
		}

		f.mu.Lock()
		exp := f.tokenExp
		f.mu.Unlock()

		// Refresh 1 minute before expiry
		refreshAt := exp.Add(-1 * time.Minute)
		if time.Now().Before(refreshAt) {
			time.Sleep(time.Until(refreshAt))
		}

		if err := f.refreshToken(); err != nil {
			time.Sleep(30 * time.Second)
		}
	}
}

// uploadImage uploads an image to Feishu and returns the image_key.
func (f *FeishuAdapter) uploadImage(imageURL string) (string, error) {
	// Download image
	resp, err := f.client.Get(imageURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	imageData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Upload to Feishu
	url := "https://open.feishu.cn/open-apis/im/v1/images"

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("image", "image.png")
	if err != nil {
		return "", err
	}
	if _, err := part.Write(imageData); err != nil {
		return "", err
	}
	writer.WriteField("image_type", "message")
	if err := writer.Close(); err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", url, &buf)
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+f.token)

	uploadResp, err := f.client.Do(req)
	if err != nil {
		return "", err
	}
	defer uploadResp.Body.Close()

	var result struct {
		Code int    `json:"code"`
		Data struct {
			ImageKey string `json:"image_key"`
		} `json:"data"`
	}

	if err := json.NewDecoder(uploadResp.Body).Decode(&result); err != nil {
		return "", err
	}

	if result.Code != 0 {
		return "", fmt.Errorf("failed to upload image")
	}

	return result.Data.ImageKey, nil
}

// escapeJSON escapes special characters in a JSON string.
func escapeJSON(s string) string {
	b, _ := json.Marshal(s)
	// Remove surrounding quotes
	return string(b[1 : len(b)-1])
}

// Feishu event types
type feishuEvent struct {
	Challenge string `json:"challenge"`
	Token     string `json:"token"`
	Header    struct {
		EventType string `json:"event_type"`
	} `json:"header"`
	Event struct {
		Message struct {
			ChatID      string `json:"chat_id"`
			ChatType    string `json:"chat_type"`
			MessageID   string `json:"message_id"`
			MessageType string `json:"message_type"`
			Content     string `json:"content"`
		} `json:"message"`
		Sender struct {
			SenderID struct {
				OpenID string `json:"open_id"`
			} `json:"sender_id"`
		} `json:"sender"`
	} `json:"event"`
}

func init() {
	DefaultRegistry.Register(&PlatformEntry{
		Platform: PlatformFeishu,
		Label:    "Feishu / Lark",
		Factory: func(config AdapterConfig) (Adapter, error) {
			return NewFeishuAdapter(config), nil
		},
		Description:    "Feishu/Lark Bot API (WebSocket + Webhook)",
		RequiredFields: []string{"appId", "appSecret"},
	})
}
