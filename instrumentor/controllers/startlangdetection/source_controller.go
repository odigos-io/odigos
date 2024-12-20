package startlangdetection

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
)

var sourceFinalizer = "odigos.io/source-finalizer"

type SourceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *SourceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	source := &v1alpha1.Source{}
	err := r.Get(ctx, req.NamespacedName, source)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	obj := workload.ClientObjectFromWorkloadKind(source.Spec.Workload.Kind)
	err = r.Client.Get(ctx, types.NamespacedName{Name: source.Spec.Workload.Name, Namespace: source.Spec.Workload.Namespace}, obj)
	if err != nil {
		// Deleted objects should be filtered in the event filter
		return ctrl.Result{}, err
	}

	if source.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(source, sourceFinalizer) {
			controllerutil.AddFinalizer(source, sourceFinalizer)
			if err := r.Update(ctx, source); err != nil {
				return ctrl.Result{}, err
			}

			instConfigName := workload.CalculateWorkloadRuntimeObjectName(req.Name, source.Spec.Workload.Kind)
			err = requestOdigletsToCalculateRuntimeDetails(ctx, r.Client, instConfigName, req.Namespace, obj, r.Scheme)
			return ctrl.Result{}, err
		}
	} else {
		// Source is being deleted
		if controllerutil.ContainsFinalizer(source, sourceFinalizer) {
			// TODO: delete resources

			controllerutil.RemoveFinalizer(source, sourceFinalizer)
			if err := r.Update(ctx, source); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	return ctrl.Result{}, err
}
