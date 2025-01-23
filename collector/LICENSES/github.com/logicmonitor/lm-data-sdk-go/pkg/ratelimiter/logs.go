package ratelimiter

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"
)

const (
	defaultRateLimitLogs = 100
)

// LogRateLimiter represents the RateLimiter config for logs
type LogRateLimiter struct {
	requestCount uint64
	maxCount     uint64
	ticker       *time.Ticker
	shutdownCh   chan struct{}
}
type LogRateLimiterSetting struct {
	RequestCount int
}
type LogPaylaodMetadata struct {
}

// NewLogRateLimiter creates RateLimiter implementation for logs using RateLimiterSetting
func NewLogRateLimiter(setting LogRateLimiterSetting) (*LogRateLimiter, error) {
	if setting.RequestCount == 0 {
		setting.RequestCount = defaultRateLimitLogs
	}
	return &LogRateLimiter{
		requestCount: 0,
		maxCount:     uint64(setting.RequestCount),
		ticker:       time.NewTicker(time.Duration(1 * time.Minute)),
		shutdownCh:   make(chan struct{}, 1),
	}, nil
}

// IncRequestCount increaments the request count associated with logs by 1
func (rateLimiter *LogRateLimiter) IncRequestCount() {
	atomic.AddUint64(&rateLimiter.requestCount, 1)
}

// ResetRequestCount resets the request count associated with logs to 0
func (rateLimiter *LogRateLimiter) ResetRequestCount() {
	atomic.StoreUint64(&rateLimiter.requestCount, 0)
}

// Acquire checks if the requests count for logs is reached to maximum allocated quota per minute.
func (rateLimiter *LogRateLimiter) Acquire(payloadMetadata interface{}) (bool, error) {
	for {
		select {
		case <-rateLimiter.shutdownCh:
			return false, fmt.Errorf("shutdown is called")
		default:
			if rateLimiter.requestCount < rateLimiter.maxCount {
				rateLimiter.IncRequestCount()
				return true, nil
			}
			return false, fmt.Errorf("request quota of (%d) requests per min for the logs is exhausted for the interval", rateLimiter.maxCount)
		}
	}
}

// Run starts the timer for reseting the logs request counter
func (rateLimiter *LogRateLimiter) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			rateLimiter.Shutdown(ctx)
			return
		case <-rateLimiter.ticker.C:
			rateLimiter.ResetRequestCount()
		}
	}
}

// Shutdown triggers the shutdown of the LogRateLimiter
func (rateLimiter *LogRateLimiter) Shutdown(_ context.Context) {
	rateLimiter.shutdownCh <- struct{}{}
}
