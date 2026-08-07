package config

import (
	"go.opentelemetry.io/collector/pdata/pcommon"
)

// TailSamplingConfigProvider resolves tail sampling configuration for a trace resource.
type TailSamplingConfigProvider interface {
	GetTailSamplingConfig(resource pcommon.Resource) (*ComputedWorkloadConfig, bool)
}
