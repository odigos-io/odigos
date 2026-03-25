package odigostailsamplingprocessor

import (
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// assertAllSpansBelongToTheSameTrace asserts that all spans in the batch belong to the same trace.
// The processor should be placed in the pipeline after the "groupbytraceid" processor.
func assertAllSpansBelongToTheSameTrace(td ptrace.Traces) (pcommon.TraceID, bool) {
	traceID := td.ResourceSpans().At(0).ScopeSpans().At(0).Spans().At(0).TraceID()
	for i := 0; i < td.ResourceSpans().Len(); i++ {
		for j := 0; j < td.ResourceSpans().At(i).ScopeSpans().Len(); j++ {
			for k := 0; k < td.ResourceSpans().At(i).ScopeSpans().At(j).Spans().Len(); k++ {
				if td.ResourceSpans().At(i).ScopeSpans().At(j).Spans().At(k).TraceID() != traceID {
					return pcommon.TraceID{}, false
				}
			}
		}
	}
	return traceID, true
}
