package config

import (
	"errors"

	"github.com/odigos-io/odigos/common"
	"gopkg.in/yaml.v2"
)

const (
	DestinationName   = "GENERIC_DESTINATION_NAME"
	DestinationType   = "GENERIC_DESTINATION_TYPE"
	ConfigurationData = "GENERIC_CONFIGURATION_DATA"
)

var (
	ErrorGenericMissingName       = errors.New("Generic destination is missing a required field (\"GENERIC_DESTINATION_NAME\"), Generic destination will not be configured")
	ErrorGenericMissingType       = errors.New("Generic destination is missing a required field (\"GENERIC_DESTINATION_TYPE\"), Generic destination will not be configured")
	ErrorGenericMissingConfData   = errors.New("Generic destination is missing a required field (\"GENERIC_CONFIGURATION_DATA\"), Generic destination will not be configured")
	ErrorGenericTracingDisabled   = errors.New("Generic destination is missing a required field (\"TRACES\"), Generic destination will not be configured")
	ErrorGenericMetricsNotAllowed = errors.New("Generic destination has a forbidden field (\"METRICS\"), Generic destination will not be configured")
	ErrorGenericLogsNotAllowed    = errors.New("Generic destination has a forbidden field (\"LOGS\"), Generic destination will not be configured")
)

type Generic struct{}

// compile time checks
var _ Configer = (*Generic)(nil)

func (g *Generic) DestType() common.DestinationType {
	return common.GenericDestinationType
}

func (g *Generic) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) ([]string, error) {
	config := dest.GetConfig()

	// destinationType, exists := config[DestinationType]
	// if !exists {
	// 	return nil, ErrorGenericMissingType
	// }
	exporterName := "generic/" + dest.GetID()

	genericData, exists := config[ConfigurationData]
	if !exists {
		return nil, ErrorGenericMissingConfData
	}

	var parsedConfig map[string]interface{}
	err := yaml.Unmarshal([]byte(genericData), &parsedConfig)
	if err != nil {
		return nil, err
	}

	// Attempt to assert genericData to GenericMap (map[string]interface{})
	//genericMap, ok := genericData.(GenericMap)
	// if !ok {
	// 	// If the type assertion fails, return an error
	// 	return nil, fmt.Errorf("expected %v to be of type GenericMap, but got %T", ConfigurationData, genericData)
	// }

	currentConfig.Exporters[exporterName] = parsedConfig

	pipelineNames := []string{}
	if isTracingEnabled(dest) {
		return nil, ErrorGenericTracingDisabled
	}

	if isMetricsEnabled(dest) {
		return nil, ErrorGenericMetricsNotAllowed
	}

	if isLoggingEnabled(dest) {
		return nil, ErrorGenericLogsNotAllowed
	}

	return pipelineNames, nil
}
