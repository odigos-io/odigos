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

func (c *Coralogix) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) ([]string, error) {
	if !IsTracingEnabled(dest) && !IsLoggingEnabled(dest) && !IsMetricsEnabled(dest) {
		return nil, ErrorCoralogixNoSignals
	}

	domain, exists := dest.GetConfig()[coralogixDomain]
	if !exists {
		return nil, errors.New("Coralogix domain not specified, gateway will not be configured for Coralogix")
	}
	appName, exists := dest.GetConfig()[coralogixApplicationName]
	if !exists {
		return nil, errors.New("Coralogix application name not specified, gateway will not be configured for Coralogix")
	}
	subName, exists := dest.GetConfig()[coralogixSubsystemName]
	if !exists {
		return nil, errors.New("Coralogix subsystem name not specified, gateway will not be configured for Coralogix")
	}

	exporterName := "coralogix/" + dest.GetID()
	currentConfig.Exporters[exporterName] = GenericMap{
		"private_key":      "${CORALOGIX_PRIVATE_KEY}",
		"domain":           domain,
		"application_name": appName,
		"subsystem_name":   subName,
	}

	var pipelineNames []string

	if IsTracingEnabled(dest) {
		tracesPipelineName := "traces/coralogix-" + dest.GetID()
		currentConfig.Service.Pipelines[tracesPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, tracesPipelineName)
	}

	if IsMetricsEnabled(dest) {
		metricsPipelineName := "metrics/coralogix-" + dest.GetID()
		currentConfig.Service.Pipelines[metricsPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, metricsPipelineName)
	}

	if IsLoggingEnabled(dest) {
		logsPipelineName := "logs/coralogix-" + dest.GetID()
		currentConfig.Service.Pipelines[logsPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, logsPipelineName)
	}

	return pipelineNames, nil
}
