// Package mcp implements the Model Context Protocol (MCP) client for Zhulong.
//
// MCP is a protocol for connecting AI models to external tools and data sources.
// This implementation follows the MCP specification and supports:
// - stdio JSON-RPC transport
// - Tool discovery and invocation
// - Resource access
// - Prompt templates
package mcp

import (
	"encoding/json"
	"time"
)

// JSON-RPC 2.0 types

// Request represents a JSON-RPC 2.0 request
type Request struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// Response represents a JSON-RPC 2.0 response
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

// RPCError represents a JSON-RPC 2.0 error
type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Notification represents a JSON-RPC 2.0 notification (no ID)
type Notification struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// MCP specific types

// ServerInfo contains information about the MCP server
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ClientInfo contains information about the MCP client
type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// InitializeRequest is sent by the client to initialize the connection
type InitializeRequest struct {
	ProtocolVersion string     `json:"protocolVersion"`
	Capabilities    Capabilities `json:"capabilities"`
	ClientInfo      ClientInfo `json:"clientInfo"`
}

// InitializeResponse is received from the server after initialization
type InitializeResponse struct {
	ProtocolVersion string       `json:"protocolVersion"`
	Capabilities    Capabilities `json:"capabilities"`
	ServerInfo      ServerInfo   `json:"serverInfo"`
}

// Capabilities represents the capabilities of client or server
type Capabilities struct {
	Tools    *ToolsCapability    `json:"tools,omitempty"`
	Resources *ResourcesCapability `json:"resources,omitempty"`
	Prompts  *PromptsCapability  `json:"prompts,omitempty"`
}

// ToolsCapability indicates support for tools
type ToolsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// ResourcesCapability indicates support for resources
type ResourcesCapability struct {
	Subscribe   bool `json:"subscribe,omitempty"`
	ListChanged bool `json:"listChanged,omitempty"`
}

// PromptsCapability indicates support for prompts
type PromptsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// Tool represents an MCP tool
type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	InputSchema interface{} `json:"inputSchema,omitempty"`
}

// ToolCall represents a request to call a tool
type ToolCall struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// ToolResult represents the result of a tool call
type ToolResult struct {
	Content []ToolContent `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

// ToolContent represents content returned by a tool
type ToolContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// Resource represents an MCP resource
type Resource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}

// ResourceContent represents the content of a resource
type ResourceContent struct {
	URI      string `json:"uri"`
	MimeType string `json:"mimeType,omitempty"`
	Text     string `json:"text,omitempty"`
	Blob     string `json:"blob,omitempty"`
}

// Prompt represents an MCP prompt template
type Prompt struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Arguments   []PromptArgument `json:"arguments,omitempty"`
}

// PromptArgument represents an argument for a prompt template
type PromptArgument struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

// PromptMessage represents a message in a prompt result
type PromptMessage struct {
	Role    string      `json:"role"`
	Content PromptContent `json:"content"`
}

// PromptContent represents the content of a prompt message
type PromptContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// Transport types

// TransportConfig contains configuration for the MCP transport
type TransportConfig struct {
	Type    string            `json:"type"`    // "stdio", "sse", "http"
	Command string            `json:"command,omitempty"` // For stdio
	Args    []string          `json:"args,omitempty"`    // For stdio
	URL     string            `json:"url,omitempty"`     // For sse/http
	Env     map[string]string `json:"env,omitempty"`     // Environment variables
	Timeout time.Duration     `json:"timeout,omitempty"`
}

// ServerConfig contains configuration for an MCP server
type ServerConfig struct {
	Name      string          `json:"name"`
	Transport TransportConfig `json:"transport"`
	Enabled   bool            `json:"enabled"`
}
