package config

import (
	"errors"
	"fmt"

	"github.com/odigos-io/odigos/common"
)

const (
	newRelicEndpoint = "NEWRELIC_ENDPOINT"
)

type NewRelic struct{}

func (n *NewRelic) DestType() common.DestinationType {
	return common.NewRelicDestinationType
}

func (n *NewRelic) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) ([]string, error) {
	endpoint, exists := dest.GetConfig()[newRelicEndpoint]
	if !exists {
		return nil, errors.New("New relic endpoint not specified, gateway will not be configured for New Relic")
	}

	exporterName := "otlp/newrelic-" + dest.GetID()
	currentConfig.Exporters[exporterName] = GenericMap{
		"endpoint": fmt.Sprintf("%s:4317", endpoint),
		"headers": GenericMap{
			"api-key": "${NEWRELIC_API_KEY}",
		},
	}
	var pipelineNames []string
	if IsTracingEnabled(dest) {
		tracesPipelineName := "traces/newrelic-" + dest.GetID()
		currentConfig.Service.Pipelines[tracesPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, tracesPipelineName)
	}

	if IsMetricsEnabled(dest) {
		metricsPipelineName := "metrics/newrelic-" + dest.GetID()
		currentConfig.Service.Pipelines[metricsPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, metricsPipelineName)
	}

	if IsLoggingEnabled(dest) {
		logsPipelineName := "logs/newrelic-" + dest.GetID()
		currentConfig.Service.Pipelines[logsPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, logsPipelineName)
	}

	return pipelineNames, nil
}
