package rollout

import (
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
)

func Test_CustomValues(t *testing.T) {
	// Arrange: custom values
	limiter := NewRolloutConcurrencyLimiter(logr.Discard().WithName("RolloutConcurrencyLimiter"))

	// Assert: limiter is not nil, first 3 TryAcquire calls should succeed, 4th should fail
	assert.NotNil(t, limiter)
	for i := 0; i < 3; i++ {
		key := workloadKey("ns", "Deployment", i)
		assert.True(t, limiter.TryAcquire(key, 3), "TryAcquire() should return true for workload %d", i+1)
	}
	assert.False(t, limiter.TryAcquire(workloadKey("ns", "Deployment", 3), 3))
}

func Test_NilReceiver_TryAcquire(t *testing.T) {
	// Arrange: nil receiver
	var limiter *RolloutConcurrencyLimiter

	// Assert: nil receiver should return true (fail-open)
	assert.True(t, limiter.TryAcquire("test/Deployment/test", 0))
}

func Test_SingleConcurrentRollout(t *testing.T) {
	// Arrange: single concurrent rollout
	limiter := NewRolloutConcurrencyLimiter(logr.Discard().WithName("RolloutConcurrencyLimiter"))

	// Assert: first request should succeed, second (different workload) should fail
	assert.True(t, limiter.TryAcquire("ns/Deployment/app1", 1))
	assert.False(t, limiter.TryAcquire("ns/Deployment/app2", 1))
}

func Test_SameWorkloadCanReacquire(t *testing.T) {
	// Arrange: single concurrent rollout
	limiter := NewRolloutConcurrencyLimiter(logr.Discard().WithName("RolloutConcurrencyLimiter"))

	// Assert: same workload can re-acquire its existing slot
	assert.True(t, limiter.TryAcquire("ns/Deployment/app1", 1))
	assert.True(t, limiter.TryAcquire("ns/Deployment/app1", 1))  // Should succeed - already has slot
	assert.False(t, limiter.TryAcquire("ns/Deployment/app2", 1)) // Different workload - denied
}

func Test_RateLimitingEnabled(t *testing.T) {
	// Arrange: rate limiting enabled (MaxConcurrentRollouts: 5)
	limiter := NewRolloutConcurrencyLimiter(logr.Discard().WithName("RolloutConcurrencyLimiter"))

	// Assert: first 5 TryAcquire calls should succeed, 6th should fail
	assert.NotNil(t, limiter)
	for i := 0; i < 5; i++ {
		key := workloadKey("ns", "Deployment", i)
		assert.True(t, limiter.TryAcquire(key, 5), "TryAcquire() should return true for workload %d", i+1)
	}
	assert.False(t, limiter.TryAcquire(workloadKey("ns", "Deployment", 5), 5))
}

func Test_RateLimitingDisabled_ZeroValue(t *testing.T) {
	// Arrange: rate limiting disabled (MaxConcurrentRollouts: 0 means unlimited)
	limiter := NewRolloutConcurrencyLimiter(logr.Discard().WithName("RolloutConcurrencyLimiter"))

	// Assert: all requests should succeed (no rate limiting when MaxConcurrentRollouts is 0)
	assert.NotNil(t, limiter)
	for i := 0; i < 100; i++ {
		key := workloadKey("ns", "Deployment", i)
		assert.True(t, limiter.TryAcquire(key, 0), "TryAcquire() should return true for workload %d when rate limiting is disabled", i+1)
	}
}

func Test_Release_ReturnsSlot(t *testing.T) {
	// Arrange: rate limiter with 1 concurrent rollout
	limiter := NewRolloutConcurrencyLimiter(logr.Discard().WithName("RolloutConcurrencyLimiter"))

	key1 := "ns/Deployment/app1"
	key2 := "ns/Deployment/app2"

	// Act: acquire slot for app1, then release it
	assert.True(t, limiter.TryAcquire(key1, 1), "first TryAcquire() should succeed")
	assert.False(t, limiter.TryAcquire(key2, 1), "second TryAcquire() should fail (exhausted)")
	limiter.ReleaseWorkloadRolloutSlot(key1)

	// Assert: slot is returned, app2 can now acquire
	assert.True(t, limiter.TryAcquire(key2, 1), "TryAcquire() should succeed after Release()")
}

func Test_Release_MultipleSlots(t *testing.T) {
	// Arrange: rate limiter with 3 concurrent rollouts
	limiter := NewRolloutConcurrencyLimiter(logr.Discard().WithName("RolloutConcurrencyLimiter"))

	keys := []string{
		"ns/Deployment/app1",
		"ns/Deployment/app2",
		"ns/Deployment/app3",
		"ns/Deployment/app4",
		"ns/Deployment/app5",
	}

	// Act: consume all 3 slots
	for i := 0; i < 3; i++ {
		assert.True(t, limiter.TryAcquire(keys[i], 3), "TryAcquire() %d should succeed", i+1)
	}
	assert.False(t, limiter.TryAcquire(keys[3], 3), "4th TryAcquire() should fail (exhausted)")

	// Release 2 slots
	limiter.ReleaseWorkloadRolloutSlot(keys[0])
	limiter.ReleaseWorkloadRolloutSlot(keys[1])

	// Assert: 2 slots available again
	assert.True(t, limiter.TryAcquire(keys[3], 3), "TryAcquire() should succeed after first Release()")
	assert.True(t, limiter.TryAcquire(keys[4], 3), "TryAcquire() should succeed after second Release()")
	assert.False(t, limiter.TryAcquire("ns/Deployment/app6", 3), "TryAcquire() should fail (exhausted again)")
}

func Test_Release_NonexistentWorkload_DoesNotAffectOtherSlots(t *testing.T) {
	limiter := NewRolloutConcurrencyLimiter(logr.Discard().WithName("RolloutConcurrencyLimiter"))

	// First acquire a slot
	assert.True(t, limiter.TryAcquire("ns/Deployment/app1", 2), "TryAcquire() 1 should succeed")

	// Release a DIFFERENT workload that never acquired a slot (should be no-op)
	limiter.ReleaseWorkloadRolloutSlot("ns/Deployment/nonexistent")

	// Verify app1's slot is still held (only 1 slot left, not 2)
	assert.True(t, limiter.TryAcquire("ns/Deployment/app2", 2), "TryAcquire() 2 should succeed")
	assert.False(t, limiter.TryAcquire("ns/Deployment/app3", 2), "TryAcquire() 3 should fail (limit is 2)")
}

func Test_NilReceiver_Release(t *testing.T) {
	// Arrange: nil receiver
	var limiter *RolloutConcurrencyLimiter

	// Act & Assert: nil receiver should not panic (no-op)
	assert.NotPanics(t, func() {
		limiter.ReleaseWorkloadRolloutSlot("test/Deployment/test")
	})
}

func Test_InFlightCount(t *testing.T) {
	limiter := NewRolloutConcurrencyLimiter(logr.Discard().WithName("RolloutConcurrencyLimiter"))

	assert.Equal(t, 0, limiter.InFlightCount())

	limiter.TryAcquire("ns/Deployment/app1", 3)
	assert.Equal(t, 1, limiter.InFlightCount())

	limiter.TryAcquire("ns/Deployment/app2", 3)
	limiter.TryAcquire("ns/Deployment/app3", 3)
	assert.Equal(t, 3, limiter.InFlightCount())

	limiter.ReleaseWorkloadRolloutSlot("ns/Deployment/app1")
	assert.Equal(t, 2, limiter.InFlightCount())
}

// Helper to generate unique workload keys
func workloadKey(namespace, kind string, index int) string {
	return namespace + "/" + kind + "/app" + string(rune('0'+index))
}
