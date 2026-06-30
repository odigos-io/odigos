package services

import (
	"encoding/json"

	"github.com/odigos-io/odigos/actions"
	"github.com/odigos-io/odigos/frontend/graph/model"
)

// GetActionTypes maps the embedded YAML action catalog into the GraphQL model.
// It mirrors GetDestinationCategories so the UI can render the list of available
// actions (and their dynamic fields) without hard-coding them.
func GetActionTypes() []*model.ActionTypeOption {
	resp := make([]*model.ActionTypeOption, 0, len(actions.Get()))

	for _, actionConfig := range actions.Get() {
		item := ActionConfigToTypeOption(actionConfig)
		resp = append(resp, &item)
	}

	return resp
}

func ActionConfigToTypeOption(actionConfig actions.Action) model.ActionTypeOption {
	fields := []*model.ActionFieldYamlProperties{}

	for _, field := range actionConfig.Spec.Fields {
		componentPropsJSON, err := json.Marshal(field.ComponentProps)
		if err != nil {
			continue
		}

		fields = append(fields, &model.ActionFieldYamlProperties{
			Name:                field.Name,
			DisplayName:         field.DisplayName,
			ComponentType:       field.ComponentType,
			ComponentProperties: string(componentPropsJSON),
			InitialValue:        field.InitialValue,
			RenderCondition:     field.RenderCondition,
		})
	}

	// Flatten the destinations-style `signals.*.supported` map into the list of
	// signals the action allows, which is what the GraphQL/UI consume.
	allowedSignals := make([]model.SignalType, 0, 4)
	if actionConfig.Spec.Signals.Traces.Supported {
		allowedSignals = append(allowedSignals, model.SignalTypeTraces)
	}
	if actionConfig.Spec.Signals.Metrics.Supported {
		allowedSignals = append(allowedSignals, model.SignalTypeMetrics)
	}
	if actionConfig.Spec.Signals.Logs.Supported {
		allowedSignals = append(allowedSignals, model.SignalTypeLogs)
	}
	if actionConfig.Spec.Signals.Profiles.Supported {
		allowedSignals = append(allowedSignals, model.SignalTypeProfiles)
	}

	return model.ActionTypeOption{
		Type:            actionConfig.Metadata.Type,
		DisplayName:     actionConfig.Metadata.DisplayName,
		Description:     actionConfig.Spec.Description,
		AllowedSignals:  allowedSignals,
		DocsEndpoint:    actionConfig.Spec.DocsEndpoint,
		DocsDescription: actionConfig.Spec.DocsDescription,
		Fields:          fields,
	}
}
