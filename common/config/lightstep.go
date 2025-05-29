package config

import (
	"github.com/odigos-io/odigos/common"
)

type Lightstep struct{}

func (l *Lightstep) DestType() common.DestinationType {
	return common.LightstepDestinationType
}

func (l *Lightstep) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) ([]string, error) {
	var pipelineNames []string
	if IsTracingEnabled(dest) {
		exporterName := "otlp/lightstep-" + dest.GetID()
		currentConfig.Exporters[exporterName] = GenericMap{
			"endpoint": "ingest.lightstep.com:443",
			"headers": GenericMap{
				"lightstep-access-token": "${LIGHTSTEP_ACCESS_TOKEN}",
			},
		}

		tracesPipelineName := "traces/lightstep-" + dest.GetID()
		currentConfig.Service.Pipelines[tracesPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, tracesPipelineName)
	}

	return pipelineNames, nil
}
