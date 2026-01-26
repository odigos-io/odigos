package rollout

import (
	"context"
	"sync"

	"github.com/go-logr/logr"
	"github.com/odigos-io/odigos/common"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	DefaultConcurrentRollouts = 5
	NoConcurrencyLimiting     = 0
)

type RolloutConcurrencyLimiter struct {
	mutex              sync.Mutex
	logger             logr.Logger
	concurrentRollouts int
	// set of workload keys currently rolling out - we use a map for fast lookup,
	// and a struct{} for the value to avoid allocating memory for the value
	inFlightRollouts map[string]struct{}
}

func NewRolloutConcurrencyLimiter() *RolloutConcurrencyLimiter {
	logger := log.FromContext(context.Background()).WithName("RolloutConcurrencyLimiter")
	return &RolloutConcurrencyLimiter{
		logger:             logger,
		concurrentRollouts: NoConcurrencyLimiting,
		inFlightRollouts:   make(map[string]struct{}),
	}
}

func (r *RolloutConcurrencyLimiter) ApplyConfig(conf *common.OdigosConfiguration) {
	if conf != nil && conf.Rollout != nil && conf.Rollout.IsConcurrentRolloutsLimiterEnabled != nil && *conf.Rollout.IsConcurrentRolloutsLimiterEnabled {
		if conf.Rollout.ConcurrentRollouts > 0 {
			r.concurrentRollouts = conf.Rollout.ConcurrentRollouts
		} else {
			r.concurrentRollouts = DefaultConcurrentRollouts
		}
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
	if r.concurrentRollouts == NoConcurrencyLimiting {
		return true
	}

	// Already has a slot for this workload (from a previous reconcile that triggered rollout)
	if _, exists := r.inFlightRollouts[workloadKey]; exists {
		return true
	}

	// Check if under limit
	if len(r.inFlightRollouts) < r.concurrentRollouts {
		r.inFlightRollouts[workloadKey] = struct{}{}
		r.logger.V(2).Info("Acquired rollout slot", "workload", workloadKey, "inFlight", len(r.inFlightRollouts), "limit", r.concurrentRollouts)
		return true
	}

	r.logger.V(2).Info("Rollout slot denied - at capacity", "workload", workloadKey, "inFlight", len(r.inFlightRollouts), "limit", r.concurrentRollouts)
	return false
}

// HasSlot returns true if the workload currently holds a slot
func (r *RolloutConcurrencyLimiter) HasSlot(workloadKey string) bool {
	if r == nil {
		return false
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	_, exists := r.inFlightRollouts[workloadKey]
	return exists
}

// Release releases the slot for a specific workload
func (r *RolloutConcurrencyLimiter) Release(workloadKey string) {
	if r == nil {
		return
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.inFlightRollouts[workloadKey]; !exists {
		// Not an error - workload may not have had a slot (e.g., no rate limiting)
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
