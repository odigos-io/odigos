package rollout

import (
	"sync"

	"github.com/go-logr/logr"
)

const (
	NoConcurrencyLimiting = 0
)

type RolloutConcurrencyLimiter struct {
	mutex  sync.Mutex
	logger logr.Logger
	// set of workload keys currently rolling out - we use a map for fast lookup,
	// and a struct{} for the value to avoid allocating memory for the value
	inFlightRollouts map[string]struct{}
}

func NewRolloutConcurrencyLimiter(logger logr.Logger) *RolloutConcurrencyLimiter {
	return &RolloutConcurrencyLimiter{
		logger:           logger,
		inFlightRollouts: make(map[string]struct{}),
	}
}

// TryAcquire attempts to acquire a slot for the given workload to allow rollout to proceed.
func (r *RolloutConcurrencyLimiter) TryAcquire(workloadKey string, limit int) bool {
	if r == nil {
		return true
	}

	// No rate limiting configured
	if limit == NoConcurrencyLimiting {
		return true
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Already has a slot for this workload (from a previous reconcile that triggered rollout)
	if _, exists := r.inFlightRollouts[workloadKey]; exists {
		return true
	}

	// Check if under limit
	if len(r.inFlightRollouts) < limit {
		r.inFlightRollouts[workloadKey] = struct{}{}
		r.logger.V(2).Info("Acquired rollout slot", "workload", workloadKey, "inFlight", len(r.inFlightRollouts), "limit", limit)
		return true
	} else if len(r.inFlightRollouts) == limit {
		r.logger.V(2).Info("Rollout slot denied - at capacity", "workload", workloadKey, "inFlight", len(r.inFlightRollouts), "limit", limit)
		return false
	} else {
		r.logger.V(2).Info("Rollout slot denied - more rollouts than the limit - this should not happen", "workload", workloadKey, "inFlight", len(r.inFlightRollouts), "limit", limit)
		return false
	}
}

// Release releases the slot for a specific workload
func (r *RolloutConcurrencyLimiter) ReleaseWorkloadRolloutSlot(workloadKey string) {
	if r == nil {
		return
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	delete(r.inFlightRollouts, workloadKey)
	r.logger.V(2).Info("Released rollout slot", "workload", workloadKey, "inFlight", len(r.inFlightRollouts))
}

// InFlightCount returns the number of workloads currently rolling out (for testing/debugging)
func (r *RolloutConcurrencyLimiter) InFlightCount() int {
	if r == nil {
		return 0
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	return len(r.inFlightRollouts)
}
