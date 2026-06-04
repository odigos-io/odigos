package computed

import "time"

type AutoRollbackConfig struct {
	Enabled         bool
	GraceTime       time.Duration
	StabilityWindow time.Duration
}
