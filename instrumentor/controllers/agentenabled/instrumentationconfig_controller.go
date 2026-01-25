package agentenabled

import (
	"context"

	"github.com/odigos-io/odigos/distros"
	"github.com/odigos-io/odigos/instrumentor/controllers/agentenabled/rollout"
	k8sutils "github.com/odigos-io/odigos/k8sutils/pkg/utils"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type InstrumentationConfigReconciler struct {
	client.Client
	DistrosProvider    *distros.Provider
	RolloutRateLimiter *rollout.RolloutRateLimiter
}

func (r *InstrumentationConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	conf, err := k8sutils.GetCurrentOdigosConfiguration(ctx, r.Client)
	if err != nil {
		return ctrl.Result{}, err
	}

	return reconcileWorkload(ctx, r.Client, req.Name, req.Namespace, r.DistrosProvider, &conf, r.RolloutRateLimiter)
}
