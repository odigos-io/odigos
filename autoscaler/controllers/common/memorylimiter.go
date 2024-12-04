package common

import (
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/config"
)

func GetMemoryLimiterConfig(memorySettings odigosv1.CollectorsGroupResourcesSettings) config.GenericMap {
	// check_interval is currently hardcoded to 1s
	// this seems to be a reasonable value for the memory limiter and what the processor uses in docs.
	// preforming memory checks is expensive, so we trade off performance with fast reaction time to memory pressure.
	return config.GenericMap{
		"check_interval":  "1s",
		"limit_mib":       memorySettings.MemoryLimiterLimitMiB,
		"spike_limit_mib": memorySettings.MemoryLimiterSpikeLimitMiB,
	}
}
