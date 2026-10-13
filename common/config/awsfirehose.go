package config

import (
	"errors"
	"strings"

	"github.com/odigos-io/odigos/common"
)

const (
	firehoseRoleARNKey    = "FIREHOSE_ROLE_ARN"
	firehoseExternalIDKey = "FIREHOSE_EXTERNAL_ID"
)

type AWSFirehose struct{}

func (f *AWSFirehose) DestType() common.DestinationType {
	return common.AWSFirehoseDestinationType
}

func (f *AWSFirehose) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) ([]string, error) {
	if len(dest.GetSignals()) != 1 || !isLoggingEnabled(dest) {
		return nil, errors.New("Amazon Data Firehose supports logs only")
	}
	for _, key := range []string{AWS_CLOUDWATCH_LOG_GROUP_NAME, AWS_CLOUDWATCH_LOG_STREAM_NAME, AWS_CLOUDWATCH_REGION} {
		if strings.TrimSpace(dest.GetConfig()[key]) == "" {
			return nil, errorMissingKey(key)
		}
	}
	if strings.ContainsAny(dest.GetConfig()[AWS_CLOUDWATCH_LOG_GROUP_NAME], "{}") {
		return nil, errors.New("Amazon Data Firehose requires a fixed CloudWatch log group with a subscription to Firehose")
	}

	exporterConfig, err := sharedConfig(dest)
	if err != nil {
		return nil, err
	}
	exporterConfig["raw_log"] = false
	for key, field := range map[string]string{firehoseRoleARNKey: "role_arn", firehoseExternalIDKey: "external_id"} {
		if value := dest.GetConfig()[key]; value != "" {
			exporterConfig[field] = value
		}
	}
	exporterName := "awscloudwatchlogs/firehose-" + dest.GetID()
	pipelineName := "logs/firehose-" + dest.GetID()
	currentConfig.Exporters[exporterName] = exporterConfig
	currentConfig.Service.Pipelines[pipelineName] = Pipeline{Exporters: []string{exporterName}}
	return []string{pipelineName}, nil
}
