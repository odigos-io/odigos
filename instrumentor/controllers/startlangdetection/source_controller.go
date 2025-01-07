package startlangdetection

import (
	"context"

	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/k8sutils/pkg/consts"
	k8sutils "github.com/odigos-io/odigos/k8sutils/pkg/utils"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
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

	// If this is a regular Source that is being created, or an Exclusion Source that is being deleted,
	// Attempt to reconcile the workloads for instrumentation.
	if source.DeletionTimestamp.IsZero() != v1alpha1.IsWorkloadExcludedSource(source) {
		if source.Spec.Workload.Kind == "Namespace" {
			// pre-process existing Sources for specific workloads so we don't have to make a bunch of API calls
			// This is used to check if a workload already has an explicit Source, so we don't overwrite its InstrumentationConfig
			sourceList := v1alpha1.SourceList{}
			err := r.Client.List(ctx, &sourceList, client.InNamespace(source.Spec.Workload.Name))
			if err != nil {
				return ctrl.Result{}, err
			}
			namespaceKindSources := make(map[workload.WorkloadKind]map[string]struct{})
			for _, source := range sourceList.Items {
				if _, exists := namespaceKindSources[source.Spec.Workload.Kind]; !exists {
					namespaceKindSources[source.Spec.Workload.Kind] = make(map[string]struct{})
				}
				// ex: map["Deployment"]["my-app"] = ...
				namespaceKindSources[source.Spec.Workload.Kind][source.Spec.Workload.Name] = struct{}{}
			}

			for _, kind := range []workload.WorkloadKind{
				workload.WorkloadKindDaemonSet,
				workload.WorkloadKindDeployment,
				workload.WorkloadKindStatefulSet,
			} {
				result, err := r.listAndReconcileWorkloadList(ctx, source, kind, namespaceKindSources)
				if err != nil {
					return result, err
				}
			}
		} else {
			_, err = reconcileWorkload(ctx,
				r.Client,
				source.Spec.Workload.Kind,
				ctrl.Request{
					NamespacedName: types.NamespacedName{
						Namespace: source.Spec.Workload.Namespace,
						Name:      source.Spec.Workload.Name,
					},
				},
				r.Scheme)
			if err != nil {
				return ctrl.Result{}, err
			}
		}

		if v1alpha1.IsWorkloadExcludedSource(source) && controllerutil.ContainsFinalizer(source, consts.StartLangDetectionFinalizer) {
			controllerutil.RemoveFinalizer(source, consts.StartLangDetectionFinalizer)
			if err := r.Update(ctx, source); err != nil {
				return k8sutils.K8SUpdateErrorHandler(err)
			}
		}
	}

	return ctrl.Result{}, err
}

func (r *SourceReconciler) listAndReconcileWorkloadList(ctx context.Context,
	source *v1alpha1.Source,
	kind workload.WorkloadKind,
	namespaceKindSources map[workload.WorkloadKind]map[string]struct{}) (ctrl.Result, error) {

	deps := workload.ClientListObjectFromWorkloadKind(kind)
	err := r.Client.List(ctx, deps, client.InNamespace(source.Spec.Workload.Name))
	if client.IgnoreNotFound(err) != nil {
		return ctrl.Result{}, err
	}

	switch obj := deps.(type) {
	case *v1.DeploymentList:
		for _, dep := range obj.Items {
			err = r.reconcileWorkloadList(ctx, ctrl.Request{NamespacedName: client.ObjectKey{Name: dep.Name, Namespace: dep.Namespace}}, kind, namespaceKindSources)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
	case *v1.DaemonSetList:
		for _, dep := range obj.Items {
			err = r.reconcileWorkloadList(ctx, ctrl.Request{NamespacedName: client.ObjectKey{Name: dep.Name, Namespace: dep.Namespace}}, kind, namespaceKindSources)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
	case *v1.StatefulSetList:
		for _, dep := range obj.Items {
			err = r.reconcileWorkloadList(ctx, ctrl.Request{NamespacedName: client.ObjectKey{Name: dep.Name, Namespace: dep.Namespace}}, kind, namespaceKindSources)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
	}
	return ctrl.Result{}, nil
}

func (r *SourceReconciler) reconcileWorkloadList(ctx context.Context,
	req ctrl.Request,
	kind workload.WorkloadKind,
	namespaceKindSources map[workload.WorkloadKind]map[string]struct{}) error {
	logger := log.FromContext(ctx)
	if _, exists := namespaceKindSources[kind][req.Name]; !exists {
		_, err := reconcileWorkload(ctx, r.Client, kind, req, r.Scheme)
		if err != nil {
			logger.Error(err, "error requesting runtime details from odiglets", "name", req.Name, "namespace", req.Namespace, "kind", kind)
		}
	}
	return nil
}
