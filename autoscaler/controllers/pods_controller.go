package controllers

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/odigos-io/odigos/autoscaler/collectormetrics"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type PodsReconciler struct {
	client.Client
	Autoscaler *collectormetrics.Autoscaler
}

func (p *PodsReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	logger := log.FromContext(ctx)
	var pod corev1.Pod
	if err := p.Get(ctx, request.NamespacedName, &pod); err != nil {
		if client.IgnoreNotFound(err) != nil {
			logger.Error(err, "Failed to get pod")
			return reconcile.Result{}, err
		}

		p.Autoscaler.Notify() <- collectormetrics.Notification{
			Reason:  collectormetrics.IPRemoved,
			PodName: request.Name,
		}
	}

	// If IP exists and pod is running
	if pod.Status.PodIP != "" && pod.Status.Phase == corev1.PodRunning {
		p.Autoscaler.Notify() <- collectormetrics.Notification{
			Reason:  collectormetrics.NewIPDiscovered,
			PodName: request.Name,
			IP:      pod.Status.PodIP,
		}
	} else {
		p.Autoscaler.Notify() <- collectormetrics.Notification{
			Reason:  collectormetrics.IPRemoved,
			PodName: request.Name,
		}
	}

	return reconcile.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (p *PodsReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Pod{}).
		WithEventFilter(p.Autoscaler.Predicate()).
		Complete(p)
}
