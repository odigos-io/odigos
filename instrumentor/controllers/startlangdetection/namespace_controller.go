package startlangdetection

import (
	"context"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/consts"

	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	k8sutils "github.com/odigos-io/odigos/k8sutils/pkg/workload"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type NamespacesReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (n *NamespacesReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	var ns corev1.Namespace
	err := n.Get(ctx, request.NamespacedName, &ns)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !k8sutils.IsObjectLabeledForInstrumentation(&ns) {
		sourceList, err := v1alpha1.GetSourceListForWorkload(ctx, n.Client, &ns)
		if err != nil {
			return ctrl.Result{}, err
		}
		if len(sourceList.Items) == 0 {
			return ctrl.Result{}, nil
		}
	}

	logger.V(0).Info("Namespace labeled for instrumentation, recalculating runtime details of relevant workloads")
	var deps appsv1.DeploymentList
	err = n.Client.List(ctx, &deps, client.InNamespace(request.Name))
	if client.IgnoreNotFound(err) != nil {
		logger.Error(err, "error fetching deployments")
		return ctrl.Result{}, err
	}

	for _, dep := range deps.Items {
		if _, exists := dep.Labels[consts.OdigosInstrumentationLabel]; !exists {
			req := ctrl.Request{NamespacedName: client.ObjectKey{Name: dep.Name, Namespace: dep.Namespace}}
			_, err = reconcileWorkload(ctx, n.Client, workload.WorkloadKindDeployment, req, n.Scheme)
			if err != nil {
				logger.Error(err, "error requesting runtime details from odiglets", "name", dep.Name, "namespace", dep.Namespace)
			}
		}
	}

	var sts appsv1.StatefulSetList
	err = n.Client.List(ctx, &sts, client.InNamespace(request.Name))
	if client.IgnoreNotFound(err) != nil {
		logger.Error(err, "error fetching statefulsets")
		return ctrl.Result{}, err
	}

	for _, st := range sts.Items {
		if _, exists := st.Labels[consts.OdigosInstrumentationLabel]; !exists {
			req := ctrl.Request{NamespacedName: client.ObjectKey{Name: st.Name, Namespace: st.Namespace}}
			_, err = reconcileWorkload(ctx, n.Client, workload.WorkloadKindStatefulSet, req, n.Scheme)
			if err != nil {
				logger.Error(err, "error requesting runtime details from odiglets", "name", st.Name, "namespace", st.Namespace)
			}
		}
	}

	var dss appsv1.DaemonSetList
	err = n.Client.List(ctx, &dss, client.InNamespace(request.Name))
	if client.IgnoreNotFound(err) != nil {
		logger.Error(err, "error fetching daemonsets")
		return ctrl.Result{}, err
	}

	for _, ds := range dss.Items {
		if _, exists := ds.Labels[consts.OdigosInstrumentationLabel]; !exists {
			req := ctrl.Request{NamespacedName: client.ObjectKey{Name: ds.Name, Namespace: ds.Namespace}}
			_, err = reconcileWorkload(ctx, n.Client, workload.WorkloadKindDaemonSet, req, n.Scheme)
			if err != nil {
				logger.Error(err, "error requesting runtime details from odiglets", "name", ds.Name, "namespace", ds.Namespace)
			}
		}
	}

	return ctrl.Result{}, nil
}
