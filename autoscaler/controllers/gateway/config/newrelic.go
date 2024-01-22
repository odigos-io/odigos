package config

import (
	"fmt"

	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	commonconf "github.com/keyval-dev/odigos/autoscaler/controllers/common"
	"github.com/keyval-dev/odigos/common"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	newRelicEndpoint = "NEWRELIC_ENDPOINT"
)

type NewRelic struct{}

func (n *NewRelic) DestType() common.DestinationType {
	return common.NewRelicDestinationType
}

func (n *NewRelic) ModifyConfig(dest *odigosv1.Destination, currentConfig *commonconf.Config) {
	endpoint, exists := dest.Spec.Data[newRelicEndpoint]
	if !exists {
		log.Log.V(0).Info("New relic endpoint not specified, gateway will not be configured for New Relic")
		return
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
			Receivers:  []string{"otlp"},
			Processors: []string{"batch"},
			Exporters:  []string{exporterName},
		}
	}

	if isMetricsEnabled(dest) {
		metricsPipelineName := "metrics/newrelic-" + dest.Name
		currentConfig.Service.Pipelines[metricsPipelineName] = commonconf.Pipeline{
			Receivers:  []string{"otlp"},
			Processors: []string{"batch"},
			Exporters:  []string{exporterName},
		}
	}

	if isLoggingEnabled(dest) {
		logsPipelineName := "logs/newrelic-" + dest.Name
		currentConfig.Service.Pipelines[logsPipelineName] = commonconf.Pipeline{
			Receivers:  []string{"otlp"},
			Processors: []string{"batch"},
			Exporters:  []string{exporterName},
		}
	}
}
