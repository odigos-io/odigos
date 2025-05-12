package config

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/odigos-io/odigos/common"
)

const (
	otlpHttpEndpointKey          = "OTLP_HTTP_ENDPOINT"
	otlpHttpTlsKey               = "OTLP_HTTP_TLS_ENABLED"
	otlpHttpCaPemKey             = "OTLP_HTTP_CA_PEM"
	otlpHttpInsecureSkipVerify   = "OTLP_HTTP_INSECURE_SKIP_VERIFY"
	otlpHttpBasicAuthUsernameKey = "OTLP_HTTP_BASIC_AUTH_USERNAME"
	otlpHttpBasicAuthPasswordKey = "OTLP_HTTP_BASIC_AUTH_PASSWORD"
	otlpHttpCompression          = "OTLP_HTTP_COMPRESSION"
	otlpHttpHeaders              = "OTLP_HTTP_HEADERS"
)

type OTLPHttp struct{}

func (g *OTLPHttp) DestType() common.DestinationType {
	return common.OtlpHttpDestinationType
}

func (g *OTLPHttp) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) ([]string, error) {
	config := dest.GetConfig()

	url, exists := config[otlpHttpEndpointKey]
	if !exists {
		return nil, errors.New("OTLP http endpoint not specified, gateway will not be configured for otlp http")
	}

	tls := dest.GetConfig()[otlpHttpTlsKey]
	tlsEnabled := tls == "true"

	parsedUrl, err := parseOtlpHttpEndpoint(url, "", "")
	if err != nil {
		return nil, errors.Join(err, errors.New("otlp http endpoint invalid, gateway will not be configured for otlp http"))
	}

	tlsConfig := GenericMap{
		"insecure": !tlsEnabled,
	}
	caPem, caExists := config[otlpHttpCaPemKey]
	if caExists && caPem != "" {
		tlsConfig["ca_pem"] = caPem
	}
	insecureSkipVerify, skipExists := config[otlpHttpInsecureSkipVerify]
	if skipExists && insecureSkipVerify != "" {
		tlsConfig["insecure_skip_verify"] = parseBool(insecureSkipVerify)
	}

	basicAuthExtensionName, basicAuthExtensionConf := applyBasicAuth(dest)

	// add authenticator extension
	if basicAuthExtensionName != "" && basicAuthExtensionConf != nil {
		currentConfig.Extensions[basicAuthExtensionName] = *basicAuthExtensionConf
		currentConfig.Service.Extensions = append(currentConfig.Service.Extensions, basicAuthExtensionName)
	}

	otlpHttpExporterName := "otlphttp/generic-" + dest.GetID()
	exporterConf := GenericMap{
		"endpoint": parsedUrl,
		"tls":      tlsConfig,
	}
	if basicAuthExtensionName != "" {
		exporterConf["auth"] = GenericMap{
			"authenticator": basicAuthExtensionName,
		}
	}
	if compression, ok := config[otlpHttpCompression]; ok {
		exporterConf["compression"] = compression
	}

	headers, exists := config[otlpHttpHeaders]
	if exists && headers != "" {
		var headersList []struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		}
		err := json.Unmarshal([]byte(headers), &headersList)
		if err != nil {
			return nil, errors.Join(err, errors.New(
				"failed to parse otlphttp destination OTLP_HTTP_HEADERS parameter as json string in the form {key: string, value: string}[]",
			))
		}
		mappedHeaders := map[string]string{}
		for _, header := range headersList {
			mappedHeaders[header.Key] = header.Value
		}
		exporterConf["headers"] = mappedHeaders
	}
	currentConfig.Exporters[otlpHttpExporterName] = exporterConf

	var pipelineNames []string
	if isTracingEnabled(dest) {
		tracesPipelineName := "traces/otlphttp-" + dest.GetID()
		currentConfig.Service.Pipelines[tracesPipelineName] = Pipeline{
			Exporters: []string{otlpHttpExporterName},
		}
		pipelineNames = append(pipelineNames, tracesPipelineName)
	}

	if isMetricsEnabled(dest) {
		metricsPipelineName := "metrics/otlphttp-" + dest.GetID()
		currentConfig.Service.Pipelines[metricsPipelineName] = Pipeline{
			Exporters: []string{otlpHttpExporterName},
		}
		pipelineNames = append(pipelineNames, metricsPipelineName)
	}

	if isLoggingEnabled(dest) {
		logsPipelineName := "logs/otlphttp-" + dest.GetID()
		currentConfig.Service.Pipelines[logsPipelineName] = Pipeline{
			Exporters: []string{otlpHttpExporterName},
		}
		pipelineNames = append(pipelineNames, logsPipelineName)
	}

	return pipelineNames, nil
}

func applyBasicAuth(dest ExporterConfigurer) (extensionName string, extensionConf *GenericMap) {
	username := dest.GetConfig()[otlpHttpBasicAuthUsernameKey]
	if username == "" {
		return "", nil
	}

	extensionName = "basicauth/otlphttp-" + dest.GetID()
	extensionConf = &GenericMap{
		"client_auth": GenericMap{
			"username": username,
			"password": fmt.Sprintf("${%s}", otlpHttpBasicAuthPasswordKey),
		},
	}

	return extensionName, extensionConf
}
