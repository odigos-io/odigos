package config

import (
	"github.com/odigos-io/odigos/common"
)

const (
	SEQ_ENDPOINT = "SEQ_ENDPOINT"
	SEQ_API_KEY  = "SEQ_API_KEY"
)

type Seq struct{}

func (j *Seq) DestType() common.DestinationType {
	return common.SeqDestinationType
}

func (j *Seq) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) ([]string, error) {
	config := dest.GetConfig()
	uniqueUri := "seq-" + dest.GetID()
	var pipelineNames []string

	endpoint, exists := config[SEQ_ENDPOINT]
	if !exists {
		return nil, errorMissingKey(SEQ_ENDPOINT)
	}
	endpoint, err := parseOtlpHttpEndpoint(endpoint)
	if err != nil {
		return nil, err
	}

	exporterName := "otlphttp/" + uniqueUri
	currentConfig.Exporters[exporterName] = GenericMap{
		"endpoint": endpoint,
		"headers": GenericMap{
			"X-Seq-ApiKey": "${SEQ_API_KEY}",
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
