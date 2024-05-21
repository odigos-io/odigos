package config

import (
	"errors"

	"github.com/odigos-io/odigos/common"
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

func (c *Coralogix) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) error {
	if !isTracingEnabled(dest) && !isLoggingEnabled(dest) && !isMetricsEnabled(dest) {
		return ErrorCoralogixNoSignals
	}

	domain, exists := dest.GetConfig()[coralogixDomain]
	if !exists {
		return errors.New("Coralogix domain not specified, gateway will not be configured for Coralogix")
	}
	appName, exists := dest.GetConfig()[coralogixApplicationName]
	if !exists {
		return errors.New("Coralogix application name not specified, gateway will not be configured for Coralogix")
	}
	subName, exists := dest.GetConfig()[coralogixSubsystemName]
	if !exists {
		return errors.New("Coralogix subsystem name not specified, gateway will not be configured for Coralogix")
	}

	exporterName := "coralogix/" + dest.GetName()
	currentConfig.Exporters[exporterName] = GenericMap{
		"private_key":      "${CORALOGIX_PRIVATE_KEY}",
		"domain":           domain,
		"application_name": appName,
		"subsystem_name":   subName,
	}

	if isTracingEnabled(dest) {
		tracesPipelineName := "traces/coralogix-" + dest.GetName()
		currentConfig.Service.Pipelines[tracesPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
	}

	if isMetricsEnabled(dest) {
		metricsPipelineName := "metrics/coralogix-" + dest.GetName()
		currentConfig.Service.Pipelines[metricsPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
	}

	if isLoggingEnabled(dest) {
		logsPipelineName := "logs/coralogix-" + dest.GetName()
		currentConfig.Service.Pipelines[logsPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
	}

	return nil
}
