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
		if !controllerutil.ContainsFinalizer(source, consts.SourceFinalizer) {
			controllerutil.AddFinalizer(source, consts.SourceFinalizer)
			// Removed by deleteinstrumentationconfig controller
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

		if source.Spec.Workload.Kind == "Namespace" {
			var deps appsv1.DeploymentList
			err = r.Client.List(ctx, &deps, client.InNamespace(source.Spec.Workload.Name))
			if client.IgnoreNotFound(err) != nil {
				logger.Error(err, "error fetching deployments")
				return ctrl.Result{}, err
			}

			for _, dep := range deps.Items {
				request := ctrl.Request{NamespacedName: client.ObjectKey{Name: dep.Name, Namespace: dep.Namespace}}
				_, err = reconcileWorkload(ctx, r.Client, workload.WorkloadKindDeployment, request, r.Scheme)
				if err != nil {
					logger.Error(err, "error requesting runtime details from odiglets", "name", dep.Name, "namespace", dep.Namespace)
				}
			}

			var sts appsv1.StatefulSetList
			err = r.Client.List(ctx, &sts, client.InNamespace(source.Spec.Workload.Name))
			if client.IgnoreNotFound(err) != nil {
				logger.Error(err, "error fetching statefulsets")
				return ctrl.Result{}, err
			}

			for _, st := range sts.Items {
				request := ctrl.Request{NamespacedName: client.ObjectKey{Name: st.Name, Namespace: st.Namespace}}
				_, err = reconcileWorkload(ctx, r.Client, workload.WorkloadKindStatefulSet, request, r.Scheme)
				if err != nil {
					logger.Error(err, "error requesting runtime details from odiglets", "name", st.Name, "namespace", st.Namespace)
				}
			}

			var dss appsv1.DaemonSetList
			err = r.Client.List(ctx, &dss, client.InNamespace(source.Spec.Workload.Name))
			if client.IgnoreNotFound(err) != nil {
				logger.Error(err, "error fetching daemonsets")
				return ctrl.Result{}, err
			}

			for _, ds := range dss.Items {
				request := ctrl.Request{NamespacedName: client.ObjectKey{Name: ds.Name, Namespace: ds.Namespace}}
				_, err = reconcileWorkload(ctx, r.Client, workload.WorkloadKindDaemonSet, request, r.Scheme)
				if err != nil {
					logger.Error(err, "error requesting runtime details from odiglets", "name", ds.Name, "namespace", ds.Namespace)
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
