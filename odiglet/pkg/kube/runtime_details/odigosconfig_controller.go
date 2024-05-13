package runtime_details

import (
	"context"

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
		if isWorkloadInstrumentationEffectiveEnabled(ctx, r.Client, &dep) {
			_, err = inspectRuntimesOfRunningPods(ctx, &logger, dep.Spec.Selector.MatchLabels, r.Client, r.Scheme, &dep)
			if err != nil {
				logger.Error(err, "error inspecting runtimes of running pods", "deployment", dep.Name, "namespace", dep.Namespace)
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
		if isWorkloadInstrumentationEffectiveEnabled(ctx, r.Client, &st) {
			_, err = inspectRuntimesOfRunningPods(ctx, &logger, st.Spec.Selector.MatchLabels, r.Client, r.Scheme, &st)
			if err != nil {
				logger.Error(err, "error inspecting runtimes of running pods", "statefulset", st.Name, "namespace", st.Namespace)
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
		if isWorkloadInstrumentationEffectiveEnabled(ctx, r.Client, &ds) {
			_, err = inspectRuntimesOfRunningPods(ctx, &logger, ds.Spec.Selector.MatchLabels, r.Client, r.Scheme, &ds)
			if err != nil {
				logger.Error(err, "error inspecting runtimes of running pods", "daemonset", ds.Name, "namespace", ds.Namespace)
			}
		}
	}

	return ctrl.Result{}, nil
}
