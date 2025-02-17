package config

import (
	"encoding/json"
	"errors"

	"github.com/odigos-io/odigos/common"
)

const (
	AWS_CLOUDWATCH_LOG_GROUP_NAME  = "AWS_CLOUDWATCH_LOG_GROUP_NAME"
	AWS_CLOUDWATCH_LOG_STREAM_NAME = "AWS_CLOUDWATCH_LOG_STREAM_NAME"
	AWS_CLOUDWATCH_REGION          = "AWS_CLOUDWATCH_REGION"
	AWS_CLOUDWATCH_ENDPOINT        = "AWS_CLOUDWATCH_ENDPOINT"
	AWS_CLOUDWATCH_LOG_RETENTION   = "AWS_CLOUDWATCH_LOG_RETENTION"
	AWS_CLOUDWATCH_TAGS            = "AWS_CLOUDWATCH_TAGS"
	AWS_CLOUDWATCH_RAW_LOG         = "AWS_CLOUDWATCH_RAW_LOG"
)

type AWSCloudWatch struct{}

func (m *AWSCloudWatch) DestType() common.DestinationType {
	return common.AWSCloudWatchDestinationType
}

func (m *AWSCloudWatch) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) ([]string, error) {
	config := dest.GetConfig()
	uniqueUri := "awscloudwatch-" + dest.GetID()
	var pipelineNames []string

	logGroupName, exists := config[AWS_CLOUDWATCH_LOG_GROUP_NAME]
	if !exists {
		return nil, errorMissingKey(AWS_CLOUDWATCH_LOG_GROUP_NAME)
	}

	logStreamName, exists := config[AWS_CLOUDWATCH_LOG_STREAM_NAME]
	if !exists {
		return nil, errorMissingKey(AWS_CLOUDWATCH_LOG_STREAM_NAME)
	}

	exporterName := "awscloudwatchlogs/" + uniqueUri
	exporterConfig := GenericMap{
		"log_group_name":  logGroupName,
		"log_stream_name": logStreamName,
	}

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
			return nil, errors.Join(err, errors.New("failed to parse awscloudwatch destination AWS_CLOUDWATCH_TAGS parameter as json string in the form {key: string, value: string}[]"))
		}

		mappedTags := map[string]string{}
		for _, tag := range tagList {
			mappedTags[tag.Key] = tag.Value
		}

		exporterConfig["tags"] = mappedTags
	}

	rawLog, exists := config[AWS_CLOUDWATCH_RAW_LOG]
	if exists {
		exporterConfig["raw_log"] = parseBool(rawLog)
	}

	currentConfig.Exporters[exporterName] = exporterConfig

	if isLoggingEnabled(dest) {
		pipeName := "logs/" + uniqueUri
		currentConfig.Service.Pipelines[pipeName] = Pipeline{
			Exporters: []string{exporterName},
		}

		pipelineNames = append(pipelineNames, pipeName)
	}

	return pipelineNames, nil
}
