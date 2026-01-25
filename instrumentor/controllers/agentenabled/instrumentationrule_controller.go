package agentenabled

import (
	"context"

	"github.com/odigos-io/odigos/distros"
	"github.com/odigos-io/odigos/instrumentor/controllers/agentenabled/rollout"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type InstrumentationRuleReconciler struct {
	client.Client
	DistrosProvider    *distros.Provider
	RolloutRateLimiter *rollout.RolloutRateLimiter
}

func (r *InstrumentationRuleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// This reconciler is fired everytime an instruentation rule that is relevant for agent injection
	// is either created, updated or deleted.
	// we might get an event here with relevant rules set to nil,
	// but they should still be processed to potentially revert thier original effects.
	// thus it is very important to have strong filtering in the predicate
	// so not to execute this reconciler too much when not needed.
	return reconcileAll(ctx, r.Client, r.DistrosProvider, r.RolloutRateLimiter)
}
