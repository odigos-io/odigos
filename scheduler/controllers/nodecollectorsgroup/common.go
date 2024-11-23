package nodecollectorsgroup

import (
	"context"
	"errors"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/k8sutils/pkg/consts"
	k8sutilsconsts "github.com/odigos-io/odigos/k8sutils/pkg/consts"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	"github.com/odigos-io/odigos/k8sutils/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func getMemorySettings(odigosConfig common.OdigosConfiguration) odigosv1.CollectorsGroupMemorySettings {
	// TODO: currently using hardcoded values, should be configurable.
	//
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
	//
	return odigosv1.CollectorsGroupMemorySettings{
		MemoryRequestMiB:           256,
		MemoryLimiterLimitMiB:      512,
		MemoryLimiterSpikeLimitMiB: 128,            // meaning that collector will start dropping data at 512-128=384MiB
		GomemlimitMiB:              512 - 128 - 32, // start aggressive GC 32 MiB before soft limit and dropping data
	}
}

func newNodeCollectorGroup(odigosConfig common.OdigosConfiguration) *odigosv1.CollectorsGroup {

	ownMetricsPort := consts.OdigosNodeCollectorOwnTelemetryPortDefault
	if odigosConfig.CollectorNode != nil && odigosConfig.CollectorNode.CollectorOwnMetricsPort != 0 {
		ownMetricsPort = odigosConfig.CollectorNode.CollectorOwnMetricsPort
	}

	return &odigosv1.CollectorsGroup{
		TypeMeta: metav1.TypeMeta{
			Kind:       "CollectorsGroup",
			APIVersion: "odigos.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      consts.OdigosNodeCollectorDaemonSetName,
			Namespace: env.GetCurrentNamespace(),
		},
		Spec: odigosv1.CollectorsGroupSpec{
			Role:                    odigosv1.CollectorsGroupRoleNodeCollector,
			CollectorOwnMetricsPort: ownMetricsPort,
			MemorySettings:          getMemorySettings(odigosConfig),
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
