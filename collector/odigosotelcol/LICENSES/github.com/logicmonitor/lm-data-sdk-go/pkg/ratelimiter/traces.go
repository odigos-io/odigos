package ratelimiter

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"
)

const (
	defaultRequestsPerMinuteLimit = 2000
	defaultSpansPerRequestLimit   = 3000
	defaultSpansPerMinuteLimit    = 139000
)

// TraceRateLimiter represents the RateLimiter config for traces
type TraceRateLimiter struct {
	spanCount              uint64
	requestCount           uint64
	requestSpanCount       uint64
	maxRequestCount        uint64
	maxSpanCount           uint64
	maxSpanCountPerRequest uint64
	ticker                 *time.Ticker
	shutdownCh             chan struct{}
}

type TraceRateLimiterSetting struct {
	RequestCount        int
	SpanCount           int
	SpanCountPerRequest int
}

type TracePayloadMetadata struct {
	RequestSpanCount uint64
}

// NewTraceRateLimiter creates RateLimiter implementation for traces using RateLimiterSetting
func NewTraceRateLimiter(setting TraceRateLimiterSetting) (*TraceRateLimiter, error) {
	if setting.RequestCount == 0 {
		setting.RequestCount = defaultRequestsPerMinuteLimit
	}
	if setting.SpanCount == 0 {
		setting.SpanCount = defaultSpansPerMinuteLimit
	}
	if setting.SpanCountPerRequest == 0 {
		setting.SpanCountPerRequest = defaultSpansPerRequestLimit
	}
	return &TraceRateLimiter{
		maxRequestCount:        uint64(setting.RequestCount),
		maxSpanCount:           uint64(setting.SpanCount),
		maxSpanCountPerRequest: uint64(setting.SpanCountPerRequest),
		ticker:                 time.NewTicker(time.Duration(1 * time.Minute)),
		shutdownCh:             make(chan struct{}, 1),
	}, nil
}

// IncRequestCount increments the request count associated with traces by 1
func (rateLimiter *TraceRateLimiter) incRequestCount() {
	atomic.AddUint64(&rateLimiter.requestCount, 1)
}

// IncSpanCount increments the span count associated with traces by no. of spans
func (rateLimiter *TraceRateLimiter) incSpanCount(requestSpanCount uint64) {
	atomic.AddUint64(&rateLimiter.spanCount, requestSpanCount)
}

// ResetRequestCount resets the request count associated with traces to 0
func (rateLimiter *TraceRateLimiter) resetRequestCount() {
	atomic.StoreUint64(&rateLimiter.requestCount, 0)
}

// ResetSpanCount resets the span count associated with traces to 0
func (rateLimiter *TraceRateLimiter) resetSpanCount() {
	atomic.StoreUint64(&rateLimiter.spanCount, 0)
}

// Acquire checks if the requests count for traces is reached to maximum allocated quota per minute.
func (rateLimiter *TraceRateLimiter) Acquire(payloadMetadata interface{}) (bool, error) {
	tracePayloadMetadata, ok := payloadMetadata.(TracePayloadMetadata)
	if !ok {
		return false, fmt.Errorf("payload metadata is not of type TracePaylaodMetadata")
	}
	select {
	case <-rateLimiter.shutdownCh:
		return false, fmt.Errorf("shutdown is called")
	default:
		// check request count rate limit
		if atomic.LoadUint64(&rateLimiter.requestCount) >= rateLimiter.maxRequestCount {
			return false, fmt.Errorf("request quota of requests per min for the traces is exhausted for the interval")
		}
		// check spans per request rate limit
		if tracePayloadMetadata.RequestSpanCount > rateLimiter.maxSpanCountPerRequest {
			return false, fmt.Errorf("request quota of span count per request for the traces is exhausted")
		}

		// check span count over a duration rate limit
		if atomic.LoadUint64(&rateLimiter.spanCount)+tracePayloadMetadata.RequestSpanCount > rateLimiter.maxSpanCount {
			return false, fmt.Errorf("request quota of span count per min for the traces is exhausted for the interval")
		}

		// if rate limit is not triggered, update counters with new data
		rateLimiter.incRequestCount()
		rateLimiter.incSpanCount(tracePayloadMetadata.RequestSpanCount)
	}
	return true, nil
}

// Run starts the timer for reseting the traces request counter
func (rateLimiter *TraceRateLimiter) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			rateLimiter.Shutdown(ctx)
			return
		case <-rateLimiter.ticker.C:
			rateLimiter.resetRequestCount()
			rateLimiter.resetSpanCount()
		}
	}
}

// Shutdown triggers the shutdown of the LogRateLimiter
func (rateLimiter *TraceRateLimiter) Shutdown(_ context.Context) {
	rateLimiter.shutdownCh <- struct{}{}
}
