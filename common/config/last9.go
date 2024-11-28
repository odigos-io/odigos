package config

import (
	"errors"

	"github.com/odigos-io/odigos/common"
)

const (
	l9OtlpEndpointKey   = "LAST9_OTLP_ENDPOINT"
	l9OtlpAuthHeaderKey = "LAST9_OTLP_BASIC_AUTH_HEADER"
)

type MyDest struct{}

func (m *MyDest) DestType() common.DestinationType {
	// DestinationType defined in common/dests.go
	return common.Last9DestinationType
}

func (m *MyDest) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) error {
	config := dest.GetConfig()
	l9OtlpEndpoint, exists := config[l9OtlpEndpointKey]
	if !exists {
		return errors.New("Last9 OpenTelemetry Endpoint key(\"LAST9_OTLP_ENDPOINT\") not specified, Last9 will not be configured")
	}

	// to make sure that the exporter name is unique, we'll ask a ID from destination
	exporterName := "otlp/last9-" + dest.GetID()
	currentConfig.Exporters["otlp/last9"] = GenericMap{
		"endpoint": l9OtlpEndpoint,
		"headers": GenericMap{
			"Authorization": "${LAST9_OTLP_BASIC_AUTH_HEADER}",
		},
	}

	// Modify the config here
	if isTracingEnabled(dest) {
		tracesPipelineName := "traces/last9-" + dest.GetID()
		currentConfig.Service.Pipelines[tracesPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
	}

	if isMetricsEnabled(dest) {
		metricsPipelineName := "metrics/last9-" + dest.GetID()
		currentConfig.Service.Pipelines[metricsPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
	}

	if isLoggingEnabled(dest) {
		logsPipelineName := "logs/last9-" + dest.GetID()
		currentConfig.Service.Pipelines[logsPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
	}

	return nil
}
