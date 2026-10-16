package clustercollector

import (
	"context"

	commonlogger "github.com/odigos-io/odigos/common/logger"
	"github.com/odigos-io/odigos/instrumentor/controllers/collectorhooks"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type SecretReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	OdigosVersion string
	Hooks         *collectorhooks.CollectorHooks
}

func (r *SecretReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := commonlogger.FromContext(ctx)
	logger.Info("Reconciling Secret")
	return reconcileClusterCollector(ctx, r.Client, r.Scheme, r.OdigosVersion, r.Hooks)
}
