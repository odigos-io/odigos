package odigostailsamplingprocessor

import (
	"errors"

	"go.opentelemetry.io/collector/component"
)

// Config holds the configuration for the odigostailsampling processor.
type Config struct {

	// the name of the extension that provides the odigos configuration
	OdigosConfigExtension *component.ID `mapstructure:"odigos_config_extension"`
}

var _ component.Config = (*Config)(nil)

// Validate validates the processor configuration.
func (cfg *Config) Validate() error {
	if cfg.OdigosConfigExtension == nil {
		return errors.New("odigos config extension is required")
	}
	return nil
}
