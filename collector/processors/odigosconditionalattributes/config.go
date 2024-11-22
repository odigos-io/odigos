package odigosconditionalattributes

import (
	"go.opentelemetry.io/collector/component"
)

type Config struct {
	Rules         []ConditionalRule `mapstructure:"rules,omitempty"`
	GlobalDefault string            `mapstructure:"global_default,omitempty"`
}

var _ component.Config = (*Config)(nil)

type ConditionalRule struct {
	AttributeToCheck string             `mapstructure:"attribute_to_check"`
	Values           map[string][]Value `mapstructure:"values"`
}

type Value struct {
	Value         string `mapstructure:"value,omitempty"`
	FromAttribute string `mapstructure:"from_attribute,omitempty"`
	NewAttribute  string `mapstructure:"new_attribute,omitempty"`
}
