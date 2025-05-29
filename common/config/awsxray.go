package config

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/odigos-io/odigos/common"
)

const (
	AWS_XRAY_REGION                     = "AWS_XRAY_REGION"
	AWS_XRAY_ENDPOINT                   = "AWS_XRAY_ENDPOINT"
	AWS_XRAY_PROXY_ADDRESS              = "AWS_XRAY_PROXY_ADDRESS"
	AWS_XRAY_DISABLE_SSL                = "AWS_XRAY_DISABLE_SSL"
	AWS_XRAY_LOCAL_MODE                 = "AWS_XRAY_LOCAL_MODE"
	AWS_XRAY_RESOURCE_ARN               = "AWS_XRAY_RESOURCE_ARN"
	AWS_XRAY_ROLE_ARN                   = "AWS_XRAY_ROLE_ARN"
	AWS_XRAY_EXTERNAL_ID                = "AWS_XRAY_EXTERNAL_ID"
	AWS_XRAY_INDEX_ALL_ATTRIBUTES       = "AWS_XRAY_INDEX_ALL_ATTRIBUTES"
	AWS_XRAY_INDEXED_ATTRIBUTES         = "AWS_XRAY_INDEXED_ATTRIBUTES"
	AWS_XRAY_LOG_GROUPS                 = "AWS_XRAY_LOG_GROUPS"
	AWS_XRAY_TELEMETRY_ENABLED          = "AWS_XRAY_TELEMETRY_ENABLED"
	AWS_XRAY_TELEMETRY_INCLUDE_METADATA = "AWS_XRAY_TELEMETRY_INCLUDE_METADATA"
	AWS_XRAY_TELEMETRY_HOSTNAME         = "AWS_XRAY_TELEMETRY_HOSTNAME"
	AWS_XRAY_TELEMETRY_INSTANCE_ID      = "AWS_XRAY_TELEMETRY_INSTANCE_ID"
	AWS_XRAY_TELEMETRY_RESOURCE_ARN     = "AWS_XRAY_TELEMETRY_RESOURCE_ARN"
	AWS_XRAY_TELEMETRY_CONTRIBUTORS     = "AWS_XRAY_TELEMETRY_CONTRIBUTORS"
)

type AWSXRay struct{}

func (m *AWSXRay) DestType() common.DestinationType {
	return common.AWSXRayDestinationType
}

func (m *AWSXRay) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) ([]string, error) {
	config := dest.GetConfig()
	uniqueUri := "awsxray-" + dest.GetID()
	exporterName := "awsxray/" + uniqueUri
	exporterConfig := make(GenericMap)
	var pipelineNames []string

	// Map string keys to their config fields
	stringFields := map[string]string{
		AWS_XRAY_REGION:                 "region",
		AWS_XRAY_ENDPOINT:               "endpoint",
		AWS_XRAY_PROXY_ADDRESS:          "proxy_address",
		AWS_XRAY_RESOURCE_ARN:           "resource_arn",
		AWS_XRAY_ROLE_ARN:               "role_arn",
		AWS_XRAY_EXTERNAL_ID:            "external_id",
		AWS_XRAY_TELEMETRY_HOSTNAME:     "telemetry.hostname",
		AWS_XRAY_TELEMETRY_INSTANCE_ID:  "telemetry.instance_id",
		AWS_XRAY_TELEMETRY_RESOURCE_ARN: "telemetry.resource_arn",
	}

	// Map boolean keys to their config fields
	boolFields := map[string]string{
		AWS_XRAY_DISABLE_SSL:                "no_verify_ssl",
		AWS_XRAY_LOCAL_MODE:                 "local_mode",
		AWS_XRAY_INDEX_ALL_ATTRIBUTES:       "index_all_attributes",
		AWS_XRAY_TELEMETRY_ENABLED:          "telemetry.enabled",
		AWS_XRAY_TELEMETRY_INCLUDE_METADATA: "telemetry.include_metadata",
	}

	// Apply string fields
	for key, field := range stringFields {
		if value, exists := config[key]; exists {
			setNestedMapValue(exporterConfig, field, value)
		}
	}

	// Apply boolean fields
	for key, field := range boolFields {
		if value, exists := config[key]; exists {
			setNestedMapValue(exporterConfig, field, parseBool(value))
		}
	}

	// Handle JSON-encoded list fields
	jsonFields := map[string]string{
		AWS_XRAY_INDEXED_ATTRIBUTES:     "indexed_attributes",
		AWS_XRAY_LOG_GROUPS:             "aws_log_groups",
		AWS_XRAY_TELEMETRY_CONTRIBUTORS: "telemetry.contributors",
	}

	for key, field := range jsonFields {
		if err := parseJSONStringArray(config, key, exporterConfig, field); err != nil {
			return nil, err
		}
	}

	currentConfig.Exporters[exporterName] = exporterConfig

	// Configure tracing pipeline
	if IsTracingEnabled(dest) {
		pipeName := "traces/" + uniqueUri
		currentConfig.Service.Pipelines[pipeName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, pipeName)
	}

	return pipelineNames, nil
}

// **Helper Functions**

// setNestedMapValue sets a nested field in a map using dot notation.
func setNestedMapValue(m GenericMap, field string, value interface{}) {
	parts := splitField(field)
	subMap := m

	// Traverse the map structure
	for i, part := range parts {
		if i == len(parts)-1 {
			subMap[part] = value
		} else {
			if _, exists := subMap[part]; !exists {
				subMap[part] = make(GenericMap)
			}
			subMap = subMap[part].(GenericMap)
		}
	}
}

// splitField splits a field path into its nested parts (e.g., "telemetry.enabled" → ["telemetry", "enabled"])
func splitField(field string) []string {
	return strings.Split(field, ".")
}

// parseJSONStringArray extracts and sets a JSON-encoded string array in the destination map.
func parseJSONStringArray(config map[string]string, key string, dest GenericMap, field string) error {
	value, exists := config[key]
	if !exists {
		return nil
	}

	var list []string
	if err := json.Unmarshal([]byte(value), &list); err != nil {
		return errors.Join(err, errors.New(
			"failed to parse AWS X-Ray destination parameter \""+key+"\" as JSON string in the format: string[]",
		))
	}

	setNestedMapValue(dest, field, list)
	return nil
}
