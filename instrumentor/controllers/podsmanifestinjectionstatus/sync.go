package podsmanifestinjectionstatus

import (
	"context"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/k8sutils/pkg/utils"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	"github.com/odigos-io/odigos/status"
	podsManifestInjection "github.com/odigos-io/odigos/status/instrumentationconfig/generated"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
)

func syncWorkload(ctx context.Context, client ctrl.Client, pw k8sconsts.PodWorkload) error {

	// get the instrumentation config for the workload.
	// if not found - there is no place to persist the info, so we skip processing.
	icName := workload.CalculateWorkloadRuntimeObjectName(pw.Name, pw.Kind)
	ic := odigosv1.InstrumentationConfig{}
	err := client.Get(ctx, types.NamespacedName{Namespace: pw.Namespace, Name: icName}, &ic)
	if err != nil {
		return ctrl.IgnoreNotFound(err)
	}

	effectiveConfig, err := utils.GetCurrentOdigosConfiguration(ctx, client)
	if err != nil {
		return err
	}

	// get the workload object to extract the label selector
	// this can be optimized later by writing the label selector into the instrumentation config
	workloadObj := workload.ClientObjectFromWorkloadKind(pw.Kind)
	err = client.Get(ctx, types.NamespacedName{Namespace: pw.Namespace, Name: pw.Name}, workloadObj)
	if err != nil {
		return ctrl.IgnoreNotFound(err)
	}

	genericWorkload, err := workload.ObjectToWorkload(workloadObj)
	if err != nil {
		return err
	}

	labelSelector := genericWorkload.LabelSelector()
	if labelSelector == nil {
		// TODO: handle this case
		return nil
	}

	// get the pods that match the label selector
	pods := &corev1.PodList{}
	err = client.List(ctx, pods, ctrl.MatchingLabels(labelSelector.MatchLabels))
	if err != nil {
		return err
	}

	podsManifestInjectionStatus := odigosv1.PodsManifestInjectionStatus{}

	for _, pod := range pods.Items {
		if agentHashValue, ok := pod.Labels[k8sconsts.OdigosAgentsMetaHashLabel]; ok {
			if agentHashValue == ic.Spec.AgentsMetaHash {
				podsManifestInjectionStatus.HasInjectedUpToDatePods = true
			} else {
				podsManifestInjectionStatus.HasInjectedOutOfDatePods = true
			}
		} else {
			podsManifestInjectionStatus.HasUninjectedPods = true
		}
	}

	injectionStatusChanged := podsManifestInjectionStatusNeedsUpdate(ic.Status.PodsManifestInjectionStatus, podsManifestInjectionStatus)
	if injectionStatusChanged {
		ic.Status.PodsManifestInjectionStatus = &podsManifestInjectionStatus
	}

	reason := calculatePodsManifestInjectionReason(
		podsManifestInjectionStatus,
		&ic,
		&effectiveConfig,
		pw.Kind,
	)
	conditionChanged := false
	if reason.Name != "" {
		conditionChanged = meta.SetStatusCondition(&ic.Status.Conditions, metav1.Condition{
			Type:    podsManifestInjection.PodsManifestInjectionType,
			Status:  reason.K8sConditionStatus,
			Reason:  reason.Name,
			Message: reason.Message,
		})
	}

	if !injectionStatusChanged && !conditionChanged {
		return nil
	}

	return client.Status().Update(ctx, &ic)
}

// podsManifestInjectionStatusNeedsUpdate reports whether the computed pod manifest
// injection status differs from what is already persisted on the InstrumentationConfig.
func podsManifestInjectionStatusNeedsUpdate(current *odigosv1.PodsManifestInjectionStatus, desired odigosv1.PodsManifestInjectionStatus) bool {
	if current == nil {
		return true
	}
	return current.HasInjectedUpToDatePods != desired.HasInjectedUpToDatePods ||
		current.HasInjectedOutOfDatePods != desired.HasInjectedOutOfDatePods ||
		current.HasUninjectedPods != desired.HasUninjectedPods
}

// calculatePodsManifestInjectionReason selects the PodsManifestInjection status reason for the observed
// pod manifest injection state.
func calculatePodsManifestInjectionReason(
	injectionStatus odigosv1.PodsManifestInjectionStatus,
	ic *odigosv1.InstrumentationConfig,
	effectiveConfig *common.OdigosConfiguration,
	workloadKind k8sconsts.WorkloadKind,
) status.Reason {

	if !injectionStatus.HasInjectedUpToDatePods && !injectionStatus.HasInjectedOutOfDatePods && !injectionStatus.HasUninjectedPods {
		return podsManifestInjection.PodsManifestInjectionNoRunningPods
	}

	if ic.Spec.AgentInjectionEnabled {
		return calculateEnabledWorkloadPodsManifestInjectionReason(injectionStatus, ic, effectiveConfig, workloadKind)
	} else {
		return calculateDisabledWorkloadPodsManifestInjectionReason(injectionStatus, ic, effectiveConfig, workloadKind)
	}
}

func calculateEnabledWorkloadPodsManifestInjectionReason(
	injectionStatus odigosv1.PodsManifestInjectionStatus,
	ic *odigosv1.InstrumentationConfig,
	effectiveConfig *common.OdigosConfiguration,
	workloadKind k8sconsts.WorkloadKind,
) status.Reason {

	// first - if all is well, then report it and that's it.
	if injectionStatus.HasInjectedUpToDatePods && !injectionStatus.HasInjectedOutOfDatePods && !injectionStatus.HasUninjectedPods {
		return podsManifestInjection.PodsManifestInjectionPodsInjectedSuccessfully
	}

	// check for "no restart" distros
	if ic.Spec.PodManifestInjectionOptional {
		return podsManifestInjection.PodsManifestInjectionPodsManifestInjectionNotRequired
	}

	// at this point, we know that there are some pods with pods injection not aligned with the desired state.
	// we need to branch the different cases to report the condition with the correct reason,
	// which is useful, informative and actionable for the user.

	if workloadKind == k8sconsts.WorkloadKindStaticPod {
		return podsManifestInjection.PodsManifestInjectionRolloutNotSupportedForStaticPods
	}

	if workloadKind == k8sconsts.WorkloadKindJob || workloadKind == k8sconsts.WorkloadKindCronJob {
		return podsManifestInjection.PodsManifestInjectionWaitingForNextJobRun
	}

	automaticRolloutDisabledInConfig := effectiveConfig.Rollout != nil &&
		effectiveConfig.Rollout.AutomaticRolloutDisabled != nil &&
		*effectiveConfig.Rollout.AutomaticRolloutDisabled
	if automaticRolloutDisabledInConfig {
		return podsManifestInjection.PodsManifestInjectionRestartRequiredAutoRolloutDisabled
	}

	var workloadRolloutReason odigosv1.WorkloadRolloutReason
	if cond := meta.FindStatusCondition(ic.Status.Conditions, odigosv1.WorkloadRolloutStatusConditionType); cond != nil {
		workloadRolloutReason = odigosv1.WorkloadRolloutReason(cond.Reason)
	}

	// some rollout info is only written in the condition, and there is no other way to figure it out.
	// In the future, it is better no to rely on the condition, and compute the rollout status from instrumentation config itself.
	switch workloadRolloutReason {
	case odigosv1.WorkloadRolloutReasonWaitingInQueue:
		return podsManifestInjection.PodsManifestInjectionWaitingInRolloutQueue
	case odigosv1.WorkloadRolloutReasonPreviousRolloutOngoing,
		odigosv1.WorkloadRolloutReasonTriggeredSuccessfully:
		return podsManifestInjection.PodsManifestInjectionRolloutInProgress
	case odigosv1.WorkloadRolloutReasonFailedToPatch:
		return podsManifestInjection.PodsManifestInjectionRestartRequiredAutoRolloutFailed
	}

	// Odigos already recorded a rollout for the current agents hash, but some pods are still
	// missing / outdated injection — typically pods that bypassed the webhook.
	if ic.Status.WorkloadRolloutHash != "" && ic.Status.WorkloadRolloutHash == ic.Spec.AgentsMetaHash {
		return podsManifestInjection.PodsManifestInjectionRestartRequiredWebhookMissed
	}

	return podsManifestInjection.PodsManifestInjectionWaitingForAutomaticRollout
}

func calculateDisabledWorkloadPodsManifestInjectionReason(
	injectionStatus odigosv1.PodsManifestInjectionStatus,
	ic *odigosv1.InstrumentationConfig,
	effectiveConfig *common.OdigosConfiguration,
	workloadKind k8sconsts.WorkloadKind,
) status.Reason {

	if !injectionStatus.HasInjectedUpToDatePods && !injectionStatus.HasInjectedOutOfDatePods && injectionStatus.HasUninjectedPods {
		return podsManifestInjection.PodsManifestInjectionAgentDisabled
	}

	// Agents that do not require pod manifest injection are disabled without restarting the workload.
	if workloadKind == k8sconsts.WorkloadKindStaticPod {
		return podsManifestInjection.PodsManifestInjectionRolloutNotSupportedForStaticPods
	}

	if workloadKind == k8sconsts.WorkloadKindJob || workloadKind == k8sconsts.WorkloadKindCronJob {
		return podsManifestInjection.PodsManifestInjectionWaitingForNextJobRunDisabledWorkloads
	}

	automaticRolloutDisabledInConfig := effectiveConfig.Rollout != nil &&
		effectiveConfig.Rollout.AutomaticRolloutDisabled != nil &&
		*effectiveConfig.Rollout.AutomaticRolloutDisabled
	if automaticRolloutDisabledInConfig {
		return podsManifestInjection.PodsManifestInjectionRestartRequiredAutoRolloutDisabledDisabledWorkloads
	}

	var workloadRolloutReason odigosv1.WorkloadRolloutReason
	if cond := meta.FindStatusCondition(ic.Status.Conditions, odigosv1.WorkloadRolloutStatusConditionType); cond != nil {
		workloadRolloutReason = odigosv1.WorkloadRolloutReason(cond.Reason)
	}

	switch workloadRolloutReason {
	case odigosv1.WorkloadRolloutReasonWaitingInQueue:
		return podsManifestInjection.PodsManifestInjectionWaitingInAutoRolloutQueueDisabledWorkloads
	case odigosv1.WorkloadRolloutReasonPreviousRolloutOngoing,
		odigosv1.WorkloadRolloutReasonTriggeredSuccessfully:
		return podsManifestInjection.PodsManifestInjectionRolloutInProgressDisabledWorkloads
	case odigosv1.WorkloadRolloutReasonFailedToPatch:
		return podsManifestInjection.PodsManifestInjectionRestartRequiredAutoRolloutFailedDisabledWorkloads
	case odigosv1.WorkloadRolloutReasonRolloutFinished:
		// The rollout completed, but injected pods remain. A fresh rollout is required.
		return podsManifestInjection.PodsManifestInjectionRestartRequiredDisabledWorkloads
	}

	return podsManifestInjection.PodsManifestInjectionWaitingForAutomaticRolloutDisabledWorkloads
}
