package controllers

import (
	"context"

	controllerconfig "github.com/odigos-io/odigos/autoscaler/controllers/controller_config"
	odigospredicate "github.com/odigos-io/odigos/k8sutils/pkg/predicate"

	"github.com/odigos-io/odigos/autoscaler/controllers/gateway"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

type OdigosConfigReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	ImagePullSecrets []string
	OdigosVersion    string
	Config           *controllerconfig.ControllerConfig
}

func (r *OdigosConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.V(0).Info("Reconciling Odigos Configuration")

	err := gateway.Sync(ctx, r.Client, r.Scheme, r.ImagePullSecrets, r.OdigosVersion, r.Config.MetricsServerEnabled)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *OdigosConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.ConfigMap{}).
		WithEventFilter(predicate.And(odigospredicate.OdigosConfigMapPredicate, odigospredicate.ConfigMapDataChangedPredicate{})).
		Complete(r)
}
