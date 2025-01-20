package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"

	"github.com/odigos-io/odigos/common"
)

const (
	grafanaCloudPrometheusRWurlKey        = "GRAFANA_CLOUD_PROMETHEUS_RW_ENDPOINT"
	grafanaCloudPrometheusUserKey         = "GRAFANA_CLOUD_PROMETHEUS_USERNAME"
	prometheusResourceAttributesLabelsKey = "PROMETHEUS_RESOURCE_ATTRIBUTES_LABELS"
	prometheusExternalLabelsKey           = "PROMETHEUS_RESOURCE_EXTERNAL_LABELS"
)

type GrafanaCloudPrometheus struct{}

func (g *GrafanaCloudPrometheus) DestType() common.DestinationType {
	return common.GrafanaCloudPrometheusDestinationType
}

func (g *GrafanaCloudPrometheus) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) error {
	if !isMetricsEnabled(dest) {
		return errors.New("Metrics not enabled, gateway will not be configured for grafana cloud prometheus")
	}

	promRwUrl, exists := dest.GetConfig()[grafanaCloudPrometheusRWurlKey]
	if !exists {
		return errors.New("Grafana Cloud Prometheus remote write endpoint not specified, gateway will not be configured" +
			" for Prometheus")
	}

	if err := validateGrafanaPrometheusUrl(promRwUrl); err != nil {
		return errors.Join(err, errors.New("failed to validate grafana cloud prometheus remote write endpoint, gateway will "+
			"not be configured for Prometheus"))
	}

	prometheusUsername, exists := dest.GetConfig()[grafanaCloudPrometheusUserKey]
	if !exists {
		return errors.New("Grafana Cloud Prometheus username not specified, gateway will not be configured for Prometheus")
	}

	resourceAttributesLabels, exists := dest.GetConfig()[prometheusResourceAttributesLabelsKey]
	processors, err := promResourceAttributesProcessors(resourceAttributesLabels, exists, dest.GetID())
	if err != nil {
		return errors.Join(err, errors.New("failed to parse grafana cloud prometheus resource attributes labels, gateway will "+
			"not be configured for Prometheus"))
	}

	authExtensionName := "basicauth/grafana" + dest.GetID()
	currentConfig.Extensions[authExtensionName] = GenericMap{
		"client_auth": GenericMap{
			"username": prometheusUsername,
			"password": "${GRAFANA_CLOUD_PROMETHEUS_PASSWORD}",
		},
	}

	exporterConf := GenericMap{
		"endpoint":            promRwUrl,
		"add_metric_suffixes": false,
		"auth": GenericMap{
			"authenticator": authExtensionName,
		},
	}

	// add external labels if they exist
	externalLabels, exists := dest.GetConfig()[prometheusExternalLabelsKey]
	if exists {
		labels := map[string]string{}
		err := json.Unmarshal([]byte(externalLabels), &labels)
		if err != nil {
			return errors.Join(err, errors.New("failed to parse grafana cloud prometheus external labels, gateway will "+
				"not be configured for Prometheus"))
		}

		exporterConf["external_labels"] = labels
	}

	rwExporterName := "prometheusremotewrite/grafana-" + dest.GetID()
	currentConfig.Exporters[rwExporterName] = exporterConf

	processorNames := []string{}
	for k, v := range processors {
		currentConfig.Processors[k] = v
		processorNames = append(processorNames, k)
	}

	metricsPipelineName := "metrics/grafana-" + dest.GetID()
	currentConfig.Service.Extensions = append(currentConfig.Service.Extensions, authExtensionName)
	currentConfig.Service.Pipelines[metricsPipelineName] = Pipeline{
		Processors: processorNames,
		Exporters:  []string{rwExporterName},
	}

	return nil
}

func validateGrafanaPrometheusUrl(input string) error {
	parsedUrl, err := url.Parse(input)
	if err != nil {
		return err
	}

	if parsedUrl.Scheme != "https" {
		return fmt.Errorf("grafana cloud prometheus remote writer endpoint scheme must be https")
	}

	if parsedUrl.Path != "/api/prom/push" {
		return fmt.Errorf("grafana cloud prometheus remote writer endpoint path should be /api/prom/push")
	}

	return nil
}

func promResourceAttributesProcessors(rawLabels string, exists bool, destName string) (GenericMap, error) {
	if !exists {
		return nil, nil
	}

	// no labels. not recommended, but ok
	if rawLabels == "" || rawLabels == "[]" {
		return nil, nil
	}

	var attributeNames []string
	err := json.Unmarshal([]byte(rawLabels), &attributeNames)
	if err != nil {
		return nil, err
	}

	transformStatements := []string{}
	for _, attr := range attributeNames {
		statement := fmt.Sprintf("set(attributes[%q], resource.attributes[%q])", attr, attr)
		transformStatements = append(transformStatements, statement)
	}

	processorName := "transform/grafana-" + destName
	return GenericMap{
		processorName: GenericMap{
			"metric_statements": []GenericMap{
				{
					"context":    "datapoint",
					"statements": transformStatements,
				},
			},
		},
	}, nil
}
