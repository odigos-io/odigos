package agentenabled

import (
	"context"

	"github.com/odigos-io/odigos/distros"
	"github.com/odigos-io/odigos/instrumentor/controllers/agentenabled/rollout"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	commonlogger "github.com/odigos-io/odigos/common/logger"
	k8sutils "github.com/odigos-io/odigos/k8sutils/pkg/utils"
)

type EffectiveConfigReconciler struct {
	client.Client
	DistrosProvider           *distros.Provider
	RolloutConcurrencyLimiter *rollout.RolloutConcurrencyLimiter
}

func (r *EffectiveConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	if cfg, err := k8sutils.GetCurrentOdigosConfiguration(ctx, r.Client); err == nil && cfg.ComponentLogLevels != nil {
		commonlogger.SetLevel(cfg.ComponentLogLevels.Resolve("instrumentor"))
	}
	return reconcileAll(ctx, r.Client, r.DistrosProvider, r.RolloutConcurrencyLimiter)
}
