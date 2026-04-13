package matchers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

func TestSpanErrorMatcher(t *testing.T) {
	tests := []struct {
		name         string
		requireError bool
		spanHasError bool
		want         bool
	}{
		{
			name:         "rule does not require error matches any span",
			requireError: false,
			spanHasError: false,
			want:         true,
		},
		{
			name:         "rule does not require error matches error span",
			requireError: false,
			spanHasError: true,
			want:         true,
		},
		{
			name:         "rule requires error and span has error matches",
			requireError: true,
			spanHasError: true,
			want:         true,
		},
		{
			name:         "rule requires error and span has no error does not match",
			requireError: true,
			spanHasError: false,
			want:         false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			span := spanWithAttrs(t, nil)
			if tt.spanHasError {
				span.Status().SetCode(ptrace.StatusCodeError)
			}
			got := SpanErrorMatcher(span, tt.requireError)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSpanDurationMatcher(t *testing.T) {
	// 50ms in nanoseconds
	const duration50msNano = 50 * 1e6
	// 100ms in nanoseconds
	const duration100msNano = 100 * 1e6

	tests := []struct {
		name        string
		durationMs  *int
		spanStartNs uint64
		spanEndNs   uint64
		want        bool
	}{
		{
			name:        "nil duration matches any span",
			durationMs:  nil,
			spanStartNs: 0,
			spanEndNs:   1,
			want:        true,
		},
		{
			name:        "span duration equals threshold matches",
			durationMs:  intPtr(50),
			spanStartNs: 0,
			spanEndNs:   duration50msNano,
			want:        true,
		},
		{
			name:        "span duration above threshold matches",
			durationMs:  intPtr(50),
			spanStartNs: 0,
			spanEndNs:   duration100msNano,
			want:        true,
		},
		{
			name:        "span duration below threshold does not match",
			durationMs:  intPtr(100),
			spanStartNs: 0,
			spanEndNs:   duration50msNano,
			want:        false,
		},
		{
			name:        "span duration zero with positive threshold does not match",
			durationMs:  intPtr(1),
			spanStartNs: 1000,
			spanEndNs:   1000,
			want:        false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			span := spanWithAttrs(t, nil)
			span.SetStartTimestamp(pcommon.Timestamp(tt.spanStartNs))
			span.SetEndTimestamp(pcommon.Timestamp(tt.spanEndNs))
			got := SpanDurationMatcher(span, tt.durationMs)
			assert.Equal(t, tt.want, got)
		})
	}
}

func intPtr(i int) *int {
	return &i
}
