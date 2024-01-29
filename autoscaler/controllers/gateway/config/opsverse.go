package config

import (
	"fmt"
	"strings"

	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	commonconf "github.com/keyval-dev/odigos/autoscaler/controllers/common"
	"github.com/keyval-dev/odigos/common"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	opsverseLogsUrl    = "OPSVERSE_LOGS_URL"
	opsverseMetricsUrl = "OPSVERSE_METRICS_URL"
	opsverseTracesUrl  = "OPSVERSE_TRACES_URL"
	opsverseUserName   = "OPSVERSE_USERNAME"
)

type OpsVerse struct{}

func (g *OpsVerse) DestType() common.DestinationType {
	return common.OpsVerseDestinationType
}

func (g *OpsVerse) ModifyConfig(dest *odigosv1.Destination, currentConfig *commonconf.Config) {
	if isMetricsEnabled(dest) && g.isMetricsVarsExists(dest) {
		url := fmt.Sprintf("%s/api/v1/write", dest.Spec.Data[opsverseMetricsUrl])
		rwExporterName := "prometheusremotewrite/opsverse-" + dest.Name
		currentConfig.Exporters[rwExporterName] = commonconf.GenericMap{
			"endpoint": url,
			"headers": commonconf.GenericMap{
				"Authorization": fmt.Sprintf("Basic %s", "${OPSVERSE_AUTH_TOKEN}"),
			},
		}

		metricsPipelineName := "metrics/opsverse-" + dest.Name
		currentConfig.Service.Pipelines[metricsPipelineName] = commonconf.Pipeline{
			Receivers:  []string{"otlp"},
			Processors: []string{"batch"},
			Exporters:  []string{rwExporterName},
		}
	}

	if isTracingEnabled(dest) && g.isTracingVarsExists(dest) {
		url := dest.Spec.Data[opsverseTracesUrl]
		url = strings.TrimPrefix(url, "http://")
		url = strings.TrimPrefix(url, "https://")
		url = fmt.Sprintf("%s:443", url)
		exporterName := "otlp/opsverse-" + dest.Name
		currentConfig.Exporters[exporterName] = commonconf.GenericMap{
			"endpoint": url,
			"headers": commonconf.GenericMap{
				"authorization": "Basic ${OPSVERSE_AUTH_TOKEN}",
			},
		}

		currentConfig.Service.Pipelines["traces/opsverse"] = commonconf.Pipeline{
			Receivers:  []string{"otlp"},
			Processors: []string{"batch"},
			Exporters:  []string{exporterName},
		}
	}

	if isLoggingEnabled(dest) && g.isLogsVarsExists(dest) {
		url := fmt.Sprintf("%s/loki/api/v1/push", dest.Spec.Data[opsverseLogsUrl])

		lokiExporterName := "loki/opsverse-" + dest.Name
		currentConfig.Exporters[lokiExporterName] = commonconf.GenericMap{
			"endpoint": url,
			"headers": commonconf.GenericMap{
				"Authorization": fmt.Sprintf("Basic %s", "${OPSVERSE_AUTH_TOKEN}"),
			},
			"labels": commonconf.GenericMap{
				"attributes": commonconf.GenericMap{
					"k8s.container.name": "k8s_container_name",
					"k8s.pod.name":       "k8s_pod_name",
					"k8s.namespace.name": "k8s_namespace_name",
				},
			},
		}

		logsPipelineName := "logs/opsverse-" + dest.Name
		currentConfig.Service.Pipelines[logsPipelineName] = commonconf.Pipeline{
			Receivers:  []string{"otlp"},
			Processors: []string{"batch"},
			Exporters:  []string{lokiExporterName},
		}
	}
}

func (g *OpsVerse) isTracingVarsExists(dest *odigosv1.Destination) bool {
	_, exists := dest.Spec.Data[opsverseTracesUrl]
	if !exists {
		log.Log.V(0).Info("OpsVerse OTLP tracing endpoint not specified, gateway will not be configured for tracing")
		return false
	}

	_, exists = dest.Spec.Data[opsverseUserName]
	if !exists {
		log.Log.V(0).Info("OpsVerse user not specified, gateway will not be configured for traces")
		return false
	}

	return true
}

func (g *OpsVerse) isLogsVarsExists(dest *odigosv1.Destination) bool {
	_, exists := dest.Spec.Data[opsverseLogsUrl]
	if !exists {
		log.Log.V(0).Info("OpsVerse logs endpoint not specified, gateway will not be configured for logs")
		return false
	}

	_, exists = dest.Spec.Data[opsverseUserName]
	if !exists {
		log.Log.V(0).Info("OpsVerse user not specified, gateway will not be configured for logs")
		return false
	}

	return true
}

func (g *OpsVerse) isMetricsVarsExists(dest *odigosv1.Destination) bool {
	_, exists := dest.Spec.Data[opsverseMetricsUrl]
	if !exists {
		log.Log.V(0).Info("OpsVerse metrics endpoint not specified, gateway will not be configured for metrics")
		return false
	}

	_, exists = dest.Spec.Data[opsverseUserName]
	if !exists {
		log.Log.V(0).Info("OpsVerse user not specified, gateway will not be configured for metrics")
		return false
	}

	return true
}
