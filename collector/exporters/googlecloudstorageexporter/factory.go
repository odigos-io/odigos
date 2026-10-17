package googlecloudstorageexporter

import (
	"context"

	"github.com/open-telemetry/opentelemetry-collector-contrib/odigos/exporter/googlecloudstorageexporter/internal/metadata"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/config/configretry"
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
		exporter.WithTraces(createTracesExporter, component.StabilityLevelBeta))
}

func createDefaultConfig() component.Config {
	return &Config{
		GCSUploader: GCSUploadConfig{
			GCSPartition: "minute",
		},

		MarshalerName: "otlp_json",
		TimeoutConfig: exporterhelper.NewDefaultTimeoutConfig(),
		RetryConfig:   configretry.NewDefaultBackOffConfig(),
		QueueConfig:   configoptional.Some(exporterhelper.NewDefaultQueueConfig()),
	}
}

func createLogsExporter(
	ctx context.Context,
	set exporter.Settings,
	cfg component.Config) (exporter.Logs, error) {

	pCfg := cfg.(*Config)
	gcsExporter, err := NewGCSExporter(pCfg, set)
	if err != nil {
		return nil, err
	}

	return exporterhelper.NewLogs(
		ctx,
		set,
		cfg,
		gcsExporter.ConsumeLogs,
		exporterhelper.WithTimeout(pCfg.TimeoutConfig),
		exporterhelper.WithRetry(pCfg.RetryConfig),
		exporterhelper.WithQueue(pCfg.QueueConfig),
		exporterhelper.WithStart(gcsExporter.Start),
		exporterhelper.WithShutdown(gcsExporter.Shutdown))
}

func createTracesExporter(
	ctx context.Context,
	set exporter.Settings,
	cfg component.Config) (exporter.Traces, error) {

	pCfg := cfg.(*Config)
	gcsExporter, err := NewGCSExporter(pCfg, set)
	if err != nil {
		return nil, err
	}

	return exporterhelper.NewTraces(
		ctx,
		set,
		cfg,
		gcsExporter.ConsumeTraces,
		exporterhelper.WithTimeout(pCfg.TimeoutConfig),
		exporterhelper.WithRetry(pCfg.RetryConfig),
		exporterhelper.WithQueue(pCfg.QueueConfig),
		exporterhelper.WithStart(gcsExporter.Start),
		exporterhelper.WithShutdown(gcsExporter.Shutdown))
}
