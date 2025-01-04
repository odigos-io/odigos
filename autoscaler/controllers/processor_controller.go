package controllers

import (
	"context"

	v1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	controllerconfig "github.com/odigos-io/odigos/autoscaler/controllers/controller_config"
	"github.com/odigos-io/odigos/autoscaler/controllers/datacollection"
	"github.com/odigos-io/odigos/autoscaler/controllers/gateway"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/version"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

type ProcessorReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	ImagePullSecrets []string
	OdigosVersion    string
	K8sVersion       *version.Version
	Config           *controllerconfig.ControllerConfig
}

func (r *ProcessorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	logger := log.FromContext(ctx)
	logger.V(0).Info("Reconciling Processor")

	err := gateway.Sync(ctx, r.Client, r.Scheme, r.ImagePullSecrets, r.OdigosVersion, r.Config)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = datacollection.Sync(ctx, r.Client, r.Scheme, r.ImagePullSecrets, r.OdigosVersion, r.K8sVersion)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *ProcessorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.Processor{}).
		// auto scaler only cares about the spec of each processor.
		// filter out events on resource status and metadata changes.
		WithEventFilter(&predicate.GenerationChangedPredicate{}).
		Complete(r)
}
