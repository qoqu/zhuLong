package bot

import (
	"bytes"
	"crypto/sha1"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

// WeComAdapter implements the Adapter interface for WeCom (企业微信).
type WeComAdapter struct {
	*BaseAdapter
	corpID     string
	agentID    string
	secret     string
	token      string
	encodingAESKey string
	client     *http.Client
	stopCh     chan struct{}
}

// NewWeComAdapter creates a a new WeCom adapter.
func NewWeComAdapter(config AdapterConfig) Adapter {
	return &WeComAdapter{
		BaseAdapter:    NewBaseAdapter(config),
		corpID:         config.AppID,
		secret:         config.AppSecret,
		token:          config.Extra["token"],
		encodingAESKey: config.Extra["encodingAESKey"],
		client:         &http.Client{Timeout: 30 * time.Second},
		stopCh:         make(chan struct{}),
	}
}

// Connect starts the WeCom adapter.
func (w *WeComAdapter) Connect() error {
	if w.corpID == "" || w.secret == "" {
		return fmt.Errorf("wecom corp_id and secret are required")
	}

	w.SetConnected(true)
	return nil
}

// Disconnect stops the WeCom adapter.
func (w *WeComAdapter) Disconnect() error {
	if !w.IsConnected() {
		return nil
	}

	close(w.stopCh)
	w.SetConnected(false)
	return nil
}

// Send sends a text message to a WeCom chat.
func (w *WeComAdapter) Send(chatID string, text string) (*SendResult, error) {
	// Get access token
	token, err := w.getAccessToken()
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/message/send?access_token=%s", token)

	payload := map[string]interface{}{
		"touser":  chatID,
		"msgtype": "text",
		"agentid": w.agentID,
		"text": map[string]string{
			"content": text,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	resp, err := w.client.Post(url, "application/json", bytes.NewReader(body))
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
		return nil, fmt.Errorf("wecom error: %s", result.ErrMsg)
	}

	return &SendResult{}, nil
}

// SendImage sends an image to a WeCom chat.
func (w *WeComAdapter) SendImage(chatID string, imageURL string, caption string) (*SendResult, error) {
	// Upload image first
	mediaID, err := w.uploadMedia(imageURL)
	if err != nil {
		return nil, err
	}

	// Get access token
	token, err := w.getAccessToken()
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/message/send?access_token=%s", token)

	payload := map[string]interface{}{
		"touser":  chatID,
		"msgtype": "image",
		"agentid": w.agentID,
		"image": map[string]string{
			"media_id": mediaID,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	resp, err := w.client.Post(url, "application/json", bytes.NewReader(body))
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
		return nil, fmt.Errorf("wecom error: %s", result.ErrMsg)
	}

	return &SendResult{}, nil
}

// GetChatInfo returns information about a WeCom chat.
func (w *WeComAdapter) GetChatInfo(chatID string) (*ChatInfo, error) {
	return &ChatInfo{
		ID:   chatID,
		Name: "WeCom Chat",
		Type: ChatTypeDM,
	}, nil
}

// SendRichText sends rich text to a WeCom chat.
func (w *WeComAdapter) SendRichText(chatID string, richText *RichTextContent) (*SendResult, error) {
	// WeCom supports markdown, so concatenate text elements
	text := ""
	for _, elem := range richText.Elements {
		if elem.Type == "text" {
			text += elem.Text
		}
	}

	return w.Send(chatID, text)
}

// SendInteractiveCard sends an interactive card to a WeCom chat.
func (w *WeComAdapter) SendInteractiveCard(chatID string, card *InteractiveCard) (*SendResult, error) {
	// WeCom supports markdown, so send as markdown
	text := ""
	if card.Title != "" {
		text += "## " + card.Title + "\n\n"
	}
	for _, elem := range card.Elements {
		if elem.Type == "text" {
			text += elem.Text + "\n\n"
		}
	}

	return w.Send(chatID, text)
}

// HandleWebhook handles an incoming webhook event from WeCom.
func (w *WeComAdapter) HandleWebhook(payload []byte, signature, timestamp, nonce string) error {
	// Verify signature
	if !w.verifySignature(signature, timestamp, nonce) {
		return fmt.Errorf("invalid signature")
	}

	// Parse XML
	var event wecomEvent
	if err := xml.Unmarshal(payload, &event); err != nil {
		return err
	}

	// Handle message events
	if event.MsgType == "text" {
		msgEvent := &MessageEvent{
			Text:        event.Content,
			MessageType: MessageTypeText,
			Source: SessionSource{
				Platform:  PlatformWeCom,
				ChatID:    event.FromUserName,
				ChatType:  ChatTypeDM,
				UserID:    event.FromUserName,
				MessageID: event.MsgID,
			},
			MessageID: event.MsgID,
			Timestamp: time.Now(),
		}

		w.HandleMessage(msgEvent)
	}

	return nil
}

// getAccessToken gets the WeCom access token.
func (w *WeComAdapter) getAccessToken() (string, error) {
	url := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/gettoken?corpid=%s&corpsecret=%s", w.corpID, w.secret)

	resp, err := w.client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		ErrCode     int    `json:"errcode"`
		AccessToken string `json:"access_token"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if result.ErrCode != 0 {
		return "", fmt.Errorf("failed to get access token")
	}

	return result.AccessToken, nil
}

// uploadMedia uploads media to WeCom.
func (w *WeComAdapter) uploadMedia(imageURL string) (string, error) {
	// Download image
	resp, err := w.client.Get(imageURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	imageData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Get access token
	token, err := w.getAccessToken()
	if err != nil {
		return "", err
	}

	// Upload to WeCom
	url := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/media/upload?access_token=%s&type=image", token)

	// Create multipart form
	// Simplified - in production, use multipart form
	_ = imageData

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return "", err
	}

	uploadResp, err := w.client.Do(req)
	if err != nil {
		return "", err
	}
	defer uploadResp.Body.Close()

	var result struct {
		MediaID string `json:"media_id"`
	}

	if err := json.NewDecoder(uploadResp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.MediaID, nil
}

// verifySignature verifies the WeCom signature.
func (w *WeComAdapter) verifySignature(signature, timestamp, nonce string) bool {
	strs := []string{w.token, timestamp, nonce}
	sort.Strings(strs)
	str := strings.Join(strs, "")

	h := sha1.New()
	h.Write([]byte(str))
	hash := fmt.Sprintf("%x", h.Sum(nil))

	return hash == signature
}

// WeCom event types
type wecomEvent struct {
	XMLName      xml.Name `xml:"xml"`
	ToUserName   string   `xml:"ToUserName"`
	FromUserName string   `xml:"FromUserName"`
	CreateTime   int64    `xml:"CreateTime"`
	MsgType      string   `xml:"MsgType"`
	Content      string   `xml:"Content"`
	MsgID        string   `xml:"MsgId"`
}

func init() {
	DefaultRegistry.Register(&PlatformEntry{
		Platform: PlatformWeCom,
		Label:    "WeCom",
		Factory: func(config AdapterConfig) (Adapter, error) {
			return NewWeComAdapter(config), nil
		},
		Description:    "WeCom Bot API (Enterprise WeChat)",
		RequiredFields: []string{"corpId", "secret"},
	})
}
