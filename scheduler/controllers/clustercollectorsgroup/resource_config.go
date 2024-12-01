package clustercollectorsgroup

import (
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
)

const (
	// the default memory request in MiB
	defaultRequestMemoryMiB = 500

	// the default CPU request in millicores
	defaultRequestCPUm = 500
	// the default CPU limit in millicores
	defaultLimitCPUm = 1000

	// MinReplicasDefault is the default number of replicas for the collector
	MinReplicasDefault = 1
	// MaxReplicasDefault is the default maximum number of replicas for the collector hpa
	MaxReplicasDefault = 10

	// this configures the processor limit_mib, which is the hard limit in MiB, afterwhich garbage collection will be forced.
	// as recommended by the processor docs, if not set, this is set to 50MiB less than the memory limit of the collector
	defaultMemoryLimiterLimitDiffMib = 50

	// the soft limit will be set to 80% of the hard limit.
	// this value is used to derive the "spike_limit_mib" parameter in the processor configuration if a value is not set
	defaultMemoryLimiterSpikePercentage = 20.0

	// the percentage out of the memory limiter hard limit, at which go runtime will start garbage collection.
	// it is used to calculate the GOMEMLIMIT environment variable value.
	defaultGoMemLimitPercentage = 80.0

	// the memory settings should prevent the collector from exceeding the memory request.
	// however, the mechanism is heuristic and does not guarantee to prevent OOMs.
	// allowing the memory limit to be slightly above the memory request can help in reducing the chances of OOMs in edge cases.
	// instead of having the process killed, it can use extra memory available on the node without allocating it preemptively.
	memoryLimitAboveRequestFactor = 1.25
)

// process the resources settings from odigos config and return the resources settings for the collectors group.
// apply any defaulting and calculations here.
func getGatewayResourceSettings(odigosConfig *common.OdigosConfiguration) *odigosv1.CollectorsGroupResourcesSettings {
	gatewayConfig := odigosConfig.CollectorGateway

	gatewayMinReplicas := getOrDefault(gatewayConfig, gatewayConfig.MinReplicas, MinReplicasDefault)
	gatewayMaxReplicas := getOrDefault(gatewayConfig, gatewayConfig.MaxReplicas, MaxReplicasDefault)
	memoryRequestMiB := getOrDefault(gatewayConfig, gatewayConfig.RequestMemoryMiB, defaultRequestMemoryMiB)
	cpuRequestm := getOrDefault(gatewayConfig, gatewayConfig.RequestCPUm, defaultRequestCPUm)
	cpuLimitm := getOrDefault(gatewayConfig, gatewayConfig.LimitCPUm, defaultLimitCPUm)
	// the memory limiter hard limit is set as 50 MiB less than the memory request
	memoryLimiterLimitMiB := getOrDefault(gatewayConfig, gatewayConfig.MemoryLimiterLimitMiB, memoryRequestMiB-defaultMemoryLimiterLimitDiffMib)
	memoryLimiterSpikeLimitMiB := getOrDefault(gatewayConfig, gatewayConfig.MemoryLimiterSpikeLimitMiB,
		memoryLimiterLimitMiB*defaultMemoryLimiterSpikePercentage/100.0)

	memoryLimitMiB := int(float64(memoryRequestMiB) * memoryLimitAboveRequestFactor)

	gomemlimitMiB := int(memoryLimiterLimitMiB * defaultGoMemLimitPercentage / 100.0)
	if odigosConfig.CollectorGateway != nil && odigosConfig.CollectorGateway.GoMemLimitMib != 0 {
		gomemlimitMiB = odigosConfig.CollectorGateway.GoMemLimitMib
	}

	return &odigosv1.CollectorsGroupResourcesSettings{
		MinReplicas:                gatewayMinReplicas,
		MaxReplicas:                gatewayMaxReplicas,
		MemoryRequestMiB:           memoryRequestMiB,
		MemoryLimitMiB:             memoryLimitMiB,
		CpuRequestMillicores:       cpuRequestm,
		CpuLimitMillicores:         cpuLimitm,
		MemoryLimiterLimitMiB:      memoryLimiterLimitMiB,
		MemoryLimiterSpikeLimitMiB: memoryLimiterSpikeLimitMiB,
		GomemlimitMiB:              gomemlimitMiB,
	}
}

// Returns the value if it is greater than 0, otherwise returns the default value.
func getOrDefault[T ~int | ~int32 | ~int64 | ~float32 | ~float64](config *common.CollectorGatewayConfiguration, value T, defaultValue T) T {
	if config != nil && value > 0 {
		return value
	}
	return defaultValue
}
