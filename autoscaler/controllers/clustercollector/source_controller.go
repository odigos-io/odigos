package clustercollector

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type InstrumentationConfigReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	OdigosVersion string
}

// Reconcile ensures that any changes to the InstrumentationConfig CRs (creation, deletion, or label modifications)
// trigger a recalculation of the ConfigMap that configures the routing filter processors.
func (r *InstrumentationConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.V(0).Info("Reconciling InstrumentationConfig")
	return reconcileClusterCollector(ctx, r.Client, r.Scheme, r.OdigosVersion)
}
