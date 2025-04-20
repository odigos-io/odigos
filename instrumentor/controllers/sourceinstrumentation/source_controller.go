package sourceinstrumentation

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	sourceutils "github.com/odigos-io/odigos/k8sutils/pkg/source"
	k8sutils "github.com/odigos-io/odigos/k8sutils/pkg/utils"
)

type SourceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *SourceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	source := &v1alpha1.Source{}
	var err error
	err = r.Get(ctx, req.NamespacedName, source)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	logger.Info("Reconciling Source object", "name", req.Name, "namespace", req.Namespace, "workload-kind", source.Spec.Workload.Kind, "workload-name", source.Spec.Workload.Name)

	var result ctrl.Result
	if source.Spec.Workload.Kind == k8sconsts.WorkloadKindNamespace {
		result, err = syncNamespaceWorkloads(ctx, r.Client, r.Scheme, source.Spec.Workload.Namespace)
	} else {
		// Get the object referenced by the Source to check whether the workload is being actively instrumented.
		// The Source itself doesn't have enough information about the global state of this workload:
		// For example, a deleted Workload Source might still be covered by a Namespace Source.
		var obj client.Object
		obj, err = sourceutils.GetClientObjectFromSource(ctx, r.Client, source)
		if client.IgnoreNotFound(err) != nil {
			// re-queue on any error besides NotFound
			return ctrl.Result{}, err
		}
		if obj != nil {
			// NotFound will return a nil object, nothing to sync without a workload obj
			result, err = syncWorkload(ctx, r.Client, r.Scheme, obj)
		}
	}
	// We could get a non-error Requeue signal from the reconcile functions,
	// such as a conflict updating the instrumentationconfig status
	if !result.IsZero() || client.IgnoreNotFound(err) != nil {
		// either the result is non-zero, or we had a non-NotFound error
		// need to filter NotFound errors out
		return result, client.IgnoreNotFound(err)
	}

	if k8sutils.IsTerminating(source) {
		// Migration: Remove old finalizers if present, these will be removed
		if controllerutil.ContainsFinalizer(source, k8sconsts.StartLangDetectionFinalizer) {
			controllerutil.RemoveFinalizer(source, k8sconsts.StartLangDetectionFinalizer)
		}
		if controllerutil.ContainsFinalizer(source, k8sconsts.DeleteInstrumentationConfigFinalizer) {
			controllerutil.RemoveFinalizer(source, k8sconsts.DeleteInstrumentationConfigFinalizer)
		}
		if controllerutil.ContainsFinalizer(source, k8sconsts.SourceInstrumentationFinalizer) {
			controllerutil.RemoveFinalizer(source, k8sconsts.SourceInstrumentationFinalizer)
		}
		if err := r.Update(ctx, source); err != nil {
			return k8sutils.K8SUpdateErrorHandler(err)
		}
	}

	return ctrl.Result{}, nil
}
