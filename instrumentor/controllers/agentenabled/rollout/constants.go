package rollout

import (
	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Predefined rollout condition states - these represent all discrete states for workload rollout
var (
	// ConditionNotRequired is used when the selected instrumentation distributions do not require application restart
	ConditionNotRequired = metav1.Condition{
		Type:    odigosv1alpha1.WorkloadRolloutStatusConditionType,
		Status:  metav1.ConditionTrue,
		Reason:  string(odigosv1alpha1.WorkloadRolloutReasonNotRequired),
		Message: "The selected instrumentation distributions do not require application restart",
	}

	// ConditionStaticPodsNotSupported is used when the workload is a static pod which doesn't support restart
	ConditionStaticPodsNotSupported = metav1.Condition{
		Type:    odigosv1alpha1.WorkloadRolloutStatusConditionType,
		Status:  metav1.ConditionTrue,
		Reason:  string(odigosv1alpha1.WorkloadRolloutReasonWorkloadNotSupporting),
		Message: "static pods don't support restart",
	}

	// ConditionWaitingForJobTrigger is used when waiting for a job/cronjob to trigger by itself
	ConditionWaitingForJobTrigger = metav1.Condition{
		Type:    odigosv1alpha1.WorkloadRolloutStatusConditionType,
		Status:  metav1.ConditionTrue,
		Reason:  string(odigosv1alpha1.WorkloadRolloutReasonWaitingForRestart),
		Message: "Waiting for job to trigger by itself",
	}

	// ConditionRolloutDisabled is used when automatic rollout is disabled in odigos configuration
	ConditionRolloutDisabled = metav1.Condition{
		Type:    odigosv1alpha1.WorkloadRolloutStatusConditionType,
		Status:  metav1.ConditionTrue,
		Reason:  string(odigosv1alpha1.WorkloadRolloutReasonDisabled),
		Message: "odigos automatic rollout is disabled",
	}

	// ConditionTriggeredSuccessfully is used when workload rollout was triggered successfully
	ConditionTriggeredSuccessfully = metav1.Condition{
		Type:    odigosv1alpha1.WorkloadRolloutStatusConditionType,
		Status:  metav1.ConditionTrue,
		Reason:  string(odigosv1alpha1.WorkloadRolloutReasonTriggeredSuccessfully),
		Message: "workload rollout triggered successfully",
	}

	// ConditionPreviousRolloutOngoing is used when waiting for a previous rollout to finish
	ConditionPreviousRolloutOngoing = metav1.Condition{
		Type:    odigosv1alpha1.WorkloadRolloutStatusConditionType,
		Status:  metav1.ConditionUnknown,
		Reason:  string(odigosv1alpha1.WorkloadRolloutReasonPreviousRolloutOngoing),
		Message: "waiting for workload rollout to finish before triggering a new one",
	}
)

// NewConditionFailedToPatch creates a condition for when the rollout failed to patch.
// This requires a dynamic message from the error, so it's a function rather than a predefined state.
func NewConditionFailedToPatch(err error) metav1.Condition {
	return metav1.Condition{
		Type:    odigosv1alpha1.WorkloadRolloutStatusConditionType,
		Status:  metav1.ConditionFalse,
		Reason:  string(odigosv1alpha1.WorkloadRolloutReasonFailedToPatch),
		Message: err.Error(),
	}
}

// NewConditionTriggeredWithMessage creates a triggered successfully condition with a custom message.
// Used for rollback scenarios where the message contains backoff details.
func NewConditionTriggeredWithMessage(message string) metav1.Condition {
	return metav1.Condition{
		Type:    odigosv1alpha1.WorkloadRolloutStatusConditionType,
		Status:  metav1.ConditionTrue,
		Reason:  string(odigosv1alpha1.WorkloadRolloutReasonTriggeredSuccessfully),
		Message: message,
	}
}

// ConditionWaitingInQueue is used when the workload is waiting for other rollouts to complete
// due to rate limiting of concurrent reconciliations
var ConditionWaitingInQueue = metav1.Condition{
	Type:    odigosv1alpha1.WorkloadRolloutStatusConditionType,
	Status:  metav1.ConditionTrue,
	Reason:  string(odigosv1alpha1.WorkloadRolloutReasonWaitingInQueue),
	Message: "Waiting for other workload rollouts to complete",
}

// NewConditionAgentDisabledDueToBackoff creates a condition for when agents are disabled due to pod backoff.
// Used when pods enter CrashLoopBackOff or ImagePullBackOff and automatic rollback is triggered.
func NewConditionAgentDisabledDueToBackoff(reason odigosv1alpha1.AgentEnabledReason, message string) metav1.Condition {
	return metav1.Condition{
		Type:    odigosv1alpha1.AgentEnabledStatusConditionType,
		Status:  metav1.ConditionFalse,
		Reason:  string(reason),
		Message: message,
	}
}
