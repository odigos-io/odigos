package config

import (
	"errors"

	"github.com/odigos-io/odigos/common"
)

const (
	defaultGCSBucket = "odigos-otlp"
	gcsBucketKey     = "GCS_BUCKET"
)

type GoogleCloudStorage struct{}

func (g *GoogleCloudStorage) DestType() common.DestinationType {
	return common.GCSDestinationType
}

func (g *GoogleCloudStorage) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) ([]string, error) {
	if !isTracingEnabled(dest) && !isLoggingEnabled(dest) {
		return nil, errors.New("GoogleCloudStorage is not enabled for any supported signals, skipping")
	}

	bucket := dest.GetConfig()[gcsBucketKey]
	if bucket == "" {
		bucket = defaultGCSBucket
	}

	exporterName := "googlecloudstorage/" + dest.GetID()
	uploaderConfig := GenericMap{"bucket": bucket}
	exporterConfig := GenericMap{"gcs": uploaderConfig}
	for field, key := range map[string]string{
		"GCS_PREFIX":      "prefix",
		"GCS_PARTITION":   "partition",
		"GCS_FILE_PREFIX": "file_prefix",
	} {
		if value := dest.GetConfig()[field]; value != "" {
			uploaderConfig[key] = value
		}
	}
	if value := dest.GetConfig()["GCS_MARSHALER"]; value != "" {
		exporterConfig["marshaler_name"] = value
	}
	currentConfig.Exporters[exporterName] = exporterConfig
	var pipelineNames []string
	if isLoggingEnabled(dest) {
		logsPipelineName := "logs/gcs-" + dest.GetID()
		currentConfig.Service.Pipelines[logsPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, logsPipelineName)
	}

	if isTracingEnabled(dest) {
		tracesPipelineName := "traces/gcs-" + dest.GetID()
		currentConfig.Service.Pipelines[tracesPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, tracesPipelineName)
	}

	return pipelineNames, nil
}
