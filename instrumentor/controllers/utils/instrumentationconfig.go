package utils

import (
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func AreConditionsLogicallySorted(conditions []metav1.Condition) bool {
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

func SortConditions(conditions []metav1.Condition) []metav1.Condition {
	conditionsByLogicalOrder := make(map[int]metav1.Condition, len(conditions))
	maxLogicalOrder := 0
	for _, condition := range conditions {
		currentLogicalOrder := odigosv1.StatusConditionTypeLogicalOrder(condition.Type)
		conditionsByLogicalOrder[currentLogicalOrder] = condition
		if currentLogicalOrder > maxLogicalOrder {
			maxLogicalOrder = currentLogicalOrder
		}
	}

	var sortedConditions []metav1.Condition
	for i := 0; i < maxLogicalOrder; i++ {
		cond, found := conditionsByLogicalOrder[i]
		if !found {
			// ok to skip condition if missing
			continue
		}
		sortedConditions = append(sortedConditions, cond)
	}

	return sortedConditions
}
