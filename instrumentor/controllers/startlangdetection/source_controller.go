package startlangdetection

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
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

	if source.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(source, consts.InstrumentedApplicationFinalizer) {
			controllerutil.AddFinalizer(source, consts.InstrumentedApplicationFinalizer)
		}
		if source.Labels == nil {
			source.Labels = make(map[string]string)
		}

		source.Labels[consts.WorkloadNameLabel] = source.Spec.Workload.Name
		source.Labels[consts.WorkloadNamespaceLabel] = source.Spec.Workload.Namespace
		source.Labels[consts.WorkloadKindLabel] = string(source.Spec.Workload.Kind)

		if err := r.Update(ctx, source); err != nil {
			return k8sutils.K8SUpdateErrorHandler(err)
		}

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

		if source.Spec.Workload.Kind == "Namespace" {
			var deps appsv1.DeploymentList
			err = r.Client.List(ctx, &deps, client.InNamespace(source.Spec.Workload.Name))
			if client.IgnoreNotFound(err) != nil {
				logger.Error(err, "error fetching deployments")
				return ctrl.Result{}, err
			}

			for _, dep := range deps.Items {
				if _, exists := namespaceKindSources[workload.WorkloadKindDeployment][dep.Name]; !exists {
					request := ctrl.Request{NamespacedName: client.ObjectKey{Name: dep.Name, Namespace: dep.Namespace}}
					_, err = reconcileWorkload(ctx, r.Client, workload.WorkloadKindDeployment, request, r.Scheme)
					if err != nil {
						logger.Error(err, "error requesting runtime details from odiglets", "name", dep.Name, "namespace", dep.Namespace)
					}
				}
			}

			var sts appsv1.StatefulSetList
			err = r.Client.List(ctx, &sts, client.InNamespace(source.Spec.Workload.Name))
			if client.IgnoreNotFound(err) != nil {
				logger.Error(err, "error fetching statefulsets")
				return ctrl.Result{}, err
			}

			for _, st := range sts.Items {
				if _, exists := namespaceKindSources[workload.WorkloadKindStatefulSet][st.Name]; !exists {
					request := ctrl.Request{NamespacedName: client.ObjectKey{Name: st.Name, Namespace: st.Namespace}}
					_, err = reconcileWorkload(ctx, r.Client, workload.WorkloadKindStatefulSet, request, r.Scheme)
					if err != nil {
						logger.Error(err, "error requesting runtime details from odiglets", "name", st.Name, "namespace", st.Namespace)
					}
				}
			}

			var dss appsv1.DaemonSetList
			err = r.Client.List(ctx, &dss, client.InNamespace(source.Spec.Workload.Name))
			if client.IgnoreNotFound(err) != nil {
				logger.Error(err, "error fetching daemonsets")
				return ctrl.Result{}, err
			}

			for _, ds := range dss.Items {
				if _, exists := namespaceKindSources[workload.WorkloadKindDaemonSet][ds.Name]; !exists {
					request := ctrl.Request{NamespacedName: client.ObjectKey{Name: ds.Name, Namespace: ds.Namespace}}
					_, err = reconcileWorkload(ctx, r.Client, workload.WorkloadKindDaemonSet, request, r.Scheme)
					if err != nil {
						logger.Error(err, "error requesting runtime details from odiglets", "name", ds.Name, "namespace", ds.Namespace)
					}
				}
			}
		} else {
			return reconcileWorkload(ctx,
				r.Client,
				source.Spec.Workload.Kind,
				ctrl.Request{
					NamespacedName: types.NamespacedName{
						Namespace: source.Spec.Workload.Namespace,
						Name:      source.Spec.Workload.Name,
					},
				},
				r.Scheme)
		}
	}

	return ctrl.Result{}, err
}
