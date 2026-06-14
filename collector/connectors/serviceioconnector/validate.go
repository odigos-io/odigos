package serviceioconnector

import (
	"errors"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// validateCompleteTraceBatch verifies that td is a complete trace batch as expected from groupbytrace.
// All spans in the batch must share one trace ID.
func validateCompleteTraceBatch(td ptrace.Traces) error {
	var traceID pcommon.TraceID
	hasSpan := false

	for i := 0; i < td.ResourceSpans().Len(); i++ {
		resourceSpan := td.ResourceSpans().At(i)
		for j := 0; j < resourceSpan.ScopeSpans().Len(); j++ {
			scopeSpan := resourceSpan.ScopeSpans().At(j)
			for k := 0; k < scopeSpan.Spans().Len(); k++ {
				currTraceID := scopeSpan.Spans().At(k).TraceID()
				if !hasSpan {
					traceID = currTraceID
					hasSpan = true
					continue
				}
				if currTraceID != traceID {
					return errors.New("not all spans belong to the same trace")
				}
			}
		}
	}

	return nil
}
