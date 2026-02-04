package rollout

import (
	"fmt"
	"time"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
)

type RollBackOptions struct {
	IsRollbackDisabled      bool
	RollbackGraceTime       time.Duration
	RollbackStabilityWindow time.Duration
	MaxConcurrentRollouts   int
}

// GetRolloutAndRollbackOptions extracts rollout and rollback configuration from OdigosConfiguration.
// Returns an error if the configuration contains invalid duration strings.
func getRolloutAndRollbackOptions(conf *common.OdigosConfiguration) (isAutomaticRolloutDisabled bool, rollBackOptions RollBackOptions, err error) {
	isAutomaticRolloutDisabled = conf.Rollout != nil && conf.Rollout.AutomaticRolloutDisabled != nil && *conf.Rollout.AutomaticRolloutDisabled

	isRollbackDisabled := conf.RollbackDisabled != nil && *conf.RollbackDisabled

	defaultRollbackGraceTime, _ := time.ParseDuration(consts.DefaultAutoRollbackGraceTime)

	rollbackGraceTime := defaultRollbackGraceTime
	if conf.RollbackGraceTime != "" {
		parsedRollbackGraceTime, parseErr := time.ParseDuration(conf.RollbackGraceTime)
		if parseErr != nil {
			return false, RollBackOptions{}, fmt.Errorf("invalid RollbackGraceTime %q: %w", conf.RollbackGraceTime, parseErr)
		}
		rollbackGraceTime = parsedRollbackGraceTime
	}

	rollbackStabilityWindow, _ := time.ParseDuration(consts.DefaultAutoRollbackStabilityWindow)
	if conf.RollbackStabilityWindow != "" {
		parsedRollbackStabilityWindow, parseErr := time.ParseDuration(conf.RollbackStabilityWindow)
		if parseErr != nil {
			return false, RollBackOptions{}, fmt.Errorf("invalid RollbackStabilityWindow %q: %w", conf.RollbackStabilityWindow, parseErr)
		}
		rollbackStabilityWindow = parsedRollbackStabilityWindow
	}

	// Concurrent rollout limiter configuration - defaults to no concurrency limiting
	maxConcurrentRollouts := NoConcurrencyLimiting
	if conf.Rollout != nil && conf.Rollout.MaxConcurrentRollouts > 0 {
		maxConcurrentRollouts = conf.Rollout.MaxConcurrentRollouts
	}

	rollBackOptions = RollBackOptions{
		IsRollbackDisabled:      isRollbackDisabled,
		RollbackGraceTime:       rollbackGraceTime,
		RollbackStabilityWindow: rollbackStabilityWindow,
		MaxConcurrentRollouts:   maxConcurrentRollouts,
	}
	return isAutomaticRolloutDisabled, rollBackOptions, nil
}
