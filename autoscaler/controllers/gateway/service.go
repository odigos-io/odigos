package gateway

import (
	"context"

	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func syncService(gateway *odigosv1.CollectorsGroup, ctx context.Context, c client.Client, scheme *runtime.Scheme) (*v1.Service, error) {
	logger := log.FromContext(ctx)
	desired, err := getDesiredService(gateway, scheme)
	if err != nil {
		logger.Error(err, "Failed to get desired service")
		return nil, err
	}

	existing := &v1.Service{}
	if err := c.Get(ctx, client.ObjectKey{Namespace: gateway.Namespace, Name: gateway.Name}, existing); err != nil {
		if apierrors.IsNotFound(err) {
			logger.V(0).Info("Creating service")
			newSvc, err := createService(desired, ctx, c)
			if err != nil {
				logger.Error(err, "failed to create service")
				return nil, err
			}
			return newSvc, nil
		} else {
			logger.Error(err, "failed to get service")
			return nil, err
		}
	}

	logger.V(0).Info("Patching service")
	newSvc, err := patchService(existing, desired, ctx, c)
	if err != nil {
		logger.Error(err, "failed to patch service")
		return nil, err
	}

	return newSvc, nil
}

func createService(desired *v1.Service, ctx context.Context, c client.Client) (*v1.Service, error) {
	if err := c.Create(ctx, desired); err != nil {
		return nil, err
	}
	return desired, nil
}

func patchService(existing *v1.Service, desired *v1.Service, ctx context.Context, c client.Client) (*v1.Service, error) {
	updated := existing.DeepCopy()
	if updated.Annotations == nil {
		updated.Annotations = map[string]string{}
	}
	if updated.Labels == nil {
		updated.Labels = map[string]string{}
	}

	updated.Spec = desired.Spec
	updated.ObjectMeta.OwnerReferences = desired.ObjectMeta.OwnerReferences
	for k, v := range desired.ObjectMeta.Annotations {
		updated.ObjectMeta.Annotations[k] = v
	}
	for k, v := range desired.ObjectMeta.Labels {
		updated.ObjectMeta.Labels[k] = v
	}

	patch := client.MergeFrom(existing)
	if err := c.Patch(ctx, updated, patch); err != nil {
		return nil, err
	}

	return updated, nil
}

func getDesiredService(gateway *odigosv1.CollectorsGroup, scheme *runtime.Scheme) (*v1.Service, error) {
	desired := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      gateway.Name,
			Namespace: gateway.Namespace,
			Labels:    commonLabels,
		},
		Spec: v1.ServiceSpec{
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
				{
					Name: "metrics",
					Port: 8888,
				},
			},
			Selector: commonLabels,
		},
	}

	if err := ctrl.SetControllerReference(gateway, desired, scheme); err != nil {
		return nil, err
	}

	return desired, nil
}
