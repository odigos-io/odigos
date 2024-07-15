package gateway

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/k8sutils/pkg/consts"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func deletePreviousServices(ctx context.Context, c client.Client, ns string) error {
	// to support multiple gateways, odigos service changed it's ClusterIP to None
	// this change is not automatically applied to existing installations, we need to delete the service
	// so that it can be recreated with the new ClusterIP value
	logger := log.FromContext(ctx)
	svc := &v1.Service{}
	err := c.Get(ctx, client.ObjectKey{Name: consts.OdigosClusterCollectorServiceName, Namespace: ns}, svc)
	if err != nil {
		return client.IgnoreNotFound(err)
	}

	if svc.Spec.ClusterIP != "None" {
		logger.Info("Deleting the Odigos gateway service to support multiple gateways.")
		err = c.Delete(ctx, svc, &client.DeleteOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}

func syncService(gateway *odigosv1.CollectorsGroup, ctx context.Context, c client.Client, scheme *runtime.Scheme) (*v1.Service, error) {
	logger := log.FromContext(ctx)
	gatewaySvc := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      consts.OdigosClusterCollectorServiceName,
			Namespace: gateway.Namespace,
			Labels:    CommonLabels,
		},
	}

	if err := ctrl.SetControllerReference(gateway, gatewaySvc, scheme); err != nil {
		logger.Error(err, "failed to set controller reference")
		return nil, err
	}

	result, err := controllerutil.CreateOrPatch(ctx, c, gatewaySvc, func() error {
		updateGatewaySvc(gatewaySvc)
		return nil
	})

	if err != nil {
		logger.Error(err, "failed to create or patch gateway service")
		return nil, err
	}

	logger.V(0).Info("gateway service synced", "result", result)
	return gatewaySvc, nil
}

func updateGatewaySvc(svc *v1.Service) {
	svc.Spec.Ports = []v1.ServicePort{
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
		{
			Name: "metrics",
			Port: 8888,
		},
	}

	svc.Spec.Selector = CommonLabels
	svc.Spec.ClusterIP = "None"
}
