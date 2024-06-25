package runtime_details

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8sutils "github.com/odigos-io/odigos/k8sutils/pkg/workload"
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
		logger.Error(err, "error fetching namespace object")
		return ctrl.Result{}, err
	}

	if !k8sutils.IsObjectLabeledForInstrumentation(&ns) {
		return ctrl.Result{}, nil
	}

	var deps appsv1.DeploymentList
	err = n.Client.List(ctx, &deps, client.InNamespace(request.Name))
	if client.IgnoreNotFound(err) != nil {
		logger.Error(err, "error fetching deployments")
		return ctrl.Result{}, err
	}

	for _, dep := range deps.Items {
		if !isInstrumentationDisabledExplicitly(&dep) {
			_, err = inspectRuntimesOfRunningPods(ctx, &logger, dep.Spec.Selector.MatchLabels, n.Client, n.Scheme, &dep)
			if err != nil {
				logger.Error(err, "error inspecting runtimes of running pods", "deployment", dep.Name, "namespace", dep.Namespace)
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
		if !isInstrumentationDisabledExplicitly(&st) {
			_, err = inspectRuntimesOfRunningPods(ctx, &logger, st.Spec.Selector.MatchLabels, n.Client, n.Scheme, &st)
			if err != nil {
				logger.Error(err, "error inspecting runtimes of running pods", "statefulset", st.Name, "namespace", st.Namespace)
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
		if !isInstrumentationDisabledExplicitly(&ds) {
			_, err = inspectRuntimesOfRunningPods(ctx, &logger, ds.Spec.Selector.MatchLabels, n.Client, n.Scheme, &ds)
			if err != nil {
				logger.Error(err, "error inspecting runtimes of running pods", "daemonset", ds.Name, "namespace", ds.Namespace)
			}
		}
	}

	return ctrl.Result{}, nil
}
