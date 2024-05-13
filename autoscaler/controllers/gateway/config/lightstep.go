package config

import (
	commonconf "github.com/odigos-io/odigos/autoscaler/controllers/common"
	"github.com/odigos-io/odigos/common"
)

type Lightstep struct{}

func (l *Lightstep) DestType() common.DestinationType {
	return common.LightstepDestinationType
}

func (l *Lightstep) ModifyConfig(dest common.ExporterConfigurer, currentConfig *commonconf.Config) error {
	if isTracingEnabled(dest) {
		exporterName := "otlp/lightstep-" + dest.GetName()
		currentConfig.Exporters[exporterName] = commonconf.GenericMap{
			"endpoint": "ingest.lightstep.com:443",
			"headers": commonconf.GenericMap{
				"lightstep-access-token": "${LIGHTSTEP_ACCESS_TOKEN}",
			},
		}

		tracesPipelineName := "traces/lightstep-" + dest.GetName()
		currentConfig.Service.Pipelines[tracesPipelineName] = commonconf.Pipeline{
			Exporters: []string{exporterName},
		}
	}

	return nil
}
