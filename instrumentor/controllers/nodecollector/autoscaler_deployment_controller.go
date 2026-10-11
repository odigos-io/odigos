package nodecollector

import (
	"context"

	ctrl "sigs.k8s.io/controller-runtime"
)

type AutoscalerDeploymentReconciler struct {
	nodeCollectorBaseReconciler
}

func (r *AutoscalerDeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.reconcileNodeCollector(ctx)
}
