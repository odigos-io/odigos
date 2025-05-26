package config

import (
	"errors"
	"strings"

	"github.com/odigos-io/odigos/common"
)

const (
	dsnKey      = "UPTRACE_DSN"
	endpointKey = "UPTRACE_ENDPOINT"
)

type Uptrace struct{}

func (s *Uptrace) DestType() common.DestinationType {
	return common.UptraceDestinationType
}

func (s *Uptrace) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) ([]string, error) {
	config := dest.GetConfig()
	dsn, exists := config[dsnKey]
	if !exists {
		return nil, errors.New("Uptrace url(\"UPTRACE_DSN\") not specified, gateway will not be configured for Uptrace")
	}

	endpoint, exists := config[endpointKey]
	if !exists {
		endpoint = "https://otlp.uptrace.dev:4317"
	}

	isHttpEndpoint := strings.HasPrefix(endpoint, "http://")
	exporterName := "otlp/uptrace-" + dest.GetID()

	currentConfig.Exporters[exporterName] = GenericMap{
		"endpoint": endpoint,
		"tls": GenericMap{
			"insecure": isHttpEndpoint,
		},
		"headers": GenericMap{
			"uptrace-dsn": dsn,
		},
	}

	var pipelineNames []string
	if IsTracingEnabled(dest) {
		tracesPipelineName := "traces/uptrace-" + dest.GetID()
		currentConfig.Service.Pipelines[tracesPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, tracesPipelineName)
	}

	if IsMetricsEnabled(dest) {
		metricsPipelineName := "metrics/uptrace-" + dest.GetID()
		currentConfig.Service.Pipelines[metricsPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, metricsPipelineName)
	}

	if IsLoggingEnabled(dest) {
		logsPipelineName := "logs/uptrace-" + dest.GetID()
		currentConfig.Service.Pipelines[logsPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, logsPipelineName)
	}

	return pipelineNames, nil
}
