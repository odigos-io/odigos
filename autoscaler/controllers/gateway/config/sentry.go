package config

import (
	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	commonconf "github.com/keyval-dev/odigos/autoscaler/controllers/common"
	"github.com/keyval-dev/odigos/common"
)

type Sentry struct{}

func (s *Sentry) DestType() common.DestinationType {
	return common.LightstepDestinationType
}

func (s *Sentry) ModifyConfig(dest *odigosv1.Destination, currentConfig *commonconf.Config) {
	if isTracingEnabled(dest) {
		currentConfig.Exporters["otlp/lightstep"] = commonconf.GenericMap{
			"endpoint": "ingest.lightstep.com:443",
			"headers": commonconf.GenericMap{
				"lightstep-access-token": "${LIGHTSTEP_ACCESS_TOKEN}",
			},
		}

		currentConfig.Service.Pipelines["traces/lightstep"] = commonconf.Pipeline{
			Receivers:  []string{"otlp"},
			Processors: []string{"batch"},
			Exporters:  []string{"otlp/lightstep"},
		}
	}
}
