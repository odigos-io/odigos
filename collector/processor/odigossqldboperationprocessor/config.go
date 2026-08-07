package odigossqldboperationprocessor

import (
	"go.opentelemetry.io/collector/component"
)

type Config struct {
	// configuration for excluding spans based on the language
	// user might opt-out of this enhancment if changing the span name is undesirable
	// due to mismatch with metric name that was already collected.
	Exclude *ExcludeConfig `mapstructure:"exclude"`
}

type ExcludeConfig struct {
	Language []string `mapstructure:"language"`
}

var _ component.Config = (*Config)(nil)

func (cfg *Config) Validate() error {

	return nil
}
