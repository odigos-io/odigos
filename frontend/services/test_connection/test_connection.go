package testconnection

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/xconfmap"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/pdata/ptrace"

	"google.golang.org/grpc/status"
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

func TestConnectionHoneycomb(ctx context.Context, dest config.ExporterConfigurer) TestConnectionResult {
	// make an http request to the honeycomb api
	// to check if the api key is valid
	// request like 	curl -i -X GET https://api.honeycomb.io/1/auth -H 'X-Honeycomb-Team: YOUR_API_KEY_HERE'

	client := &http.Client{}

	honeycombEndpoint, found := dest.GetConfig()["HONEYCOMB_ENDPOINT"]
	if !found {
		return TestConnectionResult{Succeeded: false, Message: "HONEYCOMB_ENDPOINT not found in config", Reason: FailedToConnect, DestinationType: dest.GetType(), StatusCode: http.StatusInternalServerError}
	}
	authEndpoint := fmt.Sprintf("https://%s/1/auth", honeycombEndpoint)

	req, err := http.NewRequestWithContext(ctx, "GET", authEndpoint, nil)
	if err != nil {
		return TestConnectionResult{Succeeded: false, Message: handleError(err), Reason: FailedToConnect, DestinationType: dest.GetType(), StatusCode: http.StatusInternalServerError}
	}

	apiKey, found := dest.GetConfig()["HONEYCOMB_API_KEY"]
	if !found {
		return TestConnectionResult{Succeeded: false, Message: "HONEYCOMB_API_KEY not found in config", Reason: FailedToConnect, DestinationType: dest.GetType(), StatusCode: http.StatusInternalServerError}
	}

	req.Header.Add("X-Honeycomb-Team", apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return TestConnectionResult{Succeeded: false, Message: handleError(err), Reason: FailedToConnect, DestinationType: dest.GetType(), StatusCode: http.StatusInternalServerError}
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return TestConnectionResult{Succeeded: false, Message: "failed to connect to honeycomb api", Reason: FailedToConnect, DestinationType: dest.GetType(), StatusCode: resp.StatusCode}
	}

	return TestConnectionResult{Succeeded: true, DestinationType: dest.GetType(), StatusCode: http.StatusOK}
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
		return TestConnectionResult{Succeeded: false, Message: handleError(err), Reason: InvalidConfig, DestinationType: destType, StatusCode: http.StatusInternalServerError}
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
	factory := connectionTester.Factory()
	if factory == nil {
		return TestConnectionResult{Succeeded: false, Message: "failed to create exporter factory", Reason: InvalidConfig, DestinationType: destType, StatusCode: http.StatusInternalServerError}
	}

	defaultConfig := factory.CreateDefaultConfig()
	connectionTester.ModifyConfigForConnectionTest(defaultConfig)

	// convert the user provided fields to a collector config.
	// normalizeMap converts named map types (like config.GenericMap) to plain map[string]any,
	// which is required for confmap's decoder hooks to properly handle type assertions.
	exportersConf := confmap.NewFromStringMap(normalizeMap(exporterRawConfig))
	if exportersConf == nil {
		return TestConnectionResult{Succeeded: false, Message: "failed to create exporter config", Reason: InvalidConfig, DestinationType: destType, StatusCode: http.StatusInternalServerError}
	}

	// unmarshal the user provided configuration into the default one, merging them
	err = exportersConf.Unmarshal(&defaultConfig)
	if err != nil {
		return TestConnectionResult{Succeeded: false, Message: handleError(err), Reason: InvalidConfig, DestinationType: destType, StatusCode: http.StatusInternalServerError}
	}

	if validator, ok := defaultConfig.(xconfmap.Validator); ok {
		// if the component has a Validate method, call it to validate the configuration
		err = validator.Validate()
		if err != nil {
			return TestConnectionResult{Succeeded: false, Message: handleError(err), Reason: InvalidConfig, DestinationType: destType, StatusCode: http.StatusInternalServerError}
		}
	}

	exporter, err := factory.CreateTraces(ctx, exportertest.NewNopSettings(factory.Type()), defaultConfig)
	if err != nil {
		return TestConnectionResult{Succeeded: false, Message: handleError(err), Reason: InvalidConfig, DestinationType: destType, StatusCode: http.StatusInternalServerError}
	}

	err = exporter.Start(ctx, componenttest.NewNopHost())
	if err != nil {
		return TestConnectionResult{Succeeded: false, Message: handleError(err), Reason: FailedToConnect, DestinationType: destType, StatusCode: http.StatusInternalServerError}
	}

	defer exporter.Shutdown(ctx)
	err = exporter.ConsumeTraces(ctx, ptrace.NewTraces())
	if err != nil {
		return TestConnectionResult{Succeeded: false, Message: handleError(err), Reason: FailedToConnect, DestinationType: destType, StatusCode: http.StatusInternalServerError}
	}

	return TestConnectionResult{Succeeded: true, DestinationType: destType, StatusCode: http.StatusOK}
}

func handleError(err error) string {
	msg := ""

	if s, ok := status.FromError(err); ok {
		msg = s.Message()
	} else {
		// Not a gRPC status error
		msg = err.Error()
	}

	return msg
}
