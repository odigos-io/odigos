package sourceinstrumentation

import (
	"slices"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	instrumentationConfigStatus "github.com/odigos-io/odigos/status/instrumentationconfig/generated"
)

func initiateRuntimeDetailsConditionIfMissing(ic *v1alpha1.InstrumentationConfig, workloadObj workload.Workload) bool {
	if meta.FindStatusCondition(ic.Status.Conditions, v1alpha1.RuntimeDetectionStatusConditionType) != nil {
		// avoid adding the condition if it already exists
		return false
	}

	// migration code, add this condition to previous instrumentation configs
	// which were created before this condition was introduced
	// remove this: aug 2025
	if len(ic.Status.RuntimeDetailsByContainer) > 0 {
		meta.SetStatusCondition(&ic.Status.Conditions, metav1.Condition{
			Type:    v1alpha1.RuntimeDetectionStatusConditionType,
			Status:  metav1.ConditionTrue,
			Reason:  string(v1alpha1.RuntimeDetectionReasonWaitingForDetection),
			Message: "runtime detection completed successfully",
		})
		return true
	}

	// if the workload has no available replicas, we can't detect the runtime
	if workloadObj.AvailableReplicas() == 0 &&
		workloadObj.GetObjectKind().GroupVersionKind().Kind != string(k8sconsts.WorkloadKindCronJob) {
		meta.SetStatusCondition(&ic.Status.Conditions, metav1.Condition{
			Type:    v1alpha1.RuntimeDetectionStatusConditionType,
			Status:  metav1.ConditionFalse,
			Reason:  string(v1alpha1.RuntimeDetectionReasonNoRunningPods),
			Message: "No running pods available to detect source runtime",
		})
		return true
	}

	isCronJob := false
	for _, ref := range ic.OwnerReferences {
		if ref.Controller != nil && *ref.Controller && ref.Kind == "CronJob" {
			isCronJob = true
			break
		}
	}
	if isCronJob {
		meta.SetStatusCondition(&ic.Status.Conditions, metav1.Condition{
			Type:    v1alpha1.RuntimeDetectionStatusConditionType,
			Status:  metav1.ConditionUnknown,
			Reason:  string(v1alpha1.RuntimeDetectionReasonWaitingForDetection),
			Message: "Runtime detection pending Job to start",
		})
	} else {
		meta.SetStatusCondition(&ic.Status.Conditions, metav1.Condition{
			Type:    v1alpha1.RuntimeDetectionStatusConditionType,
			Status:  metav1.ConditionUnknown,
			Reason:  string(v1alpha1.RuntimeDetectionReasonWaitingForDetection),
			Message: "Waiting for odiglet to initiate runtime detection in a node with running pod",
		})
	}
	return true
}

func initiateAgentEnabledConditionIfMissing(ic *v1alpha1.InstrumentationConfig) bool {
	if meta.FindStatusCondition(ic.Status.Conditions, instrumentationConfigStatus.AgentEnabledType) != nil {
		// avoid adding the condition if it already exists
		return false
	}

	// if the runtime detection is paused due to no running pods, we can't enable the agent
	// check for that and add a specific reason so not to have spinner in ui
	if meta.FindStatusCondition(ic.Status.Conditions, v1alpha1.RuntimeDetectionStatusConditionType).Reason == string(v1alpha1.RuntimeDetectionReasonNoRunningPods) {
		meta.SetStatusCondition(&ic.Status.Conditions, metav1.Condition{
			Type:    instrumentationConfigStatus.AgentEnabledType,
			Status:  instrumentationConfigStatus.AgentEnabledRuntimeDetailsUnavailable.K8sConditionStatus,
			Reason:  string(instrumentationConfigStatus.AgentEnabledReasonRuntimeDetailsUnavailable),
			Message: instrumentationConfigStatus.AgentEnabledRuntimeDetailsUnavailable.Message,
		})
		return true
	}

	meta.SetStatusCondition(&ic.Status.Conditions, metav1.Condition{
		Type:    instrumentationConfigStatus.AgentEnabledType,
		Status:  instrumentationConfigStatus.AgentEnabledWaitingForRuntimeInspection.K8sConditionStatus,
		Reason:  string(instrumentationConfigStatus.AgentEnabledReasonWaitingForRuntimeInspection),
		Message: instrumentationConfigStatus.AgentEnabledWaitingForRuntimeInspection.Message,
	})

	return true
}

func initiatePodsManifestInjectionConditionIfMissing(ic *v1alpha1.InstrumentationConfig) bool {
	if meta.FindStatusCondition(ic.Status.Conditions, instrumentationConfigStatus.PodsManifestInjectionType) != nil {
		return false
	}

	reason := instrumentationConfigStatus.PodsManifestInjectionNotYetReconciled
	meta.SetStatusCondition(&ic.Status.Conditions, metav1.Condition{
		Type:    instrumentationConfigStatus.PodsManifestInjectionType,
		Status:  reason.K8sConditionStatus,
		Reason:  reason.Name,
		Message: reason.Message,
	})

	return true
}

// giving the input conditions array, this function will return a new array with the conditions sorted by logical order
func sortIcConditionsByLogicalOrder(conditions []metav1.Condition) []metav1.Condition {
	slices.SortFunc(conditions, func(i, j metav1.Condition) int {
		return statusConditionTypeLogicalOrder(i.Type) - statusConditionTypeLogicalOrder(j.Type)
	})
	return conditions
}

func statusConditionTypeLogicalOrder(condType string) int {
	switch condType {
	case v1alpha1.MarkedForInstrumentationStatusConditionType:
		return 1
	case v1alpha1.RuntimeDetectionStatusConditionType:
		return 2
	case instrumentationConfigStatus.AgentEnabledType:
		return 3
	case v1alpha1.WorkloadRolloutStatusConditionType:
		return 4
	case instrumentationConfigStatus.PodsManifestInjectionType:
		return 5
	default:
		return 6
	}
}
