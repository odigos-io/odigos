package services

import (
	"encoding/json"

	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/instrumentationrules"
)

// GetInstrumentationRuleTypes maps the embedded YAML instrumentation-rules
// catalog into the GraphQL model. It mirrors GetActionTypes so the UI can render
// the list of available instrumentation rules (and their dynamic fields) without
// hard-coding them.
func GetInstrumentationRuleTypes() []*model.InstrumentationRuleTypeOption {
	resp := make([]*model.InstrumentationRuleTypeOption, 0, len(instrumentationrules.Get()))

	for _, ruleConfig := range instrumentationrules.Get() {
		item := InstrumentationRuleConfigToTypeOption(ruleConfig)
		resp = append(resp, &item)
	}

	return resp
}

func InstrumentationRuleConfigToTypeOption(ruleConfig instrumentationrules.InstrumentationRule) model.InstrumentationRuleTypeOption {
	fields := []*model.InstrumentationRuleFieldYamlProperties{}

	for _, field := range ruleConfig.Spec.Fields {
		componentPropsJSON, err := json.Marshal(field.ComponentProps)
		if err != nil {
			continue
		}

		fields = append(fields, &model.InstrumentationRuleFieldYamlProperties{
			Name:                field.Name,
			DisplayName:         field.DisplayName,
			ComponentType:       field.ComponentType,
			ComponentProperties: string(componentPropsJSON),
			InitialValue:        field.InitialValue,
			RenderCondition:     field.RenderCondition,
		})
	}

	supportedLanguages := ruleConfig.Spec.SupportedLanguages
	if supportedLanguages == nil {
		supportedLanguages = []string{}
	}

	return model.InstrumentationRuleTypeOption{
		Type:               ruleConfig.Metadata.Type,
		DisplayName:        ruleConfig.Metadata.DisplayName,
		Description:        ruleConfig.Spec.Description,
		SupportedLanguages: supportedLanguages,
		DocsURL:            ruleConfig.Spec.DocsURL,
		Fields:             fields,
	}
}
