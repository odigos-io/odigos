package config

import (
	"encoding/json"
	"fmt"
	"net/url"

	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	commonconf "github.com/keyval-dev/odigos/autoscaler/controllers/common"
	"github.com/keyval-dev/odigos/common"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	grafanaCloudPrometheusRWurlKey         = "GRAFANA_CLOUD_PROMETHEUS_RW_ENDPOINT"
	grafanaCloudPrometheusUserKey          = "GRAFANA_CLOUD_PROMETHEUS_USERNAME"
	prometheusResourceAttributesLabelsKeys = "PROMETHEUS_RESOURCE_ATTRIBUTES_LABELS"
)

type GrafanaCloudPrometheus struct{}

func (g *GrafanaCloudPrometheus) DestType() common.DestinationType {
	return common.GrafanaCloudPrometheusDestinationType
}

func (g *GrafanaCloudPrometheus) ModifyConfig(dest *odigosv1.Destination, currentConfig *commonconf.Config) {

	if !isMetricsEnabled(dest) {
		log.Log.V(0).Info("Metrics not enabled, gateway will not be configured for grafana cloud prometheus")
		return
	}

	promRwUrl, exists := dest.Spec.Data[grafanaCloudPrometheusRWurlKey]
	if !exists {
		log.Log.V(0).Info("Grafana Cloud Prometheus remote write endpoint not specified, gateway will not be configured for Prometheus")
		return
	}

	if err := validateGrafanaPrometheusUrl(promRwUrl); err != nil {
		log.Log.Error(err, "failed to validate grafana cloud prometheus remote write endpoint, gateway will not be configured for Prometheus")
		return
	}

	prometheusUsername, exists := dest.Spec.Data[grafanaCloudPrometheusUserKey]
	if !exists {
		log.Log.V(0).Info("Grafana Cloud Prometheus username not specified, gateway will not be configured for Prometheus")
		return
	}

	resourceAttributesLabels, exists := dest.Spec.Data[prometheusResourceAttributesLabelsKeys]
	processors, err := promResourceAttributesProcessors(resourceAttributesLabels, exists, dest.Name)
	if err != nil {
		log.Log.Error(err, "failed to parse grafana cloud prometheus resource attributes labels, gateway will not be configured for Prometheus")
		return
	}

	authExtensionName := "basicauth/grafana" + dest.Name
	currentConfig.Extensions[authExtensionName] = commonconf.GenericMap{
		"client_auth": commonconf.GenericMap{
			"username": prometheusUsername,
			"password": "${GRAFANA_CLOUD_PROMETHEUS_PASSWORD}",
		},
	}

	rwExporterName := "prometheusremotewrite/grafana-" + dest.Name
	currentConfig.Exporters[rwExporterName] = commonconf.GenericMap{
		"endpoint":            promRwUrl,
		"add_metric_suffixes": false,
		"auth": commonconf.GenericMap{
			"authenticator": authExtensionName,
		},
	}

	processorNames := []string{}
	for k, v := range processors {
		currentConfig.Processors[k] = v
		processorNames = append(processorNames, k)
	}

	metricsPipelineName := "metrics/grafana-" + dest.Name
	currentConfig.Service.Extensions = append(currentConfig.Service.Extensions, authExtensionName)
	currentConfig.Service.Pipelines[metricsPipelineName] = commonconf.Pipeline{
		Receivers:  []string{"otlp"},
		Processors: append([]string{"batch"}, processorNames...),
		Exporters:  []string{rwExporterName},
	}
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

func promResourceAttributesProcessors(rawLabels string, exists bool, destName string) (commonconf.GenericMap, error) {
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
		statement := fmt.Sprintf("set(attributes[\"%s\"], resource.attributes[\"%s\"])", attr, attr)
		transformStatements = append(transformStatements, statement)
	}

	processorName := "transform/grafana-" + destName
	return commonconf.GenericMap{
		processorName: commonconf.GenericMap{
			"metric_statements": []commonconf.GenericMap{
				{
					"context":    "datapoint",
					"statements": transformStatements,
				},
			},
		},
	}, nil
}
