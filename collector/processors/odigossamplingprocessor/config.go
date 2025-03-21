package odigossamplingprocessor

import (
	"errors"
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/open-telemetry/opentelemetry-collector-contrib/odigos/processor/odigossamplingprocessor/internal/sampling"
)

type Config struct {
	GlobalRules   []Rule `mapstructure:"global_rules,omitempty"`
	ServiceRules  []Rule `mapstructure:"service_rules,omitempty"`
	EndpointRules []Rule `mapstructure:"endpoint_rules,omitempty"`
}

func (cfg *Config) Validate() error {
	for _, rules := range [][]Rule{cfg.EndpointRules, cfg.ServiceRules, cfg.GlobalRules} {
		for _, rule := range rules {
			if err := rule.Validate(); err != nil {
				return err
			}
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
	if r.Name == "" || r.Type == "" || r.RuleDetails == nil {
		return errors.New("invalid rule: missing required fields")
	}

	switch r.Type {
	case "http_latency":
		var details sampling.HttpRouteLatencyRule
		if err := mapstructure.Decode(r.RuleDetails, &details); err != nil {
			return err
		}
		if err := details.Validate(); err != nil {
			return err
		}
		r.RuleDetails = &details
	case "error":
		var details sampling.ErrorRule
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
