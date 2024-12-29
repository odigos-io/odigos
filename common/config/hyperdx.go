package config

import (
	"errors"

	"github.com/odigos-io/odigos/common"
)

const (
	HyperDXEndpoint = "HYPERDX_ENDPOINT"
)

var (
	ErrorHyperDXEndpointMissing = errors.New("HyperDX is missing a required field (\"HYPERDX_ENDPOINT\"), HyperDX will not be configured")
)

type HyperDX struct{}

func (j *HyperDX) DestType() common.DestinationType {
	return common.HyperDxDestinationType
}

func (j *HyperDX) ModifyConfig(dest ExporterConfigurer, cfg *Config) error {
	config := dest.GetConfig()
	uniqueUri := "hdx-" + dest.GetID()

	url, exists := config[HyperDXEndpoint]
	if !exists {
		return ErrorHyperDXEndpointMissing
	}

	endpoint, err := parseOtlpGrpcUrl(url, true)
	if err != nil {
		return err
	}

	exporterName := "otlp/" + uniqueUri
	exporterConfig := GenericMap{
		"endpoint": endpoint,
		"headers": GenericMap{
			"authorization": "${HYPERDX_API_KEY}",
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
