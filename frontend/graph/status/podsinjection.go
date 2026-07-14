package status

import (
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/frontend/graph/model"
	podsInjectionStatus "github.com/odigos-io/odigos/status/instrumentationconfig/generated"
)

func CalculatePodsInjectionStatus(ic *v1alpha1.InstrumentationConfig) *model.DesiredConditionStatus {
	if ic == nil {
		return nil
	}

	for _, c := range ic.Status.Conditions {
		if c.Type == podsInjectionStatus.PodsInjectionType {
			r, ok := podsInjectionStatus.PodsInjectionReasonByName(c.Reason)
			if !ok {
				return nil
			}
			return &model.DesiredConditionStatus{
				Name:       c.Type,
				Status:     model.DesiredStateProgress(r.OdigosSeverity),
				ReasonEnum: &c.Reason,
				Message:    c.Message,
			}
		}
	}

	return nil
}
