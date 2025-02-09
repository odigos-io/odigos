package startlangdetection

import (
	"slices"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
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
