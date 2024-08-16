package instrumentationdevice

import (
	"context"
	"errors"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/instrumentor/instrumentation"
	"github.com/odigos-io/odigos/k8sutils/pkg/conditions"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	k8sutils "github.com/odigos-io/odigos/k8sutils/pkg/utils"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type ApplyInstrumentationDeviceReason string

const (
	ApplyInstrumentationDeviceReasonDataCollectionNotReady ApplyInstrumentationDeviceReason = "DataCollectionNotReady"
	ApplyInstrumentationDeviceReasonNoRuntimeDetails       ApplyInstrumentationDeviceReason = "NoRuntimeDetails"
	ApplyInstrumentationDeviceReasonErrApplying            ApplyInstrumentationDeviceReason = "ErrApplyingInstrumentationDevice"
	ApplyInstrumentationDeviceReasonErrRemoving            ApplyInstrumentationDeviceReason = "ErrRemovingInstrumentationDevice"
)

const (
	appliedInstrumentationDeviceType = "AppliedInstrumentationDevice"
)

func clearInstrumentationEbpf(obj client.Object) {
	annotations := obj.GetAnnotations()
	if annotations == nil {
		return
	}

	delete(annotations, consts.EbpfInstrumentationAnnotation)
}

func isDataCollectionReady(ctx context.Context, c client.Client) bool {
	logger := log.FromContext(ctx)
	var collectorGroups odigosv1.CollectorsGroupList
	err := c.List(ctx, &collectorGroups, client.InNamespace(env.GetCurrentNamespace()))
	if err != nil {
		logger.Error(err, "error getting collectors groups, skipping instrumentation")
		return false
	}

	for _, cg := range collectorGroups.Items {
		// up until v1.0.31, the collectors group role names were "GATEWAY" and "DATA_COLLECTION".
		// in v1.0.32, the role names were changed to "CLUSTER_GATEWAY" and "NODE_COLLECTOR",
		// due to adding the Processor CRD which uses these role names.
		// the new names are more descriptive and are preparations for future roles.
		// the check for "DATA_COLLECTION" is a temporary support for users that upgrade from <=v1.0.31 to >=v1.0.32.
		// once we drop support for <=v1.0.31, we can remove this comparison.
		if (cg.Spec.Role == odigosv1.CollectorsGroupRoleNodeCollector || cg.Spec.Role == "DATA_COLLECTION") && cg.Status.Ready {
			return true
		}
	}

	return false
}

func addInstrumentationDeviceToWorkload(ctx context.Context, kubeClient client.Client, runtimeDetails *odigosv1.InstrumentedApplication) error {

	logger := log.FromContext(ctx)
	obj, err := getWorkloadObject(ctx, kubeClient, runtimeDetails)
	if err != nil {
		return err
	}

	odigosConfig, err := k8sutils.GetCurrentConfig(ctx, kubeClient)
	if err != nil {
		return err
	}

	result, err := controllerutil.CreateOrPatch(ctx, kubeClient, obj, func() error {
		podSpec, err := getPodSpecFromObject(obj)
		if err != nil {
			return err
		}

		return instrumentation.ApplyInstrumentationDevicesToPodTemplate(podSpec, runtimeDetails, odigosConfig.DefaultSDKs, obj)
	})

	if err != nil {
		return err
	}

	modified := result != controllerutil.OperationResultNone
	if modified {
		logger.V(0).Info("added instrumentation device to workload", "name", obj.GetName(), "namespace", obj.GetNamespace())
	}

	return nil
}

func removeInstrumentationDeviceFromWorkload(ctx context.Context, kubeClient client.Client, namespace string, workloadKind string, workloadName string, uninstrumentReason ApplyInstrumentationDeviceReason) error {

	obj, err := getObjectFromKindString(workloadKind)
	if err != nil {
		return err
	}

	err = kubeClient.Get(ctx, client.ObjectKey{
		Namespace: namespace,
		Name:      workloadName,
	}, obj)
	if err != nil {
		return client.IgnoreNotFound(err)
	}

	result, err := controllerutil.CreateOrPatch(ctx, kubeClient, obj, func() error {

		// clear old ebpf instrumentation annotation, just in case it still exists
		clearInstrumentationEbpf(obj)
		podSpec, err := getPodSpecFromObject(obj)
		if err != nil {
			return err
		}

		instrumentation.RevertInstrumentationDevices(podSpec)
		err = instrumentation.RevertEnvOverwrites(obj, podSpec)
		if err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return err
	}

	modified := result != controllerutil.OperationResultNone
	if modified {
		logger := log.FromContext(ctx)
		logger.V(0).Info("removed instrumentation device from workload", "namespace", obj.GetNamespace(), "kind", obj.GetObjectKind(), "name", obj.GetName(), "reason", uninstrumentReason)
	}

	return nil
}

func getWorkloadObject(ctx context.Context, kubeClient client.Client, runtimeDetails *odigosv1.InstrumentedApplication) (client.Object, error) {
	name, kind, err := workload.GetWorkloadInfoRuntimeName(runtimeDetails.Name)
	if err != nil {
		return nil, err
	}

	obj, err := getObjectFromKindString(kind)
	if err != nil {
		return nil, err
	}

	err = kubeClient.Get(ctx, client.ObjectKey{
		Namespace: runtimeDetails.Namespace,
		Name:      name,
	}, obj)
	if err != nil {
		return nil, err
	}

	return obj, nil
}

func getPodSpecFromObject(obj client.Object) (*corev1.PodTemplateSpec, error) {
	switch o := obj.(type) {
	case *appsv1.Deployment:
		return &o.Spec.Template, nil
	case *appsv1.StatefulSet:
		return &o.Spec.Template, nil
	case *appsv1.DaemonSet:
		return &o.Spec.Template, nil
	default:
		return nil, errors.New("unknown kind")
	}
}

func getObjectFromKindString(kind string) (client.Object, error) {
	switch kind {
	case "Deployment":
		return &appsv1.Deployment{}, nil
	case "StatefulSet":
		return &appsv1.StatefulSet{}, nil
	case "DaemonSet":
		return &appsv1.DaemonSet{}, nil
	default:
		return nil, errors.New("unknown kind")
	}
}

// reconciles a single workload, which might be triggered by a change in multiple resources.
// each time a relevant resource changes, this function is called to reconcile the workload
// and always writes the status into the InstrumentedApplication CR
func reconcileSingleWorkload(ctx context.Context, kubeClient client.Client, runtimeDetails *odigosv1.InstrumentedApplication, isNodeCollectorReady bool) error {

	workloadName, workloadKind, err := workload.GetWorkloadInfoRuntimeName(runtimeDetails.Name)
	if err != nil {
		conditions.UpdateStatusConditions(ctx, kubeClient, runtimeDetails, &runtimeDetails.Status.Conditions, metav1.ConditionFalse, appliedInstrumentationDeviceType, string(ApplyInstrumentationDeviceReasonErrRemoving), err.Error())
		return err
	}

	if !isNodeCollectorReady {
		err := removeInstrumentationDeviceFromWorkload(ctx, kubeClient, runtimeDetails.Namespace, workloadKind, workloadName, ApplyInstrumentationDeviceReasonDataCollectionNotReady)
		if err == nil {
			conditions.UpdateStatusConditions(ctx, kubeClient, runtimeDetails, &runtimeDetails.Status.Conditions, metav1.ConditionFalse, appliedInstrumentationDeviceType, string(ApplyInstrumentationDeviceReasonDataCollectionNotReady), "OpenTelemetry pipeline not yet ready to receive data")
		} else {
			conditions.UpdateStatusConditions(ctx, kubeClient, runtimeDetails, &runtimeDetails.Status.Conditions, metav1.ConditionFalse, appliedInstrumentationDeviceType, string(ApplyInstrumentationDeviceReasonErrRemoving), err.Error())
		}
		return err
	}

	if len(runtimeDetails.Spec.RuntimeDetails) == 0 {
		err := removeInstrumentationDeviceFromWorkload(ctx, kubeClient, runtimeDetails.Namespace, workloadKind, workloadName, ApplyInstrumentationDeviceReasonNoRuntimeDetails)
		if err == nil {
			conditions.UpdateStatusConditions(ctx, kubeClient, runtimeDetails, &runtimeDetails.Status.Conditions, metav1.ConditionFalse, appliedInstrumentationDeviceType, string(ApplyInstrumentationDeviceReasonNoRuntimeDetails), "No runtime details found")
		} else {
			conditions.UpdateStatusConditions(ctx, kubeClient, runtimeDetails, &runtimeDetails.Status.Conditions, metav1.ConditionFalse, appliedInstrumentationDeviceType, string(ApplyInstrumentationDeviceReasonErrRemoving), err.Error())
		}
		return err
	}

	err = addInstrumentationDeviceToWorkload(ctx, kubeClient, runtimeDetails)
	if err == nil {
		conditions.UpdateStatusConditions(ctx, kubeClient, runtimeDetails, &runtimeDetails.Status.Conditions, metav1.ConditionTrue, appliedInstrumentationDeviceType, "InstrumentationDeviceApplied", "Instrumentation device applied successfully")
	} else {
		conditions.UpdateStatusConditions(ctx, kubeClient, runtimeDetails, &runtimeDetails.Status.Conditions, metav1.ConditionFalse, appliedInstrumentationDeviceType, string(ApplyInstrumentationDeviceReasonErrApplying), err.Error())
	}
	return err
}
