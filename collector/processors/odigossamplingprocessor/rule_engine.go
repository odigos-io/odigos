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

// ShouldSample checks each level (Global, Service, Endpoint) in order.
// - If any level is "satisfied," sample immediately.
// - If a level is matched but not satisfied, keep that probability for fallback.
// - If no level is satisfied, but at least one was matched, apply fallback probability.
func (re *RuleEngine) ShouldSample(td ptrace.Traces) bool {
	var fallbackRatio float64
	var foundFallback bool

	// Each of these slices is a "level" of rules
	levels := [][]sampling.SamplingDecision{
		re.rules.GlobalRules,
		re.rules.ServiceRules,
		re.rules.EndpointRules,
	}

	for _, rules := range levels {
		matched, satisfied, probability := evaluateLevel(td, rules)

		if satisfied {

			return (rand.Float64() * 100) < probability
		} else if matched {
			// Keep track of this fallback probability
			foundFallback = true
			fallbackRatio = probability
		}
	}

	// If we reach here, we never found a "satisfied" rule
	// If we had a fallback from a matched rule, use it
	if foundFallback {
		return (rand.Float64() * 100) < fallbackRatio
	}

	// Otherwise, we return false to indicate "drop"
	return false
}

// evaluateLevel processes all rules in the current level to find the max satisfied (Or matched) probability.
// - sampleProbabilities accumulates for "satisfied" rules
// - fallbackProbabilities accumulates for "matched but not satisfied" rules
//
// Return values:
//
//	matched:   Were any rules matched at this level (whether satisfied or not)?
//	satisfied: Were any rules satisfied at this level?
//	probability: The combined sampling probability (union) for whichever category is relevant.
func evaluateLevel(td ptrace.Traces, rules []sampling.SamplingDecision) (matched bool, satisfied bool, probability float64) {
	var sampleProbability float64
	var fallbackProbability float64

	for _, rule := range rules {
		isMatched, isSatisfied, p := rule.Evaluate(td)
		if isSatisfied {
			sampleProbability = max(sampleProbability, p)
		} else if isMatched {
			// matched but not satisfied
			fallbackProbability = max(fallbackProbability, p)
		}
	}

	if sampleProbability != 0.0 {
		return true, true, sampleProbability
	} else if fallbackProbability != 0.0 {
		return true, true, fallbackProbability
	}
	// If no rules matched at all
	return false, false, 0.0
}
