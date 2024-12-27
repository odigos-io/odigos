package config

import (
	"errors"

	"github.com/odigos-io/odigos/common"
)

const (
	HighlightEndpoint = "HIGHLIGHT_ENDPOINT"
)

var (
	ErrorHighlightEndpointMissing = errors.New("Highlight is missing a required field (\"HIGHLIGHT_ENDPOINT\"), Highlight will not be configured")
)

type Highlight struct{}

func (j *Highlight) DestType() common.DestinationType {
	return common.HighlightDestinationType
}

func (j *Highlight) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) error {
	config := dest.GetConfig()
	uniqueUri := "highlight-" + dest.GetID()

	url, exists := config[HighlightEndpoint]
	if !exists {
		return ErrorHighlightEndpointMissing
	}

	endpoint, err := parseOtlpHttpEndpoint(url)
	if err != nil {
		return err
	}

	exporterName := "otlp/" + uniqueUri
	currentConfig.Exporters[exporterName] = GenericMap{
		"endpoint": endpoint,
		"headers": GenericMap{
			"x-highlight-project": "${HIGHLIGHT_PROJECT_ID}",
		},
	}

	if isTracingEnabled(dest) {
		pipeName := "traces/" + uniqueUri
		currentConfig.Service.Pipelines[pipeName] = Pipeline{
			Exporters: []string{exporterName},
		}
	}

	if isLoggingEnabled(dest) {
		pipeName := "logs/" + uniqueUri
		currentConfig.Service.Pipelines[pipeName] = Pipeline{
			Exporters: []string{exporterName},
		}
	}

	return nil
}
