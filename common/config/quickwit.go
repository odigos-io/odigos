package config

import (
	"errors"

	"github.com/odigos-io/odigos/common"
)

const (
	qwUrlKey = "QUICKWIT_URL"
)

type Quickwit struct{}

func (e *Quickwit) DestType() common.DestinationType {
	return common.QuickwitDestinationType
}

func (e *Quickwit) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) ([]string, error) {
	if url, exists := dest.GetConfig()[qwUrlKey]; exists {
		exporterName := "otlp/quickwit-" + dest.GetID()

		currentConfig.Exporters[exporterName] = GenericMap{
			"endpoint": url,
			"tls": GenericMap{
				"insecure": true,
			},
		}

		var pipelineNames []string
		if IsTracingEnabled(dest) {
			tracesPipelineName := "traces/quickwit-" + dest.GetID()
			currentConfig.Service.Pipelines[tracesPipelineName] = Pipeline{
				Exporters: []string{exporterName},
			}
			pipelineNames = append(pipelineNames, tracesPipelineName)
		}

		if IsLoggingEnabled(dest) {
			logsPipelineName := "logs/quickwit-" + dest.GetID()
			currentConfig.Service.Pipelines[logsPipelineName] = Pipeline{
				Exporters: []string{exporterName},
			}
			pipelineNames = append(pipelineNames, logsPipelineName)
		}

		return pipelineNames, nil
	}

	return nil, errors.New("Quickwit url not specified, gateway will not be configured for Quickwit")
}
