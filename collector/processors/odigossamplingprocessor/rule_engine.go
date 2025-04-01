package odigossamplingprocessor

import (
	"math/rand"

	"github.com/open-telemetry/opentelemetry-collector-contrib/odigos/processor/odigossamplingprocessor/internal/sampling"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

// RuleEngine determines whether to sample a trace based on a set of prioritized rules.
// Rules are organized into levels (e.g., Global > Service > Endpoint) and evaluated in order.
type RuleEngine struct {
	logger *zap.Logger
	rules  RuleSet
}

// RuleSet groups sampling rules into priority levels.
// Rules are evaluated in order: Global → Service → Endpoint.
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

// extractDecisions converts generic rule definitions into SamplingDecision implementations.
func extractDecisions(rules []Rule) []sampling.SamplingDecision {
	var decisions []sampling.SamplingDecision
	for _, r := range rules {
		if decision, ok := r.RuleDetails.(sampling.SamplingDecision); ok {
			decisions = append(decisions, decision)
		}
	}
	return decisions
}

// ShouldSample determines whether to sample a trace based on rule evaluation.
// It proceeds in priority order (Global → Service → Endpoint).
//
//   - If any level contains satisfied rules, the maximum sampling ratio from that level is used.
//   - If no level is satisfied, but one or more rules matched (but didn't satisfy), the minimum fallback
//     ratio from all matched rules across all levels is used.
//   - If no rules matched at all, the trace is dropped.
func (re *RuleEngine) ShouldSample(td ptrace.Traces) bool {
	levels := [][]sampling.SamplingDecision{
		re.rules.GlobalRules,
		re.rules.ServiceRules,
		re.rules.EndpointRules,
	}

	var minFallback *float64

	for _, rules := range levels {
		ratio, satisfied, matched := evaluateLevel(td, rules)

		if satisfied {
			return (rand.Float64() * 100) < ratio
		}

		if matched {
			if minFallback == nil || ratio < *minFallback {
				minFallback = &ratio
			}
		}
	}

	if minFallback != nil {
		return (rand.Float64() * 100) < *minFallback
	}

	return false
}

// evaluateLevel runs all rules in a given level and returns:
// - ratio: the max sample ratio if any rules are satisfied, or the min fallback ratio if only matched
// - satisfied: whether any rule in this level was satisfied
// - matched: whether any rule in this level was matched (satisfied or not)
func evaluateLevel(td ptrace.Traces, rules []sampling.SamplingDecision) (
	ratio float64,
	satisfied bool,
	matched bool,
) {
	var foundFallback bool

	for _, rule := range rules {
		isMatched, isSatisfied, p := rule.Evaluate(td)

		if isSatisfied {
			satisfied = true
			ratio = max(ratio, p)
			matched = true
		} else if isMatched {
			matched = true
			if !foundFallback {
				ratio = p
				foundFallback = true
			} else {
				ratio = min(ratio, p)
			}
		}
	}

	return ratio, satisfied, matched
}
