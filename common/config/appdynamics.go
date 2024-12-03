package config

import (
	"errors"
	"strings"

	"github.com/odigos-io/odigos/common"
)

const (
	APPDYNAMICS_ACCOUNT_NAME = "APPDYNAMICS_ACCOUNT_NAME"
	APPDYNAMICS_ENDPOINT_URL = "APPDYNAMICS_ENDPOINT_URL"
	APPDYNAMICS_API_KEY      = "APPDYNAMICS_API_KEY"
)

type AppDynamics struct{}

func (m *AppDynamics) DestType() common.DestinationType {
	// DestinationType defined in common/dests.go
	return common.AppDynamicsDestinationType
}

func (m *AppDynamics) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) error {
	config := dest.GetConfig()

	uniqueUri := "appdynamics-" + dest.GetID()
	exporterName := "otlphttp/" + uniqueUri
	processorName := "resource/" + uniqueUri
	tracesPipelineName := "traces/" + uniqueUri

	// Create config for exporter

	endpoint, exists := config[APPDYNAMICS_ENDPOINT_URL]
	if !exists {
		return errors.New("AppDynamics Endpoint URL (\"APPDYNAMICS_ENDPOINT_URL\") not specified, AppDynamics will not be configured")
	}

	isHttpEndpoint := strings.HasPrefix(endpoint, "http://")
	isHttpsEndpoint := strings.HasPrefix(endpoint, "https://")

	if !isHttpEndpoint && !isHttpsEndpoint {
		return errors.New("AppDynamics Endpoint URL (\"APPDYNAMICS_ENDPOINT_URL\") malformed, HTTP prefix is required, AppDynamics will not be configured")
	}

	currentConfig.Exporters[exporterName] = GenericMap{
		"endpoint": endpoint,
		"headers": GenericMap{
			"x-api-key": "${APPDYNAMICS_API_KEY}",
		},
	}

	// Create config for processor

	accountName, exists := config[APPDYNAMICS_ACCOUNT_NAME]
	if !exists {
		return errors.New("AppDynamics Account Name (\"APPDYNAMICS_ACCOUNT_NAME\") not specified, AppDynamics will not be configured")
	}

	endpointParts := strings.Split(endpoint, ".")
	if len(endpointParts) > 0 {
		// replace the first part of the endpoint with the account name (instead of collecting another input from the user)
		endpointParts[0] = accountName
	}
	host := strings.Join(endpointParts, ".")

	currentConfig.Processors[processorName] = GenericMap{
		"attributes": []GenericMap{
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
		},
	}

	// Apply configs to serivce

	if isTracingEnabled(dest) {
		currentConfig.Service.Pipelines[tracesPipelineName] = Pipeline{
			Exporters:  []string{exporterName},
			Processors: []string{processorName},
		}
	}

	return nil
}
