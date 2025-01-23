package ratelimiter

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewLogRateLimiter(t *testing.T) {
	setting := LogRateLimiterSetting{
		RequestCount: 100,
	}
	logRateLimiter, err := NewLogRateLimiter(setting)
	assert.NoError(t, err)
	assert.Equal(t, uint64(setting.RequestCount), logRateLimiter.maxCount)
	assert.Equal(t, uint64(0), logRateLimiter.requestCount)
}

func TestNewLogRateLimiterDefaultRequestCount(t *testing.T) {
	setting := LogRateLimiterSetting{}
	logRateLimiter, err := NewLogRateLimiter(setting)
	assert.NoError(t, err)
	assert.Equal(t, uint64(defaultRateLimitLogs), logRateLimiter.maxCount)
	assert.Equal(t, uint64(0), logRateLimiter.requestCount)
}

func TestIncRequestCount_Logs(t *testing.T) {
	setting := LogRateLimiterSetting{
		RequestCount: 100,
	}
	logRateLimiter, err := NewLogRateLimiter(setting)
	assert.NoError(t, err)
	logRequestCountBeforeInc := logRateLimiter.requestCount
	logRateLimiter.IncRequestCount()
	logRequestCountAfterInc := logRateLimiter.requestCount
	assert.Equal(t, logRequestCountBeforeInc+1, logRequestCountAfterInc)
}

func TestResetRequestCount_Logs(t *testing.T) {
	setting := LogRateLimiterSetting{
		RequestCount: 100,
	}
	logRateLimiter, err := NewLogRateLimiter(setting)
	assert.NoError(t, err)
	logRateLimiter.ResetRequestCount()
	assert.Equal(t, uint64(0), logRateLimiter.requestCount)
}

func TestAcquire_Logs(t *testing.T) {
	t.Run("should acquire lock when request count is within limit and should return error when request count crosses maximum allowed requests", func(t *testing.T) {
		setting := LogRateLimiterSetting{
			RequestCount: 1,
		}
		logRateLimiter, err := NewLogRateLimiter(setting)
		assert.NoError(t, err)

		// Should allow 1 request
		ok, err := logRateLimiter.Acquire(LogPaylaodMetadata{})
		assert.Equal(t, true, ok)
		assert.NoError(t, err)

		// Should drop the second request as quota is exhausted
		ok, err = logRateLimiter.Acquire(LogPaylaodMetadata{})
		assert.Equal(t, false, ok)
		assert.Error(t, err)
	})
	t.Run("should return error when shutdown is called", func(t *testing.T) {
		setting := LogRateLimiterSetting{
			RequestCount: 100,
		}
		logRateLimiter, err := NewLogRateLimiter(setting)
		assert.NoError(t, err)
		logRateLimiter.Shutdown(context.Background())
		ok, err := logRateLimiter.Acquire(LogPaylaodMetadata{})
		assert.Equal(t, false, ok)
		assert.Error(t, err)
	})
}
func TestRun_Logs(t *testing.T) {
	setting := LogRateLimiterSetting{
		RequestCount: 100,
	}
	logRateLimiter, err := NewLogRateLimiter(setting)
	assert.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()
	logRateLimiter.ticker.Reset(3 * time.Second)
	go logRateLimiter.Run(ctx)
	logRateLimiter.IncRequestCount()
	time.Sleep(5 * time.Second)
	assert.Equal(t, uint64(0), logRateLimiter.requestCount)
	logRateLimiter.Shutdown(ctx)
}
