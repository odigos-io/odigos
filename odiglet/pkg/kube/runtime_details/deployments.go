package runtime_details

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type DeploymentsReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (d *DeploymentsReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	var dep appsv1.Deployment
	err := d.Client.Get(ctx, request.NamespacedName, &dep)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		logger.Error(err, "error fetching deployment object")
		return ctrl.Result{}, err
	}

	if !isWorkloadInstrumentationEffectiveEnabled(ctx, d.Client, &dep) {
		return ctrl.Result{}, nil
	}

	return inspectRuntimesOfRunningPods(ctx, &logger, dep.Spec.Selector.MatchLabels, d.Client, d.Scheme, &dep)
}
