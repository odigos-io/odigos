package odigosroutingfilterprocessor

import (
	"errors"

	"go.opentelemetry.io/collector/component"
)

type Config struct {
	MatchConditions []MatchCondition `mapstructure:"match_conditions"`
}

var _ component.Config = (*Config)(nil)

func (cfg *Config) Validate() error {
	if len(cfg.MatchConditions) == 0 {
		return errors.New("at least one match condition must be specified")
	}
	for _, condition := range cfg.MatchConditions {
		if err := condition.Validate(); err != nil {
			return err
		}
	}
	return nil
}

type MatchCondition struct {
	Name      string `mapstructure:"name"`
	Namespace string `mapstructure:"namespace"`
	Kind      string `mapstructure:"kind"`
}

func (mc *MatchCondition) Validate() error {
	if mc.Name == "" || mc.Namespace == "" || mc.Kind == "" {
		return errors.New("all match condition fields (name, namespace, kind) must be specified")
	}
	return nil
}
