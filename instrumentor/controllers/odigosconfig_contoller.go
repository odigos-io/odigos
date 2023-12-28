package controllers

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type OdigosConfigReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// odigos config is used to determine the default SDKs to use for each language
// when the config changes (for example, when upgrading from community to cloud tier)
// we may need to re-calculate the instrumentation devices for existing workloads
func (r *OdigosConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return ctrl.Result{}, nil
}
