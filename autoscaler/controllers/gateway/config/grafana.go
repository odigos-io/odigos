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
	grafanaUrlKey         = "GRAFANA_TEMPO_URL"
	grafanaRWurlKey       = "GRAFANA_REMOTEWRITE_URL"
	grafanaMetricsUserKey = "GRAFANA_METRICS_USER"
	grafanaLokiUserKey    = "GRAFANA_LOKI_USER"
	grafanaLokiUrl        = "GRAFANA_LOKI_URL"
)

type Grafana struct{}

func (g *Grafana) DestType() common.DestinationType {
	return common.GrafanaDestinationType
}

func (g *Grafana) ModifyConfig(dest *odigosv1.Destination, currentConfig *commonconf.Config) {
	if isMetricsEnabled(dest) && g.isMetricsVarsExists(dest) {
		url := strings.TrimPrefix(dest.Spec.Data[grafanaRWurlKey], "https://")
		user := dest.Spec.Data[grafanaMetricsUserKey]
		rwExporterName := "prometheusremotewrite/grafana"
		currentConfig.Exporters[rwExporterName] = commonconf.GenericMap{
			"endpoint": fmt.Sprintf("https://%s:%s@%s", user, "${GRAFANA_API_KEY}", url),
		}

		currentConfig.Service.Pipelines["metrics/grafana"] = commonconf.Pipeline{
			Receivers:  []string{"otlp"},
			Processors: []string{"batch"},
			Exporters:  []string{rwExporterName},
		}
	}

	if isTracingEnabled(dest) && g.isTempoVarsExists(dest) {
		url := strings.TrimSuffix(dest.Spec.Data[grafanaUrlKey], "/tempo")
		currentConfig.Exporters["otlp/grafana"] = commonconf.GenericMap{
			"endpoint": fmt.Sprintf("%s:%d", url, 443),
			"headers": commonconf.GenericMap{
				"authorization": "Basic ${GRAFANA_TEMPO_AUTH_TOKEN}",
			},
		}

		currentConfig.Service.Pipelines["traces/grafana"] = commonconf.Pipeline{
			Receivers:  []string{"otlp"},
			Processors: []string{"batch"},
			Exporters:  []string{"otlp/grafana"},
		}
	}

	if isLoggingEnabled(dest) && g.isLokiVarsExists(dest) {
		user := dest.Spec.Data[grafanaLokiUserKey]
		url := strings.TrimPrefix(dest.Spec.Data[grafanaLokiUrl], "https://")
		if !strings.HasSuffix(url, "/loki/api/v1/push") {
			url = fmt.Sprintf("%s/loki/api/v1/push", url)
		}

		lokiExporterName := "loki/grafana"
		currentConfig.Exporters[lokiExporterName] = commonconf.GenericMap{
			"endpoint": fmt.Sprintf("https://%s:%s@%s", user, "${GRAFANA_API_KEY}", url),
			"labels": commonconf.GenericMap{
				"attributes": commonconf.GenericMap{
					"k8s.container.name": "k8s_container_name",
					"k8s.pod.name":       "k8s_pod_name",
					"k8s.namespace.name": "k8s_namespace_name",
				},
			},
		}

		currentConfig.Service.Pipelines["logs/grafana"] = commonconf.Pipeline{
			Receivers:  []string{"otlp"},
			Processors: []string{"batch"},
			Exporters:  []string{lokiExporterName},
		}
	}
}

func (g *Grafana) isTempoVarsExists(dest *odigosv1.Destination) bool {
	_, exists := dest.Spec.Data[grafanaUrlKey]
	if !exists {
		log.Log.V(0).Info("Grafana URL not specified, gateway will not be configured for Tempo")
		return false
	}

	return true
}

func (g *Grafana) isLokiVarsExists(dest *odigosv1.Destination) bool {
	_, exists := dest.Spec.Data[grafanaLokiUrl]
	if !exists {
		log.Log.V(0).Info("Grafana Loki URL not specified, gateway will not be configured for metrics")
		return false
	}

	_, exists = dest.Spec.Data[grafanaLokiUserKey]
	if !exists {
		log.Log.V(0).Info("Grafana Loki user not specified, gateway will not be configured for metrics")
		return false
	}

	return true
}

func (g *Grafana) isMetricsVarsExists(dest *odigosv1.Destination) bool {
	_, exists := dest.Spec.Data[grafanaRWurlKey]
	if !exists {
		log.Log.V(0).Info("Grafana RemoteWrite URL not specified, gateway will not be configured for metrics")
		return false
	}

	_, exists = dest.Spec.Data[grafanaMetricsUserKey]
	if !exists {
		log.Log.V(0).Info("Grafana metrics user not specified, gateway will not be configured for metrics")
		return false
	}

	return true
}
