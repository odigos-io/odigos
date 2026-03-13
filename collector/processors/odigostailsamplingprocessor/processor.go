package odigostailsamplingprocessor

import (
	"context"
	"fmt"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/sampling"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"

	"github.com/odigos-io/odigos/collector/processors/odigostailsamplingprocessor/internal/metadata"
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

	// Tail sampling logic (category evaluation, drop/keep decisions) will be added in a follow-up PR.
	_ = tracePercentage
	return td, nil
}

func (p *tailSamplingProcessor) Start(ctx context.Context, host component.Host) error {
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
