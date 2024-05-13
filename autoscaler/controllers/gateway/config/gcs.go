package config

import (
	"errors"

	commonconf "github.com/odigos-io/odigos/autoscaler/controllers/common"
	"github.com/odigos-io/odigos/common"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	defaultGCSBucket = "odigos-otlp"
	gcsBucketKey     = "GCS_BUCKET"
)

type GoogleCloudStorage struct{}

func (g *GoogleCloudStorage) DestType() common.DestinationType {
	return common.GCSDestinationType
}

func (g *GoogleCloudStorage) ModifyConfig(dest common.ExporterConfigurer, currentConfig *commonconf.Config) error {

	if !isTracingEnabled(dest) && !isLoggingEnabled(dest) {
		return errors.New("GoogleCloudStorage is not enabled for any supported signals, skipping")
	}

	bucket, ok := dest.GetConfig()[gcsBucketKey]
	if !ok {
		log.Log.V(0).Info("GCS bucket not specified, using default bucket %s", defaultGCSBucket)
		bucket = defaultGCSBucket
	}

	exporterName := "googlecloudstorage/" + dest.GetName()
	currentConfig.Exporters[exporterName] = commonconf.GenericMap{
		"pcs": commonconf.GenericMap{
			"bucket": bucket,
		},
	}

	if isLoggingEnabled(dest) {
		logsPipelineName := "logs/gcs-" + dest.GetName()
		currentConfig.Service.Pipelines[logsPipelineName] = commonconf.Pipeline{
			Exporters: []string{exporterName},
		}
	}

	if isTracingEnabled(dest) {
		tracesPipelineName := "traces/gcs-" + dest.GetName()
		currentConfig.Service.Pipelines[tracesPipelineName] = commonconf.Pipeline{
			Exporters: []string{exporterName},
		}
	}

	return nil
}
