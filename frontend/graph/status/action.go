package status

import (
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/status"
	actionstatus "github.com/odigos-io/odigos/status/action/generated"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ConvertActionConditionsToStatuses maps Action CR status conditions to
// DesiredConditionStatus using the generated action status reason catalogs.
// Conditions whose ObservedGeneration does not match generation are treated as
// stale and reported as WaitingForReconcile until controllers reconcile the current spec.
func ConvertActionConditionsToStatuses(conditions []metav1.Condition, generation int64) []*model.DesiredConditionStatus {
	result := make([]*model.DesiredConditionStatus, 0, len(conditions))
	for _, c := range conditions {
		if c.ObservedGeneration != generation {
			result = append(result, waitingForReconcileStatus(c.Type))
			continue
		}
		result = append(result, convertActionConditionToStatus(c))
	}
	return result
}

func waitingForReconcileStatus(conditionType string) *model.DesiredConditionStatus {
	r, ok := waitingForReconcileReasonByConditionType(conditionType)
	if !ok {
		return &model.DesiredConditionStatus{
			Name:    conditionType,
			Status:  model.DesiredStateProgressWaiting,
			Message: "Waiting for the latest configuration to be applied",
		}
	}
	return actionReasonToDesiredConditionStatus(conditionType, r, r.Message)
}

func waitingForReconcileReasonByConditionType(conditionType string) (status.Reason, bool) {
	switch conditionType {
	case actionstatus.TransformedToProcessorType:
		return actionstatus.TransformedToProcessorWaitingForReconcile, true
	case actionstatus.AddedToCollectorConfigType:
		return actionstatus.AddedToCollectorConfigWaitingForReconcile, true
	case actionstatus.AddedToSourcesConfigType:
		return actionstatus.AddedToSourcesConfigWaitingForReconcile, true
	default:
		return status.Reason{}, false
	}
}

func convertActionConditionToStatus(c metav1.Condition) *model.DesiredConditionStatus {
	r, ok := actionReasonByCondition(c)
	if !ok {
		return &model.DesiredConditionStatus{
			Name:    c.Type,
			Status:  model.DesiredStateProgressUnknown,
			Message: c.Message,
		}
	}
	return actionReasonToDesiredConditionStatus(c.Type, r, c.Message)
}

func actionReasonByCondition(c metav1.Condition) (status.Reason, bool) {
	switch c.Type {
	case actionstatus.TransformedToProcessorType:
		return actionstatus.TransformedToProcessorReasonByName(c.Reason)
	case actionstatus.AddedToCollectorConfigType:
		return actionstatus.AddedToCollectorConfigReasonByName(c.Reason)
	case actionstatus.AddedToSourcesConfigType:
		return actionstatus.AddedToSourcesConfigReasonByName(c.Reason)
	default:
		return status.Reason{}, false
	}
}

func actionReasonToDesiredConditionStatus(name string, r status.Reason, message string) *model.DesiredConditionStatus {
	if message == "" {
		message = r.Message
	}

	actionItems := make([]*model.DesiredConditionActionItem, 0, len(r.ActionItems))
	for _, actionItem := range r.ActionItems {
		actionItems = append(actionItems, &model.DesiredConditionActionItem{
			Type:       model.DesiredConditionActionItemType(actionItem.Type),
			ButtonText: actionItem.ButtonText,
		})
	}

	return &model.DesiredConditionStatus{
		Name:        name,
		Status:      model.DesiredStateProgress(r.OdigosSeverity),
		ReasonEnum:  &r.Title,
		Message:     message,
		ActionItems: actionItems,
	}
}
