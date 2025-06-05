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
	if r.Name == "" {
		return errors.New("rule name cannot be empty")
	}
	if r.Type == "" {
		return errors.New("rule type cannot be empty")
	}
	if r.RuleDetails == nil {
		return errors.New("rule details cannot be nil")
	}

	var (
		details sampling.SamplingDecision
		err     error
	)

	switch r.Type {
	case "http_latency":
		details, err = decodeAndValidate[*sampling.HttpRouteLatencyRule](r.RuleDetails)
	case "error":
		details, err = decodeAndValidate[*sampling.ErrorRule](r.RuleDetails)
	case "span_attribute":
		details, err = decodeAndValidate[*sampling.SpanAttributeRule](r.RuleDetails)
	case "service_name":
		details, err = decodeAndValidate[*sampling.ServiceNameRule](r.RuleDetails)
	default:
		return fmt.Errorf("unknown rule type: %s", r.Type)
	}

	if err != nil {
		return err
	}

	r.RuleDetails = details
	return nil
}

func decodeAndValidate[T sampling.SamplingDecision](raw interface{}) (sampling.SamplingDecision, error) {
	var rule T
	if err := mapstructure.Decode(raw, &rule); err != nil {
		return nil, err
	}
	if err := rule.Validate(); err != nil {
		return nil, err
	}
	return rule, nil
}
