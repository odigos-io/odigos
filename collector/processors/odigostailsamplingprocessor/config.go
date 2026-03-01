package odigostailsamplingprocessor

import (
	"go.opentelemetry.io/collector/component"
)

// Config holds the configuration for the odigostailsampling processor.
type Config struct {
	// Add configuration fields here.
}

var _ component.Config = (*Config)(nil)

// Validate validates the processor configuration.
func (cfg *Config) Validate() error {
	return nil
}
