package go_hooks

import (
	"context"
)

// GetTraceContext extracts the full W3C trace context from the provided Go context.
// If no span is found in the context, it returns a zero trace context.
//
//go:noinline
func GetW3CTraceContext(ctx context.Context) []byte {
	traceContext := []byte("00-00000000000000000000000000000000-0000000000000000-00")
	return traceContext
}

// GetTraceID returns the current trace ID from the provided Go context.
// If no span is found in the context, it returns a zero ID.
//
//go:noinline
func GetTraceID(ctx context.Context) []byte {
	traceId := []byte("00000000000000000000000000000000")
	return traceId
}

// GetSpanID returns the current span ID from the provided Go context.
// If no span is found in the context, it returns a zero ID.
//
//go:noinline
func GetSpanID(ctx context.Context) []byte {
	spanId := []byte("0000000000000000")
	return spanId
}
