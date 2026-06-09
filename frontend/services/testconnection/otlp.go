package testconnection

import (
	"context"

	"github.com/odigos-io/odigos/common/config/testconnection"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/exporter/otlpexporter"
)

var _ testconnection.ExporterConnectionTester = (*otlpExporterConnectionTester)(nil)

type otlpExporterConnectionTester struct {
	factory exporter.Factory
}

func NewOTLPTester() *otlpExporterConnectionTester {
	return &otlpExporterConnectionTester{factory: otlpexporter.NewFactory()}
}

func (t *otlpExporterConnectionTester) Prefix() string {
	return t.factory.Type().String()
}

func (t *otlpExporterConnectionTester) TestExport(ctx context.Context, rawConfig map[string]any) testconnection.ExportAttempt {
	return runExport(ctx, t.factory, modifyOTLPConfig, rawConfig)
}

func modifyOTLPConfig(cfg component.Config) component.Config {
	otlpConfig, ok := cfg.(*otlpexporter.Config)
	if !ok {
		return nil
	}

	// keep the collector's default 5s timeout; disable batching and retries
	otlpConfig.QueueConfig = configoptional.None[exporterhelper.QueueBatchConfig]()
	otlpConfig.RetryConfig.Enabled = false
	return otlpConfig
}
