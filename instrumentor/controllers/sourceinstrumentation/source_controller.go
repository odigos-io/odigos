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
	err := r.Get(ctx, req.NamespacedName, source)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	logger.Info("Reconciling Source object", "name", req.Name, "namespace", req.Namespace, "workload-kind", source.Spec.Workload.Kind, "workload-name", source.Spec.Workload.Name)

	var reconcileFunc reconcileFunction
	if sourceutils.SourceStatePermitsInstrumentation(source) {
		reconcileFunc = instrumentWorkload
	} else {
		reconcileFunc = uninstrumentWorkload
	}

	// Sync based on the Source object's workload kind
	// An error from the sync functions will trigger a re-sync, except for NotFound errors
	// In a NotFound case, we still want to progress to removing the finalizer if necessary
	if source.Spec.Workload.Kind == k8sconsts.WorkloadKindNamespace {
		res, err := syncNamespaceWorkloads(
			ctx,
			r.Client,
			r.Scheme,
			source.Spec.Workload.Name,
			reconcileFunc)
		if client.IgnoreNotFound(err) != nil {
			return res, err
		}
	} else {
		res, err := reconcileFunc(
			ctx,
			r.Client,
			source.Spec.Workload,
			r.Scheme)
		if client.IgnoreNotFound(err) != nil {
			return res, err
		}
	}

	if k8sutils.IsTerminating(source) {
		if controllerutil.ContainsFinalizer(source, k8sconsts.SourceInstrumentationFinalizer) {
			controllerutil.RemoveFinalizer(source, k8sconsts.SourceInstrumentationFinalizer)
		}
		if err := r.Update(ctx, source); err != nil {
			return k8sutils.K8SUpdateErrorHandler(err)
		}
	}

	return ctrl.Result{}, nil
}
