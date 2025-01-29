package instrumentationdevice

import (
	"context"
	"errors"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/instrumentor/controllers/utils/versionsupport"
	"github.com/odigos-io/odigos/instrumentor/instrumentation"
	"github.com/odigos-io/odigos/instrumentor/sdks"
	"github.com/odigos-io/odigos/k8sutils/pkg/conditions"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	k8sutils "github.com/odigos-io/odigos/k8sutils/pkg/utils"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type ApplyInstrumentationDeviceReason string

const (
	ApplyInstrumentationDeviceReasonDataCollectionNotReady     ApplyInstrumentationDeviceReason = "DataCollectionNotReady"
	ApplyInstrumentationDeviceReasonNoRuntimeDetails           ApplyInstrumentationDeviceReason = "NoRuntimeDetails"
	ApplyInstrumentationDeviceReasonErrApplying                ApplyInstrumentationDeviceReason = "ErrApplyingInstrumentationDevice"
	ApplyInstrumentationDeviceReasonErrRemoving                ApplyInstrumentationDeviceReason = "ErrRemovingInstrumentationDevice"
	ApplyInstrumentationDeviceReasonRuntimeVersionNotSupported ApplyInstrumentationDeviceReason = "RuntimeVersionNotSupported"
)

const (
	appliedInstrumentationDeviceType = "AppliedInstrumentationDevice"
)

var (
	// can be overridden in tests
	GetDefaultSDKs = sdks.GetDefaultSDKs
)

func isDataCollectionReady(ctx context.Context, c client.Client) bool {
	logger := log.FromContext(ctx)

	nodeCollectorsGroup := odigosv1.CollectorsGroup{}
	err := c.Get(ctx, client.ObjectKey{
		Namespace: env.GetCurrentNamespace(),
		Name:      k8sconsts.OdigosNodeCollectorCollectorGroupName,
	}, &nodeCollectorsGroup)

	if err != nil {
		if apierrors.IsNotFound(err) {
			// if node collector is not yet created, then it is not ready
			return false
		} else {
			logger.Error(err, "error getting node collector group, skipping instrumentation")
			return false
		}
	}

	return nodeCollectorsGroup.Status.Ready
}

func enableOdigosInstrumentation(ctx context.Context, kubeClient client.Client, instConfig *odigosv1.InstrumentationConfig) error {

	deviceSkipped := false

	logger := log.FromContext(ctx)
	obj, err := getWorkloadObject(ctx, kubeClient, instConfig)
	if err != nil {
		return err
	}

	result, err := controllerutil.CreateOrPatch(ctx, kubeClient, obj, func() error {
		podSpec, err := getPodSpecFromObject(obj)
		if err != nil {
			return err
		}

		// get the odigos configuration to check if agents can run concurrently
		// if the configuration is not found, we assume that agents can't run concurrently [default behavior]
		odigosConfiguration, err := k8sutils.GetCurrentOdigosConfig(ctx, kubeClient)
		if err != nil {
			return err
		}

		// allowConcurrentAgents is false by default unless explicitly set to true in the OdigosConfiguration
		agentsCanRunConcurrently := false
		if odigosConfiguration.AllowConcurrentAgents != nil {
			agentsCanRunConcurrently = *odigosConfiguration.AllowConcurrentAgents
		}

		deviceSkipped, err = instrumentation.ConfigureInstrumentationForPod(podSpec, instConfig.Status.RuntimeDetailsByContainer, obj, logger, agentsCanRunConcurrently)
		if err != nil {
			return err
		}

		if !deviceSkipped {
			// add odigos.io/inject-instrumentation label to enable the webhook
			instrumentation.SetInjectInstrumentationLabel(podSpec)
		}
		return nil

	})

	// if non of the devices were applied due to the presence of another agent, return an error.
	if deviceSkipped {
		return k8sutils.OtherAgentRunError
	}

	if err != nil {
		return err
	}

	modified := result != controllerutil.OperationResultNone
	if modified {
		logger.V(0).Info("inject instrumentation label to workload pod template", "name", obj.GetName(), "namespace", obj.GetNamespace())
	}

	return nil
}

func removeInstrumentationDeviceFromWorkload(ctx context.Context, kubeClient client.Client, namespace string, workloadKind workload.WorkloadKind, workloadName string, uninstrumentReason ApplyInstrumentationDeviceReason) error {

	workloadObj := workload.ClientObjectFromWorkloadKind(workloadKind)
	if workloadObj == nil {
		return errors.New("unknown kind")
	}

	err := kubeClient.Get(ctx, client.ObjectKey{
		Namespace: namespace,
		Name:      workloadName,
	}, workloadObj)
	if err != nil {
		return client.IgnoreNotFound(err)
	}

	podSpec, err := getPodSpecFromObject(workloadObj)
	if err != nil {
		return err
	}
	// If instrumentation device is removed successfully, remove odigos.io/inject-instrumentation label to disable the webhook
	webhookLabelRemoved := instrumentation.RemoveInjectInstrumentationLabel(podSpec)
	deviceRemoved := instrumentation.RevertInstrumentationDevices(podSpec)

	// if we didn't change anything, we don't need to update the object
	// skip the api-server call, return no-op and skip the log message
	if !webhookLabelRemoved && !deviceRemoved {
		return nil
	}

	err = kubeClient.Update(ctx, workloadObj)
	if err != nil {
		// if the update fails due to a conflict, the controller will retry the operation
		return err
	}

	logger := log.FromContext(ctx)
	logger.V(0).Info("removed instrumentation device from workload", "namespace", workloadObj.GetNamespace(), "kind", workloadObj.GetObjectKind(), "name", workloadObj.GetName(), "reason", uninstrumentReason)

	return nil
}

func getWorkloadObject(ctx context.Context, kubeClient client.Client, instConfig *odigosv1.InstrumentationConfig) (client.Object, error) {
	name, kind, err := workload.ExtractWorkloadInfoFromRuntimeObjectName(instConfig.Name)
	if err != nil {
		return nil, err
	}

	workloadObject := workload.ClientObjectFromWorkloadKind(kind)
	if workloadObject == nil {
		return nil, errors.New("unknown kind")
	}

	err = kubeClient.Get(ctx, client.ObjectKey{
		Namespace: instConfig.Namespace,
		Name:      name,
	}, workloadObject)
	if err != nil {
		return nil, err
	}

	return workloadObject, nil
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

// reconciles a single workload, which might be triggered by a change in multiple resources.
// each time a relevant resource changes, this function is called to reconcile the workload
// and always writes the status into the InstrumentationConfig CR
func reconcileSingleWorkload(ctx context.Context, kubeClient client.Client, instrumentationConfig *odigosv1.InstrumentationConfig, isNodeCollectorReady bool) error {
	// RuntimeDetailsByContainer will be empty for short period until the first language detection is executed
	if len(instrumentationConfig.Status.RuntimeDetailsByContainer) == 0 {
		return nil
	}

	workloadName, workloadKind, err := workload.ExtractWorkloadInfoFromRuntimeObjectName(instrumentationConfig.Name)
	if err != nil {
		conditions.UpdateStatusConditions(ctx, kubeClient, instrumentationConfig, &instrumentationConfig.Status.Conditions, metav1.ConditionFalse, appliedInstrumentationDeviceType, string(ApplyInstrumentationDeviceReasonErrRemoving), err.Error())
		return err
	}

	if !isNodeCollectorReady {
		err := removeInstrumentationDeviceFromWorkload(ctx, kubeClient, instrumentationConfig.Namespace, workloadKind, workloadName, ApplyInstrumentationDeviceReasonDataCollectionNotReady)
		if err == nil {
			conditions.UpdateStatusConditions(ctx, kubeClient, instrumentationConfig, &instrumentationConfig.Status.Conditions, metav1.ConditionFalse, appliedInstrumentationDeviceType, string(ApplyInstrumentationDeviceReasonDataCollectionNotReady), "OpenTelemetry pipeline not yet ready to receive data")
		} else {
			conditions.UpdateStatusConditions(ctx, kubeClient, instrumentationConfig, &instrumentationConfig.Status.Conditions, metav1.ConditionFalse, appliedInstrumentationDeviceType, string(ApplyInstrumentationDeviceReasonErrRemoving), err.Error())
		}
		return err
	}

	runtimeVersionSupport, err := versionsupport.IsRuntimeVersionSupported(ctx, instrumentationConfig.Status.RuntimeDetailsByContainer)
	if !runtimeVersionSupport {
		errRemove := removeInstrumentationDeviceFromWorkload(ctx, kubeClient, instrumentationConfig.Namespace, workloadKind, workloadName, ApplyInstrumentationDeviceReasonRuntimeVersionNotSupported)
		if errRemove == nil {
			conditions.UpdateStatusConditions(ctx, kubeClient, instrumentationConfig, &instrumentationConfig.Status.Conditions, metav1.ConditionFalse, appliedInstrumentationDeviceType, string(ApplyInstrumentationDeviceReasonRuntimeVersionNotSupported), err.Error())
		} else {
			conditions.UpdateStatusConditions(ctx, kubeClient, instrumentationConfig, &instrumentationConfig.Status.Conditions, metav1.ConditionFalse, appliedInstrumentationDeviceType, string(ApplyInstrumentationDeviceReasonErrRemoving), errRemove.Error())
		}
		return nil
	}

	err = enableOdigosInstrumentation(ctx, kubeClient, instrumentationConfig)
	if err != nil {

		conditions.UpdateStatusConditions(ctx, kubeClient, instrumentationConfig, &instrumentationConfig.Status.Conditions,
			metav1.ConditionFalse, appliedInstrumentationDeviceType, string(ApplyInstrumentationDeviceReasonErrApplying),
			"Odigos instrumentation failed to apply: "+err.Error())
	} else {

		enabledMessage := "Odigos instrumentation is enabled."
		conditions.UpdateStatusConditions(ctx, kubeClient, instrumentationConfig, &instrumentationConfig.Status.Conditions,
			metav1.ConditionTrue, appliedInstrumentationDeviceType, "InstrumentationEnabled", enabledMessage)
	}

	return err
}
