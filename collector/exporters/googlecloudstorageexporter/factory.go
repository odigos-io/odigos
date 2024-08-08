package googlecloudstorageexporter

import (
	"context"

	"github.com/open-telemetry/opentelemetry-collector-contrib/odigos/exporter/googlecloudstorageexporter/internal/metadata"
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
		logger:        nil,
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

	return exporterhelper.NewLogsExporter(
		ctx,
		set,
		cfg,
		gcsExporter.ConsumeLogs)
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

	return exporterhelper.NewTracesExporter(
		ctx,
		set,
		cfg,
		gcsExporter.ConsumeTraces,
		exporterhelper.WithStart(func(ctx context.Context, host component.Host) error {
			pCfg.logger.Info("Starting GCS exporter")
			return nil
		}))
}
