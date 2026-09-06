package matchers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"

	commonapisampling "github.com/odigos-io/odigos/common/api/sampling"
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
			got := NewSpanErrorMatcher(tt.requireError).Match(span)
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
			got := NewSpanDurationMatcher(tt.durationMs).Match(span)
			assert.Equal(t, tt.want, got)
		})
	}
}

func intPtr(i int) *int {
	return &i
}

// NewHighlyRelevantOperationMatcher AND-s the operation, error and duration conditions of a rule.
// A rule that matched on any one of them instead would keep far more traces than the user asked
// for, and a rule that lost a condition would keep the wrong ones.
func TestNewHighlyRelevantOperationMatcher(t *testing.T) {
	checkoutOperation := &commonapisampling.TailSamplingOperationMatcher{
		HttpServer: &commonapisampling.TailSamplingHttpServerOperationMatcher{Route: "/checkout"},
	}

	tests := []struct {
		name         string
		operation    *commonapisampling.TailSamplingOperationMatcher
		requireError bool
		durationMs   *int
		spanRoute    string
		spanHasError bool
		spanEndNs    uint64
		want         bool
	}{
		{
			// a rule with no condition at all keeps every trace of the workload at its percentage.
			name:      "no condition matches any span",
			spanRoute: "/cart",
			want:      true,
		},
		{
			name:      "operation only matches the operation",
			operation: checkoutOperation,
			spanRoute: "/checkout",
			want:      true,
		},
		{
			name:      "operation only does not match another operation",
			operation: checkoutOperation,
			spanRoute: "/cart",
			want:      false,
		},
		{
			name:         "error only matches an error span on any operation",
			requireError: true,
			spanRoute:    "/cart",
			spanHasError: true,
			want:         true,
		},
		{
			name:         "error only does not match a healthy span",
			requireError: true,
			spanRoute:    "/cart",
			want:         false,
		},
		{
			name:       "duration only matches a slow span on any operation",
			durationMs: intPtr(100),
			spanRoute:  "/cart",
			spanEndNs:  200 * 1e6,
			want:       true,
		},
		{
			name:       "duration only does not match a fast span",
			durationMs: intPtr(100),
			spanRoute:  "/cart",
			spanEndNs:  50 * 1e6,
			want:       false,
		},
		{
			name:         "all conditions hold",
			operation:    checkoutOperation,
			requireError: true,
			durationMs:   intPtr(100),
			spanRoute:    "/checkout",
			spanHasError: true,
			spanEndNs:    200 * 1e6,
			want:         true,
		},
		{
			name:         "operation and duration hold but the error does not",
			operation:    checkoutOperation,
			requireError: true,
			durationMs:   intPtr(100),
			spanRoute:    "/checkout",
			spanEndNs:    200 * 1e6,
			want:         false,
		},
		{
			name:         "operation and error hold but the duration does not",
			operation:    checkoutOperation,
			requireError: true,
			durationMs:   intPtr(100),
			spanRoute:    "/checkout",
			spanHasError: true,
			spanEndNs:    50 * 1e6,
			want:         false,
		},
		{
			name:         "error and duration hold but the operation does not",
			operation:    checkoutOperation,
			requireError: true,
			durationMs:   intPtr(100),
			spanRoute:    "/cart",
			spanHasError: true,
			spanEndNs:    200 * 1e6,
			want:         false,
		},
		{
			// requireError false must not add a condition that rejects healthy spans.
			name:         "an explicit false error requirement matches a healthy span",
			operation:    checkoutOperation,
			requireError: false,
			spanRoute:    "/checkout",
			want:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			span := spanWithAttrsAndKind(t, ptrace.SpanKindServer, map[string]string{
				"http.request.method": "GET",
				"http.route":          tt.spanRoute,
			})
			if tt.spanHasError {
				span.Status().SetCode(ptrace.StatusCodeError)
			}
			span.SetStartTimestamp(0)
			span.SetEndTimestamp(pcommon.Timestamp(tt.spanEndNs))

			matcher := NewHighlyRelevantOperationMatcher(tt.operation, tt.requireError, tt.durationMs)
			assert.Equal(t, tt.want, matcher.Match(span))
		})
	}
}
