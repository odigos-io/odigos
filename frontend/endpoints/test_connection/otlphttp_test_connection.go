package testconnection

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/otlphttpexporter"
)

var _ ExporterConnectionTester = &otlphttpExporterConnectionTester{}

type otlphttpExporterConnectionTester struct {
	f exporter.Factory
}

func NewOTLPHTTPTester() *otlphttpExporterConnectionTester {
	return &otlphttpExporterConnectionTester{
		f: otlphttpexporter.NewFactory(),
	}
}

func (t *otlphttpExporterConnectionTester) Factory() exporter.Factory {
	return t.f
}

func (t *otlphttpExporterConnectionTester) ModifyConfigForConnectionTest(cfg component.Config) component.Config {
	otlpConf, ok := cfg.(*otlphttpexporter.Config)
	if !ok {
		return nil
	}

	// currently using the default timeout config of the collector - 5 seconds
	// Avoid batching and retries
	otlpConf.QueueConfig.Enabled = false
	otlpConf.RetryConfig.Enabled = false
	return otlpConf
}
