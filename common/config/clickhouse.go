package config

import (
	"errors"

	"github.com/odigos-io/odigos/common"
)

const (
	clickhouseEndpoint = "CLICKHOUSE_ENDPOINT"
	clickhouseUsername = "CLICKHOUSE_USERNAME"
	clickhousePassword = "CLICKHOUSE_PASSWORD"
)

type Clickhouse struct{}

func (c *Clickhouse) DestType() common.DestinationType {
	return common.ClickhouseDestinationType
}

func (c *Clickhouse) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) error {
	endpoint, exists := dest.GetConfig()[clickhouseEndpoint]
	if !exists {
		return errors.New("clickhouse endpoint not specified, gateway will not be configured for Clickhouse")
	}

	username, userExists := dest.GetConfig()[clickhouseUsername]
	password, passExists := dest.GetConfig()[clickhousePassword]
	if userExists != passExists {
		return errors.New("clickhouse username and password must be both specified, or neither")
	}

	exporterName := "clickhouse/clickhouse-" + dest.GetID()
	exporterConfig := GenericMap{
		"endpoint": endpoint,
	}
	if userExists {
		exporterConfig["username"] = username
		exporterConfig["password"] = password
	}

	currentConfig.Exporters[exporterName] = exporterConfig
	if isTracingEnabled(dest) {
		tracesPipelineName := "traces/clickhouse-" + dest.GetID()
		currentConfig.Service.Pipelines[tracesPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
	}

	if isMetricsEnabled(dest) {
		metricsPipelineName := "metrics/clickhouse-" + dest.GetID()
		currentConfig.Service.Pipelines[metricsPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
	}

	if isLoggingEnabled(dest) {
		logsPipelineName := "logs/clickhouse-" + dest.GetID()
		currentConfig.Service.Pipelines[logsPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
	}

	return nil
}
