package config

import (
	"github.com/odigos-io/odigos/common"
)

const (
	ALIBABA_ENDPOINT = "ALIBABA_ENDPOINT"
	ALIBABA_TOKEN    = "ALIBABA_TOKEN"
)

type AlibabaCloud struct{}

func (j *AlibabaCloud) DestType() common.DestinationType {
	return common.AlibabaCloudDestinationType
}

func (j *AlibabaCloud) ModifyConfig(dest ExporterConfigurer, cfg *Config) ([]string, error) {
	config := dest.GetConfig()
	uniqueUri := "alibaba-" + dest.GetID()
	var pipelineNames []string

	endpoint, exists := config[ALIBABA_ENDPOINT]
	if !exists {
		return nil, errorMissingKey(ALIBABA_ENDPOINT)
	}
	endpoint, err := parseOtlpGrpcUrl(endpoint, false)
	if err != nil {
		return nil, err
	}

	exporterName := "otlp/" + uniqueUri
	cfg.Exporters[exporterName] = GenericMap{
		"endpoint": endpoint,
		"headers": GenericMap{
			"Authentication": "${ALIBABA_TOKEN}",
		},
	}

	if isTracingEnabled(dest) {
		pipeName := "traces/" + uniqueUri
		cfg.Service.Pipelines[pipeName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, pipeName)
	}

	return pipelineNames, nil
}
