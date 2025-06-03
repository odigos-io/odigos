package config

import (
	"github.com/odigos-io/odigos/common"
)

const (
	INSTANA_ENDPOINT = "INSTANA_ENDPOINT"
)

type Instana struct{}

func (m *Instana) DestType() common.DestinationType {
	// DestinationType defined in common/dests.go
	return common.InstanaDestinationType
}

func (m *Instana) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) ([]string, error) {
	config := dest.GetConfig()
	// To make sure that the exporter and pipeline names are unique, we'll need to define a unique ID
	uniqueUri := "instana-" + dest.GetID()

	endpoint, exists := config[INSTANA_ENDPOINT]
	if !exists {
		return nil, errorMissingKey(INSTANA_ENDPOINT)
	}

	endpoint, err := parseOtlpGrpcUrl(endpoint, true)
	if err != nil {
		return nil, err
	}

	// Modify the exporter here
	exporterName := "otlp/" + uniqueUri
	exporterConfig := GenericMap{
		"endpoint": endpoint,
		"headers": GenericMap{
			"x-instana-key": "${INSTANA_AGENT_KEY}",
		},
	}

	// The Instana backend requires the host.id, faas.id, or device.id resource attribute,
	// or you can also set x-instana-host as a header in the exporter config.
	processorName := "resource/" + uniqueUri
	processorConfig := GenericMap{
		"attributes": []GenericMap{
			{
				"key":            "host.id",
				"from_attribute": "k8s.node.name",
				"action":         "insert",
			},
		},
	}

	currentConfig.Exporters[exporterName] = exporterConfig
	currentConfig.Processors[processorName] = processorConfig

	// Modify the pipelines here
	var pipelineNames []string

	if isTracingEnabled(dest) {
		pipeName := "traces/" + uniqueUri
		currentConfig.Service.Pipelines[pipeName] = Pipeline{
			Exporters:  []string{exporterName},
			Processors: []string{processorName},
		}
		pipelineNames = append(pipelineNames, pipeName)
	}

	if isMetricsEnabled(dest) {
		pipeName := "metrics/" + uniqueUri
		currentConfig.Service.Pipelines[pipeName] = Pipeline{
			Exporters:  []string{exporterName},
			Processors: []string{processorName},
		}
		pipelineNames = append(pipelineNames, pipeName)
	}

	if isLoggingEnabled(dest) {
		pipeName := "logs/" + uniqueUri
		currentConfig.Service.Pipelines[pipeName] = Pipeline{
			Exporters:  []string{exporterName},
			Processors: []string{processorName},
		}
		pipelineNames = append(pipelineNames, pipeName)
	}

	return pipelineNames, nil
}
