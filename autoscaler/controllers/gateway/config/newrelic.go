package config

import (
	"errors"
	"fmt"

	commonconf "github.com/odigos-io/odigos/autoscaler/controllers/common"
	"github.com/odigos-io/odigos/common"
)

const (
	newRelicEndpoint = "NEWRELIC_ENDPOINT"
)

type NewRelic struct{}

func (n *NewRelic) DestType() common.DestinationType {
	return common.NewRelicDestinationType
}

func (n *NewRelic) ModifyConfig(dest common.ExporterConfigurer, currentConfig *commonconf.Config) error {
	endpoint, exists := dest.GetConfig()[newRelicEndpoint]
	if !exists {
		return errors.New("New relic endpoint not specified, gateway will not be configured for New Relic")
	}

	exporterName := "otlp/newrelic-" + dest.GetName()
	currentConfig.Exporters[exporterName] = commonconf.GenericMap{
		"endpoint": fmt.Sprintf("%s:4317", endpoint),
		"headers": commonconf.GenericMap{
			"api-key": "${NEWRELIC_API_KEY}",
		},
	}

	if isTracingEnabled(dest) {
		tracesPipelineName := "traces/newrelic-" + dest.GetName()
		currentConfig.Service.Pipelines[tracesPipelineName] = commonconf.Pipeline{
			Exporters: []string{exporterName},
		}
	}

	if isMetricsEnabled(dest) {
		metricsPipelineName := "metrics/newrelic-" + dest.GetName()
		currentConfig.Service.Pipelines[metricsPipelineName] = commonconf.Pipeline{
			Exporters: []string{exporterName},
		}
	}

	if isLoggingEnabled(dest) {
		logsPipelineName := "logs/newrelic-" + dest.GetName()
		currentConfig.Service.Pipelines[logsPipelineName] = commonconf.Pipeline{
			Exporters: []string{exporterName},
		}
	}

	return nil
}
