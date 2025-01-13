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

func (g *GoogleCloudStorage) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) error {
	if !isTracingEnabled(dest) && !isLoggingEnabled(dest) {
		return errors.New("GoogleCloudStorage is not enabled for any supported signals, skipping")
	}

	bucket, ok := dest.GetConfig()[gcsBucketKey]
	if !ok {
		bucket = defaultGCSBucket
	}

	exporterName := "googlecloudstorage/" + dest.GetID()
	currentConfig.Exporters[exporterName] = GenericMap{
		"pcs": GenericMap{
			"bucket": bucket,
		},
	}

	if isLoggingEnabled(dest) {
		logsPipelineName := "logs/gcs-" + dest.GetID()
		currentConfig.Service.Pipelines[logsPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
	}

	if isTracingEnabled(dest) {
		tracesPipelineName := "traces/gcs-" + dest.GetID()
		currentConfig.Service.Pipelines[tracesPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
	}

	return nil
}
