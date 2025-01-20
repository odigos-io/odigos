package config

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/odigos-io/odigos/common"
)

const (
	otlpHttpEndpointKey          = "OTLP_HTTP_ENDPOINT"
	otlpHttpBasicAuthUsernameKey = "OTLP_HTTP_BASIC_AUTH_USERNAME"
	otlpHttpBasicAuthPasswordKey = "OTLP_HTTP_BASIC_AUTH_PASSWORD"
)

type OTLPHttp struct{}

func (g *OTLPHttp) DestType() common.DestinationType {
	return common.OtlpHttpDestinationType
}

func (g *OTLPHttp) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) error {
	url, exists := dest.GetConfig()[otlpHttpEndpointKey]
	if !exists {
		return errors.New("OTLP http endpoint not specified, gateway will not be configured for otlp http")
	}

	parsedUrl, err := parseOtlpHttpEndpoint(url)
	if err != nil {
		return errors.Join(err, errors.New("otlp http endpoint invalid, gateway will not be configured for otlp http"))
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
	}
	if basicAuthExtensionName != "" {
		exporterConf["auth"] = GenericMap{
			"authenticator": basicAuthExtensionName,
		}
	}
	currentConfig.Exporters[otlpHttpExporterName] = exporterConf

	if isTracingEnabled(dest) {
		tracesPipelineName := "traces/otlphttp-" + dest.GetID()
		currentConfig.Service.Pipelines[tracesPipelineName] = Pipeline{
			Exporters: []string{otlpHttpExporterName},
		}
	}

	if isMetricsEnabled(dest) {
		metricsPipelineName := "metrics/otlphttp-" + dest.GetID()
		currentConfig.Service.Pipelines[metricsPipelineName] = Pipeline{
			Exporters: []string{otlpHttpExporterName},
		}
	}

	if isLoggingEnabled(dest) {
		logsPipelineName := "logs/otlphttp-" + dest.GetID()
		currentConfig.Service.Pipelines[logsPipelineName] = Pipeline{
			Exporters: []string{otlpHttpExporterName},
		}
	}

	return nil
}

func parseOtlpHttpEndpoint(rawUrl string) (string, error) {
	noWhiteSpaces := strings.TrimSpace(rawUrl)
	parsedUrl, err := url.Parse(noWhiteSpaces)
	if err != nil {
		return "", fmt.Errorf("failed to parse otlp http endpoint: %w", err)
	}

	if parsedUrl.Scheme != "http" && parsedUrl.Scheme != "https" {
		return "", fmt.Errorf("invalid otlp http endpoint scheme: %s", parsedUrl.Scheme)
	}

	return noWhiteSpaces, nil
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
