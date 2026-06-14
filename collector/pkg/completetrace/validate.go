package completetrace

import (
	"errors"
	"strings"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

const OdigosTraceStateKey = "odigos"

// ValidateCompleteTrace verifies that td is a complete trace batch as expected from groupbytrace.
// All spans in the batch should share one trace ID.
//
// Validation:
//   - If spans disagree on trace ID, returns a non-nil error (misconfiguration or upstream bug).
//   - If any span's W3C tracestate contains the odigos vendor entry (see OdigosTraceStateKey),
//     head sampling already ran; returns (_, false, nil) so processing is skipped.
//   - If there are no spans, returns (_, false, nil) so the batch is skipped.
//
// Returns:
//   - traceID: the common trace ID when shouldProcess is true, or the zero value when skipping or on error.
//   - shouldProcess: true only when there is at least one span, all share traceID, and none carry odigos tracestate.
//   - spanCount: number of spans in the batch.
//   - err: non-nil only when multiple trace IDs appear in the same batch.
func ValidateCompleteTrace(td ptrace.Traces) (pcommon.TraceID, bool, int, error) {
	var traceID pcommon.TraceID
	spanCount := 0

	for i := 0; i < td.ResourceSpans().Len(); i++ {
		resourceSpan := td.ResourceSpans().At(i)
		for j := 0; j < resourceSpan.ScopeSpans().Len(); j++ {
			scopeSpan := resourceSpan.ScopeSpans().At(j)
			for k := 0; k < scopeSpan.Spans().Len(); k++ {
				span := scopeSpan.Spans().At(k)

				currTraceID := span.TraceID()
				if spanCount == 0 {
					traceID = currTraceID
					spanCount++
				} else if currTraceID != traceID {
					return pcommon.TraceID{}, false, 0, errors.New("not all spans belong to the same trace")
				}

				if ExtractOdigosTraceStateValue(span.TraceState().AsRaw()) != "" {
					return pcommon.TraceID{}, false, 0, nil
				}
			}
		}
	}

	return traceID, spanCount > 0, spanCount, nil
}

// ExtractOdigosTraceStateValue extracts the value of the "odigos" key from a W3C tracestate string.
// Tracestate format: "key1=value1,key2=value2,..."
func ExtractOdigosTraceStateValue(traceState string) string {
	for _, entry := range strings.Split(traceState, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		k, v, found := strings.Cut(entry, "=")
		if !found {
			continue
		}
		if strings.TrimSpace(k) == OdigosTraceStateKey {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
