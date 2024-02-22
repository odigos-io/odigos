package config

import (
	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	commonconf "github.com/keyval-dev/odigos/autoscaler/controllers/common"
	"github.com/keyval-dev/odigos/common"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	target = "MW_TARGET"
)

type Middleware struct{}

func (m *Middleware) DestType() common.DestinationType {
	return common.MiddlewareDestinationType
}

func (m *Middleware) ModifyConfig(dest *odigosv1.Destination, currentConfig *commonconf.Config) {

	if !isTracingEnabled(dest) && !isMetricsEnabled(dest) && !isLoggingEnabled(dest) {
		log.Log.V(0).Info("Middleware is not enabled for any supported signals, skipping")
		return
	}

	_, exists := dest.Spec.Data[target]
	if !exists {
		log.Log.V(0).Info("Middleware target not specified, gateway will not be configured for Middleware")
		return
	}

	exporterName := "otlp/middleware-" + dest.Name
	currentConfig.Exporters[exporterName] = commonconf.GenericMap{
		"endpoint": "${MW_TARGET}",
		"headers": commonconf.GenericMap{
			"authorization": "${MW_API_KEY}",
		},
	}

	if isTracingEnabled(dest) {
		tracesPipelineName := "traces/middleware-" + dest.Name
		currentConfig.Service.Pipelines[tracesPipelineName] = commonconf.Pipeline{
			Exporters: []string{exporterName},
		}
	}

	if isMetricsEnabled(dest) {
		metricsPipelineName := "metrics/middleware-" + dest.Name
		currentConfig.Service.Pipelines[metricsPipelineName] = commonconf.Pipeline{
			Exporters: []string{exporterName},
		}
	}

	if isLoggingEnabled(dest) {
		logsPipelineName := "logs/middleware-" + dest.Name
		currentConfig.Service.Pipelines[logsPipelineName] = commonconf.Pipeline{
			Exporters: []string{exporterName},
		}
	}
}
