package loglevel

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	commonlogger "github.com/odigos-io/odigos/common/logger"
	odigospredicate "github.com/odigos-io/odigos/k8sutils/pkg/predicate"
	k8sutils "github.com/odigos-io/odigos/k8sutils/pkg/utils"
)

type LogLevelReconciler struct {
	client.Client
}

func (r *LogLevelReconciler) Reconcile(ctx context.Context, _ ctrl.Request) (ctrl.Result, error) {
	cfg, err := k8sutils.GetCurrentOdigosConfiguration(ctx, r.Client)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	level := "info"
	if cfg.ComponentLogLevels != nil {
		level = cfg.ComponentLogLevels.Resolve("autoscaler")
	}
	commonlogger.SetLevel(level)
	return ctrl.Result{}, nil
}

func SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		Named("loglevel-effectiveconfig").
		For(&corev1.ConfigMap{}).
		WithEventFilter(&odigospredicate.OdigosEffectiveConfigMapPredicate).
		Complete(&LogLevelReconciler{Client: mgr.GetClient()})
}
