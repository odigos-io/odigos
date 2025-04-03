package config

import (
	"github.com/odigos-io/odigos/common"
)

const (
	VICTORIA_METRICS_ENDPOINT = "VICTORIA_METRICS_ENDPOINT"
	VICTORIA_METRICS_TOKEN    = "VICTORIA_METRICS_TOKEN"
)

type VictoriaMetrics struct{}

func (j *VictoriaMetrics) DestType() common.DestinationType {
	return common.VictoriaMetricsDestinationType
}

func (j *VictoriaMetrics) ModifyConfig(dest ExporterConfigurer, cfg *Config) ([]string, error) {
	config := dest.GetConfig()
	uniqueUri := "victoriametrics-" + dest.GetID()
	var pipelineNames []string

	endpoint, exists := config[VICTORIA_METRICS_ENDPOINT]
	if !exists {
		return nil, errorMissingKey(VICTORIA_METRICS_ENDPOINT)
	}
	endpoint, err := parseOtlpHttpEndpoint(endpoint, "", "/opentelemetry")
	if err != nil {
		return nil, err
	}

	authExtensionName := "bearertokenauth/" + uniqueUri
	cfg.Extensions[authExtensionName] = GenericMap{
		"token": "${VICTORIA_METRICS_TOKEN}",
	}

	exporterName := "otlphttp/" + uniqueUri
	cfg.Exporters[exporterName] = GenericMap{
		"endpoint": endpoint,
		"auth": GenericMap{
			"authenticator": authExtensionName,
		},
	}

	cfg.Service.Extensions = append(cfg.Service.Extensions, authExtensionName)

	if isMetricsEnabled(dest) {
		pipeName := "metrics/" + uniqueUri
		cfg.Service.Pipelines[pipeName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, pipeName)
	}

	if isLoggingEnabled(dest) {
		pipeName := "logs/" + uniqueUri
		cfg.Service.Pipelines[pipeName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, pipeName)
	}

	return pipelineNames, nil
}
