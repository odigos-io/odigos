package config

import (
	"errors"

	"github.com/odigos-io/odigos/common"
)

const (
	LumigoEndpoint = "LUMIGO_ENDPOINT"
)

var (
	ErrorLumigoEndpointMissing = errors.New("Lumigo is missing a required field (\"LUMIGO_ENDPOINT\"), Lumigo will not be configured")
)

type Lumigo struct{}

func (j *Lumigo) DestType() common.DestinationType {
	return common.LumigoDestinationType
}

func (j *Lumigo) ModifyConfig(dest ExporterConfigurer, cfg *Config) error {
	config := dest.GetConfig()
	uniqueUri := "lumigo-" + dest.GetID()

	url, exists := config[LumigoEndpoint]
	if !exists {
		return ErrorLumigoEndpointMissing
	}

	endpoint, err := parseOtlpHttpEndpoint(url)
	if err != nil {
		return err
	}

	exporterName := "otlphttp/" + uniqueUri
	exporterConfig := GenericMap{
		"endpoint": endpoint,
		"headers": GenericMap{
			"Authorization": "LumigoToken ${LUMIGO_TOKEN}",
		},
	}

	cfg.Exporters[exporterName] = exporterConfig

	if isTracingEnabled(dest) {
		pipeName := "traces/" + uniqueUri
		cfg.Service.Pipelines[pipeName] = Pipeline{
			Exporters: []string{exporterName},
		}
	}

	if isMetricsEnabled(dest) {
		pipeName := "metrics/" + uniqueUri
		cfg.Service.Pipelines[pipeName] = Pipeline{
			Exporters: []string{exporterName},
		}
	}

	if isLoggingEnabled(dest) {
		pipeName := "logs/" + uniqueUri
		cfg.Service.Pipelines[pipeName] = Pipeline{
			Exporters: []string{exporterName},
		}
	}

	return nil
}
