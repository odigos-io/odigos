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
	if !IsTracingEnabled(dest) && !IsLoggingEnabled(dest) {
		return nil, errors.New("GoogleCloudStorage is not enabled for any supported signals, skipping")
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
	var pipelineNames []string
	if IsLoggingEnabled(dest) {
		logsPipelineName := "logs/gcs-" + dest.GetID()
		currentConfig.Service.Pipelines[logsPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, logsPipelineName)
	}

	if IsTracingEnabled(dest) {
		tracesPipelineName := "traces/gcs-" + dest.GetID()
		currentConfig.Service.Pipelines[tracesPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, tracesPipelineName)
	}

	return pipelineNames, nil
}
