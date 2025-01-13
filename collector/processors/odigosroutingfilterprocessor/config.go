package odigosroutingfilterprocessor

import (
	"go.opentelemetry.io/collector/component"
)

type Config struct {
	MatchConditions map[string]bool `mapstructure:"match_conditions"`
}

var _ component.Config = (*Config)(nil)

func (cfg *Config) Validate() error {
	return nil
}
