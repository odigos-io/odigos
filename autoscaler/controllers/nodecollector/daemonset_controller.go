package nodecollector

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

type NodeCollectorDaemonSetReconciler struct {
	client.Client
}

// This reconcile will update the status of the CollectorsGroup CRD of the node collector to indicate
// that it is ready.
// the reason for doing this is to signal to instrumentor to start injecting the agents.
// we don't want to run the agents before the node collector is ready so we know that data can be exported successfully.
// and will not cause errors or memory pressure in the application runtime.
// This should be revisited in the future.
func (r *NodeCollectorDaemonSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.V(0).Info("Reconciling DaemonSet")

	var ds appsv1.DaemonSet
	if err := r.Get(ctx, req.NamespacedName, &ds); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	var datacollectionCollectortGroup odigosv1.CollectorsGroup
	if err := r.Get(ctx, types.NamespacedName{
		Namespace: ds.Namespace,
		Name:      k8sconsts.OdigosNodeCollectorDaemonSetName,
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
