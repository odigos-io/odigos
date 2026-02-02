package rollout

import (
	"context"
	"sync"

	"github.com/go-logr/logr"
	"github.com/odigos-io/odigos/common"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	NoConcurrencyLimiting = 0
)

type RolloutConcurrencyLimiter struct {
	mutex                 sync.Mutex
	logger                logr.Logger
	maxConcurrentRollouts int
	// set of workload keys currently rolling out - we use a map for fast lookup,
	// and a struct{} for the value to avoid allocating memory for the value
	inFlightRollouts map[string]struct{}
}

func NewRolloutConcurrencyLimiter() *RolloutConcurrencyLimiter {
	logger := log.FromContext(context.Background()).WithName("RolloutConcurrencyLimiter")
	return &RolloutConcurrencyLimiter{
		logger:                logger,
		maxConcurrentRollouts: NoConcurrencyLimiting,
		inFlightRollouts:      make(map[string]struct{}),
	}
}

func (r *RolloutConcurrencyLimiter) ApplyConfig(conf *common.OdigosConfiguration) {
	if conf != nil && conf.Rollout != nil && conf.Rollout.MaxConcurrentRollouts > 0 {
		r.maxConcurrentRollouts = conf.Rollout.MaxConcurrentRollouts
	}
}

// TryAcquire attempts to acquire a slot for the given workload to allow rollout to proceed.
func (r *RolloutConcurrencyLimiter) TryAcquire(workloadKey string) bool {
	if r == nil {
		return true
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// No rate limiting configured
	if r.maxConcurrentRollouts == NoConcurrencyLimiting {
		return true
	}

	// Already has a slot for this workload (from a previous reconcile that triggered rollout)
	if _, exists := r.inFlightRollouts[workloadKey]; exists {
		return true
	}

	// Check if under limit
	if len(r.inFlightRollouts) < r.maxConcurrentRollouts {
		r.inFlightRollouts[workloadKey] = struct{}{}
		r.logger.V(2).Info("Acquired rollout slot", "workload", workloadKey, "inFlight", len(r.inFlightRollouts), "limit", r.maxConcurrentRollouts)
		return true
	}

	r.logger.V(2).Info("Rollout slot denied - at capacity", "workload", workloadKey, "inFlight", len(r.inFlightRollouts), "limit", r.maxConcurrentRollouts)
	return false
}

// Release releases the slot for a specific workload
func (r *RolloutConcurrencyLimiter) ReleaseWorkloadRolloutSlot(workloadKey string) {
	if r == nil {
		return
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.inFlightRollouts[workloadKey]; !exists {
		if r.maxConcurrentRollouts == NoConcurrencyLimiting {
			// Not an error - workload may not have had a slot (e.g., no rate limiting)
			r.logger.V(2).Info("Workload does not have a slot - rate limiting is disabled", "workload", workloadKey)
		} else {
			// This should not happen, but we log it for debugging purposes
			r.logger.V(2).Info("Workload does not have a slot - this should not happen, rate limiting is enabled", "workload", workloadKey)
		}
		return
	}

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
