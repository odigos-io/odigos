package startlangdetection

import (
	"slices"
	"time"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Checks if the conditions array in the status is currently sorted by logical order
// this can be used to check if sorting is needed.
func areConditionsLogicallySorted(conditions []metav1.Condition) bool {
	var lastTypeLogicalOrder int = 0
	for _, condition := range conditions {
		currentLogicalOrder := odigosv1.StatusConditionTypeLogicalOrder(condition.Type)
		if currentLogicalOrder <= lastTypeLogicalOrder {
			return false
		}
		lastTypeLogicalOrder = currentLogicalOrder
	}
	return true
}

// giving the input conditions array, this function will return a new array with the conditions sorted by logical order
func sortIcConditionsByLogicalOrder(conditions []metav1.Condition) []metav1.Condition {
	slices.SortFunc(conditions, func(i, j metav1.Condition) int {
		return odigosv1.StatusConditionTypeLogicalOrder(i.Type) - odigosv1.StatusConditionTypeLogicalOrder(j.Type)
	})
	return conditions
}

func initiateRuntimeDetailsConditionIfMissing(ic *odigosv1.InstrumentationConfig, workloadObj workload.Workload) bool {

	if meta.FindStatusCondition(ic.Status.Conditions, odigosv1.RuntimeDetectionStatusConditionType) != nil {
		// avoid adding the condition if it already exists
		return false
	}

	// migration code, add this condition to previous instrumentation configs
	// which were created before this condition was introduced
	if len(ic.Status.RuntimeDetailsByContainer) > 0 {
		ic.Status.Conditions = append(ic.Status.Conditions, metav1.Condition{
			Type:               odigosv1.RuntimeDetectionStatusConditionType,
			Status:             metav1.ConditionTrue,
			Reason:             string(odigosv1.RuntimeDetectionReasonWaitingForDetection),
			Message:            "runtime detection completed successfully",
			LastTransitionTime: metav1.NewTime(time.Now()),
		})
		return true
	}

	// if the workload has no available replicas, we can't detect the runtime
	if workloadObj.AvailableReplicas() == 0 {
		ic.Status.Conditions = append(ic.Status.Conditions, metav1.Condition{
			Type:               odigosv1.RuntimeDetectionStatusConditionType,
			Status:             metav1.ConditionFalse,
			Reason:             string(odigosv1.RuntimeDetectionReasonNoRunningPods),
			Message:            "No running pods available to detect source runtime",
			LastTransitionTime: metav1.NewTime(time.Now()),
		})
		return true
	}

	ic.Status.Conditions = append(ic.Status.Conditions, metav1.Condition{
		Type:               odigosv1.RuntimeDetectionStatusConditionType,
		Status:             metav1.ConditionUnknown,
		Reason:             string(odigosv1.RuntimeDetectionReasonWaitingForDetection),
		Message:            "Waiting for odiglet to initiate runtime detection in a node with running pod",
		LastTransitionTime: metav1.NewTime(time.Now()),
	})

	return true
}
