package clustercollectorsgroup

import (
	"context"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/k8sutils/pkg/consts"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	"github.com/odigos-io/odigos/k8sutils/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func newClusterCollectorGroup(namespace string, resourcesSettings *odigosv1.CollectorsGroupResourcesSettings) *odigosv1.CollectorsGroup {
	return &odigosv1.CollectorsGroup{
		TypeMeta: metav1.TypeMeta{
			Kind:       "CollectorsGroup",
			APIVersion: "odigos.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      consts.OdigosClusterCollectorCollectorGroupName,
			Namespace: namespace,
		},
		Spec: odigosv1.CollectorsGroupSpec{
			Role:                    odigosv1.CollectorsGroupRoleClusterGateway,
			CollectorOwnMetricsPort: consts.OdigosClusterCollectorOwnTelemetryPortDefault,
			ResourcesSettings:       *resourcesSettings,
		},
	}
}

func sync(ctx context.Context, c client.Client) error {

	namespace := env.GetCurrentNamespace()

	var dests odigosv1.DestinationList
	err := c.List(ctx, &dests, client.InNamespace(namespace))
	if err != nil {
		return err
	}

	odigosConfig, err := utils.GetCurrentOdigosConfig(ctx, c)
	if err != nil {
		return err
	}

	resourceSettings := getGatewayResourceSettings(&odigosConfig)

	if len(dests.Items) > 0 {
		err := utils.ApplyCollectorGroup(ctx, c, newClusterCollectorGroup(namespace, resourceSettings))
		if err != nil {
			return err
		}
	}
	// once the gateway is created, it is not deleted, even if there are no destinations.
	// we might want to re-consider this behavior.

	return nil
}
