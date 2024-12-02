package config

import (
	"errors"

	"github.com/odigos-io/odigos/common"
)

type AppDynamics struct{}

func (m *AppDynamics) DestType() common.DestinationType {
	// DestinationType defined in common/dests.go
	return common.AppDynamicsDestinationType
}

func (m *AppDynamics) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) error {
	config := dest.GetConfig()
	endpoint, exists := config["${APPDYNAMICS_ENDPOINT}"]
	if !exists {
		return errors.New("AppDynamics OpenTelemetry Endpoint key(\"APPDYNAMICS_ENDPOINT\") not specified, AppDynamics will not be configured")
	}

	// to make sure that the exporter name is unique, we'll ask a ID from destination
	exporterName := "otlp/appdynamics-" + dest.GetID()
	currentConfig.Exporters[exporterName] = GenericMap{
		"endpoint": endpoint,
		"headers": GenericMap{
			"x-api-key": "${APPDYNAMICS_API_KEY}",
		},
	}

	// Modify the config here
	if isTracingEnabled(dest) {
		tracesPipelineName := "traces/appdynamics-" + dest.GetID()
		currentConfig.Service.Pipelines[tracesPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
	}

	if isMetricsEnabled(dest) {
		metricsPipelineName := "metrics/appdynamics-" + dest.GetID()
		currentConfig.Service.Pipelines[metricsPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
	}

	if isLoggingEnabled(dest) {
		logsPipelineName := "logs/appdynamics-" + dest.GetID()
		currentConfig.Service.Pipelines[logsPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
	}

	return nil
}
