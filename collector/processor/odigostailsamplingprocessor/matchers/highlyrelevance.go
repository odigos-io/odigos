package matchers

import (
	"go.opentelemetry.io/collector/pdata/ptrace"

	commonapisampling "github.com/odigos-io/odigos/common/api/sampling"
)

func NewHighlyRelevantOperationMatcher(operation *commonapisampling.TailSamplingOperationMatcher, requireError bool, durationMs *int) Matcher {
	var matchers []Matcher
	if operation != nil {
		matchers = append(matchers, NewTailSamplingOperationMatcher(operation))
	}
	if requireError {
		matchers = append(matchers, NewSpanErrorMatcher(requireError))
	}
	if durationMs != nil {
		matchers = append(matchers, NewSpanDurationMatcher(durationMs))
	}
	if len(matchers) == 0 {
		return anyMatcher{}
	}
	return newCompositeMatcher(matchers...)
}

type spanErrorMatcher struct {
	requireError bool
}

func NewSpanErrorMatcher(requireError bool) Matcher {
	return &spanErrorMatcher{requireError: requireError}
}

// Match returns true when the rule does not require an error, or when it does and the span has error status.
func (m *spanErrorMatcher) Match(span ptrace.Span) bool {
	if !m.requireError {
		return true
	}
	return span.Status().Code() == ptrace.StatusCodeError
}

type spanDurationMatcher struct {
	durationMs *int
}

func NewSpanDurationMatcher(durationMs *int) Matcher {
	return &spanDurationMatcher{durationMs: durationMs}
}

// Match returns true when no minimum duration is required, or when the span duration is at least the given threshold in milliseconds.
func (m *spanDurationMatcher) Match(span ptrace.Span) bool {
	if m.durationMs == nil {
		return true
	}
	spanDurationNano := uint64(span.EndTimestamp() - span.StartTimestamp())
	thresholdNano := uint64(*m.durationMs) * 1e6
	return spanDurationNano >= thresholdNano
}
