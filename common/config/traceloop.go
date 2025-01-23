package config

import (
	"errors"

	"github.com/odigos-io/odigos/common"
)

const (
	TraceloopEndpoint = "TRACELOOP_ENDPOINT"
)

var (
	ErrorTraceloopEndpointMissing = errors.New("Traceloop is missing a required field (\"TRACELOOP_ENDPOINT\"), Traceloop will not be configured")
)

type Traceloop struct{}

func (j *Traceloop) DestType() common.DestinationType {
	return common.TraceloopDestinationType
}

func (j *Traceloop) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) ([]string, error) {
	config := dest.GetConfig()
	uniqueUri := "traceloop-" + dest.GetID()

	url, exists := config[TraceloopEndpoint]
	if !exists {
		return nil, ErrorTraceloopEndpointMissing
	}

	endpoint, err := parseOtlpHttpEndpoint(url)
	if err != nil {
		return nil, err
	}

	exporterName := "otlphttp/" + uniqueUri
	exporterConfig := GenericMap{
		"endpoint": endpoint,
		"headers": GenericMap{
			"Authorization": "Bearer ${TRACELOOP_API_KEY}",
		},
	}

	currentConfig.Exporters[exporterName] = exporterConfig
	var pipelineNames []string
	if isTracingEnabled(dest) {
		pipeName := "traces/" + uniqueUri
		currentConfig.Service.Pipelines[pipeName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, pipeName)
	}

	if isMetricsEnabled(dest) {
		pipeName := "metrics/" + uniqueUri
		currentConfig.Service.Pipelines[pipeName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, pipeName)
	}

	return pipelineNames, nil
}
