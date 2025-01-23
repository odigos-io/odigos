package ratelimiter

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewTraceRateLimiter(t *testing.T) {
	setting := TraceRateLimiterSetting{
		RequestCount:        100,
		SpanCount:           100,
		SpanCountPerRequest: 100,
	}
	traceRateLimiter, err := NewTraceRateLimiter(setting)
	assert.NoError(t, err)
	assert.Equal(t, uint64(setting.RequestCount), traceRateLimiter.maxRequestCount)
	assert.Equal(t, uint64(setting.SpanCount), traceRateLimiter.maxSpanCount)
	assert.Equal(t, uint64(setting.SpanCountPerRequest), traceRateLimiter.maxSpanCountPerRequest)
	assert.Equal(t, uint64(0), traceRateLimiter.requestCount)
	assert.Equal(t, uint64(0), traceRateLimiter.spanCount)
	assert.Equal(t, uint64(0), traceRateLimiter.requestSpanCount)
}

func TestIncRequestCount_Traces(t *testing.T) {
	setting := TraceRateLimiterSetting{
		RequestCount: 100,
	}
	traceRateLimiter, err := NewTraceRateLimiter(setting)
	assert.NoError(t, err)
	traceRequestCountBeforeInc := traceRateLimiter.requestCount
	traceRateLimiter.incRequestCount()
	traceRequestCountAfterInc := traceRateLimiter.requestCount
	assert.Equal(t, traceRequestCountBeforeInc+1, traceRequestCountAfterInc)
}

func TestResetRequestCount_Traces(t *testing.T) {
	setting := TraceRateLimiterSetting{
		RequestCount: 100,
	}
	traceRateLimiter, err := NewTraceRateLimiter(setting)
	assert.NoError(t, err)
	traceRateLimiter.resetRequestCount()
	assert.Equal(t, uint64(0), traceRateLimiter.requestCount)
}

func TestIncSpanCount_Traces(t *testing.T) {
	setting := TraceRateLimiterSetting{
		SpanCount: 100,
	}
	traceRateLimiter, err := NewTraceRateLimiter(setting)
	assert.NoError(t, err)
	traceSpanCountBeforeInc := traceRateLimiter.spanCount
	traceRateLimiter.incSpanCount(10)
	traceSpanCountAfterInc := traceRateLimiter.spanCount
	assert.Equal(t, traceSpanCountBeforeInc+10, traceSpanCountAfterInc)
}

func TestResetSpanCount_Traces(t *testing.T) {
	setting := TraceRateLimiterSetting{
		SpanCount: 100,
	}
	traceRateLimiter, err := NewTraceRateLimiter(setting)
	assert.NoError(t, err)
	traceRateLimiter.resetSpanCount()
	assert.Equal(t, uint64(0), traceRateLimiter.spanCount)
}

func TestAcquire_Traces(t *testing.T) {
	t.Run("should acquire lock when request count is within limit and should return error when request count crosses maximum allowed requests", func(t *testing.T) {
		setting := TraceRateLimiterSetting{
			RequestCount:        1,
			SpanCount:           2,
			SpanCountPerRequest: 2,
		}
		traceRateLimiter, err := NewTraceRateLimiter(setting)
		assert.NoError(t, err)

		ok, err := traceRateLimiter.Acquire(TracePayloadMetadata{RequestSpanCount: 2})
		assert.Equal(t, true, ok)
		assert.NoError(t, err)

		// Should drop the second request as quota is exhausted
		ok, err = traceRateLimiter.Acquire(TracePayloadMetadata{RequestSpanCount: 2})
		assert.Equal(t, false, ok)
		assert.Error(t, err)
	})

	t.Run("should return error when total span count crosses maximum span count", func(t *testing.T) {
		setting := TraceRateLimiterSetting{
			RequestCount:        2,
			SpanCount:           3,
			SpanCountPerRequest: 2,
		}
		traceRateLimiter, err := NewTraceRateLimiter(setting)
		assert.NoError(t, err)

		ok, err := traceRateLimiter.Acquire(TracePayloadMetadata{RequestSpanCount: 2})
		t.Log(err)
		assert.Equal(t, true, ok)
		assert.NoError(t, err)

		// Should drop the second request as quota is exhausted
		ok, err = traceRateLimiter.Acquire(TracePayloadMetadata{RequestSpanCount: 2})
		assert.Equal(t, false, ok)
		assert.Error(t, err)
	})

	t.Run("should return error when span count in request crosses maximum allowed", func(t *testing.T) {
		setting := TraceRateLimiterSetting{
			RequestCount:        2,
			SpanCount:           10,
			SpanCountPerRequest: 2,
		}
		traceRateLimiter, err := NewTraceRateLimiter(setting)
		assert.NoError(t, err)

		ok, err := traceRateLimiter.Acquire(TracePayloadMetadata{RequestSpanCount: 3})
		assert.Equal(t, false, ok)
		assert.Error(t, err)
	})
	t.Run("should return error when shutdown is called", func(t *testing.T) {
		setting := TraceRateLimiterSetting{
			RequestCount: 100,
		}
		traceRateLimiter, err := NewTraceRateLimiter(setting)
		assert.NoError(t, err)
		traceRateLimiter.Shutdown(context.Background())
		ok, err := traceRateLimiter.Acquire(TracePayloadMetadata{})
		assert.Equal(t, false, ok)
		assert.Error(t, err)
	})
}
func TestRun_Traces(t *testing.T) {
	setting := TraceRateLimiterSetting{
		RequestCount: 100,
	}
	traceRateLimiter, err := NewTraceRateLimiter(setting)
	assert.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()
	traceRateLimiter.ticker.Reset(3 * time.Second)
	go traceRateLimiter.Run(ctx)
	traceRateLimiter.incRequestCount()
	time.Sleep(5 * time.Second)
	assert.Equal(t, uint64(0), traceRateLimiter.requestCount)
	traceRateLimiter.Shutdown(ctx)
}
