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
	cfg.Service.Extensions = append(cfg.Service.Extensions, authExtensionName)

	exporterName := "otlphttp/" + uniqueUri
	cfg.Exporters[exporterName] = GenericMap{
		"endpoint": endpoint,
		"auth": GenericMap{
			"authenticator": authExtensionName,
		},
	}

	spanMetricNames := applySpanMetricsConnector(cfg, uniqueUri)
	pipelineNames = append(pipelineNames, spanMetricNames.TracesPipeline)

	if isMetricsEnabled(dest) {
		pipeName := "metrics/" + uniqueUri
		cfg.Service.Pipelines[pipeName] = Pipeline{
			Receivers: []string{spanMetricNames.SpanMetricsConnector},
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, pipeName)
	}

	return pipelineNames, nil
}
