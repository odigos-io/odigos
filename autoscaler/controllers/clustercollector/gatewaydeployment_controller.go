package clustercollector

import (
	"context"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type ClusterCollectorDeploymentReconciler struct {
	client.Client
}

// This controller is used to track and update the collectors group status once the cluster collector is ready to receive data.
// We don't want to spin up components that needs to export to cluster collector and have errors and memory pressure.
func (r *ClusterCollectorDeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.V(0).Info("Reconciling Deployment")

	var dep appsv1.Deployment
	if err := r.Get(ctx, req.NamespacedName, &dep); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	var gatewayCollectorGroup odigosv1.CollectorsGroup
	if err := r.Get(ctx, types.NamespacedName{
		Namespace: dep.Namespace,
		Name:      k8sconsts.OdigosClusterCollectorCollectorGroupName,
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
