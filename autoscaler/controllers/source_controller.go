package controllers

import (
	"context"

	controllerconfig "github.com/odigos-io/odigos/autoscaler/controllers/controller_config"
	"github.com/odigos-io/odigos/autoscaler/controllers/gateway"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	v1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
)

type SourceReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	ImagePullSecrets []string
	OdigosVersion    string
	Config           *controllerconfig.ControllerConfig
}

// Reconcile ensures that any changes to Source CRDs (creation, deletion, or label modifications)
// trigger a recalculation of the ConfigMap that configures the routing filter processors.
func (r *SourceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.V(0).Info("Reconciling Source")

	err := gateway.Sync(ctx, r.Client, r.Scheme, r.ImagePullSecrets, r.OdigosVersion, r.Config)
	if err != nil {
		logger.Error(err, "Failed to sync gateway configuration")
		return ctrl.Result{}, err
	}

	logger.V(1).Info("Successfully synced gateway configuration for Source", "Source", req.NamespacedName)
	return ctrl.Result{}, nil
}

func (r *SourceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.Source{}).
		// Trigger reconciliation on create, delete, or label changes
		WithEventFilter(predicate.Or(
			predicate.GenerationChangedPredicate{},
			predicate.LabelChangedPredicate{},
			predicate.Funcs{
				DeleteFunc: func(e event.DeleteEvent) bool {
					return true
				},
			},
		)).
		Complete(r)
}
