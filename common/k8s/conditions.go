package k8s

import (
	"context"

	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/controller-runtime/pkg/client"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// UpdateStatusConditions updates the status conditions of the object if the passed conditions have changed.
// conditions is a pointer to the conditions of the object.
func UpdateStatusConditions(ctx context.Context, c client.Client, obj client.Object, conditions *[]metav1.Condition, 
									status metav1.ConditionStatus, conditionType string, reason string, msg string) error {
	cond := metav1.Condition{
		Type:               conditionType,
		Status:             status,
		Reason:             reason,
		Message:            msg,
		ObservedGeneration: obj.GetGeneration(),
	}

	changed := meta.SetStatusCondition(conditions, cond)

	if changed {
		err := c.Status().Update(ctx, obj)
		if err != nil {
			return err
		}
	}
	return nil
}