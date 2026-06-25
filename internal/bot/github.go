package bot

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// GitHubAdapter implements the Adapter interface for GitHub.
type GitHubAdapter struct {
	*BaseAdapter
	appID          string
	privateKey     string
	webhookSecret  string
	client         *http.Client
	stopCh         chan struct{}
}

// NewGitHubAdapter creates a new GitHub adapter.
func NewGitHubAdapter(config AdapterConfig) Adapter {
	return &GitHubAdapter{
		BaseAdapter:   NewBaseAdapter(config),
		appID:         config.AppID,
		privateKey:    config.Extra["privateKey"],
		webhookSecret: config.Extra["webhookSecret"],
		client:        &http.Client{Timeout: 30 * time.Second},
		stopCh:        make(chan struct{}),
	}
}

// Connect starts the GitHub adapter.
func (g *GitHubAdapter) Connect() error {
	if g.appID == "" {
		return fmt.Errorf("github app_id is required")
	}

	// 验证 GitHub API 连通性
	if err := g.verifyAPI(); err != nil {
		return fmt.Errorf("failed to verify GitHub API: %w", err)
	}

	g.SetConnected(true)
	return nil
}

// verifyAPI verifies the GitHub API connectivity
func (g *GitHubAdapter) verifyAPI() error {
	url := "https://api.github.com"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	// 如果有 webhook secret，用作 token
	if g.webhookSecret != "" {
		req.Header.Set("Authorization", "Bearer "+g.webhookSecret)
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := g.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// GitHub API 返回 200 或 401 都表示 API 可达
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusUnauthorized {
		return fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	return nil
}

// Disconnect stops the GitHub adapter.
func (g *GitHubAdapter) Disconnect() error {
	if !g.IsConnected() {
		return nil
	}

	close(g.stopCh)
	g.SetConnected(false)
	return nil
}

// Send sends a text message (as a comment on an issue/PR).
func (g *GitHubAdapter) Send(chatID string, text string) (*SendResult, error) {
	// chatID format: "owner/repo/issues/123" or "owner/repo/pulls/123"
	// For simplicity, we'll treat chatID as an issue/PR URL
	url := fmt.Sprintf("https://api.github.com/repos/%s/issues", chatID)

	payload := map[string]interface{}{
		"body": text,
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
	req.Header.Set("Authorization", "Bearer "+g.webhookSecret)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		ID int `json:"id"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &SendResult{
		MessageID: fmt.Sprintf("%d", result.ID),
	}, nil
}

// SendImage sends an image (as markdown in a comment).
func (g *GitHubAdapter) SendImage(chatID string, imageURL string, caption string) (*SendResult, error) {
	text := fmt.Sprintf("%s\n\n![image](%s)", caption, imageURL)
	return g.Send(chatID, text)
}

// GetChatInfo returns information about a GitHub repository.
func (g *GitHubAdapter) GetChatInfo(chatID string) (*ChatInfo, error) {
	// chatID format: "owner/repo"
	url := fmt.Sprintf("https://api.github.com/repos/%s", chatID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+g.webhookSecret)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		FullName string `json:"full_name"`
		Name     string `json:"name"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &ChatInfo{
		ID:   chatID,
		Name: result.FullName,
		Type: ChatTypeChannel,
	}, nil
}

// SendRichText sends rich text to a GitHub issue/PR.
func (g *GitHubAdapter) SendRichText(chatID string, richText *RichTextContent) (*SendResult, error) {
	// GitHub uses markdown, so concatenate text elements
	text := ""
	for _, elem := range richText.Elements {
		if elem.Type == "text" {
			text += elem.Text
		} else if elem.Type == "mention" {
			text += fmt.Sprintf("@%s", elem.Text)
		} else if elem.Type == "link" {
			text += elem.URL
		} else if elem.Type == "code" {
			text += fmt.Sprintf("`%s`", elem.Text)
		}
	}

	return g.Send(chatID, text)
}

// SendInteractiveCard sends an interactive card to a GitHub issue/PR.
func (g *GitHubAdapter) SendInteractiveCard(chatID string, card *InteractiveCard) (*SendResult, error) {
	// GitHub doesn't support interactive cards directly, so send as markdown
	text := ""
	if card.Title != "" {
		text += "## " + card.Title + "\n\n"
	}
	for _, elem := range card.Elements {
		if elem.Type == "text" {
			text += elem.Text + "\n\n"
		}
	}

	return g.Send(chatID, text)
}

// HandleWebhook handles an incoming webhook event from GitHub.
func (g *GitHubAdapter) HandleWebhook(payload []byte, signature string) error {
	// Verify signature
	if g.webhookSecret != "" && !g.verifySignature(payload, signature) {
		return fmt.Errorf("invalid signature")
	}

	var event githubEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return err
	}

	// Handle issue comment events
	if event.Action == "created" && event.Comment != nil {
		msgEvent := &MessageEvent{
			Text:        event.Comment.Body,
			MessageType: MessageTypeText,
			Source: SessionSource{
				Platform:  PlatformWebhook,
				ChatID:    fmt.Sprintf("%s/issues/%d", event.Repository.FullName, event.Issue.Number),
				ChatType:  ChatTypeChannel,
				UserID:    fmt.Sprintf("%d", event.Comment.User.ID),
				UserName:  event.Comment.User.Login,
				MessageID: fmt.Sprintf("%d", event.Comment.ID),
			},
			MessageID: fmt.Sprintf("%d", event.Comment.ID),
			Timestamp: time.Now(),
		}

		g.HandleMessage(msgEvent)
	}

	return nil
}

// verifySignature verifies the GitHub webhook signature.
func (g *GitHubAdapter) verifySignature(payload []byte, signature string) bool {
	if len(signature) < 7 {
		return false
	}

	// Remove "sha256=" prefix
	sig := signature[7:]

	mac := hmac.New(sha256.New, []byte(g.webhookSecret))
	mac.Write(payload)
	expectedMAC := fmt.Sprintf("%x", mac.Sum(nil))

	return hmac.Equal([]byte(sig), []byte(expectedMAC))
}

// GitHub event types
type githubEvent struct {
	Action string `json:"action"`
	Comment *struct {
		ID   int    `json:"id"`
		Body string `json:"body"`
		User struct {
			ID    int    `json:"id"`
			Login string `json:"login"`
		} `json:"user"`
	} `json:"comment"`
	Issue *struct {
		Number int `json:"number"`
	} `json:"issue"`
	Repository struct {
		FullName string `json:"full_name"`
	} `json:"repository"`
}

func init() {
	DefaultRegistry.Register(&PlatformEntry{
		Platform: PlatformWebhook,
		Label:    "GitHub",
		Factory: func(config AdapterConfig) (Adapter, error) {
			return NewGitHubAdapter(config), nil
		},
		Description:    "GitHub Webhook API",
		RequiredFields: []string{"appId"},
	})
}
