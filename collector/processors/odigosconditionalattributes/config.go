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
	AttributeToCheck                string                                      `mapstructure:"field_to_check"`
	NewAttributeValueConfigurations map[string][]NewAttributeValueConfiguration `mapstructure:"new_field_value_configurations"`
}

type NewAttributeValueConfiguration struct {
	Value            string `mapstructure:"value"`
	FromAttribute    string `mapstructure:"from_field,omitempty"`
	NewAttributeName string `mapstructure:"new_field,omitempty"`
}
