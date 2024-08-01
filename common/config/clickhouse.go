package config

import (
	"errors"
	"net/url"
	"strings"

	"github.com/odigos-io/odigos/common"
)

const (
	clickhouseEndpoint     = "CLICKHOUSE_ENDPOINT"
	clickhouseUsername     = "CLICKHOUSE_USERNAME"
	clickhousePassword     = "${CLICKHOUSE_PASSWORD}"
	clickhouseCreateSchema = "CLICKHOUSE_CREATE_SCHEME"
	clickhouseDatabaseName = "CLICKHOUSE_DATABASE_NAME"
	clickhouseTracesTable  = "CLICKHOUSE_TRACES_TABLE"
	clickhouseMetricsTable = "CLICKHOUSE_METRICS_TABLE"
	clickhouseLogsTable    = "CLICKHOUSE_LOGS_TABLE"
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

	if !strings.Contains(endpoint, "://") {
		endpoint = "tcp://" + endpoint
	}

	parsedUrl, err := url.Parse(endpoint)
	if err != nil {
		return errors.New("clickhouse endpoint is not a valid URL")
	}

	if parsedUrl.Port() == "" {
		endpoint = strings.Replace(endpoint, parsedUrl.Host, parsedUrl.Host+":9000", 1)
	}

	username, userExists := dest.GetConfig()[clickhouseUsername]

	exporterName := "clickhouse/clickhouse-" + dest.GetID()
	exporterConfig := GenericMap{
		"endpoint": endpoint,
	}
	if userExists {
		exporterConfig["username"] = username
		exporterConfig["password"] = clickhousePassword
	}

	createSchema, exists := dest.GetConfig()[clickhouseCreateSchema]
	createSchemaBoolValue := exists && strings.ToLower(createSchema) == "create"
	exporterConfig["create_schema"] = createSchemaBoolValue

	dbName, exists := dest.GetConfig()[clickhouseDatabaseName]
	if !exists {
		return errors.New("clickhouse database name not specified, gateway will not be configured for Clickhouse")
	}
	exporterConfig["database"] = dbName

	tracesTable, exists := dest.GetConfig()[clickhouseTracesTable]
	if exists {
		exporterConfig["traces_table_name"] = tracesTable
	}

	metricsTable, exists := dest.GetConfig()[clickhouseMetricsTable]
	if exists {
		exporterConfig["metrics_table_name"] = metricsTable
	}

	logsTable, exists := dest.GetConfig()[clickhouseLogsTable]
	if exists {
		exporterConfig["logs_table_name"] = logsTable
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
