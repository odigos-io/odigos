package config

import (
	odigosv1 "github.com/keyval-dev/odigos/api/v1alpha1"
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
	if isTracingEnabled(dest) || isMetricsEnabled(dest) || isLoggingEnabled(dest) {
		target, exists := dest.Spec.Data[target]
		if !exists {
			log.Log.V(0).Info("Middleware target not specified, gateway will not be configured for Middleware")
			return
		}

		currentConfig.Exporters["otlp/middleware"] = commonconf.GenericMap{
			"endpoint": "${MW_TARGET}",
			"headers": commonconf.GenericMap{
				"authorization": "${MW_API_KEY}",
			},
		}
	}

	if isTracingEnabled(dest) {
		currentConfig.Service.Pipelines["traces/middleware"] = commonconf.Pipeline{
			Receivers:  []string{"otlp"},
			Processors: []string{"batch"},
			Exporters:  []string{"otlp/middleware"},
		}
	}

	if isMetricsEnabled(dest) {
		currentConfig.Service.Pipelines["metrics/middleware"] = commonconf.Pipeline{
			Receivers:  []string{"otlp"},
			Processors: []string{"batch"},
			Exporters:  []string{"otlp/middleware"},
		}
	}

	if isLoggingEnabled(dest) {
		currentConfig.Service.Pipelines["logs/middleware"] = commonconf.Pipeline{
			Receivers:  []string{"otlp"},
			Processors: []string{"batch"},
			Exporters:  []string{"otlp/middleware"},
		}
	}
}
