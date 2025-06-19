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
func GetW3CTraceContext(ctx context.Context) string {
	traceContext := getW3CTraceContextBytes(ctx)
	return string(traceContext)
}

// GetTraceID returns the current trace ID from the provided Go context.
// If no span is found in the context, it returns a zero ID.
//
//go:noinline
func GetTraceID(ctx context.Context) string {
	traceId := getTraceIDBytes(ctx)
	return string(traceId)
}

// GetSpanID returns the current span ID from the provided Go context.
// If no span is found in the context, it returns a zero ID.
//
//go:noinline
func GetSpanID(ctx context.Context) string {
	spanId := getSpanIDBytes(ctx)
	return string(spanId)
}

// IsZeroTraceContext checks if the provided trace context is a zero trace context.
func IsZeroTraceContext(traceContext string) bool {
	return traceContext == ZeroTraceContext
}

// IsZeroTraceId checks if the provided trace ID is a zero trace ID.
func IsZeroTraceId(traceId string) bool {
	return traceId == ZeroTraceId
}

// IsZeroSpanId checks if the provided span ID is a zero span ID.
func IsZeroSpanId(spanId string) bool {
	return spanId == ZeroSpanId
}

//go:noinline
func getW3CTraceContextBytes(ctx context.Context) []byte {
	traceContext := []byte(ZeroTraceContext)
	return traceContext
}

//go:noinline
func getTraceIDBytes(ctx context.Context) []byte {
	traceId := []byte(ZeroTraceId)
	return traceId
}

//go:noinline
func getSpanIDBytes(ctx context.Context) []byte {
	spanId := []byte(ZeroSpanId)
	return spanId
}
