package config

import (
	"gopkg.in/yaml.v2"

	"github.com/odigos-io/odigos/common"
)

const (
	destinationTypeKey   = "DYNAMIC_DESTINATION_TYPE"
	configurationDataKey = "DYNAMIC_CONFIGURATION_DATA"
)

type Dynamic struct{}

// compile time checks
var _ Configer = (*Dynamic)(nil)

func (g *Dynamic) DestType() common.DestinationType {
	return common.DynamicDestinationType
}

func (g *Dynamic) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) ([]string, error) {
	config := dest.GetConfig()

	dynamicConfData, exists := config[configurationDataKey]
	if !exists {
		return nil, errorMissingKey(configurationDataKey)
	}

	var parsedConfig map[string]interface{}
	err := yaml.Unmarshal([]byte(dynamicConfData), &parsedConfig)
	if err != nil {
		return nil, err
	}

	destinationType, exists := config[destinationTypeKey]
	if !exists {
		return nil, errorMissingKey(destinationType)
	}

	exporterName := destinationType + "/" + dest.GetID()
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
