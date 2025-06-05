package config

import (
	"github.com/odigos-io/odigos/common"
)

type Highlight struct{}

func (j *Highlight) DestType() common.DestinationType {
	return common.HighlightDestinationType
}

func (j *Highlight) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) ([]string, error) {
	var pipelineNames []string
	uniqueUri := "highlight-" + dest.GetID()

	exporterName := "otlp/" + uniqueUri
	currentConfig.Exporters[exporterName] = GenericMap{
		"endpoint": "https://otel.highlight.io:4318",
		"headers": GenericMap{
			"x-highlight-project": "${HIGHLIGHT_PROJECT_ID}",
		},
	}

	if isTracingEnabled(dest) {
		pipeName := "traces/" + uniqueUri
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
