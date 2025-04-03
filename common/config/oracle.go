package config

import (
	"errors"

	"github.com/odigos-io/odigos/common"
)

const (
	ORACLE_ENDPOINT      = "ORACLE_ENDPOINT"
	ORACLE_DATA_KEY      = "ORACLE_DATA_KEY"
	ORACLE_DATA_KEY_TYPE = "ORACLE_DATA_KEY_TYPE"
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
	endpoint, err := parseOtlpHttpEndpoint(endpoint, "", "")
	if err != nil {
		return nil, err
	}

	exporterName := "otlphttp/" + uniqueUri
	exporterConfig := GenericMap{
		"endpoint": endpoint,
		"headers": GenericMap{
			"Authorization": "dataKey ${ORACLE_DATA_KEY}",
		},
	}

	if isTracingEnabled(dest) {
		dataKeyType, exists := config[ORACLE_DATA_KEY_TYPE]
		if !exists {
			return nil, errorMissingKey(ORACLE_DATA_KEY_TYPE)
		}
		if dataKeyType != "private" && dataKeyType != "public" {
			return nil, errors.New("invalid value for ORACLE_DATA_KEY_TYPE, must be one-of [private, public]")
		}

		exporterConfig["endpoint"] = endpoint + "/20200101/opentelemetry/" + dataKeyType
		cfg.Exporters[exporterName] = exporterConfig

		pipeName := "traces/" + uniqueUri
		cfg.Service.Pipelines[pipeName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, pipeName)
	}

	if isMetricsEnabled(dest) {
		exporterConfig["endpoint"] = endpoint + "/20200101/opentelemetry"
		cfg.Exporters[exporterName] = exporterConfig

		pipeName := "metrics/" + uniqueUri
		cfg.Service.Pipelines[pipeName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, pipeName)
	}

	return pipelineNames, nil
}
