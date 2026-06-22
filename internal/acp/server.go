// ACP (Agent Communication Protocol) - IDE原生集成
// 基于stdio/JSON-RPC，支持VS Code/Zed/JetBrains
package acp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
)

// Request JSON-RPC请求
type Request struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// Response JSON-RPC响应
type Response struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
}

// RPCError RPC错误
type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Notification 通知
type Notification struct {
	Method string      `json:"method"`
	Params interface{} `json:"params,omitempty"`
}

// ========== 工具定义 ==========

// Tool 工具
type Tool struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	InputSchema InputSchema `json:"inputSchema"`
}

// InputSchema 输入Schema
type InputSchema struct {
	Type       string                  `json:"type"`
	Properties map[string]PropertySchema `json:"properties,omitempty"`
	Required   []string                `json:"required,omitempty"`
}

// PropertySchema 属性Schema
type PropertySchema struct {
	Type        string   `json:"type"`
	Description string   `json:"description,omitempty"`
	Enum        []string `json:"enum,omitempty"`
}

// ========== MCP兼容方法 ==========

const (
	MethodListTools    = "tools/list"
	MethodCallTool     = "tools/call"
	MethodListResources = "resources/list"
	MethodReadResource  = "resources/read"
)

// ========== 服务器 ==========

// Server ACP服务器
type Server struct {
	mu       sync.Mutex
	reader   *bufio.Reader
	writer   *bufio.Writer
	tools    []Tool
	handlers map[string]ToolHandler
	done     chan struct{}
}

// ToolHandler 工具处理器
type ToolHandler func(params json.RawMessage) (interface{}, *RPCError)

// NewServer 创建ACP服务器
func NewServer(r io.Reader, w io.Writer) *Server {
	return &Server{
		reader:   bufio.NewReader(r),
		writer:   bufio.NewWriter(w),
		tools:    make([]Tool, 0),
		handlers: make(map[string]ToolHandler),
		done:     make(chan struct{}),
	}
}

// RegisterTool 注册工具
func (s *Server) RegisterTool(t Tool, handler ToolHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tools = append(s.tools, t)
	s.handlers[t.Name] = handler
}

// Start 启动服务器（阻塞）
func (s *Server) Start() error {
	decoder := json.NewDecoder(s.reader)

	for {
		var req Request
		if err := decoder.Decode(&req); err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("decode request: %w", err)
		}

		go s.handleRequest(req)
	}
}

// Stop 停止服务器
func (s *Server) Stop() {
	select {
	case <-s.done:
	default:
		close(s.done)
	}
}

func (s *Server) handleRequest(req Request) {
	switch req.Method {
	case MethodListTools:
		s.sendResponse(req.ID, map[string]interface{}{
			"tools": s.tools,
		}, nil)

	case MethodCallTool:
		s.handleToolCall(req)

	default:
		s.sendResponse(req.ID, nil, &RPCError{
			Code: -32601, Message: fmt.Sprintf("method not found: %s", req.Method),
		})
	}
}

func (s *Server) handleToolCall(req Request) {
	params, ok := req.Params.(map[string]interface{})
	if !ok {
		s.sendResponse(req.ID, nil, &RPCError{Code: -32602, Message: "invalid params"})
		return
	}

	name, _ := params["name"].(string)
	s.mu.Lock()
	handler, ok := s.handlers[name]
	s.mu.Unlock()

	if !ok {
		s.sendResponse(req.ID, nil, &RPCError{
			Code: -32601, Message: fmt.Sprintf("tool not found: %s", name),
		})
		return
	}

	// 将arguments转为json.RawMessage
	arguments, _ := json.Marshal(params["arguments"])
	result, rpcErr := handler(arguments)

	s.sendResponse(req.ID, result, rpcErr)
}

func (s *Server) sendResponse(id interface{}, result interface{}, rpcErr *RPCError) {
	resp := Response{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
		Error:   rpcErr,
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	data, _ := json.Marshal(resp)
	data = append(data, '\n')
	s.writer.Write(data)
	s.writer.Flush()
}

// ========== 客户端 ==========

// Client ACP客户端
type Client struct {
	reader *bufio.Reader
	writer *bufio.Writer
	mu     sync.Mutex
}

// NewClient 创建ACP客户端
func NewClient(r io.Reader, w io.Writer) *Client {
	return &Client{
		reader: bufio.NewReader(r),
		writer: bufio.NewWriter(w),
	}
}

// ListTools 列出工具
func (c *Client) ListTools() ([]Tool, error) {
	req := Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  MethodListTools,
	}

	resp, err := c.sendAndReceive(req)
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("RPC error: %s", resp.Error.Message)
	}

	data, _ := json.Marshal(resp.Result)
	var result struct {
		Tools []Tool `json:"tools"`
	}
	json.Unmarshal(data, &result)

	return result.Tools, nil
}

// CallTool 调用工具
func (c *Client) CallTool(name string, args map[string]interface{}) (interface{}, error) {
	req := Request{
		JSONRPC: "2.0",
		ID:      2,
		Method:  MethodCallTool,
		Params: map[string]interface{}{
			"name":      name,
			"arguments": args,
		},
	}

	resp, err := c.sendAndReceive(req)
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("RPC error: %s", resp.Error.Message)
	}

	return resp.Result, nil
}

func (c *Client) sendAndReceive(req Request) (*Response, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	data, _ := json.Marshal(req)
	data = append(data, '\n')

	if _, err := c.writer.Write(data); err != nil {
		return nil, err
	}
	if err := c.writer.Flush(); err != nil {
		return nil, err
	}

	var resp Response
	decoder := json.NewDecoder(c.reader)
	if err := decoder.Decode(&resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// ========== StdioServer - 通过标准输入输出运行 ==========

// RunStdioServer 运行stdio服务器（用于VS Code/Zed集成）
func RunStdioServer(tools []Tool, handlers map[string]ToolHandler) error {
	server := NewServer(os.Stdin, os.Stdout)
	for _, t := range tools {
		server.RegisterTool(t, handlers[t.Name])
	}
	return server.Start()
}
