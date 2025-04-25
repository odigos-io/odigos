package sourceinstrumentation

import (
	"slices"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
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
		ic.Status.Conditions = append(ic.Status.Conditions, metav1.Condition{
			Type:               v1alpha1.RuntimeDetectionStatusConditionType,
			Status:             metav1.ConditionTrue,
			Reason:             string(v1alpha1.RuntimeDetectionReasonWaitingForDetection),
			Message:            "runtime detection completed successfully",
			LastTransitionTime: metav1.NewTime(time.Now()),
		})
		return true
	}

	// if the workload has no available replicas, we can't detect the runtime
	if workloadObj.AvailableReplicas() == 0 {
		ic.Status.Conditions = append(ic.Status.Conditions, metav1.Condition{
			Type:               v1alpha1.RuntimeDetectionStatusConditionType,
			Status:             metav1.ConditionFalse,
			Reason:             string(v1alpha1.RuntimeDetectionReasonNoRunningPods),
			Message:            "No running pods available to detect source runtime",
			LastTransitionTime: metav1.NewTime(time.Now()),
		})
		return true
	}

	ic.Status.Conditions = append(ic.Status.Conditions, metav1.Condition{
		Type:               v1alpha1.RuntimeDetectionStatusConditionType,
		Status:             metav1.ConditionUnknown,
		Reason:             string(v1alpha1.RuntimeDetectionReasonWaitingForDetection),
		Message:            "Waiting for odiglet to initiate runtime detection in a node with running pod",
		LastTransitionTime: metav1.NewTime(time.Now()),
	})

	return true
}

func initiateAgentEnabledConditionIfMissing(ic *v1alpha1.InstrumentationConfig) bool {
	if meta.FindStatusCondition(ic.Status.Conditions, v1alpha1.AgentEnabledStatusConditionType) != nil {
		// avoid adding the condition if it already exists
		return false
	}

	// defaults, for most cases.
	reason := string(v1alpha1.AgentEnabledReasonWaitingForRuntimeInspection)
	message := "waiting for runtime detection to complete"

	// if the runtime detection is paused due to no running pods, we can't enable the agent
	// check for that and add a specific reason so not to have spinner in ui
	if meta.FindStatusCondition(ic.Status.Conditions, v1alpha1.RuntimeDetectionStatusConditionType).Reason == string(v1alpha1.RuntimeDetectionReasonNoRunningPods) {
		reason = string(v1alpha1.AgentEnabledReasonRuntimeDetailsUnavailable)
		message = "agent disabled while no running pods available to detect source runtime"
	}

	ic.Status.Conditions = append(ic.Status.Conditions, metav1.Condition{
		Type:               v1alpha1.AgentEnabledStatusConditionType,
		Status:             metav1.ConditionUnknown,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: metav1.NewTime(time.Now()),
	})

	return true
}

// giving the input conditions array, this function will return a new array with the conditions sorted by logical order
func sortIcConditionsByLogicalOrder(conditions []metav1.Condition) []metav1.Condition {
	slices.SortFunc(conditions, func(i, j metav1.Condition) int {
		return v1alpha1.StatusConditionTypeLogicalOrder(i.Type) - v1alpha1.StatusConditionTypeLogicalOrder(j.Type)
	})
	return conditions
}
