package startlangdetection

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
)

var (
	sourceFinalizer = "odigos.io/source-finalizer"

	workloadNameLabel      = "odigos.io/workload-name"
	workloadNamespaceLabel = "odigos.io/workload-namespace"
	workloadKindLabel      = "odigos.io/workload-kind"
)

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
		// TODO: Deleted objects should be filtered in the event filter
		return ctrl.Result{}, err
	}
	instConfigName := workload.CalculateWorkloadRuntimeObjectName(source.Spec.Workload.Name, source.Spec.Workload.Kind)

	if source.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(source, sourceFinalizer) {
			controllerutil.AddFinalizer(source, sourceFinalizer)

			if source.Labels == nil {
				source.Labels = make(map[string]string)
			}
			source.Labels[workloadNameLabel] = source.Spec.Workload.Name
			source.Labels[workloadNamespaceLabel] = source.Spec.Workload.Namespace
			source.Labels[workloadKindLabel] = string(source.Spec.Workload.Kind)

			if err := r.Update(ctx, source); err != nil {
				return ctrl.Result{}, err
			}

			err = requestOdigletsToCalculateRuntimeDetails(ctx, r.Client, instConfigName, req.Namespace, obj, r.Scheme)
			return ctrl.Result{}, err
		}
	} else {
		// Source is being deleted
		if controllerutil.ContainsFinalizer(source, sourceFinalizer) {
			// Remove the finalizer first, because if the InstrumentationConfig is not found we
			// will deadlock on the finalizer never getting removed.
			// On the other hand, this could end up deleting a Source with an orphaned InstrumentationConfig.
			controllerutil.RemoveFinalizer(source, sourceFinalizer)
			if err := r.Update(ctx, source); err != nil {
				return ctrl.Result{}, err
			}

			instConfig := &v1alpha1.InstrumentationConfig{}
			err = r.Client.Get(ctx, types.NamespacedName{Name: instConfigName, Namespace: req.Namespace}, instConfig)
			if err != nil {
				return ctrl.Result{}, client.IgnoreNotFound(err)
			}
			err = r.Client.Delete(ctx, instConfig)
			if err != nil {
				return ctrl.Result{}, client.IgnoreNotFound(err)
			}
		}
	}

	return ctrl.Result{}, err
}
