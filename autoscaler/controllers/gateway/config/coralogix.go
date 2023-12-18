package config

import (
	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	commonconf "github.com/keyval-dev/odigos/autoscaler/controllers/common"
	"github.com/keyval-dev/odigos/common"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	coralogixPrivateKey = "CORALOGIX_PRIVATE_KEY"
)

type Coralogix struct{}

func (c *Coralogix) DestType() common.DestinationType {
	return common.CoralogixDestinationType
}

func (c *Coralogix) ModifyConfig(dest *odigosv1.Destination, currentConfig *commonconf.Config) {
	if isTracingEnabled(dest) || isLoggingEnabled(dest) || isMetricsEnabled(dest) {
		privateKey, exists := dest.Spec.Data[coralogixPrivateKey]
		if !exists {
			log.Log.V(0).Info("Coralogix private key not specified, gateway will not be configured for Coralogix")
			return
		}

		currentConfig.Exporters["coralogix"] = commonconf.GenericMap{
			"private_key": 		privateKey,
			"domain":      		"${CORALOGIX_DOMAIN}",
			"application_name":	"${CORALOGIX_APPLICATION_NAME}",
			"subsystem_name":   "${CORALOGIX_SUBSYSTEM_NAME}",
		}
	}

	if isTracingEnabled(dest) {
		currentConfig.Service.Pipelines["traces/coralogix"] = commonconf.Pipeline{
			Receivers:  []string{"otlp"},
			Processors: []string{"batch"},
			Exporters:  []string{"coralogix"},
		}
	}

	if isMetricsEnabled(dest) {
		currentConfig.Service.Pipelines["metrics/coralogix"] = commonconf.Pipeline{
			Receivers:  []string{"otlp"},
			Processors: []string{"batch"},
			Exporters:  []string{"coralogix"},
		}
	}

	if isLoggingEnabled(dest) {
		currentConfig.Service.Pipelines["logs/coralogix"] = commonconf.Pipeline{
			Receivers:  []string{"otlp"},
			Processors: []string{"batch"},
			Exporters:  []string{"coralogix"},
		}
	}
}