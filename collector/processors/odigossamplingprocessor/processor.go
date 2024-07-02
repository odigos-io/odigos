package odigossamplingprocessor

import (
	"context"
	"math/rand"

	"github.com/open-telemetry/opentelemetry-collector-contrib/odigos/processor/odigossamplingprocessor/internal/sampling"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

type samplingProcessor struct {
	logger *zap.Logger
	config *Config
}

func (sp *samplingProcessor) processTraces(ctx context.Context, td ptrace.Traces) (ptrace.Traces, error) {
	globalUnsatisfiedRatio := 0.0
	serviceUnsatisfiedRatio := 0.0
	EndpointUnsatisfiedRatio := 0.0

	// Evaluate global rules first
	for _, rule := range sp.config.GlobalRules {
		switch r := rule.RuleDetails.(type) {
		case *sampling.ErrorRule: //
			if r.KeepTraceDecision(td) {
				return td, nil
			} else {
				globalUnsatisfiedRatio = max(globalUnsatisfiedRatio, r.FallbackSamplingRatio)
			}
		default:
			sp.logger.Error("Unknown rule details type", zap.String("rule", rule.Name))
		}
	}

	// Placeholder for service rules

	// Evaluate endpoint rules
	for _, rule := range sp.config.EndpointRules {
		switch r := rule.RuleDetails.(type) {
		case *sampling.HttpRouteLatencyRule:
			filterMatch, conditionMatch := r.KeepTraceDecision(td)
			if filterMatch {
				if conditionMatch {
					return td, nil
				} else {
					EndpointUnsatisfiedRatio = max(EndpointUnsatisfiedRatio, r.FallbackSamplingRatio)
				}
			}
		default:
			sp.logger.Error("Unknown rule details type", zap.String("rule", rule.Name))
		}
	}

	var finalUnsatisfiedRatio float64
	// Evaluate against the most specific unsatisfied ratio
	if EndpointUnsatisfiedRatio > 0.0 {
		finalUnsatisfiedRatio = EndpointUnsatisfiedRatio
	} else {
		if serviceUnsatisfiedRatio > 0.0 {
			finalUnsatisfiedRatio = serviceUnsatisfiedRatio
		} else {
			if globalUnsatisfiedRatio > 0.0 {
				finalUnsatisfiedRatio = globalUnsatisfiedRatio
			} else {
				// None of the rules matched, trace is sampled by default
				return td, nil
			}
		}
	}

	// Sample the trace based on the final unsatisfied ratio
	if finalUnsatisfiedRatio > 0.0 && (rand.Float64()*100) < finalUnsatisfiedRatio {
		return td, nil
	}

	sp.removeAllSpans(&td)
	return td, nil
}

////

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func (sp *samplingProcessor) removeAllSpans(td *ptrace.Traces) {
	td.ResourceSpans().RemoveIf(func(rs ptrace.ResourceSpans) bool { return true })
}
