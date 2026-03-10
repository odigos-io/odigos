package odigostailsamplingprocessor

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processorhelper"

	"github.com/odigos-io/odigos/collector/processors/odigostailsamplingprocessor/internal/metadata"
)

// NewFactory returns a new factory for the odigostailsampling processor.
func NewFactory() processor.Factory {
	return processor.NewFactory(
		metadata.Type,
		createDefaultConfig,
		processor.WithTraces(createTracesProcessor, component.StabilityLevelBeta),
	)
}

func createDefaultConfig() component.Config {
	return &Config{}
}

func createTracesProcessor(
	ctx context.Context,
	set processor.Settings,
	cfg component.Config,
	nextConsumer consumer.Traces,
) (processor.Traces, error) {
	proc := newTailSamplingProcessor(set.Logger, cfg.(*Config), set.TelemetrySettings)
	return processorhelper.NewTraces(
		ctx,
		set,
		cfg,
		nextConsumer,
		proc.processTraces,
		processorhelper.WithStart(proc.Start),
		processorhelper.WithCapabilities(consumer.Capabilities{MutatesData: true}),
	)
}
