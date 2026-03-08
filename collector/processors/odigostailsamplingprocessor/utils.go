package odigostailsamplingprocessor

import (
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// getRootSpan finds and returns the root span of the trace.
// the trace should be all spans belonging to a single trace id,
// as reported by a "groupbytraceid" processor.
// returns the root span if found, the resource of the root span, and a boolean indicating if the root span was found.
func getRootSpan(trace ptrace.Traces) (ptrace.Span, pcommon.Resource, bool) {
	resourceSpans := trace.ResourceSpans()
	for i := 0; i < resourceSpans.Len(); i++ {
		resourceSpan := resourceSpans.At(i)
		scopeSpans := resourceSpan.ScopeSpans()
		for j := 0; j < scopeSpans.Len(); j++ {
			scopeSpan := scopeSpans.At(j)
			spans := scopeSpan.Spans()
			for k := 0; k < spans.Len(); k++ {
				span := spans.At(k)
				if span.ParentSpanID().IsEmpty() {
					return span, resourceSpan.Resource(), true
				}
			}
		}
	}
	return ptrace.Span{}, pcommon.Resource{}, false
}

// add few span attributes to all spans in the trace to indicate the sampling info.
func enrichSpansWithSamplingAttributes(td ptrace.Traces, category string, ruleId string, keepPercentage float64) {
	for i := 0; i < td.ResourceSpans().Len(); i++ {
		resourceSpan := td.ResourceSpans().At(i)
		scopeSpans := resourceSpan.ScopeSpans()
		for j := 0; j < scopeSpans.Len(); j++ {
			scopeSpan := scopeSpans.At(j)
			spans := scopeSpan.Spans()
			for k := 0; k < spans.Len(); k++ {
				span := spans.At(k)
				span.Attributes().PutStr("odigos.sampling.category", category)
				span.Attributes().PutStr("odigos.sampling.trace.deciding_rule.id", ruleId)
				span.Attributes().PutDouble("odigos.sampling.trace.deciding_rule.keep_percentage", keepPercentage)
			}
		}
	}
}
