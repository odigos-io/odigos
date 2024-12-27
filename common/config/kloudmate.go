package config

import (
	"errors"

	"github.com/odigos-io/odigos/common"
)

const (
	KloudMateEndpoint = "KLOUDMATE_ENDPOINT"
)

var (
	ErrorKloudMateEndpointMissing = errors.New("KloudMate is missing a required field (\"KLOUDMATE_ENDPOINT\"), KloudMate will not be configured")
)

type KloudMate struct{}

func (j *KloudMate) DestType() common.DestinationType {
	return common.KloudMateDestinationType
}

func (j *KloudMate) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) error {
	config := dest.GetConfig()
	uniqueUri := "kloudmate-" + dest.GetID()

	url, exists := config[KloudMateEndpoint]
	if !exists {
		return ErrorKloudMateEndpointMissing
	}

	endpoint, err := parseOtlpHttpEndpoint(url)
	if err != nil {
		return err
	}

	exporterName := "otlphttp/" + uniqueUri
	exporterConfig := GenericMap{
		"endpoint": endpoint,
		"headers": GenericMap{
			"Authorization": "${KLOUDMATE_API_KEY}",
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
