package common

import (
	"testing"

	"go.opentelemetry.io/otel/trace"
)

// Test case structure
type spanKindTestCase struct {
	name     string
	input    SpanKind
	expected trace.SpanKind
}

// TestSpanKindOdigosToOtel refactors the individual tests into a single table-driven test
func TestSpanKindOdigosToOtel(t *testing.T) {
	// Define your test cases
	testCases := []spanKindTestCase{
		{"Client", ClientSpanKind, trace.SpanKindClient},
		{"Server", ServerSpanKind, trace.SpanKindServer},
		{"Producer", ProducerSpanKind, trace.SpanKindProducer},
		{"Consumer", ConsumerSpanKind, trace.SpanKindConsumer},
		{"Internal", InternalSpanKind, trace.SpanKindInternal},
		{"Unspecified", SpanKind(""), trace.SpanKindUnspecified},
	}

	// Iterate over the test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := SpanKindOdigosToOtel(tc.input)
			if got != tc.expected {
				t.Errorf("SpanKindOdigosToOtel(%v) = %v, want %v", tc.input, got, tc.expected)
			}
		})
	}
}
