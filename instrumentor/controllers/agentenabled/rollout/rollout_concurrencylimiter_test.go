package rollout

import (
	"testing"

	"github.com/odigos-io/odigos/common"
	"github.com/stretchr/testify/assert"
)

func Test_NilConfig(t *testing.T) {
	// Arrange: nil config
	limiter := NewRolloutConcurrencyLimiter()
	limiter.ApplyConfig(nil)

	// Assert: limiter is not nil, all TryAcquire calls should succeed (no rate limiting when config is nil)
	assert.NotNil(t, limiter)
	for i := 0; i < 100; i++ {
		key := workloadKey("ns", "Deployment", i)
		assert.True(t, limiter.TryAcquire(key), "TryAcquire() should return true for workload %d when config is nil", i+1)
	}
}

func Test_NilRolloutConfig(t *testing.T) {
	// Arrange: nil rollout config
	conf := &common.OdigosConfiguration{
		Rollout: nil,
	}
	limiter := NewRolloutConcurrencyLimiter()
	limiter.ApplyConfig(conf)

	// Assert: limiter is not nil
	assert.NotNil(t, limiter)
}

func Test_CustomValues(t *testing.T) {
	// Arrange: custom values
	enabled := true
	conf := &common.OdigosConfiguration{
		Rollout: &common.RolloutConfiguration{
			IsConcurrentRolloutsLimiterEnabled: &enabled,
			ConcurrentRollouts:                 3,
		},
	}
	limiter := NewRolloutConcurrencyLimiter()
	limiter.ApplyConfig(conf)

	// Assert: limiter is not nil, first 3 TryAcquire calls should succeed, 4th should fail
	assert.NotNil(t, limiter)
	for i := 0; i < 3; i++ {
		key := workloadKey("ns", "Deployment", i)
		assert.True(t, limiter.TryAcquire(key), "TryAcquire() should return true for workload %d", i+1)
	}
	assert.False(t, limiter.TryAcquire(workloadKey("ns", "Deployment", 3)))
}

func Test_NilReceiver_TryAcquire(t *testing.T) {
	// Arrange: nil receiver
	var limiter *RolloutConcurrencyLimiter

	// Assert: nil receiver should return true (fail-open)
	assert.True(t, limiter.TryAcquire("test/Deployment/test"))
}

func Test_SingleConcurrentRollout(t *testing.T) {
	// Arrange: single concurrent rollout
	enabled := true
	conf := &common.OdigosConfiguration{
		Rollout: &common.RolloutConfiguration{
			IsConcurrentRolloutsLimiterEnabled: &enabled,
			ConcurrentRollouts:                 1,
		},
	}
	limiter := NewRolloutConcurrencyLimiter()
	limiter.ApplyConfig(conf)

	// Assert: first request should succeed, second (different workload) should fail
	assert.True(t, limiter.TryAcquire("ns/Deployment/app1"))
	assert.False(t, limiter.TryAcquire("ns/Deployment/app2"))
}

func Test_SameWorkloadCanReacquire(t *testing.T) {
	// Arrange: single concurrent rollout
	enabled := true
	conf := &common.OdigosConfiguration{
		Rollout: &common.RolloutConfiguration{
			IsConcurrentRolloutsLimiterEnabled: &enabled,
			ConcurrentRollouts:                 1,
		},
	}
	limiter := NewRolloutConcurrencyLimiter()
	limiter.ApplyConfig(conf)

	// Assert: same workload can re-acquire its existing slot
	assert.True(t, limiter.TryAcquire("ns/Deployment/app1"))
	assert.True(t, limiter.TryAcquire("ns/Deployment/app1"))  // Should succeed - already has slot
	assert.False(t, limiter.TryAcquire("ns/Deployment/app2")) // Different workload - denied
}

func Test_RateLimitingEnabled(t *testing.T) {
	// Arrange: rate limiting enabled (isConcurrentRolloutsLimiterEnabled: true, concurrentRollouts: 5)
	enabled := true
	conf := &common.OdigosConfiguration{
		Rollout: &common.RolloutConfiguration{
			IsConcurrentRolloutsLimiterEnabled: &enabled,
			ConcurrentRollouts:                 5,
		},
	}
	limiter := NewRolloutConcurrencyLimiter()
	limiter.ApplyConfig(conf)

	// Assert: first 5 TryAcquire calls should succeed, 6th should fail
	assert.NotNil(t, limiter)
	for i := 0; i < 5; i++ {
		key := workloadKey("ns", "Deployment", i)
		assert.True(t, limiter.TryAcquire(key), "TryAcquire() should return true for workload %d", i+1)
	}
	assert.False(t, limiter.TryAcquire(workloadKey("ns", "Deployment", 5)))
}

func Test_RateLimitingEnabled_DefaultValues(t *testing.T) {
	// Arrange: rate limiting enabled (isConcurrentRolloutsLimiterEnabled: true, concurrentRollouts: 0)
	enabled := true
	conf := &common.OdigosConfiguration{
		Rollout: &common.RolloutConfiguration{
			IsConcurrentRolloutsLimiterEnabled: &enabled,
			ConcurrentRollouts:                 0,
		},
	}
	limiter := NewRolloutConcurrencyLimiter()
	limiter.ApplyConfig(conf)

	// Assert: first 5 TryAcquire calls should succeed (default limit), 6th should fail
	assert.NotNil(t, limiter)
	for i := 0; i < 5; i++ {
		key := workloadKey("ns", "Deployment", i)
		assert.True(t, limiter.TryAcquire(key), "TryAcquire() should return true for workload %d", i+1)
	}
	assert.False(t, limiter.TryAcquire(workloadKey("ns", "Deployment", 5)))
}

func Test_RateLimitingDisabled(t *testing.T) {
	// Arrange: rate limiting explicitly disabled (isConcurrentRolloutsLimiterEnabled: false)
	disabled := false
	conf := &common.OdigosConfiguration{
		Rollout: &common.RolloutConfiguration{
			IsConcurrentRolloutsLimiterEnabled: &disabled,
			ConcurrentRollouts:                 5, // Value should be ignored when disabled
		},
	}
	limiter := NewRolloutConcurrencyLimiter()
	limiter.ApplyConfig(conf)

	// Assert: all requests should succeed (no rate limiting)
	assert.NotNil(t, limiter)
	for i := 0; i < 100; i++ {
		key := workloadKey("ns", "Deployment", i)
		assert.True(t, limiter.TryAcquire(key), "TryAcquire() should return true for workload %d when rate limiting is disabled", i+1)
	}
}

func Test_Release_ReturnsSlot(t *testing.T) {
	// Arrange: rate limiter with 1 concurrent rollout
	enabled := true
	conf := &common.OdigosConfiguration{
		Rollout: &common.RolloutConfiguration{
			IsConcurrentRolloutsLimiterEnabled: &enabled,
			ConcurrentRollouts:                 1,
		},
	}
	limiter := NewRolloutConcurrencyLimiter()
	limiter.ApplyConfig(conf)

	key1 := "ns/Deployment/app1"
	key2 := "ns/Deployment/app2"

	// Act: acquire slot for app1, then release it
	assert.True(t, limiter.TryAcquire(key1), "first TryAcquire() should succeed")
	assert.False(t, limiter.TryAcquire(key2), "second TryAcquire() should fail (exhausted)")
	limiter.Release(key1)

	// Assert: slot is returned, app2 can now acquire
	assert.True(t, limiter.TryAcquire(key2), "TryAcquire() should succeed after Release()")
}

func Test_Release_MultipleSlots(t *testing.T) {
	// Arrange: rate limiter with 3 concurrent rollouts
	enabled := true
	conf := &common.OdigosConfiguration{
		Rollout: &common.RolloutConfiguration{
			IsConcurrentRolloutsLimiterEnabled: &enabled,
			ConcurrentRollouts:                 3,
		},
	}
	limiter := NewRolloutConcurrencyLimiter()
	limiter.ApplyConfig(conf)

	keys := []string{
		"ns/Deployment/app1",
		"ns/Deployment/app2",
		"ns/Deployment/app3",
		"ns/Deployment/app4",
		"ns/Deployment/app5",
	}

	// Act: consume all 3 slots
	for i := 0; i < 3; i++ {
		assert.True(t, limiter.TryAcquire(keys[i]), "TryAcquire() %d should succeed", i+1)
	}
	assert.False(t, limiter.TryAcquire(keys[3]), "4th TryAcquire() should fail (exhausted)")

	// Release 2 slots
	limiter.Release(keys[0])
	limiter.Release(keys[1])

	// Assert: 2 slots available again
	assert.True(t, limiter.TryAcquire(keys[3]), "TryAcquire() should succeed after first Release()")
	assert.True(t, limiter.TryAcquire(keys[4]), "TryAcquire() should succeed after second Release()")
	assert.False(t, limiter.TryAcquire("ns/Deployment/app6"), "TryAcquire() should fail (exhausted again)")
}

func Test_Release_WhenNoSlotHeld(t *testing.T) {
	// Arrange: rate limiter with slots available (none consumed)
	enabled := true
	conf := &common.OdigosConfiguration{
		Rollout: &common.RolloutConfiguration{
			IsConcurrentRolloutsLimiterEnabled: &enabled,
			ConcurrentRollouts:                 2,
		},
	}
	limiter := NewRolloutConcurrencyLimiter()
	limiter.ApplyConfig(conf)

	// Act: call Release() for workloads that don't have slots (should be no-op)
	limiter.Release("ns/Deployment/nonexistent1")
	limiter.Release("ns/Deployment/nonexistent2")
	limiter.Release("ns/Deployment/nonexistent3")

	// Assert: limiter still works correctly, allows up to 2 concurrent rollouts
	assert.True(t, limiter.TryAcquire("ns/Deployment/app1"), "TryAcquire() 1 should succeed")
	assert.True(t, limiter.TryAcquire("ns/Deployment/app2"), "TryAcquire() 2 should succeed")
	assert.False(t, limiter.TryAcquire("ns/Deployment/app3"), "TryAcquire() 3 should fail (limit is 2)")
}

func Test_NilReceiver_Release(t *testing.T) {
	// Arrange: nil receiver
	var limiter *RolloutConcurrencyLimiter

	// Act & Assert: nil receiver should not panic (no-op)
	assert.NotPanics(t, func() {
		limiter.Release("test/Deployment/test")
	})
}

func Test_HasSlot(t *testing.T) {
	enabled := true
	conf := &common.OdigosConfiguration{
		Rollout: &common.RolloutConfiguration{
			IsConcurrentRolloutsLimiterEnabled: &enabled,
			ConcurrentRollouts:                 2,
		},
	}
	limiter := NewRolloutConcurrencyLimiter()
	limiter.ApplyConfig(conf)

	key := "ns/Deployment/app1"

	// Initially no slot
	assert.False(t, limiter.HasSlot(key))

	// After acquiring
	limiter.TryAcquire(key)
	assert.True(t, limiter.HasSlot(key))

	// After releasing
	limiter.Release(key)
	assert.False(t, limiter.HasSlot(key))
}

func Test_InFlightCount(t *testing.T) {
	enabled := true
	conf := &common.OdigosConfiguration{
		Rollout: &common.RolloutConfiguration{
			IsConcurrentRolloutsLimiterEnabled: &enabled,
			ConcurrentRollouts:                 5,
		},
	}
	limiter := NewRolloutConcurrencyLimiter()
	limiter.ApplyConfig(conf)

	assert.Equal(t, 0, limiter.InFlightCount())

	limiter.TryAcquire("ns/Deployment/app1")
	assert.Equal(t, 1, limiter.InFlightCount())

	limiter.TryAcquire("ns/Deployment/app2")
	limiter.TryAcquire("ns/Deployment/app3")
	assert.Equal(t, 3, limiter.InFlightCount())

	limiter.Release("ns/Deployment/app1")
	assert.Equal(t, 2, limiter.InFlightCount())
}

// Helper to generate unique workload keys
func workloadKey(namespace, kind string, index int) string {
	return namespace + "/" + kind + "/app" + string(rune('0'+index))
}
