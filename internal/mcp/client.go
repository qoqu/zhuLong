package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"sync"
	"time"
)

// Client represents an MCP client that connects to an MCP server
type Client struct {
	config     ServerConfig
	cmd        *exec.Cmd
	stdin      io.WriteCloser
	stdout     io.ReadCloser
	stderr     io.ReadCloser
	scanner    *bufio.Scanner
	nextID     int
	mu         sync.Mutex
	tools      []Tool
	resources  []Resource
	prompts    []Prompt
	connected  bool
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewClient creates a new MCP client
func NewClient(config ServerConfig) *Client {
	ctx, cancel := context.WithCancel(context.Background())
	return &Client{
		config: config,
		ctx:    ctx,
		cancel: cancel,
	}
}

// Connect connects to the MCP server
func (c *Client) Connect() error {
	if c.connected {
		return nil
	}

	switch c.config.Transport.Type {
	case "stdio":
		return c.connectStdio()
	case "sse", "http":
		return c.connectHTTP()
	default:
		return fmt.Errorf("unsupported transport type: %s", c.config.Transport.Type)
	}
}

// connectHTTP connects to the MCP server via HTTP/SSE
func (c *Client) connectHTTP() error {
	url := c.config.Transport.URL
	if url == "" {
		return fmt.Errorf("HTTP URL is required")
	}

	// 验证 HTTP 端点
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to connect to HTTP server: %w", err)
	}
	resp.Body.Close()

	c.connected = true
	return nil
}

// sendHTTPRequest sends a JSON-RPC request via HTTP POST
func (c *Client) sendHTTPRequest(req Request) (*Response, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := c.config.Transport.URL
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var response Response
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response, nil
}

// connectStdio connects to the MCP server via stdio
func (c *Client) connectStdio() error {
	cmd := exec.CommandContext(c.ctx, c.config.Transport.Command, c.config.Transport.Args...)

	// Set environment variables
	if c.config.Transport.Env != nil {
		for k, v := range c.config.Transport.Env {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	c.cmd = cmd
	c.stdin = stdin
	c.stdout = stdout
	c.stderr = stderr
	c.scanner = bufio.NewScanner(stdout)

	// Initialize the connection
	if err := c.initialize(); err != nil {
		c.Close()
		return fmt.Errorf("failed to initialize: %w", err)
	}

	c.connected = true
	return nil
}

// initialize sends the initialize request and waits for response
func (c *Client) initialize() error {
	req := Request{
		JSONRPC: "2.0",
		ID:      c.nextRequestID(),
		Method:  "initialize",
		Params: InitializeRequest{
			ProtocolVersion: "2024-11-05",
			Capabilities: Capabilities{
				Tools: &ToolsCapability{},
			},
			ClientInfo: ClientInfo{
				Name:    "zhulong",
				Version: "0.1.0",
			},
		},
	}

	resp, err := c.sendRequest(req)
	if err != nil {
		return fmt.Errorf("initialize request failed: %w", err)
	}

	if resp.Error != nil {
		return fmt.Errorf("initialize error: %s", resp.Error.Message)
	}

	var initResp InitializeResponse
	if err := json.Unmarshal(resp.Result, &initResp); err != nil {
		return fmt.Errorf("failed to parse initialize response: %w", err)
	}

	// Send initialized notification
	notif := Notification{
		JSONRPC: "2.0",
		Method:  "notifications/initialized",
	}
	if err := c.sendNotification(notif); err != nil {
		return fmt.Errorf("failed to send initialized notification: %w", err)
	}

	// Discover tools
	if err := c.discoverTools(); err != nil {
		return fmt.Errorf("failed to discover tools: %w", err)
	}

	return nil
}

// discoverTools discovers available tools from the server
func (c *Client) discoverTools() error {
	req := Request{
		JSONRPC: "2.0",
		ID:      c.nextRequestID(),
		Method:  "tools/list",
	}

	resp, err := c.sendRequest(req)
	if err != nil {
		return fmt.Errorf("tools/list request failed: %w", err)
	}

	if resp.Error != nil {
		return fmt.Errorf("tools/list error: %s", resp.Error.Message)
	}

	var result struct {
		Tools []Tool `json:"tools"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return fmt.Errorf("failed to parse tools/list response: %w", err)
	}

	c.tools = result.Tools
	return nil
}

// CallTool calls a tool on the MCP server
func (c *Client) CallTool(ctx context.Context, name string, args map[string]interface{}) (*ToolResult, error) {
	if !c.connected {
		return nil, fmt.Errorf("not connected to MCP server")
	}

	req := Request{
		JSONRPC: "2.0",
		ID:      c.nextRequestID(),
		Method:  "tools/call",
		Params: ToolCall{
			Name:      name,
			Arguments: args,
		},
	}

	resp, err := c.sendRequestWithContext(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("tools/call request failed: %w", err)
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("tools/call error: %s", resp.Error.Message)
	}

	var result ToolResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to parse tools/call response: %w", err)
	}

	return &result, nil
}

// ListTools returns the list of available tools
func (c *Client) ListTools() []Tool {
	return c.tools
}

// GetTool returns a tool by name
func (c *Client) GetTool(name string) *Tool {
	for _, tool := range c.tools {
		if tool.Name == name {
			return &tool
		}
	}
	return nil
}

// sendRequest sends a request and waits for the response
func (c *Client) sendRequest(req Request) (*Response, error) {
	return c.sendRequestWithContext(context.Background(), req)
}

// sendRequestWithContext sends a request with context and waits for the response
func (c *Client) sendRequestWithContext(ctx context.Context, req Request) (*Response, error) {
	// HTTP transport
	if c.config.Transport.Type == "http" || c.config.Transport.Type == "sse" {
		return c.sendHTTPRequest(req)
	}

	// stdio transport
	c.mu.Lock()
	defer c.mu.Unlock()

	// Marshal request
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Send request
	data = append(data, '\n')
	if _, err := c.stdin.Write(data); err != nil {
		return nil, fmt.Errorf("failed to write request: %w", err)
	}

	// Read response with context
	type result struct {
		resp *Response
		err  error
	}
	ch := make(chan result, 1)

	go func() {
		if c.scanner.Scan() {
			var resp Response
			if err := json.Unmarshal(c.scanner.Bytes(), &resp); err != nil {
				ch <- result{nil, fmt.Errorf("failed to unmarshal response: %w", err)}
				return
			}
			ch <- result{&resp, nil}
		} else {
			if err := c.scanner.Err(); err != nil {
				ch <- result{nil, fmt.Errorf("failed to read response: %w", err)}
			} else {
				ch <- result{nil, fmt.Errorf("connection closed")}
			}
		}
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case r := <-ch:
		return r.resp, r.err
	}
}

// sendNotification sends a notification (no response expected)
func (c *Client) sendNotification(notif Notification) error {
	// HTTP transport
	if c.config.Transport.Type == "http" || c.config.Transport.Type == "sse" {
		data, err := json.Marshal(notif)
		if err != nil {
			return fmt.Errorf("failed to marshal notification: %w", err)
		}
		client := &http.Client{Timeout: 10 * time.Second}
		_, err = client.Post(c.config.Transport.URL, "application/json", bytes.NewReader(data))
		return err
	}

	// stdio transport
	c.mu.Lock()
	defer c.mu.Unlock()

	data, err := json.Marshal(notif)
	if err != nil {
		return fmt.Errorf("failed to marshal notification: %w", err)
	}

	data = append(data, '\n')
	if _, err := c.stdin.Write(data); err != nil {
		return fmt.Errorf("failed to write notification: %w", err)
	}

	return nil
}

// nextRequestID returns the next request ID
func (c *Client) nextRequestID() int {
	c.nextID++
	return c.nextID
}

// Close closes the connection to the MCP server
func (c *Client) Close() error {
	c.cancel()

	if c.stdin != nil {
		c.stdin.Close()
	}
	if c.stdout != nil {
		c.stdout.Close()
	}
	if c.stderr != nil {
		c.stderr.Close()
	}
	if c.cmd != nil && c.cmd.Process != nil {
		c.cmd.Process.Kill()
		c.cmd.Wait()
	}

	c.connected = false
	return nil
}

// IsConnected returns whether the client is connected
func (c *Client) IsConnected() bool {
	return c.connected
}
