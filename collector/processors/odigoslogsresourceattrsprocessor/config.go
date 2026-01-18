package odigoslogsresourceattrsprocessor

import (
	"go.opentelemetry.io/collector/component"
)

type Config struct {
}

var _ component.Config = (*Config)(nil)

func (cfg *Config) Validate() error {

	return nil
}
