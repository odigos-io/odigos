package odigosconfigk8sextension

import "go.opentelemetry.io/collector/component"

type Config struct {
}

func (c *Config) Validate() error {
	return nil
}

var _ component.Config = (*Config)(nil)
