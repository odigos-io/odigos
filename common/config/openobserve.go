package config

import (
	"github.com/odigos-io/odigos/common"
)

const (
	OPEN_OBSERVE_ENDPOINT    = "OPEN_OBSERVE_ENDPOINT"
	OPEN_OBSERVE_API_KEY     = "OPEN_OBSERVE_API_KEY"
	OPEN_OBSERVE_STREAM_NAME = "OPEN_OBSERVE_STREAM_NAME"
)

type OpenObserve struct{}

func (j *OpenObserve) DestType() common.DestinationType {
	return common.OpenObserveDestinationType
}

func (j *OpenObserve) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) ([]string, error) {
	config := dest.GetConfig()
	uniqueUri := "openobserve-" + dest.GetID()
	var pipelineNames []string

	endpoint, exists := config[OPEN_OBSERVE_ENDPOINT]
	if !exists {
		return nil, errorMissingKey(OPEN_OBSERVE_ENDPOINT)
	}

	streamName, exists := config[OPEN_OBSERVE_STREAM_NAME]
	if !exists {
		return nil, errorMissingKey(OPEN_OBSERVE_STREAM_NAME)
	}

	exporterName := "otlphttp/" + uniqueUri
	currentConfig.Exporters[exporterName] = GenericMap{
		"endpoint": endpoint,
		"headers": GenericMap{
			"Authorization": "Basic ${OPEN_OBSERVE_API_KEY}",
			"stream-name":   streamName,
		},
	}

	if isTracingEnabled(dest) {
		pipeName := "traces/" + uniqueUri
		currentConfig.Service.Pipelines[pipeName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, pipeName)
	}

	if isMetricsEnabled(dest) {
		pipeName := "metrics/" + uniqueUri
		currentConfig.Service.Pipelines[pipeName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, pipeName)
	}

	if isLoggingEnabled(dest) {
		pipeName := "logs/" + uniqueUri
		currentConfig.Service.Pipelines[pipeName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, pipeName)
	}

	return pipelineNames, nil
}
