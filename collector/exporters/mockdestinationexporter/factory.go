package mockdestinationexporter

import (
	"context"
	"time"

	"github.com/open-telemetry/opentelemetry-collector-contrib/odigos/exporter/mockdestinationexporter/internal/metadata"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

// NewFactory creates a factory for GCS exporter.
func NewFactory() exporter.Factory {
	return exporter.NewFactory(
		metadata.Type,
		createDefaultConfig,
		exporter.WithLogs(createLogsExporter, component.StabilityLevelBeta),
		exporter.WithTraces(createTracesExporter, component.StabilityLevelBeta),
		exporter.WithMetrics(createMetricsExporter, component.StabilityLevelBeta))
}

func createDefaultConfig() component.Config {
	return &Config{
		ResponseDuration: time.Millisecond * 100,
		RejectFraction:   0,
		TimeoutConfig:    exporterhelper.NewDefaultTimeoutConfig(),
		RetryConfig:      configretry.NewDefaultBackOffConfig(),
		QueueConfig:      exporterhelper.NewDefaultQueueConfig(),
	}
}

func createLogsExporter(
	ctx context.Context,
	set exporter.Settings,
	cfg component.Config) (exporter.Logs, error) {

	pCfg := cfg.(*Config)
	gcsExporter, err := NewMockDestinationExporter(pCfg, set)
	if err != nil {
		return nil, err
	}

	return exporterhelper.NewLogs(
		ctx,
		set,
		cfg,
		gcsExporter.ConsumeLogs,
		exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: false}),
		exporterhelper.WithTimeout(pCfg.TimeoutConfig),
		exporterhelper.WithRetry(pCfg.RetryConfig),
		exporterhelper.WithQueueBatch(pCfg.QueueConfig, exporterhelper.NewLogsQueueBatchSettings()),
	)
}

func createTracesExporter(
	ctx context.Context,
	set exporter.Settings,
	cfg component.Config) (exporter.Traces, error) {

	pCfg := cfg.(*Config)
	gcsExporter, err := NewMockDestinationExporter(pCfg, set)
	if err != nil {
		return nil, err
	}

	return exporterhelper.NewTraces(
		ctx,
		set,
		cfg,
		gcsExporter.ConsumeTraces,
		exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: false}),
		exporterhelper.WithTimeout(pCfg.TimeoutConfig),
		exporterhelper.WithRetry(pCfg.RetryConfig),
		exporterhelper.WithQueueBatch(pCfg.QueueConfig, exporterhelper.NewTracesQueueBatchSettings()),
	)
}

func createMetricsExporter(
	ctx context.Context,
	set exporter.Settings,
	cfg component.Config) (exporter.Metrics, error) {

	pCfg := cfg.(*Config)
	gcsExporter, err := NewMockDestinationExporter(pCfg, set)
	if err != nil {
		return nil, err
	}

	return exporterhelper.NewMetrics(
		ctx,
		set,
		cfg,
		gcsExporter.ConsumeMetrics,
		exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: false}),
		exporterhelper.WithTimeout(pCfg.TimeoutConfig),
		exporterhelper.WithRetry(pCfg.RetryConfig),
		exporterhelper.WithQueueBatch(pCfg.QueueConfig, exporterhelper.NewMetricsQueueBatchSettings()),
	)
}
