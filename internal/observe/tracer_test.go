package observe

import (
	"testing"
)

func TestNewTracer(t *testing.T) {
	tr := NewTracer()
	if tr == nil {
		t.Fatal("expected non-nil tracer")
	}
}

func TestStartEndSpan(t *testing.T) {
	tr := NewTracer()
	s := tr.StartSpan("test")
	tr.EndSpan(s.ID)

	if s.EndTime.IsZero() {
		t.Error("expected end time to be set")
	}
}

func TestSpans(t *testing.T) {
	tr := NewTracer()
	tr.StartSpan("a")
	tr.StartSpan("b")

	spans := tr.Spans()
	if len(spans) != 2 {
		t.Errorf("expected 2 spans, got %d", len(spans))
	}
}

func TestComputeInsights(t *testing.T) {
	i := ComputeInsights(nil)
	if i == nil {
		t.Fatal("expected non-nil insights")
	}
}
