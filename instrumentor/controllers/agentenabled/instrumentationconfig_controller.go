package agentenabled

import (
	"context"

	"github.com/odigos-io/odigos/distros"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type InstrumentationConfigReconciler struct {
	client.Client
	DistrosProvider *distros.Provider
}

func (r *InstrumentationConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return reconcileWorkload(ctx, r.Client, req.Name, req.Namespace, r.DistrosProvider)
}
