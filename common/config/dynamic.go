package config

import (
	"errors"

	"github.com/odigos-io/odigos/common"
	"gopkg.in/yaml.v2"
)

const (
	DestinationName   = "DYNAMIC_DESTINATION_NAME"
	DestinationType   = "DYNAMIC_DESTINATION_TYPE"
	ConfigurationData = "DYNAMIC_CONFIGURATION_DATA"
)

var (
	ErrorDynamicMissingName       = errors.New("Dynamic destination is missing a required field (\"DYNAMIC_DESTINATION_NAME\"), Dynamic destination will not be configured")
	ErrorDynamicMissingType       = errors.New("Dynamic destination is missing a required field (\"DYNAMIC_DESTINATION_TYPE\"), Dynamic destination will not be configured")
	ErrorDynamicMissingConfData   = errors.New("Dynamic destination is missing a required field (\"DYNAMIC_CONFIGURATION_DATA\"), Dynamic destination will not be configured")
	ErrorDynamicTracingDisabled   = errors.New("Dynamic destination is missing a required field (\"TRACES\"), Dynamic destination will not be configured")
	ErrorDynamicMetricsNotAllowed = errors.New("Dynamic destination has a forbidden field (\"METRICS\"), Dynamic destination will not be configured")
	ErrorDynamicLogsNotAllowed    = errors.New("Dynamic destination has a forbidden field (\"LOGS\"), Dynamic destination will not be configured")
)

type Dynamic struct{}

// compile time checks
var _ Configer = (*Dynamic)(nil)

func (g *Dynamic) DestType() common.DestinationType {
	return common.DynamicDestinationType
}

func (g *Dynamic) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) ([]string, error) {
	config := dest.GetConfig()

	DynamicData, exists := config[ConfigurationData]
	if !exists {
		return nil, ErrorDynamicMissingConfData
	}

	var parsedConfig map[string]interface{}
	err := yaml.Unmarshal([]byte(DynamicData), &parsedConfig)
	if err != nil {
		return nil, err
	}

	exporterName := config[DestinationType] + "/" + dest.GetID()
	currentConfig.Exporters[exporterName] = parsedConfig

	var pipelineNames []string

	if isTracingEnabled(dest) {
		tracesPipelineName := "traces/" + dest.GetID()
		currentConfig.Service.Pipelines[tracesPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, tracesPipelineName)
	}

	if isMetricsEnabled(dest) {
		metricsPipelineName := "metrics/" + dest.GetID()
		currentConfig.Service.Pipelines[metricsPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, metricsPipelineName)
	}

	if isLoggingEnabled(dest) {
		logsPipelineName := "logs/" + dest.GetID()
		currentConfig.Service.Pipelines[logsPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, logsPipelineName)
	}

	return pipelineNames, nil
}
