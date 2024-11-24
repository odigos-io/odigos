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
