package ratelimiter

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewMetricsRateLimiter(t *testing.T) {
	setting := MetricsRateLimiterSetting{
		RequestCount: 100,
	}
	metricsRateLimiter, err := NewMetricsRateLimiter(setting)
	assert.NoError(t, err)
	assert.Equal(t, uint64(setting.RequestCount), metricsRateLimiter.maxCount)
	assert.Equal(t, uint64(0), metricsRateLimiter.requestCount)
}

func TestNewMetricsRateLimiterDefaultRequestCount(t *testing.T) {
	setting := MetricsRateLimiterSetting{}
	metricsRateLimiter, err := NewMetricsRateLimiter(setting)
	assert.NoError(t, err)
	assert.Equal(t, uint64(defaultRateLimitLogs), metricsRateLimiter.maxCount)
	assert.Equal(t, uint64(0), metricsRateLimiter.requestCount)
}

func TestIncRequestCount_Metrics(t *testing.T) {
	setting := MetricsRateLimiterSetting{
		RequestCount: 100,
	}
	metricsRateLimiter, err := NewMetricsRateLimiter(setting)
	assert.NoError(t, err)
	metricsRequestCountBeforeInc := metricsRateLimiter.requestCount
	metricsRateLimiter.IncRequestCount()
	metricsRequestCountAfterInc := metricsRateLimiter.requestCount
	assert.Equal(t, metricsRequestCountBeforeInc+1, metricsRequestCountAfterInc)
}

func TestResetRequestCount_Metrics(t *testing.T) {
	setting := MetricsRateLimiterSetting{
		RequestCount: 100,
	}
	metricsRateLimiter, err := NewMetricsRateLimiter(setting)
	assert.NoError(t, err)
	metricsRateLimiter.ResetRequestCount()
	assert.Equal(t, uint64(0), metricsRateLimiter.requestCount)
}

func TestAcquire_Metrics(t *testing.T) {
	t.Run("should return error when span count in request crosses maximum allowed", func(t *testing.T) {
		setting := MetricsRateLimiterSetting{
			RequestCount: 1,
		}
		metricsRateLimiter, err := NewMetricsRateLimiter(setting)
		assert.NoError(t, err)

		// Should allow 1 request
		ok, err := metricsRateLimiter.Acquire(MetricsPaylaodMetadata{})
		assert.Equal(t, true, ok)
		assert.NoError(t, err)

		// Should drop the second request as quota is exhausted
		ok, err = metricsRateLimiter.Acquire(MetricsPaylaodMetadata{})
		assert.Equal(t, false, ok)
		assert.Error(t, err)
	})

	t.Run("should return error when shutdown is called", func(t *testing.T) {
		setting := MetricsRateLimiterSetting{
			RequestCount: 100,
		}
		metricsRateLimiter, err := NewMetricsRateLimiter(setting)
		assert.NoError(t, err)
		metricsRateLimiter.Shutdown(context.Background())
		ok, err := metricsRateLimiter.Acquire(MetricsPaylaodMetadata{})
		assert.Equal(t, false, ok)
		assert.Error(t, err)
	})
}
func TestRun_Metrics(t *testing.T) {
	setting := MetricsRateLimiterSetting{
		RequestCount: 100,
	}
	metricsRateLimiter, err := NewMetricsRateLimiter(setting)
	assert.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()
	metricsRateLimiter.ticker.Reset(3 * time.Second)
	go metricsRateLimiter.Run(ctx)
	metricsRateLimiter.IncRequestCount()
	time.Sleep(5 * time.Second)
	assert.Equal(t, uint64(0), metricsRateLimiter.requestCount)
	metricsRateLimiter.Shutdown(ctx)
}
