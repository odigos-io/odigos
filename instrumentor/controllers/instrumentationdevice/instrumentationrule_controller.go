package instrumentationdevice

import (
	"context"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type InstrumentationRuleReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *InstrumentationRuleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	var instApps odigosv1.InstrumentationConfigList
	if err := r.List(ctx, &instApps); err != nil {
		return ctrl.Result{}, err
	}
	isNodeCollectorReady := isDataCollectionReady(ctx, r.Client)

	gotConflict := false
	for _, runtimeDetails := range instApps.Items {
		err := reconcileSingleWorkload(ctx, r.Client, &runtimeDetails, isNodeCollectorReady)
		if err != nil {
			if apierrors.IsConflict(err) {
				gotConflict = true
			} else {
				return ctrl.Result{}, err
			}
		}
	}

	logger := log.FromContext(ctx)
	logger.V(0).Info("InstrumentationRule changed, recalculating instrumentation device for potential changes of otel sdks")

	if gotConflict {
		return ctrl.Result{Requeue: true}, nil
	} else {
		return ctrl.Result{}, nil
	}
}
