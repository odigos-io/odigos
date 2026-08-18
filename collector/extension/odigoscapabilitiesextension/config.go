package odigoscapabilitiesextension

import "go.opentelemetry.io/collector/component"

// Config is empty: the extension takes no options.
type Config struct{}

func (c *Config) Validate() error {
	return nil
}

var _ component.Config = (*Config)(nil)
