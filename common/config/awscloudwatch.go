package config

import (
	"encoding/json"
	"errors"

	"github.com/odigos-io/odigos/common"
)

const (
	AWS_CLOUDWATCH_LOG_GROUP_NAME                       = "AWS_CLOUDWATCH_LOG_GROUP_NAME"
	AWS_CLOUDWATCH_LOG_STREAM_NAME                      = "AWS_CLOUDWATCH_LOG_STREAM_NAME"
	AWS_CLOUDWATCH_REGION                               = "AWS_CLOUDWATCH_REGION"
	AWS_CLOUDWATCH_ENDPOINT                             = "AWS_CLOUDWATCH_ENDPOINT"
	AWS_CLOUDWATCH_LOG_RETENTION                        = "AWS_CLOUDWATCH_LOG_RETENTION"
	AWS_CLOUDWATCH_TAGS                                 = "AWS_CLOUDWATCH_TAGS"
	AWS_CLOUDWATCH_RAW_LOG                              = "AWS_CLOUDWATCH_RAW_LOG"
	AWS_CLOUDWATCH_METRICS_NAMESPACE                    = "AWS_CLOUDWATCH_METRICS_NAMESPACE"
	AWS_CLOUDWATCH_METRICS_DIMENSION_ROLLUP             = "AWS_CLOUDWATCH_METRICS_DIMENSION_ROLLUP"
	AWS_CLOUDWATCH_METRICS_DETAILED                     = "AWS_CLOUDWATCH_METRICS_DETAILED"
	AWS_CLOUDWATCH_RETAIN_INITIAL_VALUE_OF_DELTA_METRIC = "AWS_CLOUDWATCH_RETAIN_INITIAL_VALUE_OF_DELTA_METRIC"
)

type AWSCloudWatch struct{}

func (m *AWSCloudWatch) DestType() common.DestinationType {
	return common.AWSCloudWatchDestinationType
}

func (m *AWSCloudWatch) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) ([]string, error) {
	uniqueUri := "awscloudwatch-" + dest.GetID()
	var pipelineNames []string

	logsExporterName := "awscloudwatchlogs/" + uniqueUri
	logsExporterConfig, err := logsConfig(dest)
	if err != nil {
		return nil, err
	}
	currentConfig.Exporters[logsExporterName] = logsExporterConfig

	metricsExporterName := "awsemf/" + uniqueUri
	metricsExporterConfig, err := metricsConfig(dest)
	if err != nil {
		return nil, err
	}
	currentConfig.Exporters[metricsExporterName] = metricsExporterConfig

	if isLoggingEnabled(dest) {
		pipeName := "logs/" + uniqueUri
		currentConfig.Service.Pipelines[pipeName] = Pipeline{
			Exporters: []string{logsExporterName},
		}
		pipelineNames = append(pipelineNames, pipeName)
	}

	if isMetricsEnabled(dest) {
		pipeName := "metrics/" + uniqueUri
		currentConfig.Service.Pipelines[pipeName] = Pipeline{
			Exporters: []string{metricsExporterName},
		}
		pipelineNames = append(pipelineNames, pipeName)
	}

	return pipelineNames, nil
}

// Configures an exporter that shares fields between logs exporter and metrics exporter.
func sharedConfig(dest ExporterConfigurer) (GenericMap, error) {
	config := dest.GetConfig()

	// Required fields

	logGroupName, exists := config[AWS_CLOUDWATCH_LOG_GROUP_NAME]
	if !exists {
		return nil, errorMissingKey(AWS_CLOUDWATCH_LOG_GROUP_NAME)
	}

	logStreamName, exists := config[AWS_CLOUDWATCH_LOG_STREAM_NAME]
	if !exists {
		return nil, errorMissingKey(AWS_CLOUDWATCH_LOG_STREAM_NAME)
	}

	// Exporter config

	exporterConfig := GenericMap{
		"log_group_name":  logGroupName,
		"log_stream_name": logStreamName,
	}

	// Optional fields

	region, exists := config[AWS_CLOUDWATCH_REGION]
	if exists {
		exporterConfig["region"] = region
	}

	endpoint, exists := config[AWS_CLOUDWATCH_ENDPOINT]
	if exists {
		exporterConfig["endpoint"] = endpoint
	}

	logRetention, exists := config[AWS_CLOUDWATCH_LOG_RETENTION]
	if exists {
		exporterConfig["log_retention"] = parseInt(logRetention)
	}

	tags, exists := config[AWS_CLOUDWATCH_TAGS]
	if exists {
		var tagList []struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		}
		err := json.Unmarshal([]byte(tags), &tagList)
		if err != nil {
			return nil, errors.Join(err, errors.New(
				"failed to parse awscloudwatch destination AWS_CLOUDWATCH_TAGS parameter as json string in the form {key: string, value: string}[]",
			))
		}
		mappedTags := map[string]string{}
		for _, tag := range tagList {
			mappedTags[tag.Key] = tag.Value
		}
		exporterConfig["tags"] = mappedTags
	}

	return exporterConfig, nil
}

// Configures the exporter for logs.
func logsConfig(dest ExporterConfigurer) (GenericMap, error) {
	config := dest.GetConfig()
	exporterConfig, err := sharedConfig(dest)
	if err != nil {
		return nil, err
	}

	rawLog, exists := config[AWS_CLOUDWATCH_RAW_LOG]
	if exists {
		exporterConfig["raw_log"] = parseBool(rawLog)
	}

	return exporterConfig, nil
}

// Configures the exporter for metrics.
func metricsConfig(dest ExporterConfigurer) (GenericMap, error) {
	config := dest.GetConfig()
	exporterConfig, err := sharedConfig(dest)
	if err != nil {
		return nil, err
	}

	exporterConfig["output_destination"] = "cloudwatch" // other option is "stdout" which logs to stdout of gateway-collector
	exporterConfig["resource_to_telemetry_conversion"] = GenericMap{"enabled": true}

	namespace, exists := config[AWS_CLOUDWATCH_METRICS_NAMESPACE]
	if exists {
		exporterConfig["namespace"] = namespace
	}

	dimensionRollupOption, exists := config[AWS_CLOUDWATCH_METRICS_DIMENSION_ROLLUP]
	if exists {
		exporterConfig["dimension_rollup_option"] = dimensionRollupOption
	}

	metricsDetailed, exists := config[AWS_CLOUDWATCH_METRICS_DETAILED]
	if exists {
		exporterConfig["detailed_metrics"] = parseBool(metricsDetailed)
	}

	retainValueOfDelta, exists := config[AWS_CLOUDWATCH_RETAIN_INITIAL_VALUE_OF_DELTA_METRIC]
	if exists {
		exporterConfig["retain_initial_value_of_delta_metric"] = parseBool(retainValueOfDelta)
	}

	return exporterConfig, nil
}
