package startlangdetection

import (
	"context"
	"errors"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"

	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func syncNamespaceWorkloads(ctx context.Context, k8sClient client.Client, runtimeScheme *runtime.Scheme, namespace string) (ctrl.Result, error) {
	collectiveRes := ctrl.Result{}
	var errs error
	for _, kind := range []k8sconsts.WorkloadKind{
		k8sconsts.WorkloadKindDaemonSet,
		k8sconsts.WorkloadKindDeployment,
		k8sconsts.WorkloadKindStatefulSet,
	} {
		res, err := listAndReconcileWorkloadList(ctx, k8sClient, runtimeScheme, namespace, kind)
		errs = errors.Join(errs, err)
		collectiveRes = joinCtrlResultToList(res, collectiveRes)
	}
	return collectiveRes, errs
}

func listAndReconcileWorkloadList(ctx context.Context,
	k8sClient client.Client,
	runtimeScheme *runtime.Scheme,
	namespace string,
	kind k8sconsts.WorkloadKind) (ctrl.Result, error) {

	// pre-process existing Sources for specific workloads so we don't have to make a bunch of API calls
	// This is used to check if a workload already has an explicit Source, so we don't overwrite its InstrumentationConfig
	sourceList := v1alpha1.SourceList{}
	err := k8sClient.List(ctx, &sourceList, client.InNamespace(namespace))
	if err != nil {
		return ctrl.Result{}, err
	}
	namespaceKindSources := make(map[k8sconsts.WorkloadKind]map[string]struct{})
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
		return ctrl.Result{}, err
	}

	collectiveRes := ctrl.Result{}
	switch obj := deps.(type) {
	case *v1.DeploymentList:
		for _, dep := range obj.Items {
			res, err := reconcileWorkloadList(ctx, k8sClient, runtimeScheme, ctrl.Request{NamespacedName: client.ObjectKey{Name: dep.Name, Namespace: dep.Namespace}}, kind, namespaceKindSources)
			// make sure we pass any requeue request up the chain to our caller.
			if err != nil {
				return ctrl.Result{}, err
			}
			collectiveRes = joinCtrlResultToList(res, collectiveRes)
		}
	case *v1.DaemonSetList:
		for _, dep := range obj.Items {
			res, err := reconcileWorkloadList(ctx, k8sClient, runtimeScheme, ctrl.Request{NamespacedName: client.ObjectKey{Name: dep.Name, Namespace: dep.Namespace}}, kind, namespaceKindSources)
			if err != nil {
				return ctrl.Result{}, err
			}
			collectiveRes = joinCtrlResultToList(res, collectiveRes)
		}
	case *v1.StatefulSetList:
		for _, dep := range obj.Items {
			res, err := reconcileWorkloadList(ctx, k8sClient, runtimeScheme, ctrl.Request{NamespacedName: client.ObjectKey{Name: dep.Name, Namespace: dep.Namespace}}, kind, namespaceKindSources)
			if err != nil {
				return ctrl.Result{}, err
			}
			collectiveRes = joinCtrlResultToList(res, collectiveRes)
		}
	}
	return collectiveRes, nil
}

func joinCtrlResultToList(res ctrl.Result, collectiveRes ctrl.Result) ctrl.Result {

	// if the current res is not for requeue, we simply ignore it for the collectiveRes
	if res.IsZero() {
		return collectiveRes
	}

	// if the collectiveRes is not set it, set it to the current res
	if collectiveRes.IsZero() {
		return res
	}

	// notice - we ignore the requeueAfter value of the res, and only use the requeue value.

	// else, the new res is less meaningful than the collectiveRes, so keep the collectiveRes
	return collectiveRes
}

func reconcileWorkloadList(ctx context.Context,
	k8sClient client.Client,
	runtimeScheme *runtime.Scheme,
	req ctrl.Request,
	kind k8sconsts.WorkloadKind,
	namespaceKindSources map[k8sconsts.WorkloadKind]map[string]struct{}) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	if _, exists := namespaceKindSources[kind][req.Name]; !exists {
		res, err := reconcileWorkload(ctx, k8sClient, kind, req, runtimeScheme)
		if err != nil {
			logger.Error(err, "error requesting runtime details from odiglets", "name", req.Name, "namespace", req.Namespace, "kind", kind)
		}
		if !res.IsZero() {
			return res, err
		}
	}
	return ctrl.Result{}, nil
}
