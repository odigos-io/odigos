package config

import (
	"encoding/json"
	"errors"

	"github.com/odigos-io/odigos/common"
)

const (
	AWS_XRAY_REGION               = "AWS_XRAY_REGION"
	AWS_XRAY_ENDPOINT             = "AWS_XRAY_ENDPOINT"
	AWS_XRAY_PROXY_ADDRESS        = "AWS_XRAY_PROXY_ADDRESS"
	AWS_XRAY_DISABLE_SSL          = "AWS_XRAY_DISABLE_SSL"
	AWS_XRAY_LOCAL_MODE           = "AWS_XRAY_LOCAL_MODE"
	AWS_XRAY_RESOURCE_ARN         = "AWS_XRAY_RESOURCE_ARN"
	AWS_XRAY_ROLE_ARN             = "AWS_XRAY_ROLE_ARN"
	AWS_XRAY_EXTERNAL_ID          = "AWS_XRAY_EXTERNAL_ID"
	AWS_XRAY_INDEX_ALL_ATTRIBUTES = "AWS_XRAY_INDEX_ALL_ATTRIBUTES"
	AWS_XRAY_INDEXED_ATTRIBUTES   = "AWS_XRAY_INDEXED_ATTRIBUTES"
	AWS_XRAY_LOG_GROUPS           = "AWS_XRAY_LOG_GROUPS"
)

type AWSXRay struct{}

func (m *AWSXRay) DestType() common.DestinationType {
	return common.AWSXRayDestinationType
}

func (m *AWSXRay) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) ([]string, error) {
	config := dest.GetConfig()
	uniqueUri := "awsxray-" + dest.GetID()
	var pipelineNames []string

	exporterName := "awsxray/" + uniqueUri
	exporterConfig := GenericMap{}

	region, exists := config[AWS_XRAY_REGION]
	if exists {
		exporterConfig["region"] = region
	}

	endpoint, exists := config[AWS_XRAY_ENDPOINT]
	if exists {
		exporterConfig["endpoint"] = endpoint
	}

	proxyAddress, exists := config[AWS_XRAY_PROXY_ADDRESS]
	if exists {
		exporterConfig["proxy_address"] = proxyAddress
	}

	disableSsl, exists := config[AWS_XRAY_DISABLE_SSL]
	if exists {
		exporterConfig["no_verify_ssl"] = parseBool(disableSsl)
	}

	localMode, exists := config[AWS_XRAY_LOCAL_MODE]
	if exists {
		exporterConfig["local_mode"] = parseBool(localMode)
	}

	resourceArn, exists := config[AWS_XRAY_RESOURCE_ARN]
	if exists {
		exporterConfig["resource_arn"] = resourceArn
	}

	roleArn, exists := config[AWS_XRAY_ROLE_ARN]
	if exists {
		exporterConfig["role_arn"] = roleArn
	}

	externalId, exists := config[AWS_XRAY_EXTERNAL_ID]
	if exists {
		exporterConfig["external_id"] = externalId
	}

	indexAllAttributes, exists := config[AWS_XRAY_INDEX_ALL_ATTRIBUTES]
	if exists {
		exporterConfig["index_all_attributes"] = parseBool(indexAllAttributes)
	}

	indexedAttributes, exists := config[AWS_XRAY_INDEXED_ATTRIBUTES]
	if exists {
		var list []string

		err := json.Unmarshal([]byte(indexedAttributes), &list)
		if err != nil {
			return nil, errors.Join(err, errors.New(
				"failed to parse AWS X-Ray destination parameter \"AWS_XRAY_INDEXED_ATTRIBUTES\" as JSON string in the format: string[]",
			))
		}

		exporterConfig["indexed_attributes"] = indexedAttributes
	}

	logGroups, exists := config[AWS_XRAY_LOG_GROUPS]
	if exists {
		var list []string

		err := json.Unmarshal([]byte(logGroups), &list)
		if err != nil {
			return nil, errors.Join(err, errors.New(
				"failed to parse AWS X-Ray destination parameter \"AWS_XRAY_LOG_GROUPS\" as JSON string in the format: string[]",
			))
		}

		exporterConfig["aws_log_groups"] = list
	}

	currentConfig.Exporters[exporterName] = exporterConfig

	if isTracingEnabled(dest) {
		pipeName := "traces/" + uniqueUri
		currentConfig.Service.Pipelines[pipeName] = Pipeline{
			Exporters: []string{exporterName},
		}

		pipelineNames = append(pipelineNames, pipeName)
	}

	return pipelineNames, nil
}
