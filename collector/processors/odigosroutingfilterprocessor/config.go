package odigosroutingfilterprocessor

import (
	"fmt"
	"strings"

	"go.opentelemetry.io/collector/component"
)

type Config struct {
	MatchConditions map[string]bool `mapstructure:"match_conditions"`
}

var _ component.Config = (*Config)(nil)

func (cfg *Config) Validate() error {
	if len(cfg.MatchConditions) == 0 {
		return nil
	}

	for key := range cfg.MatchConditions {
		parts := strings.Split(key, "/")
		if len(parts) != 3 {
			return fmt.Errorf("invalid match condition key format: %s (expected 'namespace/name/kind')", key)
		}
		if parts[0] == "" || parts[1] == "" || parts[2] == "" {
			return fmt.Errorf("invalid match condition key: %s (namespace, name, and kind must be non-empty)", key)
		}
	}

	return nil
}
