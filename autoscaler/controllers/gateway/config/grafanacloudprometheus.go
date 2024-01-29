package config

import (
	"fmt"
	"net/url"

	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	commonconf "github.com/keyval-dev/odigos/autoscaler/controllers/common"
	"github.com/keyval-dev/odigos/common"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	grafanaCloudPrometheusRWurlKey = "GRAFANA_CLOUD_PROMETHEUS_RW_ENDPOINT"
	grafanaCloudPrometheusUserKey  = "GRAFANA_CLOUD_PROMETHEUS_USERNAME"
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

	// the following processor is found in grafana official "OpenTelemetry Collector Configuration" tool.
	// https://odigos.grafana.net/connections/add-new-connection/collector-open-telemetry
	// we should consider if these are needed for our use case and what attributes we want to add.
	//
	// transform/add_resource_attributes_as_metric_attributes:
	// # https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/processor/transformprocessor
	// error_mode: ignore
	// metric_statements:
	//   - context: datapoint
	//     statements:
	//       - set(attributes["deployment.environment"], resource.attributes["deployment.environment"])
	//       - set(attributes["service.version"], resource.attributes["service.version"])

	metricsPipelineName := "metrics/grafana-" + dest.Name
	currentConfig.Service.Extensions = append(currentConfig.Service.Extensions, authExtensionName)
	currentConfig.Service.Pipelines[metricsPipelineName] = commonconf.Pipeline{
		Receivers:  []string{"otlp"},
		Processors: []string{"batch"},
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
