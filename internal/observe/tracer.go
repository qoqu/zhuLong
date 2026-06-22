// 可观测性 - trace/span/insights
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

// ComputeInsights 汇总 spans 得出整体洞察
// 修正: 之前误写为 time.Since(time.Now())（参数反了，结果恒为0/负）
// 现在: 取所有 span 的时间跨度
func ComputeInsights(spans []*Span) *Insights {
	if len(spans) == 0 {
		return &Insights{}
	}

	// 计算总时间: 从最早 StartTime 到最晚 EndTime
	// 关键修复: 之前是 time.Since(time.Now())，now 是当前时间，time.Since(未来的now) 是 0 或负数
	var earliest, latest time.Time
	tokenSum := 0
	modelCalls := 0

	for _, s := range spans {
		if earliest.IsZero() || s.StartTime.Before(earliest) {
			earliest = s.StartTime
		}
		endT := s.EndTime
		if endT.IsZero() {
			endT = time.Now()
		}
		if latest.IsZero() || endT.After(latest) {
			latest = endT
		}

		// 累加: span 标签里的 token 消耗
		if v, ok := s.Tags["tokens"]; ok {
			var t int
			fmt.Sscanf(v, "%d", &t)
			tokenSum += t
		}
		if _, ok := s.Tags["model"]; ok {
			modelCalls++
		}
	}

	totalTime := latest.Sub(earliest)
	if totalTime < 0 {
		totalTime = 0
	}

	// 粗略成本估算: 按 DeepSeek 1元/百万token 计
	const costPerMillionTokens = 1.0
	cost := float64(tokenSum) / 1_000_000.0 * costPerMillionTokens

	return &Insights{
		TokenUsage:   tokenSum,
		CostEstimate: cost,
		TotalTime:    totalTime,
		ModelCalls:   modelCalls,
	}
}
