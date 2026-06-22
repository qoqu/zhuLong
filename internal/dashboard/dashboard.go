// Web Dashboard - 管理界面（参考Hermes dashboard）
package dashboard

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"sync"
	"time"
)

// Config Dashboard配置
type Config struct {
	Port     int
	Username string
	Password string // SHA256哈希
	DataDir  string
}

// ModuleInfo 模块信息
type ModuleInfo struct {
	Name     string `json:"name"`
	Status   string `json:"status"` // running / stopped / error
	Uptime   string `json:"uptime,omitempty"`
	Version  string `json:"version,omitempty"`
}

// Dashboard 管理仪表盘
type Dashboard struct {
	mu       sync.RWMutex
	config   *Config
	started  time.Time
	modules  []ModuleInfo
	server   *http.Server
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

	// 静态文件（前端构建产物）
	// 在生产环境中由前端服务器提供

	d.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", d.config.Port),
		Handler: mux,
	}

	go d.server.ListenAndServe()
	return nil
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
		Name: name, Status: "running",
		Uptime: time.Since(d.started).Round(time.Second).String(),
		Version: version,
	})
}

// AddFS 添加静态文件系统
func (d *Dashboard) AddFS(fsys fs.FS, prefix string) {
	// 支持嵌入前端构建产物
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
	})
}

func (d *Dashboard) handleModules(w http.ResponseWriter, r *http.Request) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(d.modules)
}

func (d *Dashboard) handleConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	// 只暴露安全信息
	json.NewEncoder(w).Encode(map[string]interface{}{
		"port":    d.config.Port,
		"dataDir": d.config.DataDir,
	})
}
