package config

import (
	"fmt"
	odigosv1 "github.com/keyval-dev/odigos/api/v1alpha1"
	commonconf "github.com/keyval-dev/odigos/autoscaler/controllers/common"
	"strings"
)

type Grafana struct{}

func (g *Grafana) DestType() odigosv1.DestinationType {
	return odigosv1.GrafanaDestinationType
}

func (g *Grafana) ModifyConfig(dest *odigosv1.Destination, currentConfig *commonconf.Config) {
	if isTracingEnabled(dest) {
		url := strings.TrimSuffix(dest.Spec.Data.Grafana.Url, "/tempo")
		currentConfig.Exporters["otlp/grafana"] = commonconf.GenericMap{
			"endpoint": fmt.Sprintf("%s:%d", url, 443),
			"headers": commonconf.GenericMap{
				"authorization": "Basic ${AUTH_TOKEN}",
			},
		}

		currentConfig.Service.Pipelines["traces/grafana"] = commonconf.Pipeline{
			Receivers:  []string{"otlp"},
			Processors: []string{"batch"},
			Exporters:  []string{"otlp/grafana"},
		}
	}
}
