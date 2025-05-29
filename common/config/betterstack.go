package config

import (
	"github.com/odigos-io/odigos/common"
)

type BetterStack struct{}

func (j *BetterStack) DestType() common.DestinationType {
	return common.BetterStackDestinationType
}

func (j *BetterStack) ModifyConfig(dest ExporterConfigurer, cfg *Config) ([]string, error) {
	uniqueUri := "betterstack-" + dest.GetID()
	pipelineNames := []string{}

	processorName := "attributes/" + uniqueUri
	cfg.Processors[processorName] = GenericMap{
		"actions": []GenericMap{
			{
				"key":    "better_stack_source_token",
				"value":  "${BETTERSTACK_TOKEN}",
				"action": "insert",
			},
		},
	}

	if IsMetricsEnabled(dest) {
		exporterName := "prometheusremotewrite/" + uniqueUri
		cfg.Exporters[exporterName] = GenericMap{
			"endpoint": "https://in-otel.logs.betterstack.com/metrics",
		}

		pipeName := "metrics/" + uniqueUri
		cfg.Service.Pipelines[pipeName] = Pipeline{
			Processors: []string{processorName},
			Exporters:  []string{exporterName},
		}
		pipelineNames = append(pipelineNames, pipeName)
	}

	if IsLoggingEnabled(dest) {
		exporterName := "otlp/" + uniqueUri
		cfg.Exporters[exporterName] = GenericMap{
			"endpoint": "https://in-otel.logs.betterstack.com:443",
		}

		pipeName := "logs/" + uniqueUri
		cfg.Service.Pipelines[pipeName] = Pipeline{
			Processors: []string{processorName},
			Exporters:  []string{exporterName},
		}
		pipelineNames = append(pipelineNames, pipeName)
	}

	return pipelineNames, nil
}
