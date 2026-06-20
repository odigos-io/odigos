package nodecollector

import (
	"context"

	v1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	commonlogger "github.com/odigos-io/odigos/common/logger"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

type ReceiverReconciler struct {
	nodeCollectorBaseReconciler
}

func (r *ReceiverReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := commonlogger.FromContext(ctx)
	logger.Info("Reconciling Receiver")
	return r.reconcileNodeCollector(ctx)
}

func (r *ReceiverReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.Receiver{}).
		WithEventFilter(&predicate.GenerationChangedPredicate{}).
		Complete(r)
}
