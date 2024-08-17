package startlangdetection

import (
	"context"

	"github.com/odigos-io/odigos/k8sutils/pkg/workload"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type OdigosConfigReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *OdigosConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.V(0).Info("Odigos Configuration changed, recalculating instrumentated application for potential changes of ignored container list")

	var deps appsv1.DeploymentList
	err := r.Client.List(ctx, &deps)
	if client.IgnoreNotFound(err) != nil {
		logger.Error(err, "error fetching deployments")
		return ctrl.Result{}, err
	}

	for _, dep := range deps.Items {
		if !workload.IsInstrumentationDisabledExplicitly(&dep) {
			req := ctrl.Request{NamespacedName: client.ObjectKey{Name: dep.Name, Namespace: dep.Namespace}}
			_, err = reconcileWorkload(ctx, r.Client, &appsv1.Deployment{}, workload.WorkloadKindPascalCaseDeployment, req, r.Scheme)
			if err != nil {
				logger.Error(err, "error requesting runtime details from odiglets", "name", dep.Name, "namespace", dep.Namespace)
			}
		}
	}

	var sts appsv1.StatefulSetList
	err = r.Client.List(ctx, &sts)
	if client.IgnoreNotFound(err) != nil {
		logger.Error(err, "error fetching statefulsets")
		return ctrl.Result{}, err
	}

	for _, st := range sts.Items {
		if !workload.IsInstrumentationDisabledExplicitly(&st) {
			req := ctrl.Request{NamespacedName: client.ObjectKey{Name: st.Name, Namespace: st.Namespace}}
			_, err = reconcileWorkload(ctx, r.Client, &appsv1.StatefulSet{}, workload.WorkloadKindPascalCaseStatefulSet, req, r.Scheme)
			if err != nil {
				logger.Error(err, "error requesting runtime details from odiglets", "name", st.Name, "namespace", st.Namespace)
			}
		}
	}

	var dss appsv1.DaemonSetList
	err = r.Client.List(ctx, &dss)
	if client.IgnoreNotFound(err) != nil {
		logger.Error(err, "error fetching daemonsets")
		return ctrl.Result{}, err
	}

	for _, ds := range dss.Items {
		if !workload.IsInstrumentationDisabledExplicitly(&ds) {
			req := ctrl.Request{NamespacedName: client.ObjectKey{Name: ds.Name, Namespace: ds.Namespace}}
			_, err = reconcileWorkload(ctx, r.Client, &appsv1.DaemonSet{}, workload.WorkloadKindPascalCaseDaemonSet, req, r.Scheme)
			if err != nil {
				logger.Error(err, "error requesting runtime details from odiglets", "name", ds.Name, "namespace", ds.Namespace)
			}
		}
	}

	return ctrl.Result{}, nil
}
