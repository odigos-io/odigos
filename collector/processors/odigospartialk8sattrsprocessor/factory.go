package odigospartialk8sattrsprocessor

import (
	"context"

	"github.com/odigos-io/odigos/collector/processor/odigospartialk8sattrsprocessor/internal/metadata"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processorhelper"
)

//go:generate mdatagen metadata.yaml

// NewFactory returns a new factory for the Service Name processor.
func NewFactory() processor.Factory {
	return processor.NewFactory(
		metadata.Type,
		createDefaultConfig,
		processor.WithLogs(createLogsProcessor, metadata.LogsStability),
	)
}

func createDefaultConfig() component.Config {
	return &Config{}
}

func createLogsProcessor(
	ctx context.Context,
	set processor.Settings,
	cfg component.Config,
	nextConsumer consumer.Logs) (processor.Logs, error) {

	proc := newPartialK8sAttrsProcessor()

	return processorhelper.NewLogs(
		ctx,
		set,
		cfg,
		nextConsumer,
		proc.processLogs,
		processorhelper.WithCapabilities(consumer.Capabilities{MutatesData: true}),
		processorhelper.WithStart(proc.Start),
	)
}
