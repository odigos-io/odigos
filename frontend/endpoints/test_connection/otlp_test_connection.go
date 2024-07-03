package testconnection

import (
	"context"
	"fmt"
	"strings"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/pdata/ptrace"

	"go.opentelemetry.io/collector/exporter/otlpexporter"
)

var configres map[common.DestinationType]config.Configer

func init() {
	var err error
	configres, err = config.LoadConfigers()
	if err != nil {
		panic(1)
	}
}

func TestOTLPConnection(ctx context.Context, dest config.ExporterConfigurer) error {
	destType := dest.GetType()
	configer, ok := configres[destType]
	if !ok {
		return fmt.Errorf("destination type %s not found", destType)
	}

	currentConfig := config.Config{
		Exporters: make(config.GenericMap),
		Service: config.Service{
			Pipelines: make(map[string]config.Pipeline),
		},
	}
	err := configer.ModifyConfig(dest, &currentConfig)
	if err != nil {
		return err
	}

	exporters := currentConfig.Exporters
	if len(exporters) == 0 {
		return fmt.Errorf("no exporters found in config")
	}

	var otlpExporterConfig config.GenericMap
	foundOTLP := false
	for componentID, cfg := range exporters {
		gm, ok := cfg.(config.GenericMap)
		if !ok {
			continue
		}
		if strings.HasPrefix(componentID, "otlp/") {
			otlpExporterConfig = gm
			foundOTLP = true
			break
		}
	}

	if !foundOTLP {
		return fmt.Errorf("no OTLP exporter found in config")
	}

	factory := otlpexporter.NewFactory()
	otlpConf, ok  := factory.CreateDefaultConfig().(*otlpexporter.Config)
	if !ok {
		return fmt.Errorf("failed to create default config")
	}
	// currently using the default timeout config of the collector - 5 seconds
	// Avoid batching and retries
	otlpConf.QueueConfig.Enabled = false
	otlpConf.RetryConfig.Enabled = false

	// convert the user provided fields to a collector config
	exportersConf := confmap.NewFromStringMap(otlpExporterConfig)
	if exportersConf == nil {
		return fmt.Errorf("failed to create exporter config")
	}

	err = exportersConf.Unmarshal(&otlpConf)
	if err != nil {
		return fmt.Errorf("failed to unmarshal exporter config: %w", err)
	}

	err = otlpConf.Validate()
	if err != nil {
		return fmt.Errorf("failed to validate exporter config: %w", err)
	}

    exporter, err := factory.CreateTracesExporter(ctx, exportertest.NewNopCreateSettings(), otlpConf)
	if err != nil {
		return fmt.Errorf("failed to create exporter: %w", err)
	}

	err = exporter.Start(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to start exporter: %w", err)
	}

	defer exporter.Shutdown(ctx)
	err = exporter.ConsumeTraces(ctx, ptrace.NewTraces())
	if err != nil {
		return fmt.Errorf("failed to consume traces by exporter: %w", err)
	}

	return nil
}
