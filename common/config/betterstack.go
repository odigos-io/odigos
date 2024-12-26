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

	processorName := "attributes/betterstack"
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
		metricsExporterName := "prometheusremotewrite/" + uniqueUri
		cfg.Exporters[metricsExporterName] = GenericMap{
			"endpoint": "https://in-otel.logs.betterstack.com/metrics",
		}

		pipeName := "metrics/" + uniqueUri
		cfg.Service.Pipelines[pipeName] = Pipeline{
			Processors: []string{processorName},
			Exporters:  []string{metricsExporterName},
		}

		if isLoggingEnabled(dest) {
			logsExporterName := "otlp/" + uniqueUri
			cfg.Exporters[logsExporterName] = GenericMap{
				"endpoint": "https://in-otel.logs.betterstack.com:443",
			}

			pipeName := "logs/" + uniqueUri
			cfg.Service.Pipelines[pipeName] = Pipeline{
				Processors: []string{processorName},
				Exporters:  []string{logsExporterName},
			}
		}

	}

	return nil
}
