package status

import (
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/status"
	actionstatus "github.com/odigos-io/odigos/status/action/generated"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ConvertActionConditionsToStatuses maps Action CR status conditions to
// DesiredConditionStatus using the generated action status reason catalogs.
func ConvertActionConditionsToStatuses(conditions []metav1.Condition) []*model.DesiredConditionStatus {
	result := make([]*model.DesiredConditionStatus, 0, len(conditions))
	for _, c := range conditions {
		result = append(result, convertActionConditionToStatus(c))
	}
	return result
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
