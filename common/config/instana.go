package config

import (
	"github.com/odigos-io/odigos/common"
)

const (
	INSTANA_HOST = "INSTANA_HOST"
)

type Instana struct{}

func (m *Instana) DestType() common.DestinationType {
	// DestinationType defined in common/dests.go
	return common.InstanaDestinationType
}

//nolint:funlen,gocyclo // This function is inherently complex due to Instana config validation, refactoring is non-trivial
func (m *Instana) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) ([]string, error) {
	config := dest.GetConfig()
	// To make sure that the exporter and pipeline names are unique, we'll need to define a unique ID
	uniqueUri := "kafka-" + dest.GetID()

	host, exists := config[INSTANA_HOST]
	if !exists {
		// return nil, errorMissingKey(INSTANA_HOST)
		host = ""
	}

	// Modify the exporter here
	exporterName := "kafka/" + uniqueUri
	exporterConfig := GenericMap{
		"endpoint": host,
		"headers": GenericMap{
			"Authorization": "apiToken ${INSTANA_API_TOKEN}",
		},
	}

	currentConfig.Exporters[exporterName] = exporterConfig

	// Modify the pipelines here
	var pipelineNames []string

	if isTracingEnabled(dest) {
		// "https://<INSTANA_ZONE>.instana.io/api/otel"

		pipeName := "traces/" + uniqueUri
		currentConfig.Service.Pipelines[pipeName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, pipeName)
	}

	if isMetricsEnabled(dest) {
		// https://<INSTANA_ZONE>.instana.io/api/prom/push

		pipeName := "metrics/" + uniqueUri
		currentConfig.Service.Pipelines[pipeName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, pipeName)
	}

	if isLoggingEnabled(dest) {
		// https://<INSTANA_ZONE>.instana.io/api/logs

		pipeName := "logs/" + uniqueUri
		currentConfig.Service.Pipelines[pipeName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, pipeName)
	}

	return pipelineNames, nil
}
