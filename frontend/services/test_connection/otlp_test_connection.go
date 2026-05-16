package testconnection

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
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
	// Avoid batching and retries (QueueBatchConfig zero value has enabled: false)
	otlpConf.QueueConfig = configoptional.None[exporterhelper.QueueBatchConfig]()
	otlpConf.RetryConfig.Enabled = false
	return otlpConf
}
