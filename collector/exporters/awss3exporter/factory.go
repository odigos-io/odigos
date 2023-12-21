package awss3exporter

import (
	"context"
	"go.opentelemetry.io/collector/exporter"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

const (
	// The value of "type" key in configuration.
	typeStr = "s3"
)

// NewFactory creates a factory for S3 exporter.
func NewFactory() exporter.Factory {
	return exporter.NewFactory(
		typeStr,
		createDefaultConfig,
		exporter.WithLogs(createLogsExporter, component.StabilityLevelBeta),
		exporter.WithTraces(createTracesExporter, component.StabilityLevelBeta))
}

func createDefaultConfig() component.Config {
	return &Config{
		AWSS3UploadConfig: AWSS3UploadConfig{
			S3Partition: "minute",
		},

		MarshalerName: "otlp_json",
		logger:        nil,
	}
}

func createLogsExporter(
	ctx context.Context,
	set exporter.CreateSettings,
	cfg component.Config) (exporter.Logs, error) {

	pCfg := cfg.(*Config)
	s3Exporter, err := NewS3Exporter(pCfg, set)
	if err != nil {
		return nil, err
	}

	return exporterhelper.NewLogsExporter(
		ctx,
		set,
		cfg,
		s3Exporter.ConsumeLogs)
}

func createTracesExporter(
	ctx context.Context,
	set exporter.CreateSettings,
	cfg component.Config) (exporter.Traces, error) {

	pCfg := cfg.(*Config)
	s3Exporter, err := NewS3Exporter(pCfg, set)
	if err != nil {
		return nil, err
	}

	return exporterhelper.NewTracesExporter(
		ctx,
		set,
		cfg,
		s3Exporter.ConsumeTraces,
		exporterhelper.WithStart(func(ctx context.Context, host component.Host) error {
			pCfg.logger.Info("Starting S3 exporter")
			return nil
		}))
}
