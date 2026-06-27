// Web Dashboard - 管理界面
package dashboard

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"sync"
	"time"
)

//go:embed static/index.html
var staticFS embed.FS

// Config Dashboard配置
type Config struct {
	Port     int
	Username string
	Password string // SHA256哈希
	DataDir  string
}

// ModuleInfo 模块信息
type ModuleInfo struct {
	Name    string `json:"name"`
	Status  string `json:"status"` // running / stopped / error
	Uptime  string `json:"uptime,omitempty"`
	Version string `json:"version,omitempty"`
}

// Dashboard 管理仪表盘
type Dashboard struct {
	mu      sync.RWMutex
	config  *Config
	started time.Time
	modules []ModuleInfo
	server  *http.Server
}

// New 创建Dashboard
func New(cfg *Config) *Dashboard {
	if cfg == nil {
		cfg = &Config{Port: 8080}
	}
	return &Dashboard{
		config:  cfg,
		started: time.Now(),
	}
}

// Start 启动Dashboard
func (d *Dashboard) Start() error {
	mux := http.NewServeMux()

	// API路由
	mux.HandleFunc("/api/status", d.handleStatus)
	mux.HandleFunc("/api/modules", d.handleModules)
	mux.HandleFunc("/api/config", d.handleConfig)
	mux.HandleFunc("/api/sessions", d.handleSessions)
	mux.HandleFunc("/api/stop", d.handleStop)

	// 静态页面 - 内嵌 HTML
	mux.HandleFunc("/", d.handleIndex)

	d.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", d.config.Port),
		Handler: mux,
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- d.server.ListenAndServe()
	}()

	// Check immediately if the server failed to start
	select {
	case err := <-errCh:
		return fmt.Errorf("dashboard server failed to start: %w", err)
	default:
		return nil
	}
}

// Stop 停止Dashboard
func (d *Dashboard) Stop() error {
	if d.server != nil {
		return d.server.Close()
	}
	return nil
}

// RegisterModule 注册模块
func (d *Dashboard) RegisterModule(name, version string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.modules = append(d.modules, ModuleInfo{
		Name:    name,
		Status:  "running",
		Uptime:  time.Since(d.started).Round(time.Second).String(),
		Version: version,
	})
}

// AddFS 添加静态文件系统
func (d *Dashboard) AddFS(fsys fs.FS, prefix string) {
	_ = fsys
	_ = prefix
}

// ========== API Handlers ==========

func (d *Dashboard) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "running",
		"uptime":  time.Since(d.started).Round(time.Second).String(),
		"port":    d.config.Port,
		"name":    "Zhulong Dashboard",
		"version": "0.4.0",
	})
}

func (d *Dashboard) handleModules(w http.ResponseWriter, r *http.Request) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	w.Header().Set("Content-Type", "application/json")
	if d.modules == nil {
		json.NewEncoder(w).Encode([]ModuleInfo{})
		return
	}
	json.NewEncoder(w).Encode(d.modules)
}

func (d *Dashboard) handleConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"port":    d.config.Port,
		"dataDir": d.config.DataDir,
	})
}

// SessionStateProvider 提供 session 状态
type SessionStateProvider func() interface{}

var sessionProvider SessionStateProvider

// SetSessionProvider 注入 session provider
func (d *Dashboard) SetSessionProvider(p SessionStateProvider) {
	sessionProvider = p
}

func (d *Dashboard) handleSessions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if sessionProvider != nil {
		json.NewEncoder(w).Encode(sessionProvider())
		return
	}
	json.NewEncoder(w).Encode([]interface{}{})
}

// handleStop 远程停止 Dashboard
func (d *Dashboard) handleStop(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "stopping"})
	go func() {
		time.Sleep(100 * time.Millisecond)
		_ = d.Stop()
	}()
}

// handleIndex 内嵌 HTML 仪表盘页面
func (d *Dashboard) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	data, err := staticFS.ReadFile("static/index.html")
	if err != nil {
		fmt.Fprint(w, "Error loading index.html: "+err.Error())
		return
	}
	w.Write(data)
}
