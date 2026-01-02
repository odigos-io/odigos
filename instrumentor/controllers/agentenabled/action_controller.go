package agentenabled

import (
	"context"

	"github.com/odigos-io/odigos/distros"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ActionReconciler struct {
	client.Client
	DistrosProvider *distros.Provider
}

func (r *ActionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// This reconciler is fired everytime an action with URLTemplatization config
	// is created, updated or deleted.
	// When URLTemplatization actions change, we need to reconcile all workloads
	// to ensure the correct instrumentation configuration is applied.
	return reconcileAll(ctx, r.Client, r.DistrosProvider)
}
