package config

import (
	"errors"

	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	commonconf "github.com/keyval-dev/odigos/autoscaler/controllers/common"
	"github.com/keyval-dev/odigos/common"
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

func (g *GoogleCloudStorage) ModifyConfig(dest *odigosv1.Destination, currentConfig *commonconf.Config) error {

	if !isTracingEnabled(dest) && !isLoggingEnabled(dest) {
		return errors.New("GoogleCloudStorage is not enabled for any supported signals, skipping")
	}

	bucket, ok := dest.Spec.Data[gcsBucketKey]
	if !ok {
		log.Log.V(0).Info("GCS bucket not specified, using default bucket %s", defaultGCSBucket)
		bucket = defaultGCSBucket
	}

	exporterName := "googlecloudstorage/" + dest.Name
	currentConfig.Exporters[exporterName] = commonconf.GenericMap{
		"pcs": commonconf.GenericMap{
			"bucket": bucket,
		},
	}

	if isLoggingEnabled(dest) {
		logsPipelineName := "logs/gcs-" + dest.Name
		currentConfig.Service.Pipelines[logsPipelineName] = commonconf.Pipeline{
			Exporters: []string{exporterName},
		}
	}

	if isTracingEnabled(dest) {
		tracesPipelineName := "traces/gcs-" + dest.Name
		currentConfig.Service.Pipelines[tracesPipelineName] = commonconf.Pipeline{
			Exporters: []string{exporterName},
		}
	}

	return nil
}
