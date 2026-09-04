package collector

import "go.opentelemetry.io/collector/pdata/pcommon"

// InterrogationCacheExtension correlates leaf-first profile stacks with spans.
type InterrogationCacheExtension interface {
	RecordSample(traceID pcommon.TraceID, spanID pcommon.SpanID, frames []string)
	GetSamples(traceID pcommon.TraceID, spanID pcommon.SpanID) (samples [][]string, ok bool)
	// TakeSamples atomically returns and removes a match, preventing trace replay
	// from producing duplicate evidence.
	TakeSamples(traceID pcommon.TraceID, spanID pcommon.SpanID) (samples [][]string, ok bool)
}
