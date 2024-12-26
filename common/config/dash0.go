package config

import (
	"errors"

	"github.com/odigos-io/odigos/common"
)

const (
	Dash0Endpoint = "DASH0_ENDPOINT"
)

var (
	ErrorDash0EndpointMissing = errors.New("Dash0 is missing a required field (\"DASH0_ENDPOINT\"), Dash0 will not be configured")
)

type Dash0 struct{}

func (j *Dash0) DestType() common.DestinationType {
	return common.Dash0DestinationType
}

func (j *Dash0) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) error {
	config := dest.GetConfig()
	uniqueUri := "dash0-" + dest.GetID()

	url, exists := config[Dash0Endpoint]
	if !exists {
		return ErrorDash0EndpointMissing
	}
	endpoint, err := parseOtlpGrpcUrl(url, false)
	if err != nil {
		return err
	}

	exporterName := "otlp/" + uniqueUri
	exporterConfig := GenericMap{
		"endpoint": endpoint,
		"headers": GenericMap{
			"Authorization": "Bearer ${DASH0_TOKEN}",
		},
	}

	currentConfig.Exporters[exporterName] = exporterConfig

	if isTracingEnabled(dest) {
		pipeName := "traces/" + uniqueUri
		currentConfig.Service.Pipelines[pipeName] = Pipeline{
			Exporters: []string{exporterName},
		}
	}

	if isMetricsEnabled(dest) {
		pipeName := "metrics/" + uniqueUri
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
