package odigosextractattributeprocessor

import (
	"fmt"

	"go.opentelemetry.io/collector/component"
)

// Config defines the configuration for the odigosextractattribute processor.
//
// The processor scans free-form string-valued span attributes for an embedded
// SourceAttribute and lifts the matched value onto a new attribute named
// TargetAttribute.
type Config struct {
	// SourceAttribute is the string name to search for in free-form string attributes, to get its value.
	SourceAttribute string `mapstructure:"source_attribute"`

	// TargetAttribute is the  new span attribute name that will get the value from SourceAttribute.
	TargetAttribute string `mapstructure:"target_attribute"`
}

var _ component.Config = (*Config)(nil)

func (cfg *Config) Validate() error {
	if cfg.SourceAttribute == "" {
		return fmt.Errorf("source_attribute is required")
	}
	if cfg.TargetAttribute == "" {
		return fmt.Errorf("target_attribute is required")
	}
	return nil
}
