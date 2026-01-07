package odigospartialk8sattrsprocessor

import (
	"context"
	"fmt"

	"github.com/odigos-io/odigos/collector/processor/odigospartialk8sattrsprocessor/internal/metadata"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processorhelper"
	"k8s.io/client-go/rest"
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

	// Get in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get in-cluster config: %w", err)
	}

	proc, err := newServiceNameProcessor(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create service name processor: %w", err)
	}

	// Start the informer
	if err := proc.start(ctx); err != nil {
		return nil, fmt.Errorf("failed to start pod metadata informer: %w", err)
	}

	return processorhelper.NewLogs(
		ctx,
		set,
		cfg,
		nextConsumer,
		proc.processLogs,
		processorhelper.WithCapabilities(consumer.Capabilities{MutatesData: true}),
	)
}
