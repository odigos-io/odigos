package config

import (
	"errors"
	"fmt"

	"github.com/odigos-io/odigos/common"
)

var ErrorHoneycombTracingDisabled = errors.New("attempting to configure Honeycomb tracing, but tracing is disabled")

const (
	honeycombEndpoint = "HONEYCOMB_ENDPOINT"
)

type Honeycomb struct{}

func (h *Honeycomb) DestType() common.DestinationType {
	return common.HoneycombDestinationType
}

func (h *Honeycomb) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) ([]string, error) {
	if !isTracingEnabled(dest) {
		return nil, ErrorHoneycombTracingDisabled
	}

	endpoint, exists := dest.GetConfig()[honeycombEndpoint]
	if !exists {
		endpoint = "api.honeycomb.io"
	}

	exporterName := "otlp/honeycomb-" + dest.GetID()
	currentConfig.Exporters[exporterName] = GenericMap{
		"endpoint": fmt.Sprintf("%s:443", endpoint),
		"headers": GenericMap{
			"x-honeycomb-team": "${HONEYCOMB_API_KEY}",
		},
	}

	var pipelineNames []string

	if isTracingEnabled(dest) {
		tracePipelineName := "traces/honeycomb-" + dest.GetID()
		currentConfig.Service.Pipelines[tracePipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, tracePipelineName)
	}

	if isMetricsEnabled(dest) {
		metricsPipelineName := "metrics/honeycomb-" + dest.GetID()
		currentConfig.Service.Pipelines[metricsPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, metricsPipelineName)
	}

	if isLoggingEnabled(dest) {
		logsPipelineName := "logs/honeycomb-" + dest.GetID()
		currentConfig.Service.Pipelines[logsPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, logsPipelineName)
	}

	return pipelineNames, nil
}
