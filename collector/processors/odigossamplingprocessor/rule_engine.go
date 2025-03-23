package odigossamplingprocessor

import (
	"math/rand"

	"github.com/open-telemetry/opentelemetry-collector-contrib/odigos/processor/odigossamplingprocessor/internal/sampling"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

type RuleEngine struct {
	logger *zap.Logger
	rules  RuleSet
}

type RuleSet struct {
	EndpointRules []sampling.SamplingDecision
	ServiceRules  []sampling.SamplingDecision
	GlobalRules   []sampling.SamplingDecision
}

func NewRuleEngine(cfg *Config, logger *zap.Logger) *RuleEngine {
	return &RuleEngine{
		logger: logger,
		rules: RuleSet{
			EndpointRules: extractDecisions(cfg.EndpointRules),
			ServiceRules:  extractDecisions(cfg.ServiceRules),
			GlobalRules:   extractDecisions(cfg.GlobalRules),
		},
	}
}

func extractDecisions(rules []Rule) []sampling.SamplingDecision {
	var decisions []sampling.SamplingDecision
	for _, r := range rules {
		decisions = append(decisions, r.RuleDetails.(sampling.SamplingDecision))
	}
	return decisions
}

func (re *RuleEngine) ShouldSample(td ptrace.Traces) bool {
	levels := [][]sampling.SamplingDecision{
		re.rules.EndpointRules,
		re.rules.ServiceRules,
		re.rules.GlobalRules,
	}

	for _, rules := range levels {
		matched, decision := evaluateLevel(td, rules)
		if matched {
			return decision
		}
	}
	return false
}

func evaluateLevel(td ptrace.Traces, rules []sampling.SamplingDecision) (matched bool, decision bool) {
	var fallbackSet bool
	var maxFallbackRatio float64

	for _, rule := range rules {
		isMatched, isSatisfied, fallback := rule.Evaluate(td)
		if isMatched {
			if isSatisfied {
				return true, true // sample immediately
			}
			fallbackSet = true
			if fallback > maxFallbackRatio {
				maxFallbackRatio = fallback
			}
		}
	}

	if fallbackSet {
		return true, (rand.Float64() * 100) < maxFallbackRatio
	}
	return false, false
}
