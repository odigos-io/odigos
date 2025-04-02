package config

import (
	"github.com/odigos-io/odigos/common"
)

const (
	GREPTIME_ENDPOINT       = "GREPTIME_ENDPOINT"
	GREPTIME_DB_NAME        = "GREPTIME_DB_NAME"
	GREPTIME_BASIC_USERNAME = "GREPTIME_BASIC_USERNAME"
	GREPTIME_BASIC_PASSWORD = "GREPTIME_BASIC_PASSWORD"
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
	err = urlHostShouldNotContainPath(endpoint)
	if err != nil {
		return nil, err
	}
	endpoint += "/v1/otlp"

	dbName, exists := config[GREPTIME_DB_NAME]
	if !exists {
		return nil, errorMissingKey(GREPTIME_DB_NAME)
	}

	basicUsername, exists := config[GREPTIME_BASIC_USERNAME]
	if !exists {
		return nil, errorMissingKey(GREPTIME_BASIC_USERNAME)
	}

	authExtensionName := "basicauth/" + uniqueUri
	cfg.Extensions[authExtensionName] = GenericMap{
		"client_auth": GenericMap{
			"username": basicUsername,
			"password": "${GREPTIME_BASIC_PASSWORD}",
		},
	}

	exporterName := "otlphttp/" + uniqueUri
	cfg.Exporters[exporterName] = GenericMap{
		"endpoint": endpoint,
		"auth": GenericMap{
			"authenticator": authExtensionName,
		},
		"headers": GenericMap{
			"X-Greptime-DB-Name": dbName,
			// "Authorization":      "Basic ",
		},
	}

	cfg.Service.Extensions = append(cfg.Service.Extensions, authExtensionName)

	if isMetricsEnabled(dest) {
		pipeName := "metrics/" + uniqueUri
		cfg.Service.Pipelines[pipeName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, pipeName)
	}

	return pipelineNames, nil
}
