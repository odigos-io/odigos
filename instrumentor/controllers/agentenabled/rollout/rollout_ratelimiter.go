package rollout

import (
	"github.com/odigos-io/odigos/common"
	"golang.org/x/time/rate"
)

const (
	DefaultConcurrentRollouts = 5.0
	NoRateLimiting            = float64(rate.Inf) // Infinite rate limiting - which means no rate limiting.
)

type RolloutRateLimiter struct {
	limiter *rate.Limiter
}

// NewRolloutRateLimiter creates a rate limiter using OdigosConfiguration values.
// Rate limiting is disabled by default and only enabled when IsConcurrentRolloutsEnabled is explicitly set to true.
func NewRolloutRateLimiter(conf *common.OdigosConfiguration) *RolloutRateLimiter {
	concurrentRollouts := NoRateLimiting

	// Only enable rate limiting if explicitly requested via IsConcurrentRolloutsEnabled=true
	if conf != nil && conf.Rollout != nil &&
		conf.Rollout.IsConcurrentRolloutsEnabled != nil &&
		*conf.Rollout.IsConcurrentRolloutsEnabled {
		// Use the configured value if positive, otherwise use the default
		if conf.Rollout.ConcurrentRollouts > 0 {
			concurrentRollouts = conf.Rollout.ConcurrentRollouts
		} else {
			concurrentRollouts = DefaultConcurrentRollouts
		}
	}

	// NOTE: this is not a mistake - we use concurrentRollouts for both burst and rate.
	// The rate limiter works by allowing a burst of X to happen immediately, and then rate limiting the subsequent requests.
	// We want to consistenly allow only X concurrent rollouts to happen at any given time,
	// with the rate limiter being used as a sort of a queueing mechanism to regulate the rate of subsequent requests.
	return &RolloutRateLimiter{
		limiter: rate.NewLimiter(rate.Limit(concurrentRollouts), int(concurrentRollouts)),
	}
}

func (r *RolloutRateLimiter) Allow() bool {
	if r == nil || r.limiter == nil {
		return true
	}
	return r.limiter.Allow()
}
