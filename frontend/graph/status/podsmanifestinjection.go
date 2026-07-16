package status

import (
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/frontend/graph/model"
	podsManifestInjectionStatus "github.com/odigos-io/odigos/status/instrumentationconfig/generated"
)

func CalculatePodsManifestInjectionStatus(ic *v1alpha1.InstrumentationConfig) *model.DesiredConditionStatus {
	if ic == nil {
		return nil
	}

	for _, c := range ic.Status.Conditions {
		if c.Type == podsManifestInjectionStatus.PodsManifestInjectionType {
			r, ok := podsManifestInjectionStatus.PodsManifestInjectionReasonByName(c.Reason)
			if !ok {
				return nil
			}

			actionItems := make([]*model.DesiredConditionActionItem, 0, len(r.ActionItems))
			for _, actionItem := range r.ActionItems {
				actionItems = append(actionItems, &model.DesiredConditionActionItem{
					Type:           model.DesiredConditionActionItemType(actionItem.Type),
					UserFacingText: actionItem.UserFacingText,
				})
			}

			return &model.DesiredConditionStatus{
				Name:        c.Type,
				Status:      model.DesiredStateProgress(r.OdigosSeverity),
				ReasonEnum:  &c.Reason,
				Message:     c.Message,
				ActionItems: actionItems,
			}
		}
	}

	return nil
}
