package nodecollector

import (
	"context"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	actionutil "github.com/odigos-io/odigos/k8sutils/pkg/action"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
)

// ActionReconciler reconciles config-extension Actions into the node collector ConfigMap.
// SQL query actions move to the node collector when span metrics are enabled so query
// attributes / span names are normalized before spanmetrics records them.
type ActionReconciler struct {
	nodeCollectorBaseReconciler
}

func (r *ActionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	action := &odigosv1.Action{}
	err := r.Get(ctx, req.NamespacedName, action)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, err
		}
		// Deleted actions still need a full node collector sync to drop processors.
		return r.reconcileNodeCollector(ctx)
	}
	if !actionutil.IsConfigExtension(action) {
		// Processor-backed actions are handled via Processor CRs.
		return ctrl.Result{}, nil
	}

	return r.reconcileNodeCollector(ctx)
}
