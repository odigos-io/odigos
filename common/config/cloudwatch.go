package config

import (
	"github.com/odigos-io/odigos/common"
)

const (
	AWS_CLOUDWATCH_TRACES_ENDPOINT  = "AWS_CLOUDWATCH_TRACES_ENDPOINT"
	AWS_CLOUDWATCH_METRICS_ENDPOINT = "AWS_CLOUDWATCH_METRICS_ENDPOINT"
	AWS_CLOUDWATCH_LOGS_ENDPOINT    = "AWS_CLOUDWATCH_LOGS_ENDPOINT"
)

type AWSCloudWatch struct{}

func (m *AWSCloudWatch) DestType() common.DestinationType {
	return common.InstanaDestinationType
}

func (m *AWSCloudWatch) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) ([]string, error) {
	config := dest.GetConfig()
	uniqueUri := "awscloudwatch-" + dest.GetID()
	var pipelineNames []string

	if isTracingEnabled(dest) {
		endpoint, exists := config[AWS_CLOUDWATCH_TRACES_ENDPOINT]
		if !exists {
			return nil, errorMissingKey(AWS_CLOUDWATCH_TRACES_ENDPOINT)
		}

		endpoint, err := parseOtlpHttpEndpoint(endpoint)
		if err != nil {
			return nil, err
		}

		exporterName := "otlphttp/" + uniqueUri
		currentConfig.Exporters[exporterName] = GenericMap{
			"endpoint": endpoint,
		}

		pipeName := "traces/" + uniqueUri
		currentConfig.Service.Pipelines[pipeName] = Pipeline{
			Exporters: []string{exporterName},
		}

		pipelineNames = append(pipelineNames, pipeName)
	}

	if isMetricsEnabled(dest) {
		endpoint, exists := config[AWS_CLOUDWATCH_METRICS_ENDPOINT]
		if !exists {
			return nil, errorMissingKey(AWS_CLOUDWATCH_METRICS_ENDPOINT)
		}

		endpoint, err := parseOtlpHttpEndpoint(endpoint)
		if err != nil {
			return nil, err
		}

		exporterName := "otlphttp/" + uniqueUri
		currentConfig.Exporters[exporterName] = GenericMap{
			"endpoint": endpoint,
		}

		pipeName := "metrics/" + uniqueUri
		currentConfig.Service.Pipelines[pipeName] = Pipeline{
			Exporters: []string{exporterName},
		}

		pipelineNames = append(pipelineNames, pipeName)
	}

	if isLoggingEnabled(dest) {
		endpoint, exists := config[AWS_CLOUDWATCH_LOGS_ENDPOINT]
		if !exists {
			return nil, errorMissingKey(AWS_CLOUDWATCH_LOGS_ENDPOINT)
		}

		endpoint, err := parseOtlpHttpEndpoint(endpoint)
		if err != nil {
			return nil, err
		}

		exporterName := "otlphttp/" + uniqueUri
		currentConfig.Exporters[exporterName] = GenericMap{
			"endpoint": endpoint,
		}

		pipeName := "logs/" + uniqueUri
		currentConfig.Service.Pipelines[pipeName] = Pipeline{
			Exporters: []string{exporterName},
		}

		pipelineNames = append(pipelineNames, pipeName)
	}

	return pipelineNames, nil
}
