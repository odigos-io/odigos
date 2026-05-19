package odigosprofilesprocessor

import (
	"fmt"

	"go.opentelemetry.io/collector/component"
)

// Config configures odigosprofilesprocessor.
type Config struct {
	// OdigosConfigExtension references the odigos_config_k8s extension (implements OdigosConfigExtension).
	OdigosConfigExtension *component.ID `mapstructure:"odigos_config_extension"`
}

// Validate checks configuration.
func (c *Config) Validate() error {
	if c.OdigosConfigExtension == nil {
		return fmt.Errorf("odigos_config_extension is required")
	}
	return nil
}
