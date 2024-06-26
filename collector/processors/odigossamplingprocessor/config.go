package odigossamplingprocessor

import (
	"errors"
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/open-telemetry/opentelemetry-collector-contrib/odigos/processor/odigossamplingprocessor/internal/sampling"
	"go.opentelemetry.io/collector/component"
)

type Config struct {
	Rules []Rule `mapstructure:"rules"`
}

var _ component.Config = (*Config)(nil)

func (cfg *Config) Validate() error {
	for _, rule := range cfg.Rules {
		if err := rule.Validate(); err != nil {
			return err
		}
	}
	return nil
}

type Rule struct {
	Name        string      `mapstructure:"name"`
	Type        string      `mapstructure:"type"`
	RuleDetails interface{} `mapstructure:"rule_details"`
}

func (r *Rule) Validate() error {
	if r.Name == "" {
		return errors.New("rule name cannot be empty")
	}
	if r.Type == "" {
		return errors.New("rule type cannot be empty")
	}
	if r.RuleDetails == nil {
		return errors.New("rule details cannot be nil")
	}

	switch r.Type {
	case "http_latency":
		var details sampling.TraceLatencyRule
		if err := mapstructure.Decode(r.RuleDetails, &details); err != nil {
			return err
		}
		if err := details.Validate(); err != nil {
			return err
		}
		r.RuleDetails = &details
	default:
		return fmt.Errorf("unknown rule type: %s", r.Type)
	}

	return nil
}
