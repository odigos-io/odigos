package config

import (
	"errors"
	"fmt"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
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

func (n *NewRelic) ModifyConfig(dest *odigosv1.Destination, currentConfig *commonconf.Config) error {
	endpoint, exists := dest.Spec.Data[newRelicEndpoint]
	if !exists {
		return errors.New("New relic endpoint not specified, gateway will not be configured for New Relic")
	}

	exporterName := "otlp/newrelic-" + dest.Name
	currentConfig.Exporters[exporterName] = commonconf.GenericMap{
		"endpoint": fmt.Sprintf("%s:4317", endpoint),
		"headers": commonconf.GenericMap{
			"api-key": "${NEWRELIC_API_KEY}",
		},
	}

	if isTracingEnabled(dest) {
		tracesPipelineName := "traces/newrelic-" + dest.Name
		currentConfig.Service.Pipelines[tracesPipelineName] = commonconf.Pipeline{
			Exporters: []string{exporterName},
		}
	}

	if isMetricsEnabled(dest) {
		metricsPipelineName := "metrics/newrelic-" + dest.Name
		currentConfig.Service.Pipelines[metricsPipelineName] = commonconf.Pipeline{
			Exporters: []string{exporterName},
		}
	}

	if isLoggingEnabled(dest) {
		logsPipelineName := "logs/newrelic-" + dest.Name
		currentConfig.Service.Pipelines[logsPipelineName] = commonconf.Pipeline{
			Exporters: []string{exporterName},
		}
	}

	return nil
}
