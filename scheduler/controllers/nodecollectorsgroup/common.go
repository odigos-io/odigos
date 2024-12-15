package nodecollectorsgroup

import (
	"context"
	"errors"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	k8sutilsconsts "github.com/odigos-io/odigos/k8sutils/pkg/consts"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	"github.com/odigos-io/odigos/k8sutils/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// the default memory request in MiB
	defaultRequestMemoryMiB = 256

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
	memoryLimitAboveRequestFactor = 2.0

	// the default CPU request in millicores
	defaultRequestCPUm = 250
	// the default CPU limit in millicores
	defaultLimitCPUm = 500
)

func getResourceSettings(odigosConfig common.OdigosConfiguration) odigosv1.CollectorsGroupResourcesSettings {
	// memory request is expensive on daemonsets since it will consume this memory
	// on each node in the cluster. setting to 256, but allowing memory to spike higher
	// to consume more available memory on the node.
	// if the node has memory to spare, we can use it to buffer more data before dropping,
	// but it also means that if no memory is available, collector might get killed by OOM killer.
	//
	// we can trade-off the memory request:
	// - more memory request: more memory allocated per collector on each node, but more buffer for bursts and transient failures.
	// - less memory request: efficient use of cluster resources, but data might be dropped earlier on spikes.
	// currently choosing 256MiB as a balance (~200MiB left for heap to handle batches and export queues).
	//
	// we can trade-off how high the memory limit is set above the request:
	// - limit is set to request: collector most stable (no OOM) but smaller buffer for bursts and early data drop.
	// - limit is set way above request: in case of memory spike, collector will use extra memory available on the node to buffer data, but might get killed by OOM killer if this memory is not available.
	// currently choosing 512MiB as a balance (200MiB guaranteed for heap, and the rest ~300MiB of buffer from node before start dropping).

	nodeCollectorConfig := odigosConfig.CollectorNode

	memoryRequestMiB := defaultRequestMemoryMiB
	if nodeCollectorConfig != nil && nodeCollectorConfig.RequestMemoryMiB > 0 {
		memoryRequestMiB = nodeCollectorConfig.RequestMemoryMiB
	}
	memoryLimitMiB := int(float64(memoryRequestMiB) * memoryLimitAboveRequestFactor)
	if nodeCollectorConfig != nil && nodeCollectorConfig.LimitMemoryMiB > 0 {
		memoryLimitMiB = nodeCollectorConfig.LimitMemoryMiB
	}

	memoryLimiterLimitMiB := memoryLimitMiB - defaultMemoryLimiterLimitDiffMib
	if nodeCollectorConfig != nil && nodeCollectorConfig.MemoryLimiterLimitMiB > 0 {
		memoryLimiterLimitMiB = nodeCollectorConfig.MemoryLimiterLimitMiB
	}
	memoryLimiterSpikeLimitMiB := memoryLimiterLimitMiB * defaultMemoryLimiterSpikePercentage / 100
	if nodeCollectorConfig != nil && nodeCollectorConfig.MemoryLimiterSpikeLimitMiB > 0 {
		memoryLimiterSpikeLimitMiB = nodeCollectorConfig.MemoryLimiterSpikeLimitMiB
	}

	gomemlimitMiB := int(memoryLimiterLimitMiB * defaultGoMemLimitPercentage / 100.0)
	if nodeCollectorConfig != nil && nodeCollectorConfig.GoMemLimitMib != 0 {
		gomemlimitMiB = nodeCollectorConfig.GoMemLimitMib
	}

	cpuRequestm := defaultRequestCPUm
	if nodeCollectorConfig != nil && nodeCollectorConfig.RequestCPUm > 0 {
		cpuRequestm = nodeCollectorConfig.RequestCPUm
	}
	cpuLimitm := defaultLimitCPUm
	if nodeCollectorConfig != nil && nodeCollectorConfig.LimitCPUm > 0 {
		cpuLimitm = nodeCollectorConfig.LimitCPUm
	}

	return odigosv1.CollectorsGroupResourcesSettings{
		MemoryRequestMiB:           memoryRequestMiB,
		MemoryLimitMiB:             memoryLimitMiB,
		MemoryLimiterLimitMiB:      memoryLimiterLimitMiB,
		MemoryLimiterSpikeLimitMiB: memoryLimiterSpikeLimitMiB,
		GomemlimitMiB:              gomemlimitMiB,
		CpuRequestMillicores:       cpuRequestm,
		CpuLimitMillicores:         cpuLimitm,
	}
}

func newNodeCollectorGroup(odigosConfig common.OdigosConfiguration) *odigosv1.CollectorsGroup {

	ownMetricsPort := k8sutilsconsts.OdigosNodeCollectorOwnTelemetryPortDefault
	if odigosConfig.CollectorNode != nil && odigosConfig.CollectorNode.CollectorOwnMetricsPort != 0 {
		ownMetricsPort = odigosConfig.CollectorNode.CollectorOwnMetricsPort
	}

	return &odigosv1.CollectorsGroup{
		TypeMeta: metav1.TypeMeta{
			Kind:       "CollectorsGroup",
			APIVersion: "odigos.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sutilsconsts.OdigosNodeCollectorDaemonSetName,
			Namespace: env.GetCurrentNamespace(),
		},
		Spec: odigosv1.CollectorsGroupSpec{
			Role:                    odigosv1.CollectorsGroupRoleNodeCollector,
			CollectorOwnMetricsPort: ownMetricsPort,
			ResourcesSettings:       getResourceSettings(odigosConfig),
		},
	}
}

func sync(ctx context.Context, c client.Client) error {

	namespace := env.GetCurrentNamespace()

	var instrumentedConfigs odigosv1.InstrumentationConfigList
	err := c.List(ctx, &instrumentedConfigs)
	if err != nil {
		return errors.Join(errors.New("failed to list InstrumentationConfigs"), err)
	}
	numberOfInstrumentedApps := len(instrumentedConfigs.Items)

	if numberOfInstrumentedApps == 0 {
		// TODO: should we delete the collector group if cluster collector is not ready?
		return utils.DeleteCollectorGroup(ctx, c, namespace, k8sutilsconsts.OdigosNodeCollectorCollectorGroupName)
	}

	clusterCollectorGroup, err := utils.GetCollectorGroup(ctx, c, namespace, k8sutilsconsts.OdigosClusterCollectorCollectorGroupName)
	if err != nil {
		return client.IgnoreNotFound(err)
	}

	odigosConfig, err := utils.GetCurrentOdigosConfig(ctx, c)
	if err != nil {
		return err
	}

	clusterCollectorReady := clusterCollectorGroup.Status.Ready
	if clusterCollectorReady {
		return utils.ApplyCollectorGroup(ctx, c, newNodeCollectorGroup(odigosConfig))
	}

	return nil
}
