package config

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	commonconf "github.com/keyval-dev/odigos/autoscaler/controllers/common"
	"github.com/keyval-dev/odigos/common"
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

func (g *OTLPHttp) ModifyConfig(dest *odigosv1.Destination, currentConfig *commonconf.Config) error {

	url, exists := dest.Spec.Data[otlpHttpEndpointKey]
	if !exists {
		return errors.New("OTLP http endpoint not specified, gateway will not be configured for otlp http")
	}

	parsedUrl, err := parseOtlpHttpEndpoint(url)
	if err != nil {
		return errors.Join(err, errors.New("otlp http endpoint invalid, gateway will not be configured for otlp http"))
	}

	basicAuthExtensionName, basicAuthExtensionConf, err := applyBasicAuth(dest)
	if err != nil {
		return errors.Join(err, errors.New("failed to apply basic auth to otlp http exporter"))
	}

	// add authenticator extension
	if basicAuthExtensionName != "" && basicAuthExtensionConf != nil {
		currentConfig.Extensions[basicAuthExtensionName] = *basicAuthExtensionConf
		currentConfig.Service.Extensions = append(currentConfig.Service.Extensions, basicAuthExtensionName)
	}

	otlpHttpExporterName := "otlphttp/generic-" + dest.Name
	exporterConf := commonconf.GenericMap{
		"endpoint": parsedUrl,
	}
	if basicAuthExtensionName != "" {
		exporterConf["auth"] = commonconf.GenericMap{
			"authenticator": basicAuthExtensionName,
		}
	}
	currentConfig.Exporters[otlpHttpExporterName] = exporterConf

	if isTracingEnabled(dest) {
		tracesPipelineName := "traces/otlphttp-" + dest.Name
		currentConfig.Service.Pipelines[tracesPipelineName] = commonconf.Pipeline{
			Exporters: []string{otlpHttpExporterName},
		}
	}

	if isMetricsEnabled(dest) {
		metricsPipelineName := "metrics/otlphttp-" + dest.Name
		currentConfig.Service.Pipelines[metricsPipelineName] = commonconf.Pipeline{
			Exporters: []string{otlpHttpExporterName},
		}
	}

	if isLoggingEnabled(dest) {
		logsPipelineName := "logs/otlphttp-" + dest.Name
		currentConfig.Service.Pipelines[logsPipelineName] = commonconf.Pipeline{
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

func applyBasicAuth(dest *odigosv1.Destination) (extensionName string, extensionConf *commonconf.GenericMap, err error) {

	username := dest.Spec.Data[otlpHttpBasicAuthUsernameKey]
	if username == "" {
		return "", nil, nil
	}

	extensionName = "basicauth/otlphttp-" + dest.Name
	extensionConf = &commonconf.GenericMap{
		"client_auth": commonconf.GenericMap{
			"username": username,
			"password": fmt.Sprintf("${%s}", otlpHttpBasicAuthPasswordKey),
		},
	}

	return extensionName, extensionConf, nil
}
