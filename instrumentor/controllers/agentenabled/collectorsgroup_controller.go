package agentenabled

import (
	"context"

	"github.com/odigos-io/odigos/distros"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type CollectorsGroupReconciler struct {
	client.Client
	DistrosProvider *distros.Provider
}

func (r *CollectorsGroupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return reconcileAll(ctx, r.Client, r.DistrosProvider)
}
