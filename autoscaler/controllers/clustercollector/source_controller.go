package clustercollector

import (
	"context"

	commonlogger "github.com/odigos-io/odigos/common/logger"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type InstrumentationConfigReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	OdigosVersion string
}

// Reconcile ensures that any changes to the InstrumentationConfig CRs (creation, deletion, or label modifications)
// trigger a recalculation of the ConfigMap that configures the routing filter processors.
func (r *InstrumentationConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := commonlogger.FromContext(ctx)
	logger.Info("Reconciling InstrumentationConfig")
	return reconcileClusterCollector(ctx, r.Client, r.Scheme, r.OdigosVersion)
}
