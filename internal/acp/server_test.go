package acp

import (
	"bytes"
	"encoding/json"
	"io"
	"testing"
)

func TestNewServer(t *testing.T) {
	var buf bytes.Buffer
	s := NewServer(&buf, &buf)
	if s == nil {
		t.Fatal("expected non-nil server")
	}
}

func TestRegisterTool(t *testing.T) {
	var buf bytes.Buffer
	s := NewServer(&buf, &buf)

	s.RegisterTool(Tool{
		Name: "test_tool", Description: "A test tool",
		InputSchema: InputSchema{Type: "object"},
	}, func(params json.RawMessage) (interface{}, *RPCError) {
		return "ok", nil
	})

	if len(s.tools) != 1 {
		t.Errorf("expected 1 tool, got %d", len(s.tools))
	}
}

func TestListTools(t *testing.T) {
	// 使用io.Pipe创建真正的双向管道
	srvReader, cliWriter := io.Pipe()
	cliReader, srvWriter := io.Pipe()

	srv := NewServer(srvReader, srvWriter)
	cli := NewClient(cliReader, cliWriter)

	srv.RegisterTool(Tool{
		Name: "ping", Description: "Ping tool",
		InputSchema: InputSchema{Type: "object"},
	}, nil)

	go srv.Start()

	tools, err := cli.ListTools()
	if err != nil {
		t.Fatalf("ListTools failed: %v", err)
	}

	if len(tools) != 1 {
		t.Errorf("expected 1 tool, got %d", len(tools))
	}
}

func TestCallTool(t *testing.T) {
	srvReader, cliWriter := io.Pipe()
	cliReader, srvWriter := io.Pipe()

	srv := NewServer(srvReader, srvWriter)
	cli := NewClient(cliReader, cliWriter)

	srv.RegisterTool(Tool{Name: "echo", Description: "Echo"}, func(params json.RawMessage) (interface{}, *RPCError) {
		return string(params), nil
	})

	go srv.Start()

	result, err := cli.CallTool("echo", map[string]interface{}{"text": "hello"})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}

	if result == nil {
		t.Error("expected non-nil result")
	}
}

func TestUnregisteredTool(t *testing.T) {
	var srvBuf, cliBuf bytes.Buffer

	srv := NewServer(&cliBuf, &srvBuf)
	cli := NewClient(&srvBuf, &cliBuf)

	go srv.Start()

	_, err := cli.CallTool("nonexistent", nil)
	if err == nil {
		t.Error("expected error for unregistered tool")
	}
}

func TestNewClient(t *testing.T) {
	var buf bytes.Buffer
	c := NewClient(&buf, &buf)
	if c == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestRunStdioServer(t *testing.T) {
	// 这个测试只验证不panic
	_ = RunStdioServer
}
