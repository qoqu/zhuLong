package bot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// SlackAdapter implements the Adapter interface for Slack.
type SlackAdapter struct {
	*BaseAdapter
	botToken       string
	signingSecret  string
	client         *http.Client
	stopCh         chan struct{}
}

// NewSlackAdapter creates a new Slack adapter.
func NewSlackAdapter(config AdapterConfig) Adapter {
	return &SlackAdapter{
		BaseAdapter:   NewBaseAdapter(config),
		botToken:      config.Token,
		signingSecret: config.Extra["signingSecret"],
		client:        &http.Client{Timeout: 30 * time.Second},
		stopCh:        make(chan struct{}),
	}
}

// Connect starts the Slack adapter.
func (s *SlackAdapter) Connect() error {
	if s.botToken == "" {
		return fmt.Errorf("slack bot token is required")
	}

	// Verify bot token
	if err := s.verifyToken(); err != nil {
		return fmt.Errorf("invalid bot token: %w", err)
	}

	s.SetConnected(true)
	return nil
}

// Disconnect stops the Slack adapter.
func (s *SlackAdapter) Disconnect() error {
	if !s.IsConnected() {
		return nil
	}

	close(s.stopCh)
	s.SetConnected(false)
	return nil
}

// Send sends a text message to a Slack channel.
func (s *SlackAdapter) Send(chatID string, text string) (*SendResult, error) {
	url := "https://slack.com/api/chat.postMessage"

	payload := map[string]interface{}{
		"channel": chatID,
		"text":    text,
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
	req.Header.Set("Authorization", "Bearer "+s.botToken)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		OK      bool   `json:"ok"`
		TS      string `json:"ts"`
		Error   string `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if !result.OK {
		return nil, fmt.Errorf("slack error: %s", result.Error)
	}

	return &SendResult{
		MessageID: result.TS,
	}, nil
}

// SendImage sends an image to a Slack channel.
func (s *SlackAdapter) SendImage(chatID string, imageURL string, caption string) (*SendResult, error) {
	url := "https://slack.com/api/chat.postMessage"

	payload := map[string]interface{}{
		"channel": chatID,
		"text":    caption,
		"blocks": []map[string]interface{}{
			{
				"type": "image",
				"image_url": imageURL,
				"alt_text": caption,
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
	req.Header.Set("Authorization", "Bearer "+s.botToken)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		OK    bool   `json:"ok"`
		TS    string `json:"ts"`
		Error string `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if !result.OK {
		return nil, fmt.Errorf("slack error: %s", result.Error)
	}

	return &SendResult{
		MessageID: result.TS,
	}, nil
}

// GetChatInfo returns information about a Slack channel.
func (s *SlackAdapter) GetChatInfo(chatID string) (*ChatInfo, error) {
	url := fmt.Sprintf("https://slack.com/api/conversations.info?channel=%s", chatID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+s.botToken)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		OK       bool `json:"ok"`
		Channel  struct {
			ID      string `json:"id"`
			Name    string `json:"name"`
			IsGroup bool   `json:"is_group"`
			IsIM    bool   `json:"is_im"`
		} `json:"channel"`
		Error string `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if !result.OK {
		return nil, fmt.Errorf("slack error: %s", result.Error)
	}

	chatType := ChatTypeChannel
	if result.Channel.IsIM {
		chatType = ChatTypeDM
	} else if result.Channel.IsGroup {
		chatType = ChatTypeGroup
	}

	return &ChatInfo{
		ID:   result.Channel.ID,
		Name: result.Channel.Name,
		Type: chatType,
	}, nil
}

// SendRichText sends rich text to a Slack channel.
func (s *SlackAdapter) SendRichText(chatID string, richText *RichTextContent) (*SendResult, error) {
	// Slack uses mrkdwn, so build from rich text elements
	text := ""
	for _, elem := range richText.Elements {
		if elem.Type == "text" {
			text += elem.Text
		} else if elem.Type == "mention" {
			text += fmt.Sprintf("<@%s>", elem.MentionID)
		} else if elem.Type == "link" {
			text += fmt.Sprintf("<%s|%s>", elem.URL, elem.Text)
		} else if elem.Type == "code" {
			text += fmt.Sprintf("`%s`", elem.Text)
		}
	}

	return s.Send(chatID, text)
}

// SendInteractiveCard sends an interactive card to a Slack channel.
func (s *SlackAdapter) SendInteractiveCard(chatID string, card *InteractiveCard) (*SendResult, error) {
	// Build Slack Block Kit message
	blocks := make([]map[string]interface{}, 0)

	// Add header
	if card.Title != "" {
		blocks = append(blocks, map[string]interface{}{
			"type": "header",
			"text": map[string]interface{}{
				"type": "plain_text",
				"text": card.Title,
			},
		})
	}

	// Add elements
	for _, elem := range card.Elements {
		if elem.Type == "text" {
			blocks = append(blocks, map[string]interface{}{
				"type": "section",
				"text": map[string]interface{}{
					"type": "mrkdwn",
					"text": elem.Text,
				},
			})
		}
	}

	// Add actions
	if len(card.Actions) > 0 {
		elements := make([]map[string]interface{}, 0)
		for _, action := range card.Actions {
			elements = append(elements, map[string]interface{}{
				"type": "button",
				"text": map[string]interface{}{
					"type": "plain_text",
					"text": action.Text,
				},
				"url": action.Value,
			})
		}
		blocks = append(blocks, map[string]interface{}{
			"type": "actions",
			"elements": elements,
		})
	}

	payload := map[string]interface{}{
		"channel": chatID,
		"blocks":  blocks,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", "https://slack.com/api/chat.postMessage", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.botToken)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		OK    bool   `json:"ok"`
		TS    string `json:"ts"`
		Error string `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if !result.OK {
		return nil, fmt.Errorf("slack error: %s", result.Error)
	}

	return &SendResult{
		MessageID: result.TS,
	}, nil
}

// verifyToken verifies the bot token by calling auth.test.
func (s *SlackAdapter) verifyToken() error {
	url := "https://slack.com/api/auth.test"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+s.botToken)

	resp, err := s.client.Do(req)
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

// HandleWebhook handles an incoming webhook event from Slack.
func (s *SlackAdapter) HandleWebhook(payload []byte) error {
	var event slackEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return err
	}

	// Handle message events
	if event.Type == "message" {
		msgEvent := &MessageEvent{
			Text:        event.Event.Text,
			MessageType: MessageTypeText,
			Source: SessionSource{
				Platform:  PlatformSlack,
				ChatID:    event.Event.Channel,
				ChatType:  ChatTypeChannel,
				UserID:    event.Event.User,
				MessageID: event.Event.TS,
			},
			MessageID: event.Event.TS,
			Timestamp: time.Now(),
		}

		s.HandleMessage(msgEvent)
	}

	return nil
}

// Slack event types
type slackEvent struct {
	Type      string `json:"type"`
	Event     struct {
		Type    string `json:"type"`
		User    string `json:"user"`
		Channel string `json:"channel"`
		Text    string `json:"text"`
		TS      string `json:"ts"`
	} `json:"event"`
}

func init() {
	DefaultRegistry.Register(&PlatformEntry{
		Platform: PlatformSlack,
		Label:    "Slack",
		Factory: func(config AdapterConfig) (Adapter, error) {
			return NewSlackAdapter(config), nil
		},
		Description:    "Slack Bot API (Socket Mode + Events API)",
		RequiredFields: []string{"token"},
	})
}
