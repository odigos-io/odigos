package odigosconditionalattributes

import (
	"go.opentelemetry.io/collector/component"
)

type Config struct {
	Rules         []ConditionalRule `mapstructure:"rules,omitempty"`
	GlobalDefault string            `mapstructure:"global_default"`
}

var _ component.Config = (*Config)(nil)

type ConditionalRule struct {
	FieldToCheck                    string                                      `mapstructure:"field_to_check"`
	NewAttributeValueConfigurations map[string][]NewAttributeValueConfiguration `mapstructure:"new_attribute_value_configurations"`
}

type NewAttributeValueConfiguration struct {
	Value            string `mapstructure:"value"`
	FromField        string `mapstructure:"from_field,omitempty"`
	NewAttributeName string `mapstructure:"new_attribute,omitempty"`
}
