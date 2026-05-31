package config

import (
	commonapisampling "github.com/odigos-io/odigos/common/api/sampling"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/common/odigosattributes"
	"go.opentelemetry.io/otel/attribute"
)

// coomputed rules add some precomputed values to each rule, so it's easier and faster to evaluate them.
//
// the changes are:
// - percentage is resolved to it's default value based on the category (0% for noise, 100% for highly relevant, percentageAtMost for cost reduction)

type ComputedRule struct {
	RuleId     string
	Name       string
	Percentage float64
	Disabled   bool

	// attributes set to use for this rule when reporting metrics
	// fast path for metrics reporting without computing the attributes set for each span.
	MetricsAttributes attribute.Set
}

type ComputedNoisyOperation struct {
	ComputedRule
	Rule commonapisampling.NoisyOperation
}

type ComputedHighlyRelevantOperation struct {
	ComputedRule
	Rule commonapisampling.HighlyRelevantOperation
}

type ComputedCostReductionRule struct {
	ComputedRule
	Rule commonapisampling.CostReductionRule
}

type ComputedWorkloadConfig struct {
	NoisyOperations          []ComputedNoisyOperation
	HighlyRelevantOperations []ComputedHighlyRelevantOperation
	CostReductionRules       []ComputedCostReductionRule
}

func compteRuleMetricsAttributes(category consts.SamplingCategory, ruleId string, ruleName string, ruleDisabled bool, dryRun bool) attribute.Set {
	rulesAttrs := []attribute.KeyValue{
		attribute.String(odigosattributes.SamplingCategory, string(category)),
		attribute.String(odigosattributes.SamplingRuleId, ruleId),
		attribute.String(odigosattributes.SamplingRuleName, ruleName),
	}

	// if rule was evaluated but disabled, add an attribute so it's visible in the metrics.
	if ruleDisabled {
		rulesAttrs = append(rulesAttrs, attribute.Bool(odigosattributes.SamplingRuleDisabled, true))
	}

	if dryRun {
		rulesAttrs = append(rulesAttrs, attribute.Bool(odigosattributes.SamplingDryRun, true))
	}

	return attribute.NewSet(rulesAttrs...)
}

func precomputeNoisyOperations(cfg *commonapisampling.TailSamplingSourceConfig, dryRun bool) []ComputedNoisyOperation {
	out := make([]ComputedNoisyOperation, 0, len(cfg.NoisyOperations))
	for _, rule := range cfg.NoisyOperations {
		percentage := GetPercentageOrDefault0(rule.PercentageAtMost)
		metricsAttributes := compteRuleMetricsAttributes(consts.SamplingCategoryNoise, rule.Id, rule.Name, rule.Disabled, dryRun)
		out = append(out, ComputedNoisyOperation{
			ComputedRule: ComputedRule{
				RuleId:            rule.Id,
				Name:              rule.Name,
				Percentage:        percentage,
				Disabled:          rule.Disabled,
				MetricsAttributes: metricsAttributes,
			},
			Rule: rule,
		})
	}
	return out
}

func precomputeHighlyRelevantOperations(cfg *commonapisampling.TailSamplingSourceConfig, dryRun bool) []ComputedHighlyRelevantOperation {
	out := make([]ComputedHighlyRelevantOperation, 0, len(cfg.HighlyRelevantOperations))
	for _, rule := range cfg.HighlyRelevantOperations {
		percentage := GetPercentageOrDefault100(rule.PercentageAtLeast)
		metricsAttributes := compteRuleMetricsAttributes(consts.SamplingCategoryHighlyRelevant, rule.Id, rule.Name, rule.Disabled, dryRun)
		out = append(out, ComputedHighlyRelevantOperation{
			ComputedRule: ComputedRule{
				RuleId:            rule.Id,
				Name:              rule.Name,
				Percentage:        percentage,
				Disabled:          rule.Disabled,
				MetricsAttributes: metricsAttributes,
			},
			Rule: rule,
		})
	}
	return out
}

func precomputeCostReductionRules(cfg *commonapisampling.TailSamplingSourceConfig, dryRun bool) []ComputedCostReductionRule {
	out := make([]ComputedCostReductionRule, 0, len(cfg.CostReductionRules))
	for _, rule := range cfg.CostReductionRules {
		percentage := rule.PercentageAtMost
		metricsAttributes := compteRuleMetricsAttributes(consts.SamplingCategoryCostReduction, rule.Id, rule.Name, rule.Disabled, dryRun)
		out = append(out, ComputedCostReductionRule{
			ComputedRule: ComputedRule{
				RuleId:            rule.Id,
				Name:              rule.Name,
				Percentage:        percentage,
				Disabled:          rule.Disabled,
				MetricsAttributes: metricsAttributes,
			},
			Rule: rule,
		})
	}
	return out
}

func precomputeWorkloadConfig(cfg *commonapisampling.TailSamplingSourceConfig, dryRun bool) *ComputedWorkloadConfig {
	return &ComputedWorkloadConfig{
		NoisyOperations:          precomputeNoisyOperations(cfg, dryRun),
		HighlyRelevantOperations: precomputeHighlyRelevantOperations(cfg, dryRun),
		CostReductionRules:       precomputeCostReductionRules(cfg, dryRun),
	}
}
