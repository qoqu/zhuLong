package compressor

import (
	"fmt"
	"testing"
)

// simpleTokenCounter 实现简单的token计数器
type simpleTokenCounter struct{}

func (c *simpleTokenCounter) Count(text string) int {
	return len(text) / 4 // 简单估算：4个字符约等于1个token
}

// BenchmarkPrune 测试修剪性能
func BenchmarkPrune(b *testing.B) {
	counter := &simpleTokenCounter{}
	compressor := NewSimpleCompressor(counter, DefaultConfig())

	// 创建测试消息
	messages := createTestMessages(100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		compressor.Prune(messages, 10)
	}
}

// BenchmarkPruneLarge 测试大消息列表修剪性能
func BenchmarkPruneLarge(b *testing.B) {
	counter := &simpleTokenCounter{}
	compressor := NewSimpleCompressor(counter, DefaultConfig())

	// 创建大测试消息列表
	messages := createTestMessages(1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		compressor.Prune(messages, 10)
	}
}

// BenchmarkAssembleContext 测试上下文组装性能
func BenchmarkAssembleContext(b *testing.B) {
	counter := &simpleTokenCounter{}
	compressor := NewSimpleCompressor(counter, DefaultConfig())

	// 创建测试数据
	systemPrompt := createStablePrefix()
	skeleton := createSkeleton()
	sessionSummary := createSessionSummary()
	workingMessages := createWorkingMessages(50)
	currentInput := "Analyze the current project structure"
	maxTokens := 8000

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		compressor.AssembleContext(systemPrompt, skeleton, sessionSummary, workingMessages, currentInput, maxTokens)
	}
}

// BenchmarkAssembleContextLarge 测试大上下文组装性能
func BenchmarkAssembleContextLarge(b *testing.B) {
	counter := &simpleTokenCounter{}
	compressor := NewSimpleCompressor(counter, DefaultConfig())

	// 创建大测试数据
	systemPrompt := createStablePrefix()
	skeleton := createSkeleton()
	sessionSummary := createSessionSummary()
	workingMessages := createWorkingMessages(200)
	currentInput := "Analyze the current project structure and provide detailed recommendations"
	maxTokens := 16000

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		compressor.AssembleContext(systemPrompt, skeleton, sessionSummary, workingMessages, currentInput, maxTokens)
	}
}

// BenchmarkCacheHitRate 测试缓存命中率
func BenchmarkCacheHitRate(b *testing.B) {
	counter := &simpleTokenCounter{}
	compressor := NewSimpleCompressor(counter, DefaultConfig())

	// 创建稳定的前缀
	systemPrompt := createStablePrefix()
	skeleton := createSkeleton()
	sessionSummary := createSessionSummary()
	currentInput := "Analyze the current project structure"
	maxTokens := 8000

	// 第一次组装（建立缓存基准）
	compressor.AssembleContext(systemPrompt, skeleton, sessionSummary, nil, currentInput, maxTokens)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// 使用相同的前缀和不同的工作消息
		workingMessages := createWorkingMessages(50 + i%10)
		compressor.AssembleContext(systemPrompt, skeleton, sessionSummary, workingMessages, currentInput, maxTokens)
	}
}

// createTestMessages 创建测试消息
func createTestMessages(count int) []Message {
	messages := make([]Message, count)
	for i := 0; i < count; i++ {
		messages[i] = Message{
			Role:       "tool",
			Content:    fmt.Sprintf("Tool result %d: Successfully processed data", i),
			ToolName:   fmt.Sprintf("tool_%d", i%5),
			LoopNumber: i / 10,
			Tokens:     100,
			Prunable:   i%3 == 0, // 每3个消息中有1个可修剪
		}
	}
	return messages
}

// createStablePrefix 创建稳定的前缀
func createStablePrefix() string {
	return `You are Zhulong, an autonomous AI agent framework.
Your goal is to help users accomplish tasks through planning, execution, and reflection.

Available tools:
- read_file: Read the contents of a file
- write_file: Write content to a file
- search_file: Search for files matching a pattern
- execute_command: Execute a shell command

`
}

// createSkeleton 创建骨架
func createSkeleton() string {
	return `## Current Session
- Goal: Analyze the project structure
- Status: executing
- Loop: 1

## Plan
1. Scan project files
2. Analyze code structure
3. Generate report

`
}

// createSessionSummary 创建会话摘要
func createSessionSummary() string {
	return `## Session Summary
- Completed 5 steps
- Used 1000 tokens
- Duration: 2 minutes

`
}

// createWorkingMessages 创建工作消息
func createWorkingMessages(count int) []Message {
	messages := make([]Message, count)
	for i := 0; i < count; i++ {
		messages[i] = Message{
			Role:       "tool",
			Content:    fmt.Sprintf("Result %d: Processed data chunk successfully", i),
			ToolName:   fmt.Sprintf("tool_%d", i%3),
			LoopNumber: i / 5,
			Tokens:     50,
			Prunable:   true,
		}
	}
	return messages
}
