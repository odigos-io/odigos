package config

import (
	"github.com/odigos-io/odigos/common"
)

type BetterStack struct{}

func (j *BetterStack) DestType() common.DestinationType {
	return common.BetterStackDestinationType
}

func (j *BetterStack) ModifyConfig(dest ExporterConfigurer, cfg *Config) error {
	uniqueUri := "betterstack-" + dest.GetID()

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

	if isMetricsEnabled(dest) {
		exporterName := "prometheusremotewrite/" + uniqueUri
		cfg.Exporters[exporterName] = GenericMap{
			"endpoint": "https://in-otel.logs.betterstack.com/metrics",
		}

		pipeName := "metrics/" + uniqueUri
		cfg.Service.Pipelines[pipeName] = Pipeline{
			Processors: []string{processorName},
			Exporters:  []string{exporterName},
		}
	}

	if isLoggingEnabled(dest) {
		exporterName := "otlp/" + uniqueUri
		cfg.Exporters[exporterName] = GenericMap{
			"endpoint": "https://in-otel.logs.betterstack.com:443",
		}

		pipeName := "logs/" + uniqueUri
		cfg.Service.Pipelines[pipeName] = Pipeline{
			Processors: []string{processorName},
			Exporters:  []string{exporterName},
		}
	}

	return nil
}
