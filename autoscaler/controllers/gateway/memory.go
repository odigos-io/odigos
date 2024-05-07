package gateway

import odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"

const (
	defaultRequestMemoryMiB = 500

	// this configures the processor limit_mib, which is the hard limit in MiB, afterwhich garbage collection will be forced.
	// as recommended by the processor docs, if not set, this is set to 50MiB less than the memory limit of the collector
	defaultMemoryLimiterLimitDiffMib = 50

	// the soft limit will be set to 80% of the hard limit.
	// this value is used to derive the "spike_limit_mib" parameter in the processor configuration if a value is not set
	defaultMemoryLimiterSpikePercentage = 20.0

	// the percentage out of the memory limiter hard limit, at which go runtime will start garbage collection.
	// it is used to calculate the GOMEMLIMIT environment variable value.
	defaultGoMemLimitPercentage = 80.0
)

type memoryConfigurations struct {
	memoryRequestMiB           int
	memoryLimiterLimitMiB      int
	memoryLimiterSpikeLimitMiB int
	gomemlimitMiB              int
}

func getMemoryConfigurations(odigosConfig *odigosv1.OdigosConfiguration) *memoryConfigurations {

	memoryRequestMiB := defaultRequestMemoryMiB
	if odigosConfig.Spec.CollectorGateway != nil && odigosConfig.Spec.CollectorGateway.RequestMemoryMiB > 0 {
		memoryRequestMiB = odigosConfig.Spec.CollectorGateway.RequestMemoryMiB
	}

	// the memory limiter hard limit is set as 50 MiB less than the memory request
	memoryLimiterLimitMiB := memoryRequestMiB - defaultMemoryLimiterLimitDiffMib
	if odigosConfig.Spec.CollectorGateway != nil && odigosConfig.Spec.CollectorGateway.MemoryLimiterLimitMiB > 0 {
		memoryLimiterLimitMiB = odigosConfig.Spec.CollectorGateway.MemoryLimiterLimitMiB
	}

	memoryLimiterSpikeLimitMiB := memoryLimiterLimitMiB * defaultMemoryLimiterSpikePercentage / 100.0
	if odigosConfig.Spec.CollectorGateway != nil && odigosConfig.Spec.CollectorGateway.MemoryLimiterSpikeLimitMiB > 0 {
		memoryLimiterSpikeLimitMiB = odigosConfig.Spec.CollectorGateway.MemoryLimiterSpikeLimitMiB
	}

	gomemlimitMiB := int(memoryLimiterLimitMiB * defaultGoMemLimitPercentage / 100.0)
	if odigosConfig.Spec.CollectorGateway != nil && odigosConfig.Spec.CollectorGateway.GoMemLimitMib != 0 {
		gomemlimitMiB = odigosConfig.Spec.CollectorGateway.GoMemLimitMib
	}

	return &memoryConfigurations{
		memoryRequestMiB:           memoryRequestMiB,
		memoryLimiterLimitMiB:      memoryLimiterLimitMiB,
		memoryLimiterSpikeLimitMiB: memoryLimiterSpikeLimitMiB,
		gomemlimitMiB:              gomemlimitMiB,
	}
}
