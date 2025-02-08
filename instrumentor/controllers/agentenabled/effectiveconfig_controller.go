package agentenabled

import (
	"context"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type EffectiveConfigReconciler struct {
	client.Client
}

func (r *EffectiveConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return reconcileAll(ctx, r.Client)
}
