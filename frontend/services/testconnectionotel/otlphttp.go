package testconnectionotel

import (
	"context"

	"github.com/odigos-io/odigos/common/config/testconnection"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/exporter/otlphttpexporter"
)

var _ testconnection.ExporterConnectionTester = (*otlphttpExporterConnectionTester)(nil)

type otlphttpExporterConnectionTester struct {
	factory exporter.Factory
}

func NewOTLPHTTPTester() *otlphttpExporterConnectionTester {
	return &otlphttpExporterConnectionTester{factory: otlphttpexporter.NewFactory()}
}

func (t *otlphttpExporterConnectionTester) Prefix() string {
	return t.factory.Type().String()
}

func (t *otlphttpExporterConnectionTester) TestExport(ctx context.Context, rawConfig map[string]any) testconnection.ExportAttempt {
	return runExport(ctx, t.factory, modifyOTLPHTTPConfig, rawConfig)
}

func modifyOTLPHTTPConfig(cfg component.Config) component.Config {
	otlphttpConfig, ok := cfg.(*otlphttpexporter.Config)
	if !ok {
		return nil
	}

	// keep the collector's default 5s timeout; disable batching and retries
	otlphttpConfig.QueueConfig = configoptional.None[exporterhelper.QueueBatchConfig]()
	otlphttpConfig.RetryConfig.Enabled = false
	return otlphttpConfig
}
