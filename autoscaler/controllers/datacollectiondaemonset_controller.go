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

type DataCollectionDaemonSetReconciler struct {
	client.Client
}

func (r *DataCollectionDaemonSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.V(0).Info("Reconciling DaemonSet")

	var ds appsv1.DaemonSet
	if err := r.Get(ctx, req.NamespacedName, &ds); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	var datacollectionCollectortGroup odigosv1.CollectorsGroup
	if err := r.Get(ctx, types.NamespacedName{
		Namespace: ds.Namespace,
		Name:      consts.OdigosNodeCollectorDaemonSetName,
	}, &datacollectionCollectortGroup); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	isNowReady := calcDataCollectionReadyStatus(&ds)
	if !datacollectionCollectortGroup.Status.Ready && isNowReady {
		if err := r.Status().Patch(ctx, &datacollectionCollectortGroup, client.RawPatch(
			types.MergePatchType,
			[]byte(`{"status": { "ready": true }}`),
		)); err != nil {
			logger.Error(err, "Failed to update data collection status")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// Data collection is ready if at least 50% of the pods are ready
func calcDataCollectionReadyStatus(ds *appsv1.DaemonSet) bool {
	return ds.Status.DesiredNumberScheduled > 0 && float64(ds.Status.NumberReady) >= float64(ds.Status.DesiredNumberScheduled)/float64(2)
}

// SetupWithManager sets up the controller with the Manager.
func (r *DataCollectionDaemonSetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.DaemonSet{}).
		WithEventFilter(&predicate.NodeCollectorsDaemonSetPredicate).
		Complete(r)
}
