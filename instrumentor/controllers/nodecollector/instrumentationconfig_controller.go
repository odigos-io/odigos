package nodecollector

import (
	"context"

	ctrl "sigs.k8s.io/controller-runtime"
)

type InstrumentationConfigReconciler struct {
	nodeCollectorBaseReconciler
}

func (r *InstrumentationConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.reconcileNodeCollector(ctx)
}
