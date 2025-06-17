package go_hooks

import (
	"context"
)

const (
	// ZeroTraceContext is a zero trace context.
	ZeroTraceContext = "00-00000000000000000000000000000000-0000000000000000-00"
	// ZeroTraceId is a zero trace ID.
	ZeroTraceId = "00000000000000000000000000000000"
	// ZeroSpanId is a zero span ID.
	ZeroSpanId = "0000000000000000"
)

// GetTraceContext extracts the full W3C trace context from the provided Go context.
// If no span is found in the context, it returns a zero trace context.
//
//go:noinline
func GetW3CTraceContext(ctx context.Context) []byte {
	traceContext := []byte(ZeroTraceContext)
	return traceContext
}

// GetTraceID returns the current trace ID from the provided Go context.
// If no span is found in the context, it returns a zero ID.
//
//go:noinline
func GetTraceID(ctx context.Context) []byte {
	traceId := []byte(ZeroTraceId)
	return traceId
}

// GetSpanID returns the current span ID from the provided Go context.
// If no span is found in the context, it returns a zero ID.
//
//go:noinline
func GetSpanID(ctx context.Context) []byte {
	spanId := []byte(ZeroSpanId)
	return spanId
}

// IsZeroTraceContext checks if the provided trace context is a zero trace context.
func IsZeroTraceContext(traceContext []byte) bool {
	return string(traceContext) == ZeroTraceContext
}

// IsZeroTraceId checks if the provided trace ID is a zero trace ID.
func IsZeroTraceId(traceId []byte) bool {
	return string(traceId) == ZeroTraceId
}

// IsZeroSpanId checks if the provided span ID is a zero span ID.
func IsZeroSpanId(spanId []byte) bool {
	return string(spanId) == ZeroSpanId
}
