package config

import (
	"fmt"
	odigosv1 "github.com/keyval-dev/odigos/api/v1alpha1"
	commonconf "github.com/keyval-dev/odigos/autoscaler/controllers/common"
	"github.com/keyval-dev/odigos/common"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"strings"
)

const (
	grafanaUrlKey = "GRAFANA_URL"
)

type Grafana struct{}

func (g *Grafana) DestType() common.DestinationType {
	return common.GrafanaDestinationType
}

func (g *Grafana) ModifyConfig(dest *odigosv1.Destination, currentConfig *commonconf.Config) {
	if isTracingEnabled(dest) {
		val, exists := dest.Spec.Data[grafanaUrlKey]
		if !exists {
			log.Log.V(0).Info("Grafana URL not specified, gateway will not be configured for Grafana")
			return
		}

		url := strings.TrimSuffix(val, "/tempo")
		currentConfig.Exporters["otlp/grafana"] = commonconf.GenericMap{
			"endpoint": fmt.Sprintf("%s:%d", url, 443),
			"headers": commonconf.GenericMap{
				"authorization": "Basic ${GRAFANA_AUTH_TOKEN}",
			},
		}

		currentConfig.Service.Pipelines["traces/grafana"] = commonconf.Pipeline{
			Receivers:  []string{"otlp"},
			Processors: []string{"batch"},
			Exporters:  []string{"otlp/grafana"},
		}
	}
}
