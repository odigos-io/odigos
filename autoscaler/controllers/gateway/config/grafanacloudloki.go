package config

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	commonconf "github.com/keyval-dev/odigos/autoscaler/controllers/common"
	"github.com/keyval-dev/odigos/common"
	"sigs.k8s.io/controller-runtime/pkg/log"
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

func (g *GrafanaCloudLoki) ModifyConfig(dest *odigosv1.Destination, currentConfig *commonconf.Config) {

	if !isLoggingEnabled(dest) {
		log.Log.V(0).Info("Logging not enabled, gateway will not be configured for grafana cloud Loki")
		return
	}

	lokiUrl, exists := dest.Spec.Data[grafanaCloudLokiEndpointKey]
	if !exists {
		log.Log.V(0).Info("Grafana Cloud Loki endpoint not specified, gateway will not be configured for Loki")
		return
	}

	lokiExporterEndpoint, err := grafanaLokiUrlFromInput(lokiUrl)
	if err != nil {
		log.Log.Error(err, "failed to parse grafana cloud loki endpoint, gateway will not be configured for Loki")
		return
	}

	lokiUsername, exists := dest.Spec.Data[grafanaCloudLokiUsernameKey]
	if !exists {
		log.Log.V(0).Info("Grafana Cloud Loki username not specified, gateway will not be configured for Loki")
		return
	}

	rawLokiLabels, exists := dest.Spec.Data[grafanaCloudLokiLabelsKey]
	lokiProcessors, err := lokiLabelsProcessors(rawLokiLabels, exists, dest.Name)
	if err != nil {
		log.Log.Error(err, "failed to parse grafana cloud loki labels, gateway will not be configured for Loki")
		return
	}

	authExtensionName := "basicauth/grafana" + dest.Name
	currentConfig.Extensions[authExtensionName] = commonconf.GenericMap{
		"client_auth": commonconf.GenericMap{
			"username": lokiUsername,
			"password": "${GRAFANA_CLOUD_LOKI_PASSWORD}",
		},
	}

	exporterName := "loki/grafana-" + dest.Name
	currentConfig.Exporters[exporterName] = commonconf.GenericMap{
		"endpoint": lokiExporterEndpoint,
		"auth": commonconf.GenericMap{
			"authenticator": authExtensionName,
		},
	}

	processorNames := []string{}
	for k, v := range lokiProcessors {
		currentConfig.Processors[k] = v
		processorNames = append(processorNames, k)
	}

	logsPipelineName := "logs/grafana-" + dest.Name
	currentConfig.Service.Extensions = append(currentConfig.Service.Extensions, authExtensionName)
	currentConfig.Service.Pipelines[logsPipelineName] = commonconf.Pipeline{
		Receivers:  []string{"otlp"},
		Processors: append([]string{"batch"}, processorNames...),
		Exporters:  []string{exporterName},
	}

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
		parsedUrl.Path = "/loki/api/v1/push"
	}
	if parsedUrl.Path != "/loki/api/v1/push" {
		return "", fmt.Errorf("unexpected path for loki endpoint %s", parsedUrl.Path)
	}

	// the username and password should be givin as input fields, and not coded into the url
	if parsedUrl.User != nil {
		return "", fmt.Errorf("unexpected user info for loki endpoint url %s", parsedUrl.User)
	}

	return parsedUrl.String(), nil
}

func lokiLabelsProcessors(rawLabels string, exists bool, destName string) (commonconf.GenericMap, error) {

	// backwards compatibility, if the user labels are not provided, we use the default
	if !exists {
		processorName := "attributes/grafana-" + destName
		return commonconf.GenericMap{
			processorName: commonconf.GenericMap{
				"actions": []commonconf.GenericMap{
					{
						"key":    "loki.attribute.labels",
						"action": "insert",
						"value":  "k8s.container.name, k8s.pod.name, k8s.namespace.name",
					},
				},
			},
		}, nil
	}

	// no labels. not recommended, but ok
	if rawLabels == "" || rawLabels == "[]" {
		return commonconf.GenericMap{}, nil
	}

	var attributeNames []string
	err := json.Unmarshal([]byte(rawLabels), &attributeNames)
	if err != nil {
		return nil, err
	}
	attributeHint := strings.Join(attributeNames, ", ")

	processors := commonconf.GenericMap{}

	// since we don't know if the attributes are logs attributes or resource attributes, we will add them to both processors
	attributesProcessorName := "attributes/grafana-" + destName
	processors[attributesProcessorName] = commonconf.GenericMap{
		"actions": []commonconf.GenericMap{
			{
				"key":    "loki.attribute.labels",
				"action": "insert",
				"value":  attributeHint,
			},
		},
	}

	resourceProcessorName := "resource/grafana-" + destName
	processors[resourceProcessorName] = commonconf.GenericMap{
		"attributes": []commonconf.GenericMap{
			{
				"key":    "loki.resource.labels",
				"action": "insert",
				"value":  attributeHint,
			},
		},
	}

	return processors, nil
}
