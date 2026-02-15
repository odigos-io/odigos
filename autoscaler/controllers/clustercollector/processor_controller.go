package clustercollector

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type ProcessorReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	OdigosVersion string
}

func (r *ProcessorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.V(0).Info("Reconciling Processor")
	return reconcileClusterCollector(ctx, r.Client, r.Scheme, r.OdigosVersion)
}
