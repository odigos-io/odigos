package controllers

import (
	"context"

	"github.com/odigos-io/odigos/autoscaler/collectormetrics"

	v1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/autoscaler/controllers/gateway"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type OdigosConfigReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	ImagePullSecrets []string
	OdigosVersion    string
	Autoscaler       *collectormetrics.Autoscaler
}

func (r *OdigosConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	logger := log.FromContext(ctx)
	logger.V(0).Info("Reconciling Odigos Configuration")

	err := gateway.Sync(ctx, r.Client, r.Scheme, r.ImagePullSecrets, r.OdigosVersion)
	if err != nil {
		return ctrl.Result{}, err
	}

	r.Autoscaler.Notify() <- collectormetrics.Notification{
		Reason: collectormetrics.OdigosConfigUpdated,
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *OdigosConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.OdigosConfiguration{}).
		Complete(r)
}
