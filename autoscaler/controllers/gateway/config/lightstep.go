package config

import (
	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	commonconf "github.com/keyval-dev/odigos/autoscaler/controllers/common"
	"github.com/keyval-dev/odigos/common"
)

type Lightstep struct{}

func (l *Lightstep) DestType() common.DestinationType {
	return common.LightstepDestinationType
}

func (l *Lightstep) ModifyConfig(dest *odigosv1.Destination, currentConfig *commonconf.Config) error {
	if isTracingEnabled(dest) {
		exporterName := "otlp/lightstep-" + dest.Name
		currentConfig.Exporters[exporterName] = commonconf.GenericMap{
			"endpoint": "ingest.lightstep.com:443",
			"headers": commonconf.GenericMap{
				"lightstep-access-token": "${LIGHTSTEP_ACCESS_TOKEN}",
			},
		}

		tracesPipelineName := "traces/lightstep-" + dest.Name
		currentConfig.Service.Pipelines[tracesPipelineName] = commonconf.Pipeline{
			Exporters: []string{exporterName},
		}
	}

	return nil
}
