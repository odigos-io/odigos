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
	commonapi "github.com/odigos-io/odigos/common/api"

	"github.com/odigos-io/odigos/common/collector"
)

type tailSamplingProcessor struct {
	logger                *zap.Logger
	config                *Config
	odigosConfigExtension collector.OdigosConfigExtension
}

func (p *tailSamplingProcessor) processTraces(ctx context.Context, td ptrace.Traces) (ptrace.Traces, error) {

	if p.odigosConfigExtension == nil {
		p.logger.Error("odigos config extension is not set, skipping tail sampling")
		return td, nil // for auto generated tests, and not to crash in case it somehow happens
	}
	if td.ResourceSpans().Len() == 0 {
		return td, nil // no spans to process
	}

	// assuming that all the spans have the same trace ID, so take just the first one.
	traceID := td.ResourceSpans().At(0).ScopeSpans().At(0).Spans().At(0).TraceID()
	rnd := sampling.TraceIDToRandomness(traceID)
	// convert from range [0-MaxAdjustedCount] to range [0-100]
	tracePercentage := float64(rnd.Unsigned()) / float64(sampling.MaxAdjustedCount) * 100.0

	// Noisy operations category.
	matched, rule := p.evaluateNoisyOperations(td)
	if matched {
		percentageAtMost := 0.0
		if rule.PercentageAtMost != nil {
			percentageAtMost = *rule.PercentageAtMost
		}

		// either drop it, or keep it and add relevant sampling attributes to all spans.
		keepTrace := tracePercentage <= percentageAtMost

		if keepTrace {
			enrichSpansWithSamplingAttributes(td, "noisy", rule.Id, percentageAtMost)
			return td, nil
		} else {
			// drop the trace by not returning anything in the result.
			return ptrace.NewTraces(), nil
		}
	}

	return td, nil
}

// evaluateNoisyOperations evaluates the noisy operations category for the trace.
// it return the result of the evaluation.
func (p *tailSamplingProcessor) evaluateNoisyOperations(td ptrace.Traces) (bool, *commonapi.WorkloadNoisyOperation) {

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

func (p *tailSamplingProcessor) getTailSamplingConfig(resource pcommon.Resource) (*commonapi.SamplingCollectorConfig, bool) {
	collectorConfig, ok := p.odigosConfigExtension.GetFromResource(resource)
	if !ok {
		return nil, false
	}
	if collectorConfig.TailSampling == nil {
		return nil, false
	}
	return collectorConfig.TailSampling, true
}

func newTailSamplingProcessor(logger *zap.Logger, cfg *Config) *tailSamplingProcessor {
	return &tailSamplingProcessor{
		logger: logger,
		config: cfg,
	}
}
