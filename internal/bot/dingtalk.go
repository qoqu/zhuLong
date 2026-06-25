package bot

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// DingTalkAdapter implements the Adapter interface for DingTalk.
type DingTalkAdapter struct {
	*BaseAdapter
	appKey    string
	appSecret string
	webhook   string
	secret    string
	token     string
	client    *http.Client
	stopCh    chan struct{}
}

// NewDingTalkAdapter creates a new DingTalk adapter.
func NewDingTalkAdapter(config AdapterConfig) Adapter {
	return &DingTalkAdapter{
		BaseAdapter: NewBaseAdapter(config),
		appKey:      config.AppID,
		appSecret:   config.AppSecret,
		webhook:     config.WebhookURL,
		secret:      config.Extra["secret"],
		client:      &http.Client{Timeout: 30 * time.Second},
		stopCh:      make(chan struct{}),
	}
}

// Connect starts the DingTalk adapter.
func (d *DingTalkAdapter) Connect() error {
	if d.webhook == "" && (d.appKey == "" || d.appSecret == "") {
		return fmt.Errorf("dingtalk webhook or app_key/app_secret is required")
	}

	// 如果有 appKey/appSecret，获取 access token 并启动轮询
	if d.appKey != "" && d.appSecret != "" {
		token, err := d.getAccessToken()
		if err != nil {
			return fmt.Errorf("failed to get access token: %w", err)
		}
		d.token = token

		// 启动 token 自动刷新
		go d.refreshTokenLoop()

		// 启动消息轮询
		go d.pollMessages()
	}

	d.SetConnected(true)
	return nil
}

// Disconnect stops the DingTalk adapter.
func (d *DingTalkAdapter) Disconnect() error {
	if !d.IsConnected() {
		return nil
	}

	close(d.stopCh)
	d.SetConnected(false)
	return nil
}

// Send sends a text message to a DingTalk chat.
func (d *DingTalkAdapter) Send(chatID string, text string) (*SendResult, error) {
	// Use webhook if available
	if d.webhook != "" {
		return d.sendWebhook(text)
	}
	// Otherwise use API
	return d.sendAPI(chatID, text)
}

// sendWebhook sends a message via DingTalk webhook.
func (d *DingTalkAdapter) sendWebhook(text string) (*SendResult, error) {
	webhookURL := d.webhook

	// Add signature if secret is set
	if d.secret != "" {
		timestamp := time.Now().UnixMilli()
		sign, err := d.sign(timestamp)
		if err != nil {
			return nil, err
		}
		webhookURL = fmt.Sprintf("%s&timestamp=%d&sign=%s", webhookURL, timestamp, url.QueryEscape(sign))
	}

	payload := map[string]interface{}{
		"msgtype": "text",
		"text": map[string]string{
			"content": text,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	resp, err := d.client.Post(webhookURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result.ErrCode != 0 {
		return nil, fmt.Errorf("dingtalk error: %s", result.ErrMsg)
	}

	return &SendResult{}, nil
}

// sendAPI sends a message via DingTalk API.
func (d *DingTalkAdapter) sendAPI(chatID string, text string) (*SendResult, error) {
	// Get access token
	token, err := d.getAccessToken()
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("https://oapi.dingtalk.com/chat/send?access_token=%s", token)

	payload := map[string]interface{}{
		"chatid":  chatID,
		"msgtype": "text",
		"text": map[string]string{
			"content": text,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	resp, err := d.client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result.ErrCode != 0 {
		return nil, fmt.Errorf("dingtalk error: %s", result.ErrMsg)
	}

	return &SendResult{}, nil
}

// SendImage sends an image to a DingTalk chat.
// SendImage sends an image to a DingTalk chat.
func (d *DingTalkAdapter) SendImage(chatID string, imageURL string, caption string) (*SendResult, error) {
	if d.webhook != "" {
		return d.sendWebhookImage(imageURL)
	}
	return nil, fmt.Errorf("dingtalk image send via API not implemented")
}

// SendRichText sends rich text to a DingTalk chat.
func (d *DingTalkAdapter) SendRichText(chatID string, richText *RichTextContent) (*SendResult, error) {
	// Simplified: concatenate all text elements
	text := ""
	for _, elem := range richText.Elements {
		if elem.Type == "text" {
			text += elem.Text
		}
	}
	return d.Send(chatID, text)
}

// SendInteractiveCard sends an interactive card to a DingTalk chat.
func (d *DingTalkAdapter) SendInteractiveCard(chatID string, card *InteractiveCard) (*SendResult, error) {
	// Simplified: send as markdown
	text := ""
	if card.Title != "" {
		text += "## " + card.Title + "\n\n"
	}
	for _, elem := range card.Elements {
		if elem.Type == "text" {
			text += elem.Text + "\n\n"
		}
	}
	return d.Send(chatID, text)
}

// sendWebhookImage sends an image via DingTalk webhook.
func (d *DingTalkAdapter) sendWebhookImage(imageURL string) (*SendResult, error) {
	webhookURL := d.webhook

	if d.secret != "" {
		timestamp := time.Now().UnixMilli()
		sign, err := d.sign(timestamp)
		if err != nil {
			return nil, err
		}
		webhookURL = fmt.Sprintf("%s&timestamp=%d&sign=%s", webhookURL, timestamp, url.QueryEscape(sign))
	}

	payload := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"title": "图片",
			"text":  fmt.Sprintf("![图片](%s)", imageURL),
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	resp, err := d.client.Post(webhookURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result.ErrCode != 0 {
		return nil, fmt.Errorf("dingtalk error: %s", result.ErrMsg)
	}

	return &SendResult{}, nil
}

// GetChatInfo returns information about a DingTalk chat.
func (d *DingTalkAdapter) GetChatInfo(chatID string) (*ChatInfo, error) {
	token, err := d.getAccessToken()
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("https://oapi.dingtalk.com/chat/get?access_token=%s&chatid=%s", token, chatID)

	resp, err := d.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		ErrCode int    `json:"errcode"`
		ChatInfo struct {
			ChatID string `json:"chatid"`
			Name   string `json:"name"`
		} `json:"chatInfo"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result.ErrCode != 0 {
		return nil, fmt.Errorf("failed to get chat info")
	}

	return &ChatInfo{
		ID:   result.ChatInfo.ChatID,
		Name: result.ChatInfo.Name,
		Type: ChatTypeGroup,
	}, nil
}

// HandleWebhook handles an incoming webhook event from DingTalk.
func (d *DingTalkAdapter) HandleWebhook(payload []byte) error {
	var event dingtalkEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return err
	}

	// Handle message events
	if event.MsgType == "text" {
		msgEvent := &MessageEvent{
			Text:        event.Text.Content,
			MessageType: MessageTypeText,
			Source: SessionSource{
				Platform:  PlatformDingTalk,
				ChatID:    event.ConversationID,
				ChatType:  ChatTypeGroup,
				UserID:    event.SenderID,
				UserName:  event.SenderNick,
				MessageID: event.MsgID,
			},
			MessageID: event.MsgID,
			Timestamp: time.Unix(event.CreateAt/1000, 0),
		}

		d.HandleMessage(msgEvent)
	}

	return nil
}

// getAccessToken gets the DingTalk access token.
func (d *DingTalkAdapter) getAccessToken() (string, error) {
	url := "https://oapi.dingtalk.com/gettoken"

	payload := map[string]string{
		"appkey":    d.appKey,
		"appsecret": d.appSecret,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	resp, err := d.client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		ErrCode     int    `json:"errcode"`
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if result.ErrCode != 0 {
		return "", fmt.Errorf("failed to get access token")
	}

	return result.AccessToken, nil
}

// refreshTokenLoop refreshes the access token periodically
func (d *DingTalkAdapter) refreshTokenLoop() {
	for {
		select {
		case <-d.stopCh:
			return
		default:
		}

		time.Sleep(7000 * time.Second) // Token valid for ~7200s

		token, err := d.getAccessToken()
		if err != nil {
			continue
		}
		d.token = token
	}
}

// pollMessages polls for new messages from DingTalk (simplified)
func (d *DingTalkAdapter) pollMessages() {
	for {
		select {
		case <-d.stopCh:
			return
		default:
		}

		// DingTalk doesn't have a standard polling API for bot messages
		// Messages are received via webhook callback
		time.Sleep(5 * time.Second)
	}
}

// sign generates a DingTalk webhook signature.
func (d *DingTalkAdapter) sign(timestamp int64) (string, error) {
	stringToSign := fmt.Sprintf("%d\n%s", timestamp, d.secret)
	mac := hmac.New(sha256.New, []byte(d.secret))
	mac.Write([]byte(stringToSign))
	sign := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	return sign, nil
}

// DingTalk event types
type dingtalkEvent struct {
	MsgType        string `json:"msgtype"`
	Text           struct {
		Content string `json:"content"`
	} `json:"text"`
	MsgID          string `json:"msgId"`
	ConversationID string `json:"conversationId"`
	SenderID       string `json:"senderId"`
	SenderNick     string `json:"senderNick"`
	CreateAt       int64  `json:"createAt"`
}

func init() {
	DefaultRegistry.Register(&PlatformEntry{
		Platform: PlatformDingTalk,
		Label:    "DingTalk",
		Factory: func(config AdapterConfig) (Adapter, error) {
			return NewDingTalkAdapter(config), nil
		},
		Description:    "DingTalk Bot API (Stream Mode + Webhook)",
		RequiredFields: []string{},
	})
}
