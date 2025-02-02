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

func syncNamespaceWorkloads(ctx context.Context, k8sClient client.Client, runtimeScheme *runtime.Scheme, namespace string) error {
	var err error
	for _, kind := range []k8sconsts.WorkloadKind{
		k8sconsts.WorkloadKindDaemonSet,
		k8sconsts.WorkloadKindDeployment,
		k8sconsts.WorkloadKindStatefulSet,
	} {
		err = errors.Join(err, listAndReconcileWorkloadList(ctx, k8sClient, runtimeScheme, namespace, kind))
	}
	return err
}

func listAndReconcileWorkloadList(ctx context.Context,
	k8sClient client.Client,
	runtimeScheme *runtime.Scheme,
	namespace string,
	kind k8sconsts.WorkloadKind) error {

	// pre-process existing Sources for specific workloads so we don't have to make a bunch of API calls
	// This is used to check if a workload already has an explicit Source, so we don't overwrite its InstrumentationConfig
	sourceList := v1alpha1.SourceList{}
	err := k8sClient.List(ctx, &sourceList, client.InNamespace(namespace))
	if err != nil {
		return err
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
	kind k8sconsts.WorkloadKind,
	namespaceKindSources map[k8sconsts.WorkloadKind]map[string]struct{}) error {
	logger := log.FromContext(ctx)
	if _, exists := namespaceKindSources[kind][req.Name]; !exists {
		_, err := reconcileWorkload(ctx, k8sClient, kind, req, runtimeScheme)
		if err != nil {
			logger.Error(err, "error requesting runtime details from odiglets", "name", req.Name, "namespace", req.Namespace, "kind", kind)
		}
	}
	return nil
}
