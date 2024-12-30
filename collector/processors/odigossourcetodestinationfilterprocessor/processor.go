package odigossourcetodestinationfilterprocessor

import (
	"context"

	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

type filterProcessor struct {
	logger *zap.Logger
	config *Config
}

func (fp *filterProcessor) processTraces(ctx context.Context, td ptrace.Traces) (ptrace.Traces, error) {
	rspans := td.ResourceSpans()

	for i := 0; i < rspans.Len(); i++ {
		resourceSpan := rspans.At(i)
		ilSpans := resourceSpan.ScopeSpans()

		for j := 0; j < ilSpans.Len(); j++ {
			scopeSpan := ilSpans.At(j)
			spans := scopeSpan.Spans()

			spans.RemoveIf(func(span ptrace.Span) bool {
				return !fp.matches(span, resourceSpan)
			})
		}
	}

	return td, nil
}

func (fp *filterProcessor) matches(span ptrace.Span, resourceSpan ptrace.ResourceSpans) bool {
	attributes := resourceSpan.Resource().Attributes()

	name, _ := attributes.Get("name")
	namespace, _ := attributes.Get("namespace")
	kind, _ := attributes.Get("kind")

	for _, condition := range fp.config.MatchConditions {
		if name.AsString() == condition.Name &&
			namespace.AsString() == condition.Namespace &&
			kind.AsString() == condition.Kind {
			return true
		}
	}

	return false
}
