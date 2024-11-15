package controllers

import (
	"context"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/k8sutils/pkg/consts"
	predicate "github.com/odigos-io/odigos/k8sutils/pkg/predicate"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type GatewayDeploymentReconciler struct {
	client.Client
}

func (r *GatewayDeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.V(0).Info("Reconciling Deployment")

	var dep appsv1.Deployment
	if err := r.Get(ctx, req.NamespacedName, &dep); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	var gatewayCollectorGroup odigosv1.CollectorsGroup
	if err := r.Get(ctx, types.NamespacedName{
		Namespace: dep.Namespace,
		Name:      consts.OdigosClusterCollectorDeploymentName,
	}, &gatewayCollectorGroup); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	isReady := dep.Status.ReadyReplicas > 0

	if !gatewayCollectorGroup.Status.Ready && isReady {
		err := r.Status().Patch(ctx, &gatewayCollectorGroup, client.RawPatch(
			types.MergePatchType,
			[]byte(`{"status": { "ready": true }}`),
		))
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *GatewayDeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.Deployment{}).
		WithEventFilter(&predicate.ClusterCollectorDeploymentPredicate).
		Complete(r)
}
