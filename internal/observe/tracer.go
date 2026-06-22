// 可观测性 - trace/span/insights（参考Hermes Langfuse集成）
package observe

import (
	"fmt"
	"sync"
	"time"
)

// Span 跨度
type Span struct {
	ID        string
	Name      string
	StartTime time.Time
	EndTime   time.Time
	Tags      map[string]string
	Events    []SpanEvent
}

// SpanEvent 跨度事件
type SpanEvent struct {
	Time    time.Time
	Message string
	Level   string
}

// Tracer 追踪器
type Tracer struct {
	mu     sync.Mutex
	spans  []*Span
	active map[string]*Span
}

func NewTracer() *Tracer {
	return &Tracer{active: make(map[string]*Span)}
}

func (t *Tracer) StartSpan(name string) *Span {
	t.mu.Lock()
	defer t.mu.Unlock()

	s := &Span{
		ID: fmt.Sprintf("span-%d", time.Now().UnixNano()),
		Name: name, StartTime: time.Now(),
		Tags: make(map[string]string), Events: make([]SpanEvent, 0),
	}
	t.spans = append(t.spans, s)
	t.active[s.ID] = s
	return s
}

func (t *Tracer) EndSpan(id string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if s, ok := t.active[id]; ok {
		s.EndTime = time.Now()
		delete(t.active, id)
	}
}

func (t *Tracer) Spans() []*Span {
	t.mu.Lock()
	defer t.mu.Unlock()
	cp := make([]*Span, len(t.spans))
	copy(cp, t.spans)
	return cp
}

// Insights 分析
type Insights struct {
	TokenUsage   int
	CostEstimate float64
	TotalTime    time.Duration
	ModelCalls   int
}

func ComputeInsights(spans []*Span) *Insights {
	return &Insights{
		TokenUsage:   0,
		CostEstimate: 0,
		TotalTime:    time.Since(time.Now()),
		ModelCalls:   len(spans),
	}
}
