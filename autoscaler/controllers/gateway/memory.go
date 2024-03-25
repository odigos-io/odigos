package gateway

import odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"

const (
	defaultRequestMemoryMiB uint = 500

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
	memoryRequestMib           uint
	memoryLimiterLimitMib      uint
	memoryLimiterSpikeLimitMiB uint
	gomemlimitMib              uint
}

func getMemoryConfigurations(odigosConfig *odigosv1.OdigosConfiguration) *memoryConfigurations {

	memoryRequestMib := defaultRequestMemoryMiB
	if odigosConfig.Spec.CollectorGatewayRequestMemoryMi != 0 {
		memoryRequestMib = odigosConfig.Spec.CollectorGatewayRequestMemoryMi
	}

	// the memory limiter hard limit is set as 50 MiB less than the memory request
	memoryLimiterLimitMib := memoryRequestMib - defaultMemoryLimiterLimitDiffMib
	if odigosConfig.Spec.CollectorGatewayMemoryLimiterLimitMib != 0 {
		memoryLimiterLimitMib = odigosConfig.Spec.CollectorGatewayMemoryLimiterLimitMib
	}

	memoryLimiterSpikeLimitMiB := memoryLimiterLimitMib * defaultMemoryLimiterSpikePercentage / 100.0
	if odigosConfig.Spec.CollectorGatewayMemoryLimiterSpikeLimitMiB != 0 {
		memoryLimiterSpikeLimitMiB = odigosConfig.Spec.CollectorGatewayMemoryLimiterSpikeLimitMiB
	}

	gomemlimitMib := uint(memoryLimiterLimitMib * defaultGoMemLimitPercentage / 100.0)
	if odigosConfig.Spec.CollectorGatewayGoMemLimitMib != 0 {
		gomemlimitMib = odigosConfig.Spec.CollectorGatewayGoMemLimitMib
	}

	return &memoryConfigurations{
		memoryRequestMib:           memoryRequestMib,
		memoryLimiterLimitMib:      memoryLimiterLimitMib,
		memoryLimiterSpikeLimitMiB: memoryLimiterSpikeLimitMiB,
		gomemlimitMib:              gomemlimitMib,
	}
}
