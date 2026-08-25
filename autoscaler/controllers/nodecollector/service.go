package nodecollector

import (
	"context"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	commonlogger "github.com/odigos-io/odigos/common/logger"
	"github.com/odigos-io/odigos/k8sutils/pkg/feature"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var (
	ClusterCollectorGateway = map[string]string{
		k8sconsts.OdigosCollectorRoleLabel: string(k8sconsts.CollectorsRoleNodeCollector),
	}
)

func (b *nodeCollectorBaseReconciler) syncService(ctx context.Context, dc *odigosv1.CollectorsGroup) error {
	if !feature.ServiceInternalTrafficPolicy(feature.GA) {
		return nil
	}
	if dc == nil {
		// it is valid to not have a data collection collector group until it is created by the scheduler
		return nil
	}

	logger := commonlogger.FromContext(ctx)

	dcService := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.OdigosNodeCollectorLocalTrafficServiceName,
			Namespace: dc.Namespace,
			Labels:    ClusterCollectorGateway,
		},
	}

	if err := ctrl.SetControllerReference(dc, dcService, b.scheme); err != nil {
		logger.Error(err, "failed to set controller reference")
		return err
	}

	_, err := controllerutil.CreateOrPatch(ctx, b.Client, dcService, func() error {
		updateNodeCollectorLocalTrafficSvc(dcService)
		return nil
	})
	return err
}

// updateNodeCollectorLocalTrafficSvc updates desired fields in place.
// Do not replace Spec wholesale: CreateOrPatch uses a JSON merge patch, and
// zeroing ClusterIP/Type/etc. produces nulls that the API rejects as immutable,
// failing every reconcile and blocking SyncConfigMap.
func updateNodeCollectorLocalTrafficSvc(svc *v1.Service) {
	localTrafficPolicy := v1.ServiceInternalTrafficPolicyLocal
	svc.Spec.Selector = map[string]string{
		k8sconsts.OdigosCollectorRoleLabel: string(k8sconsts.CollectorsRoleNodeCollector),
	}
	svc.Spec.Ports = []v1.ServicePort{
		{
			Name:        "otlp",
			Protocol:    "TCP",
			AppProtocol: ptr.To("grpc"),
			Port:        4317,
			TargetPort:  intstr.FromInt(4317),
		},
		{
			Name:       "otlphttp",
			Protocol:   "TCP",
			Port:       4318,
			TargetPort: intstr.FromInt(4318),
		},
		{
			Name:       "metrics",
			Protocol:   "TCP",
			Port:       8888,
			TargetPort: intstr.FromInt32(k8sconsts.OdigosNodeCollectorOwnTelemetryPortDefault),
		},
	}
	svc.Spec.InternalTrafficPolicy = &localTrafficPolicy
}
