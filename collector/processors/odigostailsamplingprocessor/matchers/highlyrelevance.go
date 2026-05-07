package matchers

import (
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// SpanErrorMatcher returns true when the rule does not require an error, or when it does and the span has error status.
func SpanErrorMatcher(span ptrace.Span, requireError bool) bool {
	if !requireError {
		return true
	}
	return span.Status().Code() == ptrace.StatusCodeError
}

// SpanDurationMatcher returns true when no minimum duration is required, or when the span duration is at least the given threshold in milliseconds.
func SpanDurationMatcher(span ptrace.Span, durationMs *int) bool {
	if durationMs == nil {
		return true
	}
	spanDurationNano := uint64(span.EndTimestamp() - span.StartTimestamp())
	thresholdNano := uint64(*durationMs) * 1e6
	return spanDurationNano >= thresholdNano
}
