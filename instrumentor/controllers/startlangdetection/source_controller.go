package startlangdetection

import (
	"context"
	"errors"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	sourceutils "github.com/odigos-io/odigos/k8sutils/pkg/source"
	k8sutils "github.com/odigos-io/odigos/k8sutils/pkg/utils"
)

type SourceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// StartLangDetectionSourcePredicate returns true if the Source object is relevant to starting language detection.
// This means that the Source must be either:
// 1) A normal (non-excluding) Source AND NOT terminating, or
// 2) An excluding Source AND terminating
// In either of these cases, we want to check if workloads should start to be instrumented.
var StartLangDetectionSourcePredicate = predicate.Funcs{
	UpdateFunc: func(e event.UpdateEvent) bool {
		source := e.ObjectNew.(*v1alpha1.Source)
		return sourceutils.IsSourceRelevant(source)
	},

	CreateFunc: func(e event.CreateEvent) bool {
		source := e.Object.(*v1alpha1.Source)
		return sourceutils.IsSourceRelevant(source)
	},

	DeleteFunc: func(e event.DeleteEvent) bool {
		return false
	},

	// Allow generic events (e.g., external triggers)
	GenericFunc: func(e event.GenericEvent) bool {
		source := e.Object.(*v1alpha1.Source)
		return sourceutils.IsSourceRelevant(source)
	},
}

func (r *SourceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling Source object", "name", req.Name, "namespace", req.Namespace)
	source := &v1alpha1.Source{}
	err := r.Get(ctx, req.NamespacedName, source)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if source.Spec.Workload.Kind == k8sconsts.WorkloadKindNamespace {
		err = errors.Join(err, syncNamespaceWorkloads(ctx, r.Client, r.Scheme, source.Spec.Workload.Name))
	} else {
		_, reconcileErr := reconcileWorkload(ctx,
			r.Client,
			source.Spec.Workload.Kind,
			ctrl.Request{
				NamespacedName: types.NamespacedName{
					Namespace: source.Spec.Workload.Namespace,
					Name:      source.Spec.Workload.Name,
				},
			},
			r.Scheme)
		if reconcileErr != nil {
			err = errors.Join(err, reconcileErr)
		}
	}

	if client.IgnoreNotFound(err) != nil {
		return ctrl.Result{}, err
	}

	if v1alpha1.IsDisabledSource(source) &&
		k8sutils.IsTerminating(source) &&
		controllerutil.ContainsFinalizer(source, k8sconsts.StartLangDetectionFinalizer) {
		controllerutil.RemoveFinalizer(source, k8sconsts.StartLangDetectionFinalizer)
		if err := r.Update(ctx, source); err != nil {
			return k8sutils.K8SUpdateErrorHandler(err)
		}
	}

	return ctrl.Result{}, client.IgnoreNotFound(err)
}
