package config

import (
	"github.com/odigos-io/odigos/common"
)

const (
	CHECKLY_ENDOINT     = "CHECKLY_ENDOINT"
	CHECKLY_API_KEY     = "CHECKLY_API_KEY"
	CHECKLY_WITH_FILTER = "CHECKLY_WITH_FILTER"
)

type Checkly struct{}

func (j *Checkly) DestType() common.DestinationType {
	return common.ChecklyDestinationType
}

func (j *Checkly) ModifyConfig(dest ExporterConfigurer, cfg *Config) ([]string, error) {
	config := dest.GetConfig()
	uniqueUri := "checkly-" + dest.GetID()
	processorName := "filter/" + uniqueUri
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

	withFilter, exists := config[CHECKLY_WITH_FILTER]
	addFilterProcessor := exists && withFilter == "true"
	if addFilterProcessor {
		cfg.Processors[processorName] = GenericMap{
			"error_mode": "ignore",
			"traces": []GenericMap{
				{
					"span": GenericMap{
						"include": GenericMap{
							"match_type": "expr",
							"expressions": []string{
								`trace_state["checkly"] == "true"`,
							},
						},
					},
				},
			},
		}
	}

	if isTracingEnabled(dest) {
		pipeName := "traces/" + uniqueUri
		pipeline := Pipeline{
			Exporters: []string{exporterName},
		}
		if addFilterProcessor {
			pipeline.Processors = []string{processorName}
		}

		cfg.Service.Pipelines[pipeName] = pipeline
		pipelineNames = append(pipelineNames, pipeName)
	}

	return pipelineNames, nil
}
