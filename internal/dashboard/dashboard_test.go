package dashboard

import (
	"net/http"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	d := New(nil)
	if d == nil {
		t.Fatal("expected non-nil dashboard")
	}
}

func TestNew_WithConfig(t *testing.T) {
	d := New(&Config{Port: 9090, Username: "admin"})
	if d.config.Port != 9090 {
		t.Errorf("expected port=9090, got %d", d.config.Port)
	}
}

func TestStartStop(t *testing.T) {
	d := New(&Config{Port: 19876}) // 使用不常见端口
	err := d.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	// 测试API是否响应
	resp, err := http.Get("http://localhost:19876/api/status")
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	d.Stop()
}

func TestRegisterModule(t *testing.T) {
	d := New(&Config{Port: 8080})
	d.RegisterModule("agent", "1.0.0")
	d.RegisterModule("canvas", "2.0.0")

	d.mu.RLock()
	count := len(d.modules)
	d.mu.RUnlock()
	if count != 2 {
		t.Errorf("expected 2 modules, got %d", count)
	}
}

func TestHandleStatus(t *testing.T) {
	d := New(&Config{Port: 8080})
	d.Start()
	time.Sleep(10 * time.Millisecond)

	// 创建请求
	req, _ := http.NewRequest("GET", "/api/status", nil)
	w := &mockResponseWriter{header: make(http.Header)}
	d.handleStatus(w, req)

	if w.statusCode != 0 && w.statusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", w.statusCode)
	}
}

func TestHandleModules(t *testing.T) {
	d := New(&Config{Port: 8080})
	d.RegisterModule("test", "0.1.0")

	req, _ := http.NewRequest("GET", "/api/modules", nil)
	w := &mockResponseWriter{header: make(http.Header)}
	d.handleModules(w, req)

	_ = w
}

type mockResponseWriter struct {
	header     http.Header
	statusCode int
	body       []byte
}

func (m *mockResponseWriter) Header() http.Header { return m.header }

func (m *mockResponseWriter) Write(b []byte) (int, error) {
	m.body = append(m.body, b...)
	return len(b), nil
}

func (m *mockResponseWriter) WriteHeader(code int) { m.statusCode = code }
