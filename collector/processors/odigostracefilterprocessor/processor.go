package odigostracefilterprocessor

import (
	"context"

	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

type traceFilterProcessor struct {
	logger     *zap.Logger
	evaluators []SpanFilterEvaluator
}

func (p *traceFilterProcessor) processTraces(_ context.Context, td ptrace.Traces) (ptrace.Traces, error) {
	if len(p.evaluators) == 0 {
		return td, nil
	}

	// Filter first, then prune. RemoveIf drops every matching element, so calling it
	// while iterating the same slice by index can shrink it by more than one entry and
	// leave the loop indexing out of range.
	for i := 0; i < td.ResourceSpans().Len(); i++ {
		rs := td.ResourceSpans().At(i)

		for j := 0; j < rs.ScopeSpans().Len(); j++ {
			p.filterSpans(rs.ScopeSpans().At(j).Spans())
		}

		rs.ScopeSpans().RemoveIf(func(s ptrace.ScopeSpans) bool {
			return s.Spans().Len() == 0
		})
	}

	td.ResourceSpans().RemoveIf(func(r ptrace.ResourceSpans) bool {
		return r.ScopeSpans().Len() == 0
	})

	return td, nil
}

func (p *traceFilterProcessor) filterSpans(spans ptrace.SpanSlice) {
	spans.RemoveIf(func(span ptrace.Span) bool {
		for _, eval := range p.evaluators {
			if eval.ShouldDrop(span) {
				p.logger.Debug("odigos_trace_filter: dropping span",
					zap.String("span_name", span.Name()),
					zap.Uint32("flags", span.Flags()),
					zap.String("trace_id", span.TraceID().String()),
				)
				return true
			}
		}
		return false
	})
}
