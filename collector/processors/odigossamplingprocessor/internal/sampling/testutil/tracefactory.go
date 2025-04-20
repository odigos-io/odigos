package testutil

import (
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

type TraceBuilder struct {
	traces ptrace.Traces
}

func NewTrace() *TraceBuilder {
	return &TraceBuilder{
		traces: ptrace.NewTraces(),
	}
}

func (tb *TraceBuilder) AddResource(serviceName string) *ResourceBuilder {
	rs := tb.traces.ResourceSpans().AppendEmpty()
	if serviceName != "" {
		rs.Resource().Attributes().PutStr("service.name", serviceName)
	}
	return &ResourceBuilder{tb, rs}
}

func (tb *TraceBuilder) AddEmptyResource() *ResourceBuilder {
	rs := tb.traces.ResourceSpans().AppendEmpty()
	return &ResourceBuilder{tb, rs}
}

func (tb *TraceBuilder) Build() ptrace.Traces {
	return tb.traces
}

type ResourceBuilder struct {
	tb *TraceBuilder
	rs ptrace.ResourceSpans
}

func (rb *ResourceBuilder) AddSpan(name string, opts ...SpanOption) *ResourceBuilder {
	ss := rb.rs.ScopeSpans().AppendEmpty()
	span := ss.Spans().AppendEmpty()
	span.SetName(name)

	start := time.Now()
	end := start.Add(10 * time.Millisecond)
	span.SetStartTimestamp(pcommon.NewTimestampFromTime(start))
	span.SetEndTimestamp(pcommon.NewTimestampFromTime(end))

	for _, opt := range opts {
		opt(span)
	}
	return rb
}

func (rb *ResourceBuilder) Done() *TraceBuilder {
	return rb.tb
}

type SpanOption func(span ptrace.Span)

func WithStatus(code ptrace.StatusCode) SpanOption {
	return func(span ptrace.Span) {
		span.Status().SetCode(code)
	}
}

func WithAttribute(key, value string) SpanOption {
	return func(span ptrace.Span) {
		span.Attributes().PutStr(key, value)
	}
}

func WithLatency(d time.Duration) SpanOption {
	return func(span ptrace.Span) {
		start := time.Now()
		span.SetStartTimestamp(pcommon.NewTimestampFromTime(start))
		span.SetEndTimestamp(pcommon.NewTimestampFromTime(start.Add(d)))
	}
}
