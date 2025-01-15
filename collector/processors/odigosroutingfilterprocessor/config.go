package odigosroutingfilterprocessor

import (
	"go.opentelemetry.io/collector/component"
)

type Config struct {
	MatchConditions []string `mapstructure:"match_conditions"`
	MatchMap        map[string]struct{}
}

var _ component.Config = (*Config)(nil)

func (cfg *Config) Validate() error {
	return nil
}

func (cfg *Config) InitMatchMap() {
	cfg.MatchMap = make(map[string]struct{}, len(cfg.MatchConditions))
	for _, condition := range cfg.MatchConditions {
		cfg.MatchMap[condition] = struct{}{}
	}
}
