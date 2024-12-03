package config

import (
	"errors"
	"strings"

	"github.com/odigos-io/odigos/common"
)

const (
	APPDYNAMICS_APPLICATION_NAME = "APPDYNAMICS_APPLICATION_NAME"
	APPDYNAMICS_ACCOUNT_NAME     = "APPDYNAMICS_ACCOUNT_NAME"
	APPDYNAMICS_ENDPOINT_URL     = "APPDYNAMICS_ENDPOINT_URL"
	APPDYNAMICS_API_KEY          = "APPDYNAMICS_API_KEY"
)

type AppDynamics struct{}

func (m *AppDynamics) DestType() common.DestinationType {
	// DestinationType defined in common/dests.go
	return common.AppDynamicsDestinationType
}

func (m *AppDynamics) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) error {
	config := dest.GetConfig()
	uniqueUri := "appdynamics-" + dest.GetID()

	endpoint, endpointExists := config[APPDYNAMICS_ENDPOINT_URL]
	if !endpointExists {
		return errors.New("AppDynamics Endpoint URL (\"APPDYNAMICS_ENDPOINT_URL\") not specified, AppDynamics will not be configured")
	}

	isHttpEndpoint := strings.HasPrefix(endpoint, "http://")
	isHttpsEndpoint := strings.HasPrefix(endpoint, "https://")

	if !isHttpEndpoint && !isHttpsEndpoint {
		return errors.New("AppDynamics Endpoint URL (\"APPDYNAMICS_ENDPOINT_URL\") malformed, HTTP prefix is required, AppDynamics will not be configured")
	}

	accountName, accountNameExists := config[APPDYNAMICS_ACCOUNT_NAME]
	if !accountNameExists {
		return errors.New("AppDynamics Account Name (\"APPDYNAMICS_ACCOUNT_NAME\") not specified, AppDynamics will not be configured")
	}

	applicationName, applicationNameExists := config[APPDYNAMICS_APPLICATION_NAME]
	if !applicationNameExists {
		applicationName = "odigos"
	}

	endpointParts := strings.Split(endpoint, ".")
	if len(endpointParts) > 0 {
		// Replace the first part of the endpoint with the account name (instead of collecting another input from the user).
		// Example:
		// endpoint - "https://<something-with-dashes>.saas.appdynamics.com"
		// host - "<account-name>.saas.appdynamics.com"
		endpointParts[0] = accountName
	}
	host := strings.Join(endpointParts, ".")

	// Create config for exporter

	exporterName := "otlphttp/" + uniqueUri
	currentConfig.Exporters[exporterName] = GenericMap{
		"endpoint": endpoint,
		"headers": GenericMap{
			"x-api-key": "${APPDYNAMICS_API_KEY}",
		},
	}

	// Create config for processor

	processorName := "resource/" + uniqueUri
	currentConfig.Processors[processorName] = GenericMap{
		"attributes": []GenericMap{
			{
				// This is required by AppDynamics, without it they will accept the data but not display it.
				// This key will be used to identify the cluster in AppDynamics.
				"key":    "service.namespace",
				"value":  applicationName,
				"action": "insert",
			},
			{
				"key":    "appdynamics.controller.account",
				"value":  accountName,
				"action": "insert",
			},
			{
				"key":    "appdynamics.controller.host",
				"value":  host,
				"action": "insert",
			},
			{
				"key":    "appdynamics.controller.port",
				"value":  443,
				"action": "insert",
			},
		},
	}

	// Apply configs to serivce

	if isTracingEnabled(dest) {
		tracesPipelineName := "traces/" + uniqueUri
		currentConfig.Service.Pipelines[tracesPipelineName] = Pipeline{
			Exporters:  []string{exporterName},
			Processors: []string{processorName},
		}
	}

	if isMetricsEnabled(dest) {
		metricsPipelineName := "metrics/" + uniqueUri
		currentConfig.Service.Pipelines[metricsPipelineName] = Pipeline{
			Exporters:  []string{exporterName},
			Processors: []string{processorName},
		}
	}

	if isLoggingEnabled(dest) {
		logsPipelineName := "logs/" + uniqueUri
		currentConfig.Service.Pipelines[logsPipelineName] = Pipeline{
			Exporters:  []string{exporterName},
			Processors: []string{processorName},
		}
	}

	return nil
}
