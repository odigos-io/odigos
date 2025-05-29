package config

import (
	"github.com/odigos-io/odigos/common"
)

const (
	CHECKLY_ENDOINT = "CHECKLY_ENDOINT"
	CHECKLY_API_KEY = "CHECKLY_API_KEY"
)

type Checkly struct{}

func (j *Checkly) DestType() common.DestinationType {
	return common.ChecklyDestinationType
}

func (j *Checkly) ModifyConfig(dest ExporterConfigurer, cfg *Config) ([]string, error) {
	config := dest.GetConfig()
	uniqueUri := "checkly-" + dest.GetID()
	var pipelineNames []string

	endpoint, exists := config[CHECKLY_ENDOINT]
	if !exists {
		return nil, errorMissingKey(CHECKLY_ENDOINT)
	}
	endpoint, err := parseOtlpGrpcUrl(endpoint, true)
	if err != nil {
		return nil, err
	}

	exporterName := "otlp/" + uniqueUri
	cfg.Exporters[exporterName] = GenericMap{
		"endpoint": endpoint,
		"headers": GenericMap{
			"authorization": "${CHECKLY_API_KEY}",
		},
	}

	processorName := "filter/" + uniqueUri
	cfg.Processors[processorName] = GenericMap{
		"error_mode": "ignore",
		"traces": GenericMap{
			"span": []string{
				`trace_state["checkly"] == "true"`,
			},
		},
	}

	if IsTracingEnabled(dest) {
		pipeName := "traces/" + uniqueUri
		cfg.Service.Pipelines[pipeName] = Pipeline{
			Processors: []string{processorName},
			Exporters:  []string{exporterName},
		}

		pipelineNames = append(pipelineNames, pipeName)
	}

	return pipelineNames, nil
}
