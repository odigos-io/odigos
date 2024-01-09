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

	currentConfig.Exporters["otlp/newrelic"] = commonconf.GenericMap{
		"endpoint": fmt.Sprintf("%s:4317", endpoint),
		"headers": commonconf.GenericMap{
			"api-key": "${NEWRELIC_API_KEY}",
		},
	}

	if isTracingEnabled(dest) {
		currentConfig.Service.Pipelines["traces/newrelic"] = commonconf.Pipeline{
			Receivers:  []string{"otlp"},
			Processors: []string{"batch"},
			Exporters:  []string{"otlp/newrelic"},
		}
	}

	if isMetricsEnabled(dest) {
		currentConfig.Service.Pipelines["metrics/newrelic"] = commonconf.Pipeline{
			Receivers:  []string{"otlp"},
			Processors: []string{"batch"},
			Exporters:  []string{"otlp/newrelic"},
		}
	}

	if isLoggingEnabled(dest) {
		currentConfig.Service.Pipelines["logs/newrelic"] = commonconf.Pipeline{
			Receivers:  []string{"otlp"},
			Processors: []string{"batch"},
			Exporters:  []string{"otlp/newrelic"},
		}
	}
}
