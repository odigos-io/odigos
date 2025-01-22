package testconnection

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

var (
	configres         map[common.DestinationType]config.Configer
	connectionTesters = []ExporterConnectionTester{
		NewOTLPTester(),     // "otlp/" prefix
		NewOTLPHTTPTester(), // "otlphttp/" prefix
	}
)

func init() {
	var err error
	configres, err = config.LoadConfigers()
	if err != nil {
		panic(1)
	}
}

type TestConnectionErrorReason string

const (
	UnKnownDestination      TestConnectionErrorReason = "unknown destination"
	InvalidConfig           TestConnectionErrorReason = "invalid config"
	UnsupportedExporterType TestConnectionErrorReason = "unsupported exporter type"
	FailedToConnect         TestConnectionErrorReason = "failed to connect"
)

type TestConnectionResult struct {
	Succeeded       bool
	Message         string
	Reason          TestConnectionErrorReason
	StatusCode      int
	DestinationType common.DestinationType
}

type ExporterConnectionTester interface {
	// Factory returns the exporter factory for the exporter type.
	// This is used to create the exporter instance for testing the connection.
	Factory() exporter.Factory
	// ModifyConfigForConnectionTest modifies the exporter configuration for testing the connection.
	// Since the default configuration may have batching, retries, etc. which may not be suitable for testing the connection.
	ModifyConfigForConnectionTest(component.Config) component.Config
}

func getConnectionTester(exporterID string) ExporterConnectionTester {
	for _, tester := range connectionTesters {
		prefix := fmt.Sprintf("%s/", tester.Factory().Type().String())
		if strings.HasPrefix(exporterID, prefix) {
			return tester
		}
	}
	return nil
}

func TestConnection(ctx context.Context, dest config.ExporterConfigurer) TestConnectionResult {
	destType := dest.GetType()
	configer, ok := configres[destType]
	if !ok {
		return TestConnectionResult{Succeeded: false, Reason: UnKnownDestination, DestinationType: destType, StatusCode: http.StatusNotImplemented}
	}

	currentConfig := config.Config{
		Exporters: make(config.GenericMap),
		Service: config.Service{
			Pipelines: make(map[string]config.Pipeline),
		},
	}
	_, err := configer.ModifyConfig(dest, &currentConfig)
	if err != nil {
		return TestConnectionResult{Succeeded: false, Message: err.Error(), Reason: InvalidConfig, DestinationType: destType, StatusCode: http.StatusInternalServerError}
	}

	exporters := currentConfig.Exporters
	if len(exporters) == 0 {
		return TestConnectionResult{Message: "no exporters found in config", Reason: InvalidConfig, DestinationType: destType, StatusCode: http.StatusInternalServerError, Succeeded: false}
	}

	var exporterRawConfig config.GenericMap
	var connectionTester ExporterConnectionTester
	foundTester := false
	for componentID, cfg := range exporters {
		gm, ok := cfg.(config.GenericMap)
		if !ok {
			continue
		}
		ct := getConnectionTester(componentID)
		if ct != nil {
			connectionTester = ct
			foundTester = true
			exporterRawConfig = gm
			break
		}
	}

	if !foundTester {
		return TestConnectionResult{Succeeded: false, Message: "no supported exporter found in config", Reason: UnsupportedExporterType, DestinationType: destType, StatusCode: http.StatusNotFound}
	}

	// before testing the connection, replace placeholders (if exists) in the config with actual values
	replacePlaceholders(exporterRawConfig, dest.GetConfig())
	defaultConfig := connectionTester.Factory().CreateDefaultConfig()
	connectionTester.ModifyConfigForConnectionTest(defaultConfig)

	// convert the user provided fields to a collector config
	exportersConf := confmap.NewFromStringMap(exporterRawConfig)
	if exportersConf == nil {
		return TestConnectionResult{Succeeded: false, Message: "failed to create exporter config", Reason: InvalidConfig, DestinationType: destType, StatusCode: http.StatusInternalServerError}
	}

	// unmarshal the user provided configuration into the default one, merging them
	err = exportersConf.Unmarshal(&defaultConfig)
	if err != nil {
		return TestConnectionResult{Succeeded: false, Message: err.Error(), Reason: InvalidConfig, DestinationType: destType, StatusCode: http.StatusInternalServerError}
	}

	if validator, ok := defaultConfig.(component.ConfigValidator); ok {
		// if the component has a Validate method, call it to validate the configuration
		err = validator.Validate()
		if err != nil {
			return TestConnectionResult{Succeeded: false, Message: err.Error(), Reason: InvalidConfig, DestinationType: destType, StatusCode: http.StatusInternalServerError}
		}
	}

	exporter, err := connectionTester.Factory().CreateTracesExporter(ctx, exportertest.NewNopSettings(), defaultConfig)
	if err != nil {
		return TestConnectionResult{Succeeded: false, Message: err.Error(), Reason: InvalidConfig, DestinationType: destType, StatusCode: http.StatusInternalServerError}
	}

	err = exporter.Start(ctx, nil)
	if err != nil {
		return TestConnectionResult{Succeeded: false, Message: err.Error(), Reason: FailedToConnect, DestinationType: destType, StatusCode: http.StatusInternalServerError}
	}

	defer exporter.Shutdown(ctx)
	err = exporter.ConsumeTraces(ctx, ptrace.NewTraces())
	if err != nil {
		return TestConnectionResult{Succeeded: false, Message: err.Error(), Reason: FailedToConnect, DestinationType: destType, StatusCode: http.StatusInternalServerError}
	}

	return TestConnectionResult{Succeeded: true, DestinationType: destType, StatusCode: http.StatusOK}
}
