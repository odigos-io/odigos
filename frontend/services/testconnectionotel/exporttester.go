// Package testconnectionotel implements the destinations/testconnection.ExporterConnectionTester port using
// the OTel collector exporters.
package testconnectionotel

import (
	"context"
	"net/http"

	"github.com/odigos-io/odigos/destinations/testconnection"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/xconfmap"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/pdata/ptrace"

	"google.golang.org/grpc/status"
)

var testers = []testconnection.ExporterConnectionTester{NewOTLPTester(), NewOTLPHTTPTester()}

func Testers() []testconnection.ExporterConnectionTester {
	return testers
}

// configModifier strips batching/retries unsuitable for a single-shot connection test.
type configModifier func(component.Config) component.Config

// runExport builds the exporter from rawConfig and exports an empty trace payload, mapping config
// problems to InvalidConfig and connectivity problems to FailedToConnect.
func runExport(ctx context.Context, factory exporter.Factory, modify configModifier, rawConfig map[string]any) testconnection.ExportAttempt {
	defaultConfig := factory.CreateDefaultConfig()
	modify(defaultConfig)

	exportersConf := confmap.NewFromStringMap(rawConfig)
	if err := exportersConf.Unmarshal(&defaultConfig); err != nil {
		return invalidConfig(handleError(err))
	}
	if validator, ok := defaultConfig.(xconfmap.Validator); ok {
		if err := validator.Validate(); err != nil {
			return invalidConfig(handleError(err))
		}
	}

	exp, err := factory.CreateTraces(ctx, exportertest.NewNopSettings(factory.Type()), defaultConfig)
	if err != nil {
		return invalidConfig(handleError(err))
	}
	if err := exp.Start(ctx, componenttest.NewNopHost()); err != nil {
		return failedToConnect(handleError(err))
	}
	defer exp.Shutdown(ctx)
	if err := exp.ConsumeTraces(ctx, ptrace.NewTraces()); err != nil {
		return failedToConnect(handleError(err))
	}

	return testconnection.ExportAttempt{Succeeded: true, StatusCode: http.StatusOK}
}

func invalidConfig(message string) testconnection.ExportAttempt {
	return testconnection.ExportAttempt{
		Reason:     testconnection.InvalidConfig,
		StatusCode: http.StatusInternalServerError,
		Message:    message,
	}
}

func failedToConnect(message string) testconnection.ExportAttempt {
	return testconnection.ExportAttempt{
		Reason:     testconnection.FailedToConnect,
		StatusCode: http.StatusInternalServerError,
		Message:    message,
	}
}

func handleError(err error) string {
	if s, ok := status.FromError(err); ok {
		return s.Message()
	}
	// not a gRPC status error
	return err.Error()
}
