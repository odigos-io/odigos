package testconnection

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
)

var configers map[common.DestinationType]config.Configer

func init() {
	var err error
	configers, err = config.LoadConfigers()
	if err != nil {
		panic(err)
	}
}

type TestConnectionErrorReason string

const (
	UnknownDestination      TestConnectionErrorReason = "unknown destination"
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

type ExportAttempt struct {
	Succeeded  bool
	Reason     TestConnectionErrorReason
	StatusCode int
	Message    string
}

// An interface to prevent importing the collector - dependency inverts and delegates the implementation to the caller modules (OSS/VMAgent)
type ExporterConnectionTester interface {
	// Prefix is the exporter "type" segment (before the "/") that this tester knows how to test,
	// e.g. "otlp" matches generated exporter ids like "otlp/my-destination".
	Prefix() string
	// builds the exporter from the destination's rawConfig, connects, and exports an empty payload.
	TestExport(ctx context.Context, rawConfig map[string]any) ExportAttempt
}

// builds the destination's exporter config and connects using the tester whose Prefix matches the generated exporter id
func TestConnection(ctx context.Context, dest config.ExporterConfigurer, testers []ExporterConnectionTester) TestConnectionResult {
	destType := dest.GetType()

	// Honeycomb is a special case:
	// rejects the empty test payload, so verify the API key against its auth endpoint instead
	// this still hasn't been patched with Honeycomb (correct as of 31-05-26)
	if destType == common.HoneycombDestinationType {
		return testConnectionHoneycomb(ctx, dest)
	}

	configer, ok := configers[destType]
	if !ok {
		return failResult(destType, UnknownDestination, http.StatusNotImplemented, "")
	}

	currentConfig := config.Config{
		Exporters: make(config.GenericMap),
		Service: config.Service{
			Pipelines: make(map[string]config.Pipeline),
		},
	}
	if _, err := configer.ModifyConfig(dest, &currentConfig); err != nil {
		return failResult(destType, InvalidConfig, http.StatusInternalServerError, err.Error())
	}
	if len(currentConfig.Exporters) == 0 {
		return failResult(destType, InvalidConfig, http.StatusInternalServerError, "no exporters found in config")
	}

	var exporterRawConfig config.GenericMap
	var connectionTester ExporterConnectionTester
	for componentID, cfg := range currentConfig.Exporters {
		gm, ok := cfg.(config.GenericMap)
		if !ok {
			continue
		}
		if t := getConnectionTester(testers, componentID); t != nil {
			connectionTester, exporterRawConfig = t, gm
			break
		}
	}
	if connectionTester == nil {
		return failResult(destType, UnsupportedExporterType, http.StatusNotFound, "no supported exporter found in config")
	}

	// resolve ${KEY} placeholders, then flatten named maps so confmap's decoder hooks work downstream
	replacePlaceholders(exporterRawConfig, dest.GetConfig())
	connectionAttempt := connectionTester.TestExport(ctx, normalizeMap(exporterRawConfig))
	if !connectionAttempt.Succeeded {
		return failResult(destType, connectionAttempt.Reason, connectionAttempt.StatusCode, connectionAttempt.Message)
	}

	return TestConnectionResult{Succeeded: true, DestinationType: destType, StatusCode: http.StatusOK}
}

func getConnectionTester(testers []ExporterConnectionTester, exporterID string) ExporterConnectionTester {
	for _, tester := range testers {
		if strings.HasPrefix(exporterID, tester.Prefix()+"/") {
			return tester
		}
	}
	return nil
}

// Special case for testing Honeycomb
func testConnectionHoneycomb(ctx context.Context, dest config.ExporterConfigurer) TestConnectionResult {
	destType := dest.GetType()

	honeycombEndpoint, found := dest.GetConfig()["HONEYCOMB_ENDPOINT"]
	if !found {
		return failResult(destType, FailedToConnect, http.StatusInternalServerError, "HONEYCOMB_ENDPOINT not found in config")
	}
	apiKey, found := dest.GetConfig()["HONEYCOMB_API_KEY"]
	if !found {
		return failResult(destType, FailedToConnect, http.StatusInternalServerError, "HONEYCOMB_API_KEY not found in config")
	}

	authEndpoint := fmt.Sprintf("https://%s/1/auth", honeycombEndpoint)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, authEndpoint, http.NoBody)
	if err != nil {
		return failResult(destType, FailedToConnect, http.StatusInternalServerError, err.Error())
	}
	req.Header.Add("X-Honeycomb-Team", apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return failResult(destType, FailedToConnect, http.StatusInternalServerError, err.Error())
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return failResult(destType, FailedToConnect, resp.StatusCode, "failed to connect to honeycomb api")
	}

	return TestConnectionResult{Succeeded: true, DestinationType: destType, StatusCode: http.StatusOK}
}

func failResult(destType common.DestinationType, reason TestConnectionErrorReason, statusCode int, message string) TestConnectionResult {
	return TestConnectionResult{
		Succeeded:       false,
		Reason:          reason,
		StatusCode:      statusCode,
		Message:         message,
		DestinationType: destType,
	}
}
