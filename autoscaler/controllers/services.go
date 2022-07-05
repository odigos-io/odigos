package controllers

import (
	"context"
	odigosv1 "github.com/keyval-dev/odigos/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *CollectorReconciler) syncServices(ctx context.Context, collector *odigosv1.Collector) (bool, error) {
	svcList, err := r.listServices(ctx, collector)
	if err != nil {
		return false, err
	}

	if len(svcList.Items) == 0 {
		err = r.createServices(ctx, collector)
		if err != nil {
			return false, err
		}
		return true, nil
	}

	if !r.isServicesUpToDate(svcList) {
		err = r.updateServices(ctx, svcList)
		if err != nil {
			return false, err
		}

		return true, nil
	}

	return false, nil
}

func (r *CollectorReconciler) isServicesUpToDate(svcList *v1.ServiceList) bool {
	expected := []string{"zipkin", "otlp"}
	for _, exp := range expected {
		for _, svc := range svcList.Items {
			found := false
			for _, port := range svc.Spec.Ports {
				if port.Name == exp {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
	}

	return true
}

func (r *CollectorReconciler) updateServices(ctx context.Context, svcList *v1.ServiceList) error {
	for _, svc := range svcList.Items {
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
				Name:       "zipkin",
				Protocol:   "TCP",
				Port:       9411,
				TargetPort: intstr.FromInt(9411),
			},
			{
				Name: "metrics",
				Port: 8888,
			},
		}

		err := r.Update(ctx, &svc)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *CollectorReconciler) createServices(ctx context.Context, collector *odigosv1.Collector) error {
	svc := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      collector.Name,
			Namespace: collector.Namespace,
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
					Name:       "zipkin",
					Protocol:   "TCP",
					Port:       9411,
					TargetPort: intstr.FromInt(9411),
				},
				{
					Name: "metrics",
					Port: 8888,
				},
			},
			Selector: commonLabels,
		},
	}

	err := ctrl.SetControllerReference(collector, svc, r.Scheme)
	if err != nil {
		return err
	}

	err = r.Create(ctx, svc)
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			return nil
		}
		return err
	}

	return nil
}

func (r *CollectorReconciler) listServices(ctx context.Context, collector *odigosv1.Collector) (*v1.ServiceList, error) {
	var svcList v1.ServiceList
	err := r.List(ctx, &svcList, client.InNamespace(collector.Namespace), client.MatchingFields{ownerKey: collector.Name})
	if err != nil {
		return nil, err
	}

	return &svcList, nil
}
