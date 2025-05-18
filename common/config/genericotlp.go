package config

import (
	"encoding/json"
	"errors"

	"github.com/odigos-io/odigos/common"
)

const (
	genericOtlpUrlKey             = "OTLP_GRPC_ENDPOINT"
	genericOtlpTlsKey             = "OTLP_GRPC_TLS_ENABLED"
	genericOtlpCaPemKey           = "OTLP_GRPC_CA_PEM"
	genericOtlpInsecureSkipVerify = "OTLP_GRPC_INSECURE_SKIP_VERIFY"
	otlpGrpcCompression           = "OTLP_GRPC_COMPRESSION"
	otlpGrpcHeaders               = "OTLP_GRPC_HEADERS"
)

type GenericOTLP struct{}

func (g *GenericOTLP) DestType() common.DestinationType {
	return common.GenericOTLPDestinationType
}

func (g *GenericOTLP) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) ([]string, error) {
	config := dest.GetConfig()

	url, exists := config[genericOtlpUrlKey]
	if !exists {
		return nil, errors.New("generic OTLP gRPC endpoint not specified, gateway will not be configured for otlp")
	}

	tls := dest.GetConfig()[genericOtlpTlsKey]
	tlsEnabled := tls == "true"

	grpcEndpoint, err := parseOtlpGrpcUrl(url, tlsEnabled)
	if err != nil {
		return nil, errorMissingKey(genericOtlpUrlKey)
	}

	tlsConfig := GenericMap{
		"insecure": !tlsEnabled,
	}
	caPem, caExists := config[genericOtlpCaPemKey]
	if caExists && caPem != "" {
		tlsConfig["ca_pem"] = caPem
	}
	insecureSkipVerify, skipExists := config[genericOtlpInsecureSkipVerify]
	if skipExists && insecureSkipVerify != "" {
		tlsConfig["insecure_skip_verify"] = parseBool(insecureSkipVerify)
	}

	genericOtlpExporterName := "otlp/generic-" + dest.GetID()
	exporterConf := GenericMap{
		"endpoint": grpcEndpoint,
		"tls":      tlsConfig,
	}

	if compression, ok := config[otlpGrpcCompression]; ok {
		exporterConf["compression"] = compression
	}

	headers, exists := config[otlpGrpcHeaders]
	if exists && headers != "" {
		var headersList []struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		}
		err := json.Unmarshal([]byte(headers), &headersList)
		if err != nil {
			return nil, errors.Join(err, errors.New(
				"failed to parse otlpGrpc destination OTLP_GRPC_HEADERS parameter as json string in the form {key: string, value: string}[]",
			))
		}
		mappedHeaders := map[string]string{}
		for _, header := range headersList {
			mappedHeaders[header.Key] = header.Value
		}
		exporterConf["headers"] = mappedHeaders
	}
	currentConfig.Exporters[genericOtlpExporterName] = exporterConf

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
