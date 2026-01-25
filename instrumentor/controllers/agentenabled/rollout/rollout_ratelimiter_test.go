package rollout

import (
	"testing"

	"github.com/odigos-io/odigos/common"
	"github.com/stretchr/testify/assert"
)

func Test_NilConfig(t *testing.T) {
	// Arrange: nil config
	limiter := NewRolloutRateLimiter(nil)

	// Assert: limiter is not nil, all Allow calls should succeed (no rate limiting when config is nil)
	assert.NotNil(t, limiter)
	for i := 0; i < 100; i++ {
		assert.True(t, limiter.Allow(), "Allow() should return true for call %d when config is nil", i+1)
	}
}

func Test_NilRolloutConfig(t *testing.T) {
	// Arrange: nil rollout config
	conf := &common.OdigosConfiguration{
		Rollout: nil,
	}
	limiter := NewRolloutRateLimiter(conf)

	// Assert: limiter is not nil, no Allow calls should be made
	assert.NotNil(t, limiter)
}

func Test_CustomValues(t *testing.T) {
	// Arrange: custom values
	enabled := true
	conf := &common.OdigosConfiguration{
		Rollout: &common.RolloutConfiguration{
			IsConcurrentRolloutsEnabled: &enabled,
			ConcurrentRollouts:          3.0,
		},
	}
	limiter := NewRolloutRateLimiter(conf)

	// Assert: limiter is not nil, first 3 Allow calls should succeed, 4th should fail (burst exhausted)
	assert.NotNil(t, limiter)
	for i := 0; i < 3; i++ {
		assert.True(t, limiter.Allow(), "Allow() should return true for call %d", i+1)
	}
	assert.False(t, limiter.Allow())
}

func Test_NilReceiver_Allow(t *testing.T) {
	// Arrange: nil receiver
	var limiter *RolloutRateLimiter

	// Assert: nil receiver should return true (fail-open)
	assert.True(t, limiter.Allow())
}

func Test_SingleConcurrentRollout(t *testing.T) {
	// Arrange: single concurrent rollout
	enabled := true
	conf := &common.OdigosConfiguration{
		Rollout: &common.RolloutConfiguration{
			IsConcurrentRolloutsEnabled: &enabled,
			ConcurrentRollouts:          1.0,
		},
	}
	limiter := NewRolloutRateLimiter(conf)

	// Assert: first request should succeed, second request should fail (burst exhausted)
	assert.True(t, limiter.Allow())
	assert.False(t, limiter.Allow())
}

func Test_RateLimitingEnabled(t *testing.T) {
	// Arrange: rate limiting enabled (isConcurrentRolloutsEnabled: true, concurrentRollouts: 5.0)
	enabled := true
	conf := &common.OdigosConfiguration{
		Rollout: &common.RolloutConfiguration{
			IsConcurrentRolloutsEnabled: &enabled,
			ConcurrentRollouts:          5.0,
		},
	}
	limiter := NewRolloutRateLimiter(conf)

	// Assert: first 5 Allow calls should succeed, 6th should fail
	assert.NotNil(t, limiter)
	for i := 0; i < 5; i++ {
		assert.True(t, limiter.Allow(), "Allow() should return true for call %d", i+1)
	}
	assert.False(t, limiter.Allow())
}

func Test_RateLimitingEnabled_DefaultValues(t *testing.T) {
	// Arrange: rate limiting enabled (isConcurrentRolloutsEnabled: true, concurrentRollouts: 0.0)
	enabled := true
	conf := &common.OdigosConfiguration{
		Rollout: &common.RolloutConfiguration{
			IsConcurrentRolloutsEnabled: &enabled,
			ConcurrentRollouts:          0.0,
		},
	}
	limiter := NewRolloutRateLimiter(conf)

	// Assert: first 5 Allow calls should succeed, 6th should fail
	assert.NotNil(t, limiter)
	for i := 0; i < 5; i++ {
		assert.True(t, limiter.Allow(), "Allow() should return true for call %d", i+1)
	}
	assert.False(t, limiter.Allow())
}

func Test_RateLimitingDisabled(t *testing.T) {
	// Arrange: rate limiting explicitly disabled (isConcurrentRolloutsEnabled: false)
	disabled := false
	conf := &common.OdigosConfiguration{
		Rollout: &common.RolloutConfiguration{
			IsConcurrentRolloutsEnabled: &disabled,
			ConcurrentRollouts:          5.0, // Value should be ignored when disabled
		},
	}
	limiter := NewRolloutRateLimiter(conf)

	// Assert: all requests should succeed (no rate limiting)
	assert.NotNil(t, limiter)
	for i := 0; i < 100; i++ {
		assert.True(t, limiter.Allow(), "Allow() should return true for call %d when rate limiting is disabled", i+1)
	}
}
