package config

import (
	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	commonconf "github.com/keyval-dev/odigos/autoscaler/controllers/common"
	"github.com/keyval-dev/odigos/common"
)

const (
	defaultGCSBucket = "odigos-otlp"
	gcsBucketKey     = "GCS_BUCKET"
)

type GoogleCloudStorage struct{}

func (g *GoogleCloudStorage) DestType() common.DestinationType {
	return common.GCSDestinationType
}

func (g *GoogleCloudStorage) ModifyConfig(dest *odigosv1.Destination, currentConfig *commonconf.Config) {
	bucket, ok := dest.Spec.Data[gcsBucketKey]
	if !ok {
		bucket = defaultGCSBucket
	}

	if isLoggingEnabled(dest) {
		currentConfig.Exporters["googlecloudstorage"] = commonconf.GenericMap{
			"pcs": commonconf.GenericMap{
				"bucket": bucket,
			},
		}

		currentConfig.Service.Pipelines["logs/gcs"] = commonconf.Pipeline{
			Receivers:  []string{"otlp"},
			Processors: []string{"batch"},
			Exporters:  []string{"googlecloudstorage"},
		}
	}

	if isTracingEnabled(dest) {
		currentConfig.Exporters["googlecloudstorage"] = commonconf.GenericMap{
			"pcs": commonconf.GenericMap{
				"bucket": bucket,
			},
		}

		currentConfig.Service.Pipelines["traces/gcs"] = commonconf.Pipeline{
			Receivers:  []string{"otlp"},
			Processors: []string{"batch"},
			Exporters:  []string{"googlecloudstorage"},
		}
	}
}
