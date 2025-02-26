package sourceinstrumentation

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

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
	logger.Info("Reconciling Source object", "name", req.Name, "namespace", req.Namespace)
	source := &v1alpha1.Source{}
	err := r.Get(ctx, req.NamespacedName, source)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	var reconcileFunc reconcileFunction
	if sourceutils.SourceEnablesInstrumentation(source) {
		reconcileFunc = instrumentWorkload
	} else {
		reconcileFunc = uninstrumentWorkload
	}

	if source.Spec.Workload.Kind == k8sconsts.WorkloadKindNamespace {
		res, err := syncNamespaceWorkloads(
			ctx,
			r.Client,
			r.Scheme,
			source.Spec.Workload.Name,
			reconcileFunc)
		if err != nil {
			return res, err
		}
	} else {
		res, err := reconcileFunc(
			ctx,
			r.Client,
			source.Spec.Workload.Kind,
			types.NamespacedName{
				Namespace: source.Spec.Workload.Namespace,
				Name:      source.Spec.Workload.Name,
			},
			r.Scheme)
		if err != nil {
			return res, err
		}
	}

	if controllerutil.ContainsFinalizer(source, k8sconsts.StartLangDetectionFinalizer) {
		controllerutil.RemoveFinalizer(source, k8sconsts.StartLangDetectionFinalizer)
	}
	if controllerutil.ContainsFinalizer(source, k8sconsts.DeleteInstrumentationConfigFinalizer) {
		controllerutil.RemoveFinalizer(source, k8sconsts.DeleteInstrumentationConfigFinalizer)
	}

	if !source.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(source, k8sconsts.SourceInstrumentationFinalizer) {
			controllerutil.RemoveFinalizer(source, k8sconsts.SourceInstrumentationFinalizer)
		}
		if err := r.Update(ctx, source); err != nil {
			return k8sutils.K8SUpdateErrorHandler(err)
		}
	}

	return ctrl.Result{}, client.IgnoreNotFound(err)
}
