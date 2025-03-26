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
			// If satisfied at this level, sample immediately using that probability
			return rand.Float64() < probability
		} else if matched {
			// Keep track of this fallback probability
			foundFallback = true
			fallbackRatio = probability
			// Do NOT return yet; maybe a deeper level is fully satisfied
		}
	}

	// If we reach here, we never found a "satisfied" rule
	// If we had a fallback from a matched rule, use it
	if foundFallback {
		return rand.Float64() < fallbackRatio
	}

	// Otherwise, we return false to indicate "drop"
	return false
}

// evaluateLevel processes all rules in the current level to find combined probabilities.
// - sampleProbabilities accumulates for "satisfied" rules
// - fallbackProbabilities accumulates for "matched but not satisfied" rules
//
// Return values:
//
//	matched:   Were any rules matched at this level (whether satisfied or not)?
//	satisfied: Were any rules satisfied at this level?
//	probability: The combined sampling probability (union) for whichever category is relevant.
func evaluateLevel(td ptrace.Traces, rules []sampling.SamplingDecision) (matched bool, satisfied bool, probability float64) {
	var sampleProbabilities []float64
	var fallbackProbabilities []float64

	for _, rule := range rules {
		isMatched, isSatisfied, p := rule.Evaluate(td)
		if isSatisfied {
			sampleProbabilities = append(sampleProbabilities, p)
		} else if isMatched {
			// matched but not satisfied
			fallbackProbabilities = append(fallbackProbabilities, p)
		}
	}

	// If we have ANY satisfied rules, we combine them (union of independent probabilities)
	if len(sampleProbabilities) > 0 {
		return true, true, UnionOfIndependents(sampleProbabilities)
	}

	// If we have no satisfied but we do have matched, combine fallback probabilities
	if len(fallbackProbabilities) > 0 {
		return true, false, UnionOfIndependents(fallbackProbabilities)
	}

	// If no rules matched at all
	return false, false, 0.0
}

func UnionOfIndependents(probs []float64) float64 {
	product := 1.0
	for _, p := range probs {
		product *= (1.0 - p)
	}
	return 1.0 - product
}
