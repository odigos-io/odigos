package otlpproxygrpcexporter

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

// componentType is the config key for this exporter (e.g. otlpproxygrpc/dynatrace).
var componentType = component.MustNewType("otlpproxygrpc")

// NewFactory returns the factory for the OTLP/gRPC-over-proxy exporter.
func NewFactory() exporter.Factory {
	return exporter.NewFactory(
		componentType,
		createDefaultConfig,
		exporter.WithTraces(createTraces, component.StabilityLevelBeta),
		exporter.WithMetrics(createMetrics, component.StabilityLevelBeta),
		exporter.WithLogs(createLogs, component.StabilityLevelBeta),
	)
}

func createDefaultConfig() component.Config {
	return &Config{
		TimeoutConfig: exporterhelper.NewDefaultTimeoutConfig(),
		RetryConfig:   configretry.NewDefaultBackOffConfig(),
		QueueConfig:   configoptional.Some(exporterhelper.NewDefaultQueueConfig()),
		ClientConfig:  configgrpc.ClientConfig{},
	}
}

func createTraces(ctx context.Context, set exporter.Settings, cfg component.Config) (exporter.Traces, error) {
	c := cfg.(*Config)
	e := newExporter(c, set)
	return exporterhelper.NewTraces(ctx, set, cfg, e.pushTraces,
		exporterhelper.WithStart(e.start),
		exporterhelper.WithShutdown(e.shutdown),
		exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: false}),
		exporterhelper.WithTimeout(c.TimeoutConfig),
		exporterhelper.WithRetry(c.RetryConfig),
		exporterhelper.WithQueue(c.QueueConfig),
	)
}

func createMetrics(ctx context.Context, set exporter.Settings, cfg component.Config) (exporter.Metrics, error) {
	c := cfg.(*Config)
	e := newExporter(c, set)
	return exporterhelper.NewMetrics(ctx, set, cfg, e.pushMetrics,
		exporterhelper.WithStart(e.start),
		exporterhelper.WithShutdown(e.shutdown),
		exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: false}),
		exporterhelper.WithTimeout(c.TimeoutConfig),
		exporterhelper.WithRetry(c.RetryConfig),
		exporterhelper.WithQueue(c.QueueConfig),
	)
}

func createLogs(ctx context.Context, set exporter.Settings, cfg component.Config) (exporter.Logs, error) {
	c := cfg.(*Config)
	e := newExporter(c, set)
	return exporterhelper.NewLogs(ctx, set, cfg, e.pushLogs,
		exporterhelper.WithStart(e.start),
		exporterhelper.WithShutdown(e.shutdown),
		exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: false}),
		exporterhelper.WithTimeout(c.TimeoutConfig),
		exporterhelper.WithRetry(c.RetryConfig),
		exporterhelper.WithQueue(c.QueueConfig),
	)
}
