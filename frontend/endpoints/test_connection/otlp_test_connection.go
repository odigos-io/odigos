package testconnection

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter"

	"go.opentelemetry.io/collector/exporter/otlpexporter"
)

var _ ExporterConnectionTester = &otlpExporterConnectionTester{}

type otlpExporterConnectionTester struct {
	f exporter.Factory
}

func NewOTLPTester() *otlpExporterConnectionTester {
	return &otlpExporterConnectionTester{
		f: otlpexporter.NewFactory(),
	}
}

func (t *otlpExporterConnectionTester) Factory() exporter.Factory {
	return t.f
}

func (t *otlpExporterConnectionTester) ModifyConfigForConnectionTest(cfg component.Config) component.Config {
	otlpConf, ok := cfg.(*otlpexporter.Config)
	if !ok {
		return nil
	}

	// currently using the default timeout config of the collector - 5 seconds
	// Avoid batching and retries
	otlpConf.QueueConfig.Enabled = false
	otlpConf.RetryConfig.Enabled = false
	return otlpConf
}
