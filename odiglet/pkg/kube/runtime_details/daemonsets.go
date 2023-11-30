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

type DaemonSetsReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (d *DaemonSetsReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var ds appsv1.DaemonSet
	err := d.Client.Get(ctx, request.NamespacedName, &ds)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		logger.Error(err, "error fetching daemonset object")
		return ctrl.Result{}, err
	}

	if isInstrumentationDisabledExplicitly(&ds) {
		return ctrl.Result{}, nil
	}

	if isObjectLabeled(&ds) || isNamespaceLabeled(ctx, &ds, d.Client) {
		return inspectRuntimesOfRunningPods(ctx, &logger, ds.Spec.Selector.MatchLabels, d.Client, d.Scheme, &ds)
	}

	return ctrl.Result{}, nil
}
