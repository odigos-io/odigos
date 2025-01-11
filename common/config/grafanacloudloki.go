package config

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/odigos-io/odigos/common"
)

const (
	grafanaCloudLokiEndpointKey = "GRAFANA_CLOUD_LOKI_ENDPOINT"
	grafanaCloudLokiUsernameKey = "GRAFANA_CLOUD_LOKI_USERNAME"
	grafanaCloudLokiLabelsKey   = "GRAFANA_CLOUD_LOKI_LABELS"
)

type GrafanaCloudLoki struct{}

func (g *GrafanaCloudLoki) DestType() common.DestinationType {
	return common.GrafanaCloudLokiDestinationType
}

func (g *GrafanaCloudLoki) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) error {
	if !isLoggingEnabled(dest) {
		return errors.New("Logging not enabled, gateway will not be configured for grafana cloud Loki")
	}

	lokiUrl, exists := dest.GetConfig()[grafanaCloudLokiEndpointKey]
	if !exists {
		return errors.New("Grafana Cloud Loki endpoint not specified, gateway will not be configured for Loki")
	}

	lokiExporterEndpoint, err := grafanaLokiUrlFromInput(lokiUrl)
	if err != nil {
		return errors.Join(err, errors.New("failed to parse grafana cloud loki endpoint, gateway will not be configured for Loki"))
	}

	lokiUsername, exists := dest.GetConfig()[grafanaCloudLokiUsernameKey]
	if !exists {
		return errors.New("Grafana Cloud Loki username not specified, gateway will not be configured for Loki")
	}

	rawLokiLabels, exists := dest.GetConfig()[grafanaCloudLokiLabelsKey]
	lokiProcessors, err := lokiLabelsProcessors(rawLokiLabels, exists, dest.GetID())
	if err != nil {
		return errors.Join(err, errors.New("failed to parse grafana cloud loki labels, gateway will not be configured for Loki"))
	}

	authExtensionName := "basicauth/grafana" + dest.GetID()
	currentConfig.Extensions[authExtensionName] = GenericMap{
		"client_auth": GenericMap{
			"username": lokiUsername,
			"password": "${GRAFANA_CLOUD_LOKI_PASSWORD}",
		},
	}

	exporterName := "loki/grafana-" + dest.GetID()
	currentConfig.Exporters[exporterName] = GenericMap{
		"endpoint": lokiExporterEndpoint,
		"auth": GenericMap{
			"authenticator": authExtensionName,
		},
	}

	processorNames := []string{}
	for k, v := range lokiProcessors {
		currentConfig.Processors[k] = v
		processorNames = append(processorNames, k)
	}

	logsPipelineName := "logs/grafana-" + dest.GetID()
	currentConfig.Service.Extensions = append(currentConfig.Service.Extensions, authExtensionName)
	currentConfig.Service.Pipelines[logsPipelineName] = Pipeline{
		Processors: processorNames,
		Exporters:  []string{exporterName},
	}

	return nil
}

// to send logs to grafana cloud we use the loki exporter, which uses a loki
// endpoint url like: "https://logs-prod-012.grafana.net/loki/api/v1/push"
// Unfortunately, the grafana cloud website does not provide this url in
// an easily parseable format, so we have to parse it ourselves.
//
// the grafana account page provides the url in the form:
//   - "https://logs-prod-012.grafana.net"
//     for data source, in which case we need to append the path
//   - "https://<User Id>:<Your Grafana.com API Token>@logs-prod-012.grafana.net/loki/api/v1/push"
//     for promtail, in which case we need error as we are expecting this info to be provided as input fields
//
// this function will attempt to parse and prepare the url for use with the
// otelcol loki exporter
func grafanaLokiUrlFromInput(rawUrl string) (string, error) {
	rawUrl = strings.TrimSpace(rawUrl)
	urlWithScheme := rawUrl

	// the user should provide the url with the scheme, but if they don't we add it ourselves
	if !strings.Contains(rawUrl, "://") {
		urlWithScheme = "https://" + rawUrl
	}

	parsedUrl, err := url.Parse(urlWithScheme)
	if err != nil {
		return "", err
	}

	if parsedUrl.Scheme != "https" {
		return "", fmt.Errorf("unexpected scheme %s, only https is supported", parsedUrl.Scheme)
	}

	if parsedUrl.Path == "" {
		parsedUrl.Path = lokiApiPath
	}
	if parsedUrl.Path != lokiApiPath {
		return "", fmt.Errorf("unexpected path for loki endpoint %s", parsedUrl.Path)
	}

	// the username and password should be givin as input fields, and not coded into the url
	if parsedUrl.User != nil {
		return "", fmt.Errorf("unexpected user info for loki endpoint url %s", parsedUrl.User)
	}

	return parsedUrl.String(), nil
}
