package agentenabled

import (
	"context"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type InstrumentationRuleReconciler struct {
	client.Client
}

func (r *InstrumentationRuleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	// Fetch the InstrumentationRule instance
	ir := &odigosv1.InstrumentationRule{}
	err := r.Get(ctx, req.NamespacedName, ir)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// avoid processing instrumentation rules which are not relevant to this controller
	if ir.Spec.OtelSdks == nil {
		return ctrl.Result{}, nil
	}

	return reconcileAll(ctx, r.Client)
}
