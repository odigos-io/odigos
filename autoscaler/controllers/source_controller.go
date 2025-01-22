package controllers

import (
	"context"

	controllerconfig "github.com/odigos-io/odigos/autoscaler/controllers/controller_config"
	"github.com/odigos-io/odigos/autoscaler/controllers/gateway"
	odigospredicate "github.com/odigos-io/odigos/k8sutils/pkg/predicate"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *SourceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.Source{}).
		WithEventFilter(predicate.Or(
			odigospredicate.ExistencePredicate{},
			predicate.LabelChangedPredicate{},
		)).
		Complete(r)
}
