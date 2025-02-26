package deleteinstrumentationconfig

import (
	"context"
	"errors"

	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/consts"
	sourceutils "github.com/odigos-io/odigos/k8sutils/pkg/source"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
)

func reconcileWorkloadObject(ctx context.Context, kubeClient client.Client, workloadObject client.Object) error {
	logger := log.FromContext(ctx)

	instrumented, _, err := sourceutils.IsObjectInstrumentedBySource(ctx, kubeClient, workloadObject)
	if err != nil {
		return err
	}
	if instrumented {
		return nil
	}

	// check if the workload is enabled by deprecated labels,
	// once we fully remove the support for the instrumentation labels, we can remove this check
	enabledByDeprecatedLabels, err := workload.IsWorkloadInstrumentationEffectiveEnabled(ctx, kubeClient, workloadObject)
	if err != nil {
		return err
	}
	if enabledByDeprecatedLabels {
		return nil
	}

	if err := deleteWorkloadInstrumentationConfig(ctx, kubeClient, workloadObject); err != nil {
		logger.Error(err, "error removing runtime details")
		return err
	}
	err = removeReportedNameAnnotation(ctx, kubeClient, workloadObject)
	if err != nil {
		logger.Error(err, "error removing reported name annotation ")
		return err
	}

	return nil
}

func deleteWorkloadInstrumentationConfig(ctx context.Context, kubeClient client.Client, workloadObject client.Object) error {
	logger := log.FromContext(ctx)
	ns := workloadObject.GetNamespace()
	name := workloadObject.GetName()
	kind := workload.WorkloadKindFromClientObject(workloadObject)
	instrumentationConfigName := workload.CalculateWorkloadRuntimeObjectName(name, kind)
	logger.V(1).Info("deleting instrumentationconfig", "name", instrumentationConfigName, "namespace", ns)

	instConfigErr := kubeClient.Delete(ctx, &odigosv1.InstrumentationConfig{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      instrumentationConfigName,
		},
	})

	if instConfigErr != nil {
		return client.IgnoreNotFound(instConfigErr)
	}

	return nil
}

func removeReportedNameAnnotation(ctx context.Context, kubeClient client.Client, workloadObject client.Object) error {
	if _, exists := workloadObject.GetAnnotations()[consts.OdigosReportedNameAnnotation]; !exists {
		return nil
	}

	return kubeClient.Patch(ctx, workloadObject, client.RawPatch(types.MergePatchType, []byte(`{"metadata":{"annotations":{"`+consts.OdigosReportedNameAnnotation+`":null}}}`)))
}

func syncNamespaceWorkloads(ctx context.Context, k8sClient client.Client, req ctrl.Request) error {
	var err error
	for _, kind := range []k8sconsts.WorkloadKind{
		k8sconsts.WorkloadKindDaemonSet,
		k8sconsts.WorkloadKindDeployment,
		k8sconsts.WorkloadKindStatefulSet,
	} {
		err = errors.Join(err, listAndSyncWorkloadList(ctx, k8sClient, req, kind))
	}
	return err
}

func listAndSyncWorkloadList(ctx context.Context,
	k8sClient client.Client,
	req ctrl.Request,
	kind k8sconsts.WorkloadKind) error {
	logger := log.FromContext(ctx)
	logger.V(2).Info("Uninstrumenting workloads for Namespace Source", "name", req.Name, "namespace", req.Namespace, "kind", kind)

	workloads := workload.ClientListObjectFromWorkloadKind(kind)
	err := k8sClient.List(ctx, workloads, client.InNamespace(req.Namespace))
	if client.IgnoreNotFound(err) != nil {
		return err
	}

	switch obj := workloads.(type) {
	case *appsv1.DeploymentList:
		for _, dep := range obj.Items {
			err = syncGenericWorkload(ctx, k8sClient, kind, client.ObjectKey{Namespace: dep.Namespace, Name: dep.Name})
			if err != nil {
				return err
			}
		}
	case *appsv1.DaemonSetList:
		for _, dep := range obj.Items {
			err = syncGenericWorkload(ctx, k8sClient, kind, client.ObjectKey{Namespace: dep.Namespace, Name: dep.Name})
			if err != nil {
				return err
			}
		}
	case *appsv1.StatefulSetList:
		for _, dep := range obj.Items {
			err = syncGenericWorkload(ctx, k8sClient, kind, client.ObjectKey{Namespace: dep.Namespace, Name: dep.Name})
			if err != nil {
				return err
			}
		}
	}
	return err
}

func syncGenericWorkload(ctx context.Context, c client.Client, kind k8sconsts.WorkloadKind, key client.ObjectKey) error {
	// it is very important that we make the changes based on a fresh copy of the workload object
	// if a list operation pulled in state and is now slowly iterating over it, we might be working with stale data
	freshWorkloadCopy := workload.ClientObjectFromWorkloadKind(kind)
	workloadGetErr := c.Get(ctx, key, freshWorkloadCopy)
	if workloadGetErr != nil {
		if apierrors.IsNotFound(workloadGetErr) {
			// if the workload been deleted, we don't need to do anything
			return nil
		} else {
			return workloadGetErr
		}
	}

	instrumented, _, err := sourceutils.IsObjectInstrumentedBySource(ctx, c, freshWorkloadCopy)
	if err != nil {
		return err
	}
	if instrumented {
		return nil
	}

	// check if the workload is enabled by deprecated labels,
	// once we fully remove the support for the instrumentation labels, we can remove this check
	enabledByDeprecatedLabels, err := workload.IsWorkloadInstrumentationEffectiveEnabled(ctx, c, freshWorkloadCopy)
	if err != nil {
		return err
	}
	if enabledByDeprecatedLabels {
		return nil
	}

	err = errors.Join(err, deleteWorkloadInstrumentationConfig(ctx, c, freshWorkloadCopy))
	err = errors.Join(err, removeReportedNameAnnotation(ctx, c, freshWorkloadCopy))
	return err
}
