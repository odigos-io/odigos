package config

import (
	"errors"

	"github.com/odigos-io/odigos/common"
)

const (
	BetterStackMetricsEndpoint = "BETTERSTACK_METRICS_ENDPOINT"
	BetterStackLogsEndpoint    = "BETTERSTACK_LOGS_ENDPOINT"
)

var (
	ErrorBetterStackMetricsEndpointMissing = errors.New("BetterStack is missing a required field (\"BETTERSTACK_METRICS_ENDPOINT\"), BetterStack will not be configured")
	ErrorBetterStackLogsEndpointMissing    = errors.New("BetterStack is missing a required field (\"BETTERSTACK_LOGS_ENDPOINT\"), BetterStack will not be configured")
)

type BetterStack struct{}

func (j *BetterStack) DestType() common.DestinationType {
	return common.BetterStackDestinationType
}

func (j *BetterStack) ModifyConfig(dest ExporterConfigurer, cfg *Config) error {
	config := dest.GetConfig()
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
		url, exists := config[BetterStackMetricsEndpoint]
		if !exists {
			return ErrorBetterStackMetricsEndpointMissing
		}

		endpoint, err := parseOtlpHttpEndpoint(url)
		if err != nil {
			return err
		}

		exporterName := "prometheusremotewrite/" + uniqueUri
		cfg.Exporters[exporterName] = GenericMap{
			"endpoint": endpoint,
		}

		pipeName := "metrics/" + uniqueUri
		cfg.Service.Pipelines[pipeName] = Pipeline{
			Processors: []string{processorName},
			Exporters:  []string{exporterName},
		}
	}

	if isLoggingEnabled(dest) {
		url, exists := config[BetterStackLogsEndpoint]
		if !exists {
			return ErrorBetterStackLogsEndpointMissing
		}

		endpoint, err := parseOtlpHttpEndpoint(url)
		if err != nil {
			return err
		}

		exporterName := "otlp/" + uniqueUri
		cfg.Exporters[exporterName] = GenericMap{
			"endpoint": endpoint,
		}

		pipeName := "logs/" + uniqueUri
		cfg.Service.Pipelines[pipeName] = Pipeline{
			Processors: []string{processorName},
			Exporters:  []string{exporterName},
		}
	}

	return nil
}
