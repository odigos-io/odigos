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

func (g *GenericOTLP) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) error {

	url, exists := dest.GetConfig()[genericOtlpUrlKey]
	if !exists {
		return errors.New("Generic OTLP gRPC endpoint not specified, gateway will not be configured for otlp")
	}

	grpcEndpoint, err := parseOtlpGrpcUrl(url, false)
	if err != nil {
		return errors.Join(err, errors.New("otlp endpoint invalid, gateway will not be configured for otlp"))
	}

	genericOtlpExporterName := "otlp/generic-" + dest.GetID()
	currentConfig.Exporters[genericOtlpExporterName] = GenericMap{
		"endpoint": grpcEndpoint,
		"tls": GenericMap{
			"insecure": true,
		},
	}

	if isTracingEnabled(dest) {
		tracesPipelineName := "traces/generic-" + dest.GetID()
		currentConfig.Service.Pipelines[tracesPipelineName] = Pipeline{
			Exporters: []string{genericOtlpExporterName},
		}
	}

	if isMetricsEnabled(dest) {
		metricsPipelineName := "metrics/generic-" + dest.GetID()
		currentConfig.Service.Pipelines[metricsPipelineName] = Pipeline{
			Exporters: []string{genericOtlpExporterName},
		}
	}

	if isLoggingEnabled(dest) {
		logsPipelineName := "logs/generic-" + dest.GetID()
		currentConfig.Service.Pipelines[logsPipelineName] = Pipeline{
			Exporters: []string{genericOtlpExporterName},
		}
	}

	return nil
}
