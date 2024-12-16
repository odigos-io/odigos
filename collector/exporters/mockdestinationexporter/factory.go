package mockdestinationexporter

import (
	"context"
	"time"

	"github.com/open-telemetry/opentelemetry-collector-contrib/odigos/exporter/mockdestinationexporter/internal/metadata"
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
	gcsExporter, err := NewMockDestinationExporter(pCfg, set)
	if err != nil {
		return nil, err
	}

	return exporterhelper.NewTracesExporter(
		ctx,
		set,
		cfg,
		gcsExporter.ConsumeTraces,
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

	return exporterhelper.NewMetricsExporter(
		ctx,
		set,
		cfg,
		gcsExporter.ConsumeMetrics,
	)
}
