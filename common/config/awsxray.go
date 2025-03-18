package config

import (
	"github.com/odigos-io/odigos/common"
)

const (
	AWS_XRAY_REGION        = "AWS_XRAY_REGION"
	AWS_XRAY_ENDPOINT      = "AWS_XRAY_ENDPOINT"
	AWS_XRAY_PROXY_ADDRESS = "AWS_XRAY_PROXY_ADDRESS"
	AWS_XRAY_DISABLE_SSL   = "AWS_XRAY_DISABLE_SSL"
	AWS_XRAY_LOCAL_MODE    = "AWS_XRAY_LOCAL_MODE"
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
