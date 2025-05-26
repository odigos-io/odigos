package config

import (
	"errors"

	"github.com/odigos-io/odigos/common"
)

type Sentry struct{}

func (s *Sentry) DestType() common.DestinationType {
	return common.SentryDestinationType
}

func (s *Sentry) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) ([]string, error) {
	if !IsTracingEnabled(dest) {
		return nil, errors.New("Sentry is not enabled for any supported signals, skipping")
	}
	var pipelineNames []string
	if IsTracingEnabled(dest) {
		exporterName := "sentry/" + dest.GetID()
		currentConfig.Exporters[exporterName] = GenericMap{
			"dsn": "${DSN}",
		}

		tracesPipelineName := "traces/sentry-" + dest.GetID()
		currentConfig.Service.Pipelines[tracesPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, tracesPipelineName)
	}

	return pipelineNames, nil
}
