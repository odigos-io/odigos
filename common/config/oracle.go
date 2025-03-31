package config

import (
	"github.com/odigos-io/odigos/common"
)

const (
	ORACLE_ENDPOINT = "ORACLE_ENDPOINT"
	ORACLE_DATA_KEY = "ORACLE_DATA_KEY"
)

type Oracle struct{}

func (j *Oracle) DestType() common.DestinationType {
	return common.OracleDestinationType
}

func (j *Oracle) ModifyConfig(dest ExporterConfigurer, cfg *Config) ([]string, error) {
	config := dest.GetConfig()
	uniqueUri := "oracle-" + dest.GetID()
	var pipelineNames []string

	endpoint, exists := config[ORACLE_ENDPOINT]
	if !exists {
		return nil, errorMissingKey(ORACLE_ENDPOINT)
	}
	endpoint, err := parseOtlpHttpEndpoint(endpoint)
	if err != nil {
		return nil, err
	}

	exporterName := "otlphttp/" + uniqueUri
	cfg.Exporters[exporterName] = GenericMap{
		"endpoint": endpoint,
		"headers": GenericMap{
			"Authorization": "dataKey ${ORACLE_DATA_KEY}",
		},
	}

	if isTracingEnabled(dest) {
		pipeName := "traces/" + uniqueUri
		cfg.Service.Pipelines[pipeName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, pipeName)
	}

	if isMetricsEnabled(dest) {
		pipeName := "metrics/" + uniqueUri
		cfg.Service.Pipelines[pipeName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, pipeName)
	}

	return pipelineNames, nil
}
