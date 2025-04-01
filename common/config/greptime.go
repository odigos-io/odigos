package config

import (
	"github.com/odigos-io/odigos/common"
)

const (
	GREPTIME_ENDPOINT    = "GREPTIME_ENDPOINT"
	GREPTIME_DB_NAME     = "GREPTIME_DB_NAME"
	GREPTIME_BASIC_TOKEN = "GREPTIME_BASIC_TOKEN"
)

type Greptime struct{}

func (j *Greptime) DestType() common.DestinationType {
	return common.GreptimeDestinationType
}

func (j *Greptime) ModifyConfig(dest ExporterConfigurer, cfg *Config) ([]string, error) {
	config := dest.GetConfig()
	uniqueUri := "greptime-" + dest.GetID()
	var pipelineNames []string

	endpoint, exists := config[GREPTIME_ENDPOINT]
	if !exists {
		return nil, errorMissingKey(GREPTIME_ENDPOINT)
	}
	endpoint, err := parseOtlpHttpEndpoint(endpoint)
	if err != nil {
		return nil, err
	}

	dbName, exists := config[GREPTIME_DB_NAME]
	if !exists {
		return nil, errorMissingKey(GREPTIME_DB_NAME)
	}

	exporterName := "otlphttp/" + uniqueUri
	cfg.Exporters[exporterName] = GenericMap{
		"endpoint": endpoint,
		"headers": GenericMap{
			"X-Greptime-DB-Name": dbName,
			// TODO: handle Base64 encoding of GREPTIME_BASIC_TOKEN
			"Authorization": "Basic ${GREPTIME_BASIC_TOKEN}",
		},
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
