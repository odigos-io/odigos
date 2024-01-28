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
	grafanaRWurlKey       = "GRAFANA_REMOTEWRITE_URL"
	grafanaMetricsUserKey = "GRAFANA_METRICS_USER"
)

type Grafana struct{}

func (g *Grafana) DestType() common.DestinationType {
	return common.GrafanaDestinationType
}

func (g *Grafana) ModifyConfig(dest *odigosv1.Destination, currentConfig *commonconf.Config) {
	if isMetricsEnabled(dest) && g.isMetricsVarsExists(dest) {
		url := strings.TrimPrefix(dest.Spec.Data[grafanaRWurlKey], "https://")
		user := dest.Spec.Data[grafanaMetricsUserKey]
		rwExporterName := "prometheusremotewrite/grafana-" + dest.Name
		currentConfig.Exporters[rwExporterName] = commonconf.GenericMap{
			"endpoint": fmt.Sprintf("https://%s:%s@%s", user, "${GRAFANA_API_KEY}", url),
		}

		metricsPipelineName := "metrics/grafana-" + dest.Name
		currentConfig.Service.Pipelines[metricsPipelineName] = commonconf.Pipeline{
			Receivers:  []string{"otlp"},
			Processors: []string{"batch"},
			Exporters:  []string{rwExporterName},
		}
	}
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
