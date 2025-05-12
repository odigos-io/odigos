package nodecollector

import (
	"context"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/k8sutils/pkg/feature"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	ClusterCollectorGateway = map[string]string{
		k8sconsts.OdigosCollectorRoleLabel: string(k8sconsts.CollectorsRoleNodeCollector),
	}
)

func syncService(ctx context.Context, c client.Client, scheme *runtime.Scheme, dc *odigosv1.CollectorsGroup) error {
	if !feature.ServiceInternalTrafficPolicy(feature.GA) {
		return nil
	}
	logger := log.FromContext(ctx)

	localTrafficPolicy := v1.ServiceInternalTrafficPolicyLocal
	dcService := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.OdigosNodeCollectorLocalTrafficServiceName,
			Namespace: dc.Namespace,
			Labels:    ClusterCollectorGateway,
		},
		Spec: v1.ServiceSpec{
			Selector: map[string]string{
				k8sconsts.OdigosCollectorRoleLabel: string(k8sconsts.CollectorsRoleNodeCollector),
			},
			Ports: []v1.ServicePort{
				{
					Name:       "otlp",
					Protocol:   "TCP",
					Port:       4317,
					TargetPort: intstr.FromInt(4317),
				},
				{
					Name:       "otlphttp",
					Protocol:   "TCP",
					Port:       4318,
					TargetPort: intstr.FromInt(4318),
				},
			},
			InternalTrafficPolicy: &localTrafficPolicy,
		},
	}

	if err := ctrl.SetControllerReference(dc, dcService, scheme); err != nil {
		logger.Error(err, "failed to set controller reference")
		return err
	}

	err := c.Create(ctx, dcService)
	return client.IgnoreAlreadyExists(err)
}
