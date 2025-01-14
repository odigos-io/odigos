package startlangdetection

import (
	"context"
	"errors"

	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/k8sutils/pkg/consts"
	sourceutils "github.com/odigos-io/odigos/k8sutils/pkg/source"
	k8sutils "github.com/odigos-io/odigos/k8sutils/pkg/utils"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
)

type SourceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// SourcePredicates returns true if the Source object is relevant to starting language detection.
// This means that the Source must be either:
// 1) A normal (non-excluding) Source AND NOT terminating, or
// 2) An excluding Source AND terminating
// In either of these cases, we want to check if workloads should start to be instrumented.
var SourcePredicates = predicate.Funcs{
	UpdateFunc: func(e event.UpdateEvent) bool {
		source := e.ObjectNew.(*v1alpha1.Source)
		return sourceutils.IsActiveSource(source)
	},

	CreateFunc: func(e event.CreateEvent) bool {
		source := e.Object.(*v1alpha1.Source)
		return sourceutils.IsActiveSource(source)
	},

	DeleteFunc: func(e event.DeleteEvent) bool {
		return false
	},

	// Allow generic events (e.g., external triggers)
	GenericFunc: func(e event.GenericEvent) bool {
		source := e.Object.(*v1alpha1.Source)
		return sourceutils.IsActiveSource(source)
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

	// If this is a regular Source that is being created, or an Exclusion Source that is being deleted,
	// Attempt to reconcile the workloads for instrumentation.
	// if (terminating && exclude) || (!terminating && !exclude)
	if sourceutils.IsActiveSource(source) {
		if source.Spec.Workload.Kind == "Namespace" {
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

		if v1alpha1.IsExcludedSource(source) &&
			k8sutils.IsTerminating(source) &&
			controllerutil.ContainsFinalizer(source, consts.StartLangDetectionFinalizer) {
			controllerutil.RemoveFinalizer(source, consts.StartLangDetectionFinalizer)
			if err := r.Update(ctx, source); err != nil {
				return k8sutils.K8SUpdateErrorHandler(err)
			}
		}
	}

	return ctrl.Result{}, client.IgnoreNotFound(err)
}

func syncNamespaceWorkloads(ctx context.Context, k8sClient client.Client, runtimeScheme *runtime.Scheme, namespace string) error {
	var err error
	for _, kind := range []workload.WorkloadKind{
		workload.WorkloadKindDaemonSet,
		workload.WorkloadKindDeployment,
		workload.WorkloadKindStatefulSet,
	} {
		err = errors.Join(err, listAndReconcileWorkloadList(ctx, k8sClient, runtimeScheme, namespace, kind))
	}
	return err
}

func listAndReconcileWorkloadList(ctx context.Context,
	k8sClient client.Client,
	runtimeScheme *runtime.Scheme,
	namespace string,
	kind workload.WorkloadKind) error {

	// pre-process existing Sources for specific workloads so we don't have to make a bunch of API calls
	// This is used to check if a workload already has an explicit Source, so we don't overwrite its InstrumentationConfig
	sourceList := v1alpha1.SourceList{}
	err := k8sClient.List(ctx, &sourceList, client.InNamespace(namespace))
	if err != nil {
		return err
	}
	namespaceKindSources := make(map[workload.WorkloadKind]map[string]struct{})
	for _, s := range sourceList.Items {
		if _, exists := namespaceKindSources[s.Spec.Workload.Kind]; !exists {
			namespaceKindSources[s.Spec.Workload.Kind] = make(map[string]struct{})
		}
		// ex: map["Deployment"]["my-app"] = ...
		namespaceKindSources[s.Spec.Workload.Kind][s.Spec.Workload.Name] = struct{}{}
	}

	deps := workload.ClientListObjectFromWorkloadKind(kind)
	err = k8sClient.List(ctx, deps, client.InNamespace(namespace))
	if client.IgnoreNotFound(err) != nil {
		return err
	}

	switch obj := deps.(type) {
	case *v1.DeploymentList:
		for _, dep := range obj.Items {
			err = reconcileWorkloadList(ctx, k8sClient, runtimeScheme, ctrl.Request{NamespacedName: client.ObjectKey{Name: dep.Name, Namespace: dep.Namespace}}, kind, namespaceKindSources)
			if err != nil {
				return err
			}
		}
	case *v1.DaemonSetList:
		for _, dep := range obj.Items {
			err = reconcileWorkloadList(ctx, k8sClient, runtimeScheme, ctrl.Request{NamespacedName: client.ObjectKey{Name: dep.Name, Namespace: dep.Namespace}}, kind, namespaceKindSources)
			if err != nil {
				return err
			}
		}
	case *v1.StatefulSetList:
		for _, dep := range obj.Items {
			err = reconcileWorkloadList(ctx, k8sClient, runtimeScheme, ctrl.Request{NamespacedName: client.ObjectKey{Name: dep.Name, Namespace: dep.Namespace}}, kind, namespaceKindSources)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func reconcileWorkloadList(ctx context.Context,
	k8sClient client.Client,
	runtimeScheme *runtime.Scheme,
	req ctrl.Request,
	kind workload.WorkloadKind,
	namespaceKindSources map[workload.WorkloadKind]map[string]struct{}) error {
	logger := log.FromContext(ctx)
	if _, exists := namespaceKindSources[kind][req.Name]; !exists {
		_, err := reconcileWorkload(ctx, k8sClient, kind, req, runtimeScheme)
		if err != nil {
			logger.Error(err, "error requesting runtime details from odiglets", "name", req.Name, "namespace", req.Namespace, "kind", kind)
		}
	}
	return nil
}
