package odigostailsamplingprocessor

import (
	"context"
	"fmt"

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
	// Tail sampling logic will be added in a follow-up PR.
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
