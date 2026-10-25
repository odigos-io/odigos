package clustercollector

import (
	"context"

	"github.com/odigos-io/odigos/common"
	k8slogger "github.com/odigos-io/odigos/k8sutils/pkg/logger"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type CollectorsGroupReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	OdigosVersion string
	Tier          common.OdigosTier
}

func (r *CollectorsGroupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := k8slogger.FromContext(ctx)
	logger.Info("Reconciling CollectorsGroup")
	return reconcileClusterCollector(ctx, r.Client, r.Scheme, r.OdigosVersion, r.Tier)
}
