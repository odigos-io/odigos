package config

import (
	"errors"

	"github.com/odigos-io/odigos/common"
)

const (
	GroundcoverEndpoint = "GROUNDCOVER_ENDPOINT"
	GroundcoverApiKey   = "GROUNDCOVER_API_KEY"
)

var (
	ErrorGroundcoverEndpointMissing = errors.New("Groundcover is missing a required field " +
		"(\"GROUNDCOVER_ENDPOINT\"), Groundcover will not be configured")
	ErrorGroundcoverApiKeyMissing = errors.New("Groundcover is missing a required field " +
		"(\"GROUNDCOVER_API_KEY\"), Groundcover will not be configured")
)

type Groundcover struct{}

func (j *Groundcover) DestType() common.DestinationType {
	return common.GroundcoverDestinationType
}

func (j *Groundcover) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) error {
	config := dest.GetConfig()
	uniqueUri := "groundcover-" + dest.GetID()

	url, exists := config[GroundcoverEndpoint]
	if !exists {
		return ErrorGroundcoverEndpointMissing
	}

	endpoint, err := parseOtlpGrpcUrl(url, true)
	if err != nil {
		return err
	}

	exporterName := "otlp/" + uniqueUri
	exporterConfig := GenericMap{
		"endpoint": endpoint,
		"headers": GenericMap{
			"apikey": "${GROUNDCOVER_API_KEY}",
		},
	}

	currentConfig.Exporters[exporterName] = exporterConfig

	if isTracingEnabled(dest) {
		tracesPipelineName := "traces/" + uniqueUri
		currentConfig.Service.Pipelines[tracesPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
	}

	if isMetricsEnabled(dest) {
		tracesPipelineName := "metrics/" + uniqueUri
		currentConfig.Service.Pipelines[tracesPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
	}

	if isLoggingEnabled(dest) {
		tracesPipelineName := "logs/" + uniqueUri
		currentConfig.Service.Pipelines[tracesPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
	}

	return nil
}
