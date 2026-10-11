package odigosrouterconnector

import (
	"errors"

	"go.opentelemetry.io/collector/component"
)

type Config struct {
	component.Config
	OdigosConfigExtension *component.ID `mapstructure:"odigos_config_extension"`
}

func (c *Config) Validate() error {
	if c.OdigosConfigExtension == nil {
		return errors.New("odigos_config_extension is required")
	}
	return nil
}
