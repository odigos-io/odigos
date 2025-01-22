package odigossourcesfilter

import (
	"go.opentelemetry.io/collector/component"
)

type Config struct {
	MatchConditions []string `mapstructure:"match_conditions"`
}

var _ component.Config = (*Config)(nil)

func (cfg *Config) Validate() error {
	return nil
}
