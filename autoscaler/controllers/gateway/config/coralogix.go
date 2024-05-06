package config

import (
	"errors"

	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	commonconf "github.com/keyval-dev/odigos/autoscaler/controllers/common"
	"github.com/keyval-dev/odigos/common"
)

var (
	ErrorCoralogixNoSignals = errors.New("coralogix destination does not have any signals to export")
)

const (
	coralogixDomain          = "CORALOGIX_DOMAIN"
	coralogixApplicationName = "CORALOGIX_APPLICATION_NAME"
	coralogixSubsystemName   = "CORALOGIX_SUBSYSTEM_NAME"
)

type Coralogix struct{}

func (c *Coralogix) DestType() common.DestinationType {
	return common.CoralogixDestinationType
}

func (c *Coralogix) ModifyConfig(dest *odigosv1.Destination, currentConfig *commonconf.Config) error {
	if !isTracingEnabled(dest) && !isLoggingEnabled(dest) && !isMetricsEnabled(dest) {
		return ErrorCoralogixNoSignals
	}

	domain, exists := dest.Spec.Data[coralogixDomain]
	if !exists {
		return errors.New("Coralogix domain not specified, gateway will not be configured for Coralogix")
	}
	appName, exists := dest.Spec.Data[coralogixApplicationName]
	if !exists {
		return errors.New("Coralogix application name not specified, gateway will not be configured for Coralogix")
	}
	subName, exists := dest.Spec.Data[coralogixSubsystemName]
	if !exists {
		return errors.New("Coralogix subsystem name not specified, gateway will not be configured for Coralogix")
	}

	exporterName := "coralogix/" + dest.Name
	currentConfig.Exporters[exporterName] = commonconf.GenericMap{
		"private_key":      "${CORALOGIX_PRIVATE_KEY}",
		"domain":           domain,
		"application_name": appName,
		"subsystem_name":   subName,
	}

	if isTracingEnabled(dest) {
		tracesPipelineName := "traces/coralogix-" + dest.Name
		currentConfig.Service.Pipelines[tracesPipelineName] = commonconf.Pipeline{
			Exporters: []string{exporterName},
		}
	}

	if isMetricsEnabled(dest) {
		metricsPipelineName := "metrics/coralogix-" + dest.Name
		currentConfig.Service.Pipelines[metricsPipelineName] = commonconf.Pipeline{
			Exporters: []string{exporterName},
		}
	}

	if isLoggingEnabled(dest) {
		logsPipelineName := "logs/coralogix-" + dest.Name
		currentConfig.Service.Pipelines[logsPipelineName] = commonconf.Pipeline{
			Exporters: []string{exporterName},
		}
	}

	return nil
}
