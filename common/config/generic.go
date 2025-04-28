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

	// destinationType, exists := config[DestinationType]
	// if !exists {
	// 	return nil, ErrorDynamicMissingType
	// }
	exporterName := "dynamic/" + dest.GetID()

	DynamicData, exists := config[ConfigurationData]
	if !exists {
		return nil, ErrorDynamicMissingConfData
	}

	var parsedConfig map[string]interface{}
	err := yaml.Unmarshal([]byte(DynamicData), &parsedConfig)
	if err != nil {
		return nil, err
	}

	// Attempt to assert DynamicData to DynamicMap (map[string]interface{})
	//DynamicMap, ok := DynamicData.(DynamicMap)
	// if !ok {
	// 	// If the type assertion fails, return an error
	// 	return nil, fmt.Errorf("expected %v to be of type DynamicMap, but got %T", ConfigurationData, DynamicData)
	// }

	currentConfig.Exporters[exporterName] = parsedConfig

	pipelineNames := []string{}
	if isTracingEnabled(dest) {
		return nil, ErrorDynamicTracingDisabled
	}

	if isMetricsEnabled(dest) {
		return nil, ErrorDynamicMetricsNotAllowed
	}

	if isLoggingEnabled(dest) {
		return nil, ErrorDynamicLogsNotAllowed
	}

	return pipelineNames, nil
}
