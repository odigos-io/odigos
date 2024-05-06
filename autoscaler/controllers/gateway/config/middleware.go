package config

import (
	"errors"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	commonconf "github.com/odigos-io/odigos/autoscaler/controllers/common"
	"github.com/odigos-io/odigos/common"
)

const (
	target = "MW_TARGET"
)

type Middleware struct{}

func (m *Middleware) DestType() common.DestinationType {
	return common.MiddlewareDestinationType
}

func (m *Middleware) ModifyConfig(dest *odigosv1.Destination, currentConfig *commonconf.Config) error {

	if !isTracingEnabled(dest) && !isMetricsEnabled(dest) && !isLoggingEnabled(dest) {
		return errors.New("Middleware is not enabled for any supported signals, skipping")
	}

	_, exists := dest.Spec.Data[target]
	if !exists {
		return errors.New("Middleware target not specified, gateway will not be configured for Middleware")
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

	return nil
}
