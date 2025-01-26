package config

import (
	"errors"

	"github.com/odigos-io/odigos/common"
)

const (
	genericOtlpUrlKey = "OTLP_GRPC_ENDPOINT"
)

type GenericOTLP struct{}

func (g *GenericOTLP) DestType() common.DestinationType {
	return common.GenericOTLPDestinationType
}

func (g *GenericOTLP) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) ([]string, error) {
	url, exists := dest.GetConfig()[genericOtlpUrlKey]
	if !exists {
		return nil, errors.New("Generic OTLP gRPC endpoint not specified, gateway will not be configured for otlp")
	}

	grpcEndpoint, err := parseOtlpGrpcUrl(url, false)
	if err != nil {
		return nil, errors.Join(err, errors.New("otlp endpoint invalid, gateway will not be configured for otlp"))
	}

	genericOtlpExporterName := "otlp/generic-" + dest.GetID()
	currentConfig.Exporters[genericOtlpExporterName] = GenericMap{
		"endpoint": grpcEndpoint,
		"tls": GenericMap{
			"insecure": true,
		},
	}
	var pipelineNames []string
	if isTracingEnabled(dest) {
		tracesPipelineName := "traces/generic-" + dest.GetID()
		currentConfig.Service.Pipelines[tracesPipelineName] = Pipeline{
			Exporters: []string{genericOtlpExporterName},
		}
		pipelineNames = append(pipelineNames, tracesPipelineName)
	}

	if isMetricsEnabled(dest) {
		metricsPipelineName := "metrics/generic-" + dest.GetID()
		currentConfig.Service.Pipelines[metricsPipelineName] = Pipeline{
			Exporters: []string{genericOtlpExporterName},
		}
		pipelineNames = append(pipelineNames, metricsPipelineName)
	}

	if isLoggingEnabled(dest) {
		logsPipelineName := "logs/generic-" + dest.GetID()
		currentConfig.Service.Pipelines[logsPipelineName] = Pipeline{
			Exporters: []string{genericOtlpExporterName},
		}
		pipelineNames = append(pipelineNames, logsPipelineName)
	}

	return pipelineNames, nil
}
