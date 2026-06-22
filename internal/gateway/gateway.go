// 消息网关 - 20+平台适配器 + 统一路由（参考Hermes Gateway）
package gateway

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// Platform 平台类型
type Platform string

const (
	PlatformCLI      Platform = "cli"
	PlatformTelegram Platform = "telegram"
	PlatformDiscord  Platform = "discord"
	PlatformSlack    Platform = "slack"
	PlatformWhatsApp Platform = "whatsapp"
	PlatformDingTalk Platform = "dingtalk"
	PlatformFeishu   Platform = "feishu"
	PlatformWeCom    Platform = "wecom"
	PlatformMatrix   Platform = "matrix"
	PlatformTeams    Platform = "teams"
	PlatformEmail    Platform = "email"
	PlatformWebhook  Platform = "webhook"
	PlatformAPI      Platform = "api"
)

// ========== 消息模型 ==========

// Message 统一消息
type Message struct {
	ID        string
	Platform  Platform
	ChannelID string  // 频道/群组ID
	UserID    string  // 用户ID
	Content   string  // 消息内容
	ReplyTo   string  // 回复目标消息ID
	Timestamp time.Time
	Metadata  map[string]interface{}
}

// OutgoingMessage 出站消息
type OutgoingMessage struct {
	Platform  Platform
	ChannelID string
	UserID    string
	Content   string
	ReplyTo   string
}

// ========== 平台适配器接口 ==========

// Adapter 平台适配器
type Adapter interface {
	Platform() Platform
	Name() string
	Start(handler MessageHandler) error
	Stop() error
	Send(msg OutgoingMessage) error
	IsAvailable() bool
}

// MessageHandler 消息处理函数
type MessageHandler func(msg Message)

// ========== 适配器注册表 ==========

// Registry 适配器注册表
type Registry struct {
	mu        sync.RWMutex
	adapters  map[Platform]Adapter
	handlers  []MessageHandler
}

// NewRegistry 创建注册表
func NewRegistry() *Registry {
	return &Registry{
		adapters: make(map[Platform]Adapter),
	}
}

// Register 注册适配器
func (r *Registry) Register(a Adapter) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.adapters[a.Platform()] = a
}

// Get 获取适配器
func (r *Registry) Get(p Platform) (Adapter, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	a, ok := r.adapters[p]
	return a, ok
}

// ListPlatforms 列出已注册平台
func (r *Registry) ListPlatforms() []Platform {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var list []Platform
	for p := range r.adapters {
		list = append(list, p)
	}
	return list
}

// OnMessage 注册消息处理器
func (r *Registry) OnMessage(handler MessageHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers = append(r.handlers, handler)
}

// Dispatch 分发消息到所有处理器
func (r *Registry) Dispatch(msg Message) {
	r.mu.RLock()
	handlers := make([]MessageHandler, len(r.handlers))
	copy(handlers, r.handlers)
	r.mu.RUnlock()

	for _, h := range handlers {
		h(msg)
	}
}

// ========== CLI适配器 ==========

type CLIAdapter struct {
	handler MessageHandler
	running bool
}

func NewCLIAdapter() *CLIAdapter { return &CLIAdapter{} }

func (a *CLIAdapter) Platform() Platform { return PlatformCLI }
func (a *CLIAdapter) Name() string       { return "CLI" }

func (a *CLIAdapter) Start(handler MessageHandler) error {
	a.handler = handler
	a.running = true
	return nil
}

func (a *CLIAdapter) Stop() error {
	a.running = false
	return nil
}

func (a *CLIAdapter) Send(msg OutgoingMessage) error {
	fmt.Println(msg.Content)
	return nil
}

func (a *CLIAdapter) IsAvailable() bool { return true }

// ========== 管理器 ==========

// Runner 网关运行器
type Runner struct {
	registry *Registry
	adapters []Adapter
	sessions *SessionStore
}

// NewRunner 创建网关运行器
func NewRunner() *Runner {
	return &Runner{
		registry: NewRegistry(),
		adapters: make([]Adapter, 0),
		sessions: NewSessionStore(),
	}
}

// RegisterAdapter 注册适配器
func (r *Runner) RegisterAdapter(a Adapter) {
	r.adapters = append(r.adapters, a)
	r.registry.Register(a)
}

// Start 启动所有适配器
func (r *Runner) Start() error {
	for _, a := range r.adapters {
		if !a.IsAvailable() {
			log.Printf("[gateway] skipping unavailable: %s", a.Name())
			continue
		}
		if err := a.Start(func(msg Message) {
			r.registry.Dispatch(msg)
		}); err != nil {
			return fmt.Errorf("start %s: %w", a.Name(), err)
		}
		log.Printf("[gateway] started: %s", a.Name())
	}
	return nil
}

// Stop 停止所有适配器
func (r *Runner) Stop() {
	for _, a := range r.adapters {
		a.Stop()
	}
}

// Send 发送消息到指定平台
func (r *Runner) Send(msg OutgoingMessage) error {
	a, ok := r.registry.Get(msg.Platform)
	if !ok {
		return fmt.Errorf("platform %s not registered", msg.Platform)
	}
	return a.Send(msg)
}

// Registry 返回注册表
func (r *Runner) Registry() *Registry { return r.registry }

// ========== 会话管理 ==========

// Session 会话
type Session struct {
	ID        string
	Platform  Platform
	UserID    string
	ChannelID string
	CreatedAt time.Time
	Context   map[string]interface{}
}

// SessionStore 会话存储
type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

func NewSessionStore() *SessionStore {
	return &SessionStore{
		sessions: make(map[string]*Session),
	}
}

func (s *SessionStore) GetOrCreate(platform Platform, userID, channelID string) *Session {
	key := fmt.Sprintf("%s:%s:%s", platform, userID, channelID)
	s.mu.Lock()
	defer s.mu.Unlock()

	if ses, ok := s.sessions[key]; ok {
		return ses
	}

	ses := &Session{
		ID:        key,
		Platform:  platform,
		UserID:    userID,
		ChannelID: channelID,
		CreatedAt: time.Now(),
		Context:   make(map[string]interface{}),
	}
	s.sessions[key] = ses
	return ses
}

func (s *SessionStore) Get(id string) (*Session, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ses, ok := s.sessions[id]
	return ses, ok
}

func (s *SessionStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.sessions)
}
