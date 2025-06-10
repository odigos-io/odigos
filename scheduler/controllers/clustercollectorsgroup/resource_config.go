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
	defaultMinReplicas = 1
	// MaxReplicasDefault is the default maximum number of replicas for the collector hpa
	defaultMaxReplicas = 10

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

	gatewayMinReplicas := defaultMinReplicas
	if gatewayConfig != nil && gatewayConfig.MinReplicas > 0 {
		gatewayMinReplicas = gatewayConfig.MinReplicas
	}
	gatewayMaxReplicas := defaultMaxReplicas
	if gatewayConfig != nil && gatewayConfig.MaxReplicas > 0 {
		gatewayMaxReplicas = gatewayConfig.MaxReplicas
	}
	memoryRequestMiB := defaultRequestMemoryMiB
	if gatewayConfig != nil && gatewayConfig.RequestMemoryMiB > 0 {
		memoryRequestMiB = gatewayConfig.RequestMemoryMiB
	}
	memoryLimitMiB := int(float64(memoryRequestMiB) * memoryLimitAboveRequestFactor)
	if odigosConfig.CollectorGateway != nil && odigosConfig.CollectorGateway.LimitMemoryMiB > 0 {
		memoryLimitMiB = odigosConfig.CollectorGateway.LimitMemoryMiB
	}
	cpuRequestm := defaultRequestCPUm
	if gatewayConfig != nil && gatewayConfig.RequestCPUm > 0 {
		cpuRequestm = gatewayConfig.RequestCPUm
	}
	cpuLimitm := defaultLimitCPUm
	if gatewayConfig != nil && gatewayConfig.LimitCPUm > 0 {
		cpuLimitm = gatewayConfig.LimitCPUm
	}

	// the memory limiter hard limit is set as 50 MiB less than the memory request

	memoryLimiterLimitMiB := memoryRequestMiB - defaultMemoryLimiterLimitDiffMib
	if odigosConfig.CollectorGateway != nil && odigosConfig.CollectorGateway.MemoryLimiterLimitMiB > 0 {
		memoryLimiterLimitMiB = odigosConfig.CollectorGateway.MemoryLimiterLimitMiB
	}
	memoryLimiterSpikeLimitMiB := memoryLimiterLimitMiB * defaultMemoryLimiterSpikePercentage / 100
	if odigosConfig.CollectorGateway != nil && odigosConfig.CollectorGateway.MemoryLimiterSpikeLimitMiB > 0 {
		memoryLimiterSpikeLimitMiB = odigosConfig.CollectorGateway.MemoryLimiterSpikeLimitMiB
	}

	gomemlimitMiB := int(memoryLimiterLimitMiB * defaultGoMemLimitPercentage / 100.0)
	if odigosConfig.CollectorGateway != nil && odigosConfig.CollectorGateway.GoMemLimitMib != 0 {
		gomemlimitMiB = odigosConfig.CollectorGateway.GoMemLimitMib
	}

	return &odigosv1.CollectorsGroupResourcesSettings{
		MinReplicas:                &gatewayMinReplicas,
		MaxReplicas:                &gatewayMaxReplicas,
		MemoryRequestMiB:           memoryRequestMiB,
		MemoryLimitMiB:             memoryLimitMiB,
		CpuRequestMillicores:       cpuRequestm,
		CpuLimitMillicores:         cpuLimitm,
		MemoryLimiterLimitMiB:      memoryLimiterLimitMiB,
		MemoryLimiterSpikeLimitMiB: memoryLimiterSpikeLimitMiB,
		GomemlimitMiB:              gomemlimitMiB,
	}
}
