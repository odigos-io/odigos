package odigostracefilterprocessor

import (
	"context"

	"go.opentelemetry.io/collector/pdata/pcommon"
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

	for i := td.ResourceSpans().Len() - 1; i >= 0; i-- {
		rs := td.ResourceSpans().At(i)

		for j := rs.ScopeSpans().Len() - 1; j >= 0; j-- {
			ss := rs.ScopeSpans().At(j)
			p.filterSpans(rs.Resource(), ss.Spans())
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

func (p *traceFilterProcessor) filterSpans(resource pcommon.Resource, spans ptrace.SpanSlice) {
	spans.RemoveIf(func(span ptrace.Span) bool {
		for _, eval := range p.evaluators {
			if eval.ShouldDrop(resource, span) {
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
