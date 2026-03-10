package odigostailsamplingprocessor

import (
	"context"
	"fmt"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/sampling"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"

	"github.com/odigos-io/odigos/collector/processors/odigostailsamplingprocessor/category"
	"github.com/odigos-io/odigos/collector/processors/odigostailsamplingprocessor/internal/metadata"
	commonapisampling "github.com/odigos-io/odigos/common/api/sampling"

	"github.com/odigos-io/odigos/common/collector"
)

type tailSamplingProcessor struct {
	logger                *zap.Logger
	config                *Config
	odigosConfigExtension collector.OdigosConfigExtension

	telemetryBuilder *metadata.TelemetryBuilder
}

func (p *tailSamplingProcessor) processTraces(ctx context.Context, td ptrace.Traces) (ptrace.Traces, error) {

	if p.odigosConfigExtension == nil {
		p.logger.Error("odigos config extension is not set, skipping tail sampling")
		return td, nil // for auto generated tests, and not to crash in case it somehow happens
	}
	if td.ResourceSpans().Len() == 0 || td.ResourceSpans().At(0).ScopeSpans().Len() == 0 || td.ResourceSpans().At(0).ScopeSpans().At(0).Spans().Len() == 0 {
		return td, nil // no spans to process
	}

	traceID, ok := assertAllSpansBelongToTheSameTrace(td)
	if !ok {
		p.logger.Error("not all spans belong to the same trace", zap.String("trace_id", traceID.String()))
		return td, nil // not all spans belong to the same trace
	}

	rnd := sampling.TraceIDToRandomness(traceID)
	// convert from range [0-MaxAdjustedCount] to range [0-100]
	tracePercentage := float64(rnd.Unsigned()) / float64(sampling.MaxAdjustedCount) * 100.0

	// Noisy operations category.
	matched, noisyOperationRule := p.evaluateNoisyOperations(td)
	if matched {
		percentageAtMost := 0.0
		if noisyOperationRule.PercentageAtMost != nil {
			percentageAtMost = *noisyOperationRule.PercentageAtMost
		}

		// either drop it, or keep it and add relevant sampling attributes to all spans.
		keepTrace := tracePercentage <= percentageAtMost

		if keepTrace || p.config.DryRun {
			enrichSpansWithSamplingAttributes(td, "noisy", noisyOperationRule.Id, noisyOperationRule.Name, percentageAtMost, p.config.DryRun, keepTrace)
			return td, nil
		} else {
			// drop the trace by not returning anything in the result.
			return ptrace.NewTraces(), nil
		}
	}

	matched, highlyRelevantOperationRule, _ := category.EvaluateHighlyRelevantOperations(ctx, td, p.odigosConfigExtension, tracePercentage, p.telemetryBuilder)
	if matched {
		percentageAtLeast := category.GetPercentageOrDefault(highlyRelevantOperationRule.PercentageAtLeast, 100.0)
		keepTrace := tracePercentage <= percentageAtLeast

		if keepTrace || p.config.DryRun {
			enrichSpansWithSamplingAttributes(td, "highly_relevant", highlyRelevantOperationRule.Id, highlyRelevantOperationRule.Name, percentageAtLeast, p.config.DryRun, keepTrace)
			return td, nil
		} else {
			return ptrace.NewTraces(), nil
		}
	}

	matched, costReductionRule, _ := category.EvaluateCostReductionOperations(td, p.odigosConfigExtension, tracePercentage)
	if matched {
		percentageAtMost := costReductionRule.PercentageAtMost
		keepTrace := tracePercentage <= percentageAtMost

		if keepTrace || p.config.DryRun {
			enrichSpansWithSamplingAttributes(td, "cost_reduction", costReductionRule.Id, costReductionRule.Name, percentageAtMost, p.config.DryRun, keepTrace)
			return td, nil
		} else {
			return ptrace.NewTraces(), nil
		}
	}

	return td, nil
}

// evaluateNoisyOperations evaluates the noisy operations category for the trace.
// it return the result of the evaluation.
func (p *tailSamplingProcessor) evaluateNoisyOperations(td ptrace.Traces) (bool, *commonapisampling.NoisyOperation) {

	rootSpan, resource, found := getRootSpan(td)
	if !found {
		// the root span is missing, so we cannot apply noisy operations category
		// as the rules are evaluated only on the root span.
		return false, nil
	}

	tailSamplingConfig, ok := p.getTailSamplingConfig(resource)
	if !ok {
		// the tail sampling config is set only if there are actually any rules.
		// this source is not relevant for noisy operations category.
		return false, nil
	}

	return category.EvaluateNoisyOperations(rootSpan, tailSamplingConfig.NoisyOperations)
}

func (p *tailSamplingProcessor) Start(ctx context.Context, host component.Host) error {
	// the extension name is validated as not nil in the config validate function
	// and can be nil in tests
	if p.config.OdigosConfigExtension != nil {
		ext, found := host.GetExtensions()[*p.config.OdigosConfigExtension]
		if !found || ext == nil {
			return fmt.Errorf("odigos config extension not found")
		}
		odigosConfigExtension, ok := ext.(collector.OdigosConfigExtension)
		if !ok {
			return fmt.Errorf("the collector extension instance %s is not a valid odigos config extension", *p.config.OdigosConfigExtension)
		}
		p.odigosConfigExtension = odigosConfigExtension
	}
	return nil
}

func (p *tailSamplingProcessor) getTailSamplingConfig(resource pcommon.Resource) (*commonapisampling.TailSamplingSourceConfig, bool) {
	collectorConfig, ok := p.odigosConfigExtension.GetFromResource(resource)
	if !ok {
		return nil, false
	}
	if collectorConfig.TailSampling == nil {
		return nil, false
	}
	return collectorConfig.TailSampling, true
}

func newTailSamplingProcessor(logger *zap.Logger, cfg *Config, set component.TelemetrySettings) *tailSamplingProcessor {
	telemetryBuilder, err := metadata.NewTelemetryBuilder(set)
	if err != nil {
		logger.Error("failed to create telemetry builder", zap.Error(err))
	}
	return &tailSamplingProcessor{
		logger:           logger,
		config:           cfg,
		telemetryBuilder: telemetryBuilder,
	}
}
