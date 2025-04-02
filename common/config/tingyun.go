package config

import (
	"github.com/odigos-io/odigos/common"
)

const (
	TINGYUN_ENDPOINT    = "TINGYUN_ENDPOINT"
	TINGYUN_LICENSE_KEY = "TINGYUN_LICENSE_KEY"
)

type Tingyun struct{}

func (j *Tingyun) DestType() common.DestinationType {
	return common.TingyunDestinationType
}

func (j *Tingyun) ModifyConfig(dest ExporterConfigurer, cfg *Config) ([]string, error) {
	config := dest.GetConfig()
	uniqueUri := "tingyun-" + dest.GetID()
	var pipelineNames []string

	endpoint, exists := config[TINGYUN_ENDPOINT]
	if !exists {
		return nil, errorMissingKey(TINGYUN_ENDPOINT)
	}
	endpoint, err := parseOtlpHttpEndpoint(endpoint)
	if err != nil {
		return nil, err
	}

	exporterName := "otlphttp/" + uniqueUri
	cfg.Exporters[exporterName] = GenericMap{
		"endpoint": endpoint,
	}

	resourceProcessorName := "resource/" + uniqueUri
	cfg.Processors[resourceProcessorName] = GenericMap{
		"attributes": []GenericMap{
			{
				"key":    "tingyun.license",
				"value":  "${TINGYUN_LICENSE_KEY}",
				"action": "insert",
			},
		},
	}

	if isTracingEnabled(dest) {
		pipeName := "traces/" + uniqueUri
		cfg.Service.Pipelines[pipeName] = Pipeline{
			Processors: []string{resourceProcessorName},
			Exporters:  []string{exporterName},
		}
		pipelineNames = append(pipelineNames, pipeName)
	}

	if isMetricsEnabled(dest) {
		pipeName := "metrics/" + uniqueUri
		cfg.Service.Pipelines[pipeName] = Pipeline{
			Processors: []string{resourceProcessorName},
			Exporters:  []string{exporterName},
		}
		pipelineNames = append(pipelineNames, pipeName)
	}

	return pipelineNames, nil
}
